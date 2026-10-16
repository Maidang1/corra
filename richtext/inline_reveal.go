package richtext

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Maidang1/corra/internal/mdext"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// InlineReveal maps native UTF-16 caret boundaries to canonical Markdown.
// The interior boundary of a surrogate pair maps to its leading boundary.
// At adjoining styled fragments the next fragment's content start wins.
type InlineReveal struct {
	Source    string `json:"source"`
	Positions []int  `json:"positions"`
}

type revealAtom struct {
	value      rune
	start, end int // UTF-8 source offsets until the final conversion.
}

type revealSourceSpan struct {
	start, end int
	atoms      []revealAtom
}

// RevealInline uses the storage serializer and its real parsed source spans,
// so link destinations, titles and delimiter text cannot steal a caret match.
// An error leaves callers free to retain the rendered fragment unchanged.
func RevealInline(runs []Run, table bool) (InlineReveal, error) {
	for _, run := range runs {
		if !utf8.ValidString(run.Text) || !utf8.ValidString(run.Math) || !utf8.ValidString(run.Footnote) || !utf8.ValidString(run.Link) || !utf8.ValidString(run.LinkTitle) {
			return InlineReveal{}, fmt.Errorf("invalid UTF-8 inline run")
		}
	}
	source := serializeParagraph(Paragraph{Kind: "body", Runs: runs})
	if table {
		source = serializeCell(runs)
	}
	source, _ = revealPreserveWhitespace(source, table)
	var expected []Run
	for _, run := range runs {
		if run.Source {
			for _, p := range ParseInline(run.Text, table).Paragraphs {
				expected = append(expected, p.Runs...)
			}
		} else {
			expected = append(expected, run)
		}
	}
	preservesStyles := func(source string) bool {
		parsed := ParseInline(source, table)
		return len(parsed.Paragraphs) == 1 && parsed.Paragraphs[0].Kind == "body" && sameRevealRuns(expected, parsed.Paragraphs[0].Runs)
	}
	if !preservesStyles(source) {
		// Markdown flanking rules move literal boundary whitespace outside
		// emphasis. Entity tokens keep that whitespace and its native style.
		source = serializeRuns(revealEncodedRuns(runs, table), escapeReveal)
		if table {
			source = escapeCellPipes(source)
		}
	}
	if !preservesStyles(source) {
		return InlineReveal{}, fmt.Errorf("inline source cannot preserve run semantics")
	}
	spans, err := revealSourceSpans(runs, table, source)
	if err != nil {
		return InlineReveal{}, err
	}
	atoms, err := inlineSourceAtoms(source, table, spans)
	if err != nil {
		return InlineReveal{}, err
	}
	var visible []rune
	for _, run := range runs {
		if run.Math != "" || run.Footnote != "" {
			visible = append(visible, '\ufffc')
		} else if run.Break {
			visible = append(visible, '\u2028')
		} else {
			visible = append(visible, []rune(run.Text)...)
		}
	}
	if len(visible) != len(atoms) {
		return InlineReveal{}, fmt.Errorf("inline source changed visible length (%d != %d)", len(atoms), len(visible))
	}
	offsets := revealUTF16Offsets(source)
	positions := []int{0}
	for i, value := range visible {
		atom := atoms[i]
		if value != atom.value || atom.start < 0 || atom.end > len(source) || atom.end < atom.start {
			return InlineReveal{}, fmt.Errorf("inline source changed visible text at %d", i)
		}
		positions[len(positions)-1] = offsets[atom.start]
		if value > 0xffff {
			positions = append(positions, positions[len(positions)-1])
		}
		positions = append(positions, offsets[atom.end])
	}
	return InlineReveal{Source: source, Positions: positions}, nil
}

func sameRevealRuns(expected, actual []Run) bool {
	nextCharacter := func(runs []Run) func() (Run, bool) {
		offset := 0
		return func() (Run, bool) {
			for len(runs) > 0 {
				run := runs[0]
				if run.Break {
					runs, offset = runs[1:], 0
					return Run{Break: true}, true
				}
				if run.Math != "" || run.Footnote != "" {
					runs, offset = runs[1:], 0
					run.Text = run.Math
					return run, true
				}
				if offset == len(run.Text) {
					runs, offset = runs[1:], 0
					continue
				}
				value, size := utf8.DecodeRuneInString(run.Text[offset:])
				offset += size
				run.Text = string(value)
				return run, true
			}
			return Run{}, false
		}
	}
	wantNext, gotNext := nextCharacter(expected), nextCharacter(actual)
	for {
		want, wantOK := wantNext()
		got, gotOK := gotNext()
		if !wantOK || !gotOK {
			return wantOK == gotOK
		}
		if want != got {
			return false
		}
	}
}

func inlineSourceAtoms(source string, table bool, spans []revealSourceSpan) ([]revealAtom, error) {
	input, prefix := source, 0
	var original []int
	if table {
		// ParseInline protects newly typed pipes before making its temporary
		// table. Keep that projection reversible for source caret positions.
		for i := range len(source) {
			if source[i] == '|' && (i == 0 || source[i-1] != '\\') {
				original = append(original, i)
			}
			original = append(original, i)
		}
		original = append(original, len(source))
		input, prefix = "| "+escapeCellPipes(source)+" |\n| --- |", 2
	}
	src := []byte(input)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, mdext.InlineExtender))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(inlineMathParser{}, 50)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(footnoteInlineParser{}, 49)))
	root := md.Parser().Parse(text.NewReader(src))
	mdext.TransformHTML(root, src)
	var atoms []revealAtom
	add := func(value rune, start, end int) {
		start, end = start-prefix, end-prefix
		if table && start >= 0 && end >= start && end < len(original) {
			start, end = original[start], original[end]
		}
		index := sort.Search(len(spans), func(i int) bool { return spans[i].end > start })
		if index < len(spans) && spans[index].start < end {
			return
		}
		atoms = append(atoms, revealAtom{value, start, end})
	}
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *footnoteInline:
			start := n.Pos()
			if start < 0 || start+len(n.source) > len(src) || string(src[start:start+len(n.source)]) != n.source {
				return ast.WalkStop, fmt.Errorf("footnote source has no exact position")
			}
			add('\ufffc', start, start+len(n.source))
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			segment := n.Segment
			if n.HardLineBreak() && segment.Start == segment.Stop && n.Pos() >= 0 && n.Pos() < len(src) && src[n.Pos()] == '<' {
				end := n.Pos() + strings.IndexByte(string(src[n.Pos():]), '>') + 1
				if end > n.Pos() && isLineBreakTag(string(src[n.Pos():end])) {
					add('\u2028', n.Pos(), end)
					return ast.WalkSkipChildren, nil
				}
			}
			if segment.Padding != 0 || segment.ForceNewline {
				return ast.WalkStop, fmt.Errorf("synthetic inline text has no exact source boundary")
			}
			for _, atom := range revealTextAtoms(src, segment.Start, segment.Stop, n.IsRaw()) {
				add(atom.value, atom.start, atom.end)
			}
			if n.HardLineBreak() || n.SoftLineBreak() {
				end := segment.Stop
				for end < len(src) && src[end] != '\n' {
					end++
				}
				if end < len(src) {
					end++
				}
				add('\u2028', segment.Stop, end)
			}
		case *inlineMath:
			start := n.Pos()
			if start < 0 || start+len(n.source) > len(src) || string(src[start:start+len(n.source)]) != n.source {
				return ast.WalkStop, fmt.Errorf("math source has no exact position")
			}
			delim := 1
			if strings.HasPrefix(n.source, "$$") {
				delim = 2
			}
			add('\ufffc', start+delim, start+len(n.source)-delim)
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			start := n.Pos()
			if start >= 0 && start < len(src) && strings.ContainsRune(" <*_~(", rune(src[start])) {
				start++
			}
			label := n.Label(src)
			if start < 0 || start+len(label) > len(src) || string(src[start:start+len(label)]) != string(label) {
				return ast.WalkStop, fmt.Errorf("autolink has no exact source position")
			}
			for _, atom := range revealTextAtoms(src, start, start+len(label), true) {
				add(atom.value, atom.start, atom.end)
			}
			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			if n.Segments.Len() == 1 {
				seg := n.Segments.At(0)
				if isLineBreakTag(string(seg.Value(src))) {
					add('\u2028', seg.Start, seg.Stop)
				}
			}
		case *ast.String:
			return ast.WalkStop, fmt.Errorf("generated inline text has no exact source position")
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	for _, span := range spans {
		atoms = append(atoms, span.atoms...)
	}
	if len(spans) > 0 {
		sort.SliceStable(atoms, func(i, j int) bool { return atoms[i].start < atoms[j].start })
	}
	return atoms, nil
}

// Escapes and entities each form one source token, even when their decoded
// text has several Unicode scalars. Every returned boundary stays in UTF-8.
func revealTextAtoms(src []byte, start, end int, raw bool) []revealAtom {
	var out []revealAtom
	for i := start; i < end; {
		stop := i
		var value string
		if raw && src[i] == '\r' && i+1 < end && src[i+1] == '\n' {
			stop, value = i+2, " "
		} else if !raw && src[i] == '\\' && i+1 < end && util.IsPunct(src[i+1]) {
			stop, value = i+2, string(src[i+1])
		} else if !raw && src[i] == '&' {
			if size := strings.IndexByte(string(src[i:min(end, i+40)]), ';'); size > 0 && size < 40 {
				token := src[i : i+size+1]
				decoded := decodeText(token)
				if string(decoded) != string(token) {
					stop, value = i+size+1, string(decoded)
				}
			}
		}
		if stop == i {
			r, size := utf8.DecodeRune(src[i:end])
			stop, value = i+size, string(r)
			if raw && r == '\n' {
				value = " "
			}
		}
		for _, r := range value {
			out = append(out, revealAtom{r, i, stop})
		}
		i = stop
	}
	return out
}

// Only expanded source runs need serializer emission ranges: their native
// characters include the delimiters that the Markdown AST hides.
func revealSourceSpans(runs []Run, table bool, source string) ([]revealSourceSpan, error) {
	hasSource := false
	for _, run := range runs {
		hasSource = hasSource || run.Source
	}
	if !hasSource {
		return nil, nil
	}
	runs = revealEncodedRuns(runs, table)
	levels := []escapeLevel{escapeNone, escapeInline, escapeAll, escapeEntities, escapeReveal}
	if table {
		levels = levels[1:]
	}
	for _, level := range levels {
		var spans []revealSourceSpan
		raw := serializeRunsObserved(runs, level, func(i, start, end int) {
			run := runs[i]
			if run.Source {
				span := revealSourceSpan{start: start, end: end}
				for offset, r := range run.Text {
					span.atoms = append(span.atoms, revealAtom{r, start + offset, start + offset + utf8.RuneLen(r)})
				}
				spans = append(spans, span)
			}
		})
		value, boundaries := revealTransform(raw, table, level != escapeReveal)
		if level != escapeReveal {
			var whitespaceBoundaries []int
			value, whitespaceBoundaries = revealPreserveWhitespace(value, table)
			for i := range boundaries {
				boundaries[i] = whitespaceBoundaries[boundaries[i]]
			}
		}
		if value != source {
			continue
		}
		for i := range spans {
			spans[i].start, spans[i].end = boundaries[spans[i].start], boundaries[spans[i].end]
			for j := range spans[i].atoms {
				atom := &spans[i].atoms[j]
				atom.start, atom.end = boundaries[atom.start], boundaries[atom.end]
			}
		}
		return spans, nil
	}
	return nil, fmt.Errorf("source runs have no canonical emission ranges")
}

func revealEncodedRuns(runs []Run, table bool) []Run {
	normalized := Paragraph{}
	for _, run := range runs {
		if table && run.Math != "" {
			delim := "$"
			if strings.HasPrefix(run.Text, "$$") {
				delim = "$$"
			}
			run.Text = delim + strings.ReplaceAll(run.Math, "|", `\|`) + delim
		}
		appendRun(&normalized, run)
	}
	runs = append([]Run(nil), normalized.Runs...)
	if table {
		for i := range runs {
			if runs[i].Code && runs[i].Math == "" && !runs[i].Source {
				runs[i].Text = strings.ReplaceAll(runs[i].Text, "|", `\|`)
			}
		}
	}
	return runs
}

func revealTransform(value string, table bool, protect bool) (string, []int) {
	var out strings.Builder
	boundaries := make([]int, len(value)+1)
	leading := true
	for offset, r := range value {
		boundaries[offset] = out.Len()
		if table && r == '|' && (offset == 0 || value[offset-1] != '\\') {
			out.WriteByte('\\')
		}
		if protect && !table && leading && (r == ' ' || r == '\t') {
			fmt.Fprintf(&out, "&#%d;", r)
		} else {
			out.WriteRune(r)
			leading = r == '\n'
		}
		boundaries[offset+utf8.RuneLen(r)] = out.Len()
	}
	return out.String(), boundaries
}

// The storage serializer intentionally ignores whitespace style differences.
// Revealing cannot lose actual native characters to Markdown's line/cell trim.
func revealPreserveWhitespace(value string, table bool) (string, []int) {
	encode := make([]bool, len(value))
	for lineStart := 0; lineStart < len(value); {
		end := strings.IndexByte(value[lineStart:], '\n')
		if end < 0 {
			end = len(value)
		} else {
			end += lineStart
		}
		trimEnd := end
		if end < len(value) && end-lineStart >= 2 && value[end-1] == ' ' && value[end-2] == ' ' {
			trimEnd -= 2
		}
		for i := trimEnd - 1; i >= lineStart && (value[i] == ' ' || value[i] == '\t'); i-- {
			encode[i] = true
		}
		if table {
			for i := lineStart; i < end && (value[i] == ' ' || value[i] == '\t'); i++ {
				encode[i] = true
			}
		}
		lineStart = end + 1
	}
	var out strings.Builder
	boundaries := make([]int, len(value)+1)
	for i := range len(value) {
		boundaries[i] = out.Len()
		if encode[i] {
			fmt.Fprintf(&out, "&#%d;", value[i])
		} else {
			out.WriteByte(value[i])
		}
	}
	boundaries[len(value)] = out.Len()
	return out.String(), boundaries
}

func revealUTF16Offsets(source string) []int {
	positions := make([]int, len(source)+1)
	offset := 0
	for i, r := range source {
		for j := range utf8.RuneLen(r) {
			positions[i+j] = offset
		}
		offset++
		if r > 0xffff {
			offset++
		}
		positions[i+utf8.RuneLen(r)] = offset
	}
	return positions
}

// InlineCaretMap maps UTF-16 boundaries in actual edited Markdown to native
// rendered boundaries. Delimiters and link metadata stay at the corresponding
// content edge. An interior surrogate boundary snaps to its leading edge.
func InlineCaretMap(source string, table bool) ([]int, error) {
	if !utf8.ValidString(source) {
		return nil, fmt.Errorf("invalid UTF-8 inline source")
	}
	doc := ParseInline(source, table)
	if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "body" {
		return nil, fmt.Errorf("source is not one editable inline paragraph")
	}
	atoms, err := inlineSourceAtoms(source, table, nil)
	if err != nil {
		return nil, err
	}
	var visible []rune
	for _, run := range doc.Paragraphs[0].Runs {
		switch {
		case run.Math != "" || run.Footnote != "":
			visible = append(visible, '\ufffc')
		case run.Break:
			visible = append(visible, '\u2028')
		default:
			visible = append(visible, []rune(run.Text)...)
		}
	}
	if len(atoms) != len(visible) {
		return nil, fmt.Errorf("parsed inline projection has a different length")
	}
	for i, atom := range atoms {
		if atom.value != visible[i] || atom.start < 0 || atom.end < atom.start || atom.end > len(source) {
			return nil, fmt.Errorf("parsed inline projection differs at %d", i)
		}
	}
	offsets := revealUTF16Offsets(source)
	positions := make([]int, offsets[len(source)]+1)
	rendered, index := 0, 0
	for s := range positions {
		for index < len(atoms) && offsets[atoms[index].end] <= s {
			rendered++
			if atoms[index].value > 0xffff {
				rendered++
			}
			index++
		}
		positions[s] = rendered
		if index == len(atoms) {
			continue
		}
		atom := atoms[index]
		start, end := offsets[atom.start], offsets[atom.end]
		if s > start && s < end && s-start > end-s {
			positions[s]++
			if atom.value > 0xffff {
				positions[s]++
			}
		}
	}
	return positions, nil
}
