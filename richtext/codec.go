// Package richtext bridges Markdown storage and a deliberately conservative
// native rich-text model. Unsupported syntax remains a protected source block.
package richtext

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/mdext"
)

type Document struct {
	Paragraphs []Paragraph `json:"paragraphs"`
}

type Paragraph struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Level   int    `json:"level,omitempty"`
	Checked bool   `json:"checked,omitempty"`
	Ordinal int    `json:"ordinal,omitempty"`
	Depth   int    `json:"depth,omitempty"`
	// Alert is the GitHub callout label ("NOTE", "TIP", ...) of a quote whose
	// first line is "[!LABEL]". The marker itself stays in Raw and Runs, so an
	// untouched alert still round-trips byte for byte.
	Alert string `json:"alert,omitempty"`
	// List metadata preserves original indentation until the tree is changed.
	ListIndent       string   `json:"listIndent,omitempty"`
	ListID           string   `json:"listId,omitempty"`
	ListContainer    string   `json:"listContainer,omitempty"`
	ListMarker       string   `json:"listMarker,omitempty"`
	ListBaseline     string   `json:"listBaseline,omitempty"`
	ListItem         string   `json:"listItem,omitempty"`
	ListContinuation bool     `json:"listContinuation,omitempty"`
	Runs             []Run    `json:"runs"`
	Raw              string   `json:"raw,omitempty"`
	RawBase64        string   `json:"rawBase64,omitempty"`
	Prefix           string   `json:"prefix,omitempty"`
	Separator        string   `json:"separator,omitempty"`
	Baseline         string   `json:"baseline,omitempty"`
	NextID           string   `json:"nextId,omitempty"`
	Rows             [][]Cell `json:"rows,omitempty"`
	Alignments       []string `json:"alignments,omitempty"`
	// Open and Close hold a code block's fences so the lines between them can
	// be edited without the fences on screen (Typora-style).
	Open          string `json:"open,omitempty"`
	Close         string `json:"close,omitempty"`
	CodeNewline   string `json:"codeNewline,omitempty"`
	CodeClose     string `json:"codeClose,omitempty"`
	CodeSeparator string `json:"codeSeparator,omitempty"`
	// Definitions retain their inner kind. Group distinguishes repeated labels;
	// marker and indent on every child recover the container after deletions.
	FootnoteID           string `json:"footnoteId,omitempty"`
	FootnoteGroup        string `json:"footnoteGroup,omitempty"`
	FootnoteContinuation bool   `json:"footnoteContinuation,omitempty"`
	FootnoteMarker       string `json:"footnoteMarker,omitempty"`
	FootnoteIndent       string `json:"footnoteIndent,omitempty"`
	FootnoteRaw          string `json:"footnoteRaw,omitempty"`
	FootnoteBaseline     string `json:"footnoteBaseline,omitempty"`
}

type Cell struct {
	Runs []Run `json:"runs"`
}

type Run struct {
	Text string `json:"text"`
	// Source is an expanded inline fragment being edited in the native view.
	// Its live text is Markdown source, rather than text to escape.
	Source    bool   `json:"source,omitempty"`
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
	Code      bool   `json:"code,omitempty"`
	Link      string `json:"link,omitempty"`
	LinkTitle string `json:"linkTitle,omitempty"`
	Underline bool   `json:"underline,omitempty"` // <u>x</u>
	Kbd       bool   `json:"kbd,omitempty"`       // <kbd>x</kbd>
	Highlight bool   `json:"highlight,omitempty"` // ==x== or <mark>
	Sup       bool   `json:"sup,omitempty"`       // ^x^ or <sup>
	Sub       bool   `json:"sub,omitempty"`       // ~x~ or <sub>
	// Math holds an inline math expression; Text keeps its exact $...$ source.
	Math string `json:"math,omitempty"`
	// Footnote is the reference label; Text retains its exact [^label] token.
	Footnote string `json:"footnote,omitempty"`
	// Break is a hard line break inside the paragraph (two trailing spaces, a
	// trailing backslash, or <br>). It carries no text and renders as a soft
	// visual line that keeps the paragraph's block identity.
	Break bool `json:"break,omitempty"`
	// Soft marks a Break that was a plain source newline: it shows as its own
	// line and saves back as a bare newline instead of a hard break.
	Soft bool `json:"soft,omitempty"`
}

// Parse retains each block's exact source and boundary whitespace. Baseline is
// an opaque semantic fingerprint; native clients must preserve all metadata.
func Parse(source string) Document {
	fm, rest := mdext.SplitFrontMatter(source)
	if fm != "" {
		// Front matter is a protected source block ahead of the body. The
		// blank lines between the closing fence and the body fold into the
		// fence line's separator so serialization stays byte-exact.
		nl := "\n"
		if strings.Contains(fm, "\r\n") {
			nl = "\r\n"
		}
		sep := nl
		for rest != "" {
			if strings.HasPrefix(rest, "\r\n") {
				sep += "\r\n"
				rest = rest[2:]
			} else if rest[0] == '\n' {
				sep += "\n"
				rest = rest[1:]
			} else {
				break
			}
		}
		paragraphs := sourceLines(strings.TrimSuffix(strings.TrimSuffix(fm, "\n"), "\r"))
		paragraphs[len(paragraphs)-1].Separator = sep
		body := Parse(rest)
		paragraphs = append(paragraphs, body.Paragraphs...)
		return identifyParagraphs(Document{Paragraphs: paragraphs})
	}
	src := []byte(source)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, mdext.InlineExtender))
	md.Parser().AddOptions(parser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 50)))
	md.Parser().AddOptions(parser.WithBlockParsers(util.Prioritized(footnoteBlockParser{}, 49)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(inlineMathParser{}, 50)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(footnoteInlineParser{}, 49)))
	root := md.Parser().Parse(text.NewReader(src))
	mdext.TransformHTML(root, src)
	type block struct {
		node         ast.Node
		start        int
		kind         string
		ordinal      int
		depth        int
		listID       string
		container    string
		marker       string
		item         string
		continuation bool
	}
	var blocks []block
	var flattenList func(*ast.List, int, string)
	flattenList = func(list *ast.List, depth int, id string) {
		ordinal := list.Start
		for item := list.FirstChild(); item != nil; item = item.NextSibling() {
			kind := "bullet"
			if list.IsOrdered() {
				kind = "ordered"
			}
			if isTask(item) {
				kind = "checklist"
			}
			itemID := fmt.Sprintf("item%d", item.Pos())
			first := true
			if item.FirstChild() == nil {
				blocks = append(blocks, block{node: item, start: lineStart(source, item.Pos()), kind: kind, ordinal: ordinal, depth: depth, listID: id, container: fmt.Sprintf("list%d", list.Pos()), marker: string(list.Marker), item: itemID})
			}
			for child := item.FirstChild(); child != nil; child = child.NextSibling() {
				if nested, ok := child.(*ast.List); ok {
					flattenList(nested, depth+1, id)
				} else {
					blocks = append(blocks, block{node: child, start: lineStart(source, child.Pos()), kind: kind, ordinal: ordinal, depth: depth, listID: id, container: fmt.Sprintf("list%d", list.Pos()), marker: string(list.Marker), item: itemID, continuation: !first})
					first = false
				}
			}
			ordinal++
		}
	}
	for node := root.FirstChild(); node != nil; node = node.NextSibling() {
		if list, ok := node.(*ast.List); ok && editableList(list, src) {
			flattenList(list, 0, fmt.Sprintf("list%d", node.Pos()))
		} else {
			blocks = append(blocks, block{node: node, start: lineStart(source, node.Pos())})
		}
	}
	doc := Document{Paragraphs: make([]Paragraph, 0, len(blocks))}
	if len(blocks) == 0 {
		for _, line := range strings.SplitAfter(source, "\n") {
			raw, separator := line, ""
			if strings.HasSuffix(line, "\n") {
				raw, separator = strings.TrimSuffix(line, "\n"), "\n"
			}
			doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "body", Runs: []Run{}, Raw: raw, Separator: separator})
		}
		return identifyParagraphs(doc)
	}
	for i, b := range blocks {
		end := len(source)
		if i+1 < len(blocks) {
			end = blocks[i+1].start
		}
		if b.start < 0 || b.start > end || end > len(source) {
			return identifyParagraphs(Document{Paragraphs: sourceLines(source)})
		}
		raw, separator := splitSeparator(source[b.start:end])
		if !utf8.ValidString(raw) {
			p := opaque(base64.StdEncoding.EncodeToString([]byte(raw)), "Invalid UTF-8 source")
			p.Separator = separator
			if i == 0 {
				p.Prefix = source[:b.start]
			}
			doc.Paragraphs = append(doc.Paragraphs, p)
			continue
		}
		ps := convertBlock(b.node, src, raw)
		if b.kind != "" && len(ps) == 1 && ps[0].Kind != "opaque" && ps[0].Kind != "source" {
			ps[0].Kind = b.kind
			if ps[0].Checked || isTask(b.node) {
				ps[0].Kind = "checklist"
			}
			if b.marker == "." || b.marker == ")" {
				ps[0].Ordinal = b.ordinal
			}
			ps[0].Depth, ps[0].ListID = b.depth, b.listID
			ps[0].ListContainer, ps[0].ListMarker = b.container, b.marker
			ps[0].ListItem, ps[0].ListContinuation = b.item, b.continuation
			ps[0].ListIndent = raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
			ps[0].ListBaseline = listFingerprint(ps[0])
		}
		if len(ps) == 1 && ps[0].Kind != "code" && ps[0].FootnoteID == "" {
			// codeLines already set Raw per content line.
			ps[0].Raw = raw
		}
		ps[len(ps)-1].Separator = separator
		if i == 0 {
			ps[0].Prefix = source[:b.start]
		}
		if !utf8.ValidString(ps[0].Prefix) || !utf8.ValidString(ps[len(ps)-1].Separator) {
			p := opaque(base64.StdEncoding.EncodeToString([]byte(source)), "Invalid UTF-8 source")
			return identifyParagraphs(Document{Paragraphs: []Paragraph{p}})
		}
		doc.Paragraphs = append(doc.Paragraphs, ps...)
	}
	return expandBlankParagraphs(doc)
}

func identifyParagraphs(doc Document) Document {
	for i := range doc.Paragraphs {
		p := &doc.Paragraphs[i]
		p.ID, p.NextID = fmt.Sprintf("p%d", i), ""
		if i+1 < len(doc.Paragraphs) {
			p.NextID = fmt.Sprintf("p%d", i+1)
		}
		p.Baseline = fingerprint(*p)
	}
	for i := 0; i < len(doc.Paragraphs); {
		p := &doc.Paragraphs[i]
		end := i + 1
		if p.FootnoteID != "" {
			for end < len(doc.Paragraphs) && doc.Paragraphs[end].FootnoteGroup == p.FootnoteGroup {
				end++
			}
			p.FootnoteBaseline = footnoteFingerprint(doc.Paragraphs[i:end])
		}
		i = end
	}
	return doc
}

func blankLine(line string) Paragraph {
	raw, separator := strings.TrimSuffix(line, "\n"), "\n"
	if strings.HasSuffix(raw, "\r") {
		raw, separator = strings.TrimSuffix(raw, "\r"), "\r\n"
	}
	return Paragraph{Kind: "body", Runs: []Run{}, Raw: raw, Separator: separator}
}

func expandBlankParagraphs(doc Document) Document {
	out := Document{}
	for i, p := range doc.Paragraphs {
		if i == 0 {
			lines := strings.SplitAfter(p.Prefix, "\n")
			for _, line := range lines[:len(lines)-1] {
				out.Paragraphs = append(out.Paragraphs, blankLine(line))
			}
			p.Prefix = lines[len(lines)-1]
		}
		lines := strings.SplitAfter(p.Separator, "\n")
		keep := 2
		if i+1 == len(doc.Paragraphs) {
			keep = 1
		}
		if len(lines)-1 <= keep {
			out.Paragraphs = append(out.Paragraphs, p)
			continue
		}
		p.Separator = strings.Join(lines[:keep], "")
		out.Paragraphs = append(out.Paragraphs, p)
		for _, line := range lines[keep : len(lines)-1] {
			out.Paragraphs = append(out.Paragraphs, blankLine(line))
		}
		out.Paragraphs[len(out.Paragraphs)-1].Separator += lines[len(lines)-1]
	}
	// Each line keeps the hidden ending metadata so native paragraph joins and
	// IME replacement can remove the last line without losing the fence.
	for start := 0; start < len(out.Paragraphs); {
		if out.Paragraphs[start].Kind != "code" {
			start++
			continue
		}
		end := start + 1
		for end < len(out.Paragraphs) && out.Paragraphs[end].Kind == "code" && out.Paragraphs[end].Open == "" {
			end++
		}
		last := out.Paragraphs[end-1]
		for i := start; i < end; i++ {
			out.Paragraphs[i].CodeClose = last.Close
			out.Paragraphs[i].CodeSeparator = last.Separator
		}
		start = end
	}
	return identifyParagraphs(out)
}

func lineStart(source string, pos int) int {
	if pos < 0 || pos > len(source) {
		return -1
	}
	return strings.LastIndexByte(source[:pos], '\n') + 1
}

func splitSeparator(source string) (string, string) {
	end := len(source)
	for end > 0 {
		start := strings.LastIndexByte(source[:end], '\n') + 1
		if strings.Trim(source[start:end], " \t\r") != "" {
			return source[:end], source[end:]
		}
		if start == 0 {
			return "", source
		}
		end = start - 1
		if end > 0 && source[end-1] == '\r' {
			end--
		}
	}
	return "", source
}

func editableList(list *ast.List, src []byte) bool {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		if item.ChildCount() == 0 {
			continue // an empty item ("- ") is an empty editable paragraph
		}
		child := item.FirstChild()
		if child.Kind() != ast.KindTextBlock && child.Kind() != ast.KindParagraph {
			return false
		}
		p := Paragraph{}
		if !collectRuns(child, src, Run{}, &p) {
			return false
		}
		for child = child.NextSibling(); child != nil; child = child.NextSibling() {
			nested, ok := child.(*ast.List)
			if ok {
				if !editableList(nested, src) {
					return false
				}
				continue
			}
			if child.Kind() != ast.KindParagraph && child.Kind() != ast.KindTextBlock {
				return false
			}
			p := Paragraph{}
			if !collectRuns(child, src, Run{}, &p) {
				return false
			}
		}
	}
	return true
}

func isTask(node ast.Node) bool {
	if node.Kind() == ast.KindListItem {
		node = node.FirstChild()
	}
	if node == nil {
		return false
	}
	_, ok := node.FirstChild().(*east.TaskCheckBox)
	return ok
}

// opaque marks the rare block a text view cannot edit safely (invalid UTF-8).
func opaque(rawBase64, label string) Paragraph {
	return Paragraph{Kind: "opaque", Runs: []Run{{Text: "[" + label + " - edit in source mode]"}}, RawBase64: rawBase64}
}

// sourceLines turns an unsupported block into editable source lines: one
// paragraph per line, verbatim in, verbatim out. Unedited lines round-trip
// byte-exact through Raw and Baseline; edited lines serialize as PlainText.
func sourceLines(raw string) []Paragraph {
	if raw == "" {
		return []Paragraph{{Kind: "source", Runs: []Run{{Text: ""}}}}
	}
	lines := strings.Split(raw, "\n")
	out := make([]Paragraph, 0, len(lines))
	for _, line := range lines {
		text, separator := line, "\n"
		if strings.HasSuffix(text, "\r") {
			text, separator = strings.TrimSuffix(text, "\r"), "\r\n"
		}
		out = append(out, Paragraph{Kind: "source", Runs: []Run{{Text: text}}, Raw: text, Separator: separator})
	}
	out[len(out)-1].Separator = ""
	return out
}

// codeLines splits a fenced code block into editable lines: the fences live
// in Open and Close, only the code is text. An unclosed block stays plain
// source lines so its bytes round-trip exactly.
func codeLines(raw string) []Paragraph {
	nl := "\n"
	if strings.Contains(raw, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	open, body := lines[0], lines[1:]
	fence := "```"
	if strings.HasPrefix(strings.TrimSpace(open), "~~~") {
		fence = "~~~"
	}
	close := ""
	if n := len(body); n > 0 && strings.HasPrefix(strings.TrimSpace(body[n-1]), fence) {
		close = body[n-1]
		body = body[:n-1]
	} else {
		return sourceLines(raw)
	}
	if len(body) == 0 {
		body = []string{""}
	}
	out := make([]Paragraph, len(body))
	for i, line := range body {
		out[i] = Paragraph{Kind: "code", Runs: []Run{{Text: line}}, Raw: line, Separator: nl, CodeNewline: nl}
	}
	out[0].Open = open
	out[len(out)-1].Close = close
	out[len(out)-1].Separator = ""
	return out
}

// fenceLanguage returns the lowercased info string of a fenced code block.
func fenceLanguage(n *ast.FencedCodeBlock, src []byte) string {
	if n.Info == nil {
		return ""
	}
	info := string(n.Info.Segment.Value(src))
	return strings.ToLower(strings.TrimSpace(info))
}

// loneImage recognizes a paragraph holding nothing but one image.
func loneImage(node ast.Node, raw string) *Paragraph {
	if node.ChildCount() != 1 {
		// goldmark rejects a destination holding a space unless it is wrapped
		// in angle brackets, so a Typora size suffix arrives as a run of plain
		// text. Read the figure off the source instead; Raw stays byte for byte
		// what the document holds either way.
		if _, _, _, ok := mdext.SizedImage(raw); ok {
			return &Paragraph{Kind: "image", Raw: raw, Runs: []Run{{Text: raw}}}
		}
		return nil
	}
	if _, ok := node.FirstChild().(*ast.Image); !ok {
		return nil
	}
	return &Paragraph{Kind: "image", Raw: raw, Runs: []Run{{Text: raw}}}
}

func convertBlock(node ast.Node, src []byte, raw string) []Paragraph {
	p := Paragraph{Kind: "body", Runs: []Run{}}
	switch n := node.(type) {
	case *ast.Heading:
		p.Kind, p.Level = "heading", n.Level
	case *ast.Paragraph, *ast.TextBlock:
		if img := loneImage(node, raw); img != nil {
			return []Paragraph{*img}
		}
		if mdext.IsTOC(raw) {
			// The marker is one paragraph that renders the document's headings,
			// so it keeps its own kind like math and diagrams do.
			return []Paragraph{{Kind: "toc", Raw: raw, Runs: []Run{{Text: raw}}}}
		}
	case *ast.ListItem:
		if n.FirstChild() == nil {
			return []Paragraph{p}
		}
		node = n.FirstChild()
	case *ast.Blockquote:
		return quoteParagraphs(n, src, raw)
	case *ast.FencedCodeBlock:
		if lang := fenceLanguage(n, src); lang == "chart" || lang == "mermaid" || lang == "d2" || lang == "goat" || lang == "flow" || lang == "sequence" {
			// A diagram/chart fence renders as a figure (like math): the raw
			// fence stays for click-to-edit and byte-exact round-trips.
			return []Paragraph{{Kind: "diagram", Raw: raw, Runs: []Run{{Text: lang}}}}
		}
		return codeLines(raw)
	case *ast.CodeBlock:
		return sourceLines(raw)
	case *mathBlock:
		return []Paragraph{{Kind: "math", Raw: raw, Runs: []Run{{Text: strings.Join(strings.Fields(raw), " ")}}}}
	case *footnoteBlock:
		return footnoteParagraphs(n, raw)
	case *ast.List:
		return sourceLines(raw)
	case *east.Table:
		return convertTable(n, src, raw)
	case *ast.HTMLBlock:
		return sourceLines(raw)
	case *ast.LinkReferenceDefinition:
		return sourceLines(raw)
	default:
		return sourceLines(raw)
	}
	if !collectRuns(node, src, Run{}, &p) {
		return sourceLines(raw)
	}
	return []Paragraph{p}
}

func styleWithText(style Run, text string) Run {
	style.Text = text
	return style
}

func containsMath(raw string) bool {
	escaped := false
	for _, r := range raw {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '$' {
			return true
		}
	}
	return false
}

func collectRuns(parent ast.Node, src []byte, style Run, p *Paragraph) bool {
	// Coalesce plain text split by unmatched Markdown delimiter scanners.
	var pending strings.Builder
	flush := func() {
		if pending.Len() > 0 {
			appendRun(p, styleWithText(style, pending.String()))
			pending.Reset()
		}
	}
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		run := style
		switch n := node.(type) {
		case *footnoteInline:
			flush()
			run.Text, run.Footnote = n.source, n.label
			appendRun(p, run)
		case *inlineMath:
			flush()
			run.Text, run.Math = n.source, n.tex
			appendRun(p, run)
		case *ast.Text:
			content := n.Segment.Value(src)
			if !n.IsRaw() {
				content = decodeText(content)
			}
			pending.Write(content)
			if n.HardLineBreak() {
				flush()
				p.Runs = append(p.Runs, Run{Break: true})
			} else if n.SoftLineBreak() {
				flush()
				p.Runs = append(p.Runs, Run{Break: true, Soft: true})
			}
		case *ast.String:
			pending.Write(n.Value)
		case *ast.Emphasis:
			flush()
			if n.Level == 2 {
				run.Bold = true
			} else {
				run.Italic = true
			}
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *east.Strikethrough:
			flush()
			run.Strike = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *ast.CodeSpan:
			flush()
			run.Code = true
			var value strings.Builder
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				childText, ok := child.(*ast.Text)
				if !ok {
					return false
				}
				value.Write(childText.Segment.Value(src))
			}
			run.Text = strings.ReplaceAll(strings.ReplaceAll(value.String(), "\r\n", " "), "\n", " ")
			appendRun(p, run)
		case *ast.Link:
			flush()
			run.Link = string(decodeText(n.Destination))
			run.LinkTitle = string(decodeText(n.Title))
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *east.TaskCheckBox:
			p.Checked = n.IsChecked
		case *mdext.Underline:
			flush()
			run.Underline = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *mdext.Mark:
			flush()
			run.Highlight = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *mdext.Superscript:
			flush()
			run.Sup = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *mdext.Subscript:
			flush()
			run.Sub = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *mdext.Kbd:
			flush()
			run.Kbd = true
			if !collectRuns(node, src, run, p) {
				return false
			}
		case *ast.AutoLink:
			// Bare URLs (GFM linkify) and <url> stay ordinary rich text; the
			// link survives because the URL itself re-links on parse.
			label, url := string(n.Label(src)), string(n.URL(src))
			if n.AutoLinkType == ast.AutoLinkEmail {
				url = "mailto:" + label
			} else if url != label && "http://"+label != url {
				return false
			}
			flush()
			run.Link = url
			run.Text = label
			appendRun(p, run)
		case *ast.RawHTML:
			var tag strings.Builder
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				tag.Write(seg.Value(src))
			}
			if !isLineBreakTag(tag.String()) {
				return false
			}
			flush()
			p.Runs = append(p.Runs, Run{Break: true})
		default:
			return false
		}
	}
	flush()
	return true
}

// isLineBreakTag reports whether raw HTML is a <br> in any spelling.
func isLineBreakTag(tag string) bool {
	t := strings.ToLower(strings.TrimSpace(tag))
	t = strings.TrimPrefix(t, "<")
	t = strings.TrimSuffix(t, ">")
	t = strings.TrimSuffix(t, "/")
	return strings.TrimSpace(t) == "br"
}

func decodeText(source []byte) []byte {
	var out strings.Builder
	for i := 0; i < len(source); i++ {
		if source[i] == '\\' && i+1 < len(source) && util.IsPunct(source[i+1]) {
			i++
			out.WriteByte(source[i])
			continue
		}
		if source[i] == '&' {
			if end := strings.IndexByte(string(source[i:]), ';'); end > 0 && end < 40 {
				entity := source[i : i+end+1]
				decoded := util.ResolveEntityNames(util.ResolveNumericReferences(entity))
				if string(decoded) != string(entity) {
					out.Write(decoded)
					i += end
					continue
				}
			}
		}
		out.WriteByte(source[i])
	}
	return []byte(out.String())
}

func appendRun(p *Paragraph, run Run) {
	if run.Break {
		p.Runs = append(p.Runs, Run{Break: true, Soft: run.Soft})
		return
	}
	if run.Text == "" {
		return
	}
	if len(p.Runs) != 0 && run.Footnote == "" {
		last := &p.Runs[len(p.Runs)-1]
		other := *last
		other.Text = run.Text
		if other == run {
			last.Text += run.Text
			return
		}
	}
	p.Runs = append(p.Runs, run)
}

func fingerprint(p Paragraph) string {
	p.ID, p.Raw, p.RawBase64, p.Prefix, p.Separator, p.Baseline, p.NextID = "", "", "", "", "", "", ""
	p.ListIndent, p.ListID, p.ListBaseline = "", "", ""
	p.ListContainer, p.ListMarker = "", ""
	p.ListItem = ""
	p.FootnoteRaw, p.FootnoteBaseline = "", ""
	normalized := Paragraph{Runs: []Run{}}
	for _, run := range p.Runs {
		appendRun(&normalized, run)
	}
	p.Runs = normalized.Runs
	if p.Kind == "table" {
		rows := make([][]Cell, len(p.Rows))
		for rowIndex, row := range p.Rows {
			rows[rowIndex] = make([]Cell, len(row))
			for column, cell := range row {
				normalized := Paragraph{Runs: []Run{}}
				for _, run := range cell.Runs {
					appendRun(&normalized, run)
				}
				rows[rowIndex][column] = Cell{Runs: normalized.Runs}
			}
		}
		p.Rows = rows
	}
	data, _ := json.Marshal(p)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Serialize reuses original blocks when possible. Opaque blocks are protected
// even if a native text system accidentally changes their visible label.
func serializeDocument(doc Document) string {
	var out strings.Builder
	lists := listSources(doc)
	for i, p := range doc.Paragraphs {
		out.WriteString(p.Prefix)
		if line, ok := lists[i]; ok {
			out.WriteString(line)
		} else if p.Kind == "code" {
			out.WriteString(codeLineSource(doc, i))
		} else if p.Kind == "opaque" || (p.Baseline != "" && p.Baseline == fingerprint(p)) {
			raw := p.Raw
			if p.RawBase64 != "" {
				if decoded, err := base64.StdEncoding.DecodeString(p.RawBase64); err == nil {
					raw = string(decoded)
				}
			}
			out.WriteString(raw)
		} else {
			out.WriteString(serializeParagraph(p))
		}
		separator := p.Separator
		if p.Kind == "code" && p.CodeClose != "" && (i+1 == len(doc.Paragraphs) || doc.Paragraphs[i+1].Kind != "code" || doc.Paragraphs[i+1].Open != "") {
			separator = p.CodeSeparator
		}
		if emptyParagraph(p) && p.Baseline != fingerprint(p) && i+1 < len(doc.Paragraphs) {
			separator = newlineStyle(separator)
		}
		if i+1 == len(doc.Paragraphs) && p.NextID != "" && strings.Count(separator, "\n") > 1 {
			separator = separator[:strings.IndexByte(separator, '\n')+1]
		}
		if separator == "" && i+1 < len(doc.Paragraphs) {
			separator = "\n\n"
			line := p.Kind == "source" || p.Kind == "code"
			nextLine := doc.Paragraphs[i+1].Kind == p.Kind
			if emptyParagraph(p) || (line && nextLine) || sameList(p, doc.Paragraphs[i+1]) {
				separator = "\n"
			} else if p.Kind == "quote" && doc.Paragraphs[i+1].Kind == "quote" {
				// A new paragraph inside the same quote, the way Typora writes it.
				separator = "\n>\n"
			}
		}
		if !emptyParagraph(p) && i+1 < len(doc.Paragraphs) && emptyParagraph(doc.Paragraphs[i+1]) && (p.NextID == "" || p.NextID != doc.Paragraphs[i+1].ID) {
			for strings.Count(separator, "\n") < 2 {
				separator += newlineStyle(separator)
			}
		}
		if i+1 < len(doc.Paragraphs) && strings.Contains(separator, ">") && (p.Kind != "quote" || doc.Paragraphs[i+1].Kind != "quote") {
			// The quote's ">" line no longer joins two quote paragraphs.
			separator = newlineStyle(separator) + newlineStyle(separator)
		}
		if i+1 < len(doc.Paragraphs) && strings.Count(separator, "\n") < 2 {
			next := doc.Paragraphs[i+1]
			adjacencyChanged := p.NextID != next.ID
			contentChanged := fingerprint(p) != p.Baseline || fingerprint(next) != next.Baseline
			flowingBlocks := (p.Kind == "body" || p.Kind == "quote") && (next.Kind == "body" || next.Kind == "quote")
			// A list item or quote followed by an edited body line would
			// swallow it as a lazy continuation.
			lazy := p.Kind == "bullet" || p.Kind == "ordered" || p.Kind == "checklist" || p.Kind == "quote"
			// Raw source can be reused after a deletion or move, but its old
			// separator may no longer keep the new neighboring blocks apart.
			linePair := ((p.Kind == "source" || p.Kind == "code") && p.Kind == next.Kind) || sameList(p, next)
			if !emptyParagraph(p) && !emptyParagraph(next) && !linePair && (adjacencyChanged || ((flowingBlocks || lazy) && contentChanged)) {
				if strings.Contains(separator, "\r\n") {
					separator += "\r\n"
				} else {
					separator += "\n"
				}
			}
		}
		if i+1 < len(doc.Paragraphs) {
			next := doc.Paragraphs[i+1]
			if next.ListContinuation && isList(p) && p.ListID == next.ListID {
				for strings.Count(separator, "\n") < 2 {
					separator += newlineStyle(separator)
				}
			}
			// CommonMark needs a blank line when a new nested list starts
			// above 1, or its first item is empty ("- " can be Setext).
			ordered := next.Kind == "ordered" || (next.Kind == "checklist" && next.Ordinal > 0)
			emptyItem := next.Kind != "checklist" && PlainText(next) == ""
			if isList(p) && isList(next) && next.Depth > p.Depth && ((ordered && next.Ordinal != 1) || emptyItem) {
				for strings.Count(separator, "\n") < 2 {
					separator += newlineStyle(separator)
				}
			}
		}
		out.WriteString(separator)
	}
	return out.String()
}

// codeLineSource writes one code line with its fences at the group edges.
// Missing fences (a split line drops them) regenerate as plain ``` so the
// block stays a code block.
func codeLineSource(doc Document, i int) string {
	p := doc.Paragraphs[i]
	nl := newlineStyle(p.Separator)
	if p.CodeNewline != "" {
		nl = p.CodeNewline
	}
	start := p.Open != "" || i == 0 || doc.Paragraphs[i-1].Kind != "code"
	end := i+1 == len(doc.Paragraphs) || doc.Paragraphs[i+1].Kind != "code" || doc.Paragraphs[i+1].Open != ""
	line := p.Raw
	if p.Baseline == "" || p.Baseline != fingerprint(p) {
		line = PlainText(p)
	}
	var b strings.Builder
	if start {
		open := p.Open
		if open == "" {
			open = "```"
		}
		b.WriteString(open)
		// An untouched empty block has no content line: "```go\n```".
		if end && p.Close != "" && line == "" {
			b.WriteString(nl + p.Close)
			return b.String()
		}
		b.WriteString(nl)
	}
	b.WriteString(line)
	if end {
		close := p.Close
		if close == "" {
			close = p.CodeClose
		}
		for previous := i - 1; close == "" && previous >= 0 && doc.Paragraphs[previous].Kind == "code" && p.Open == ""; previous-- {
			close = doc.Paragraphs[previous].Close
			if doc.Paragraphs[previous].Open != "" {
				break
			}
		}
		if close == "" {
			close = "```"
		}
		b.WriteString(nl + close)
	}
	return b.String()
}

// sameList reports whether two adjacent items belong to one tight list, so
// they are written on consecutive lines instead of as separate lists.
func sameList(p, next Paragraph) bool {
	if isList(p) && isList(next) && p.Depth == next.Depth && p.ListContainer != "" && next.ListContainer != "" && p.ListContainer != next.ListContainer {
		return false
	}
	if isList(p) && isList(next) && (p.Depth > 0 || next.Depth > 0) {
		return true
	}
	bullet := func(p Paragraph) bool { return p.Kind == "bullet" || (p.Kind == "checklist" && p.Ordinal == 0) }
	ordered := func(p Paragraph) bool { return p.Kind == "ordered" || (p.Kind == "checklist" && p.Ordinal > 0) }
	return (bullet(p) && bullet(next)) || (ordered(p) && ordered(next))
}

func emptyParagraph(p Paragraph) bool { return p.Kind == "body" && PlainText(p) == "" }

func newlineStyle(separator string) string {
	if strings.Contains(separator, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func serializeParagraph(p Paragraph) string {
	if p.ListContinuation {
		body := p
		body.Kind = "body"
		body.Depth = 0
		body.ListContinuation = false
		return serializeParagraph(body)
	}
	if p.Kind == "table" {
		return serializeTable(p)
	}
	if p.Kind == "source" {
		return PlainText(p)
	}
	normalized := Paragraph{}
	for _, run := range p.Runs {
		appendRun(&normalized, run)
	}
	// Escape only as much as the text needs, like Typora: try the plain
	// spelling first and fall back to stricter escaping (then entities)
	// only when a re-parse would change the paragraph's semantics.
	var value string
	for _, level := range []escapeLevel{escapeNone, escapeInline, escapeAll, escapeEntities, escapeReveal} {
		value = protectIndentation(serializeRuns(normalized.Runs, level))
		if sameParagraphSemantics(blockPrefix(p, value), p, normalized.Runs) {
			break
		}
	}
	return blockPrefix(p, value)
}

// serializeCell writes a table cell. A bare | would end the cell, so cells
// start at the inline level, which escapes it.
func serializeCell(runs []Run) string {
	normalized := Paragraph{}
	for _, run := range runs {
		if run.Math != "" {
			delimiter := "$"
			if strings.HasPrefix(run.Text, "$$") {
				delimiter = "$$"
			}
			run.Text = delimiter + strings.ReplaceAll(run.Math, "|", `\|`) + delimiter
		}
		appendRun(&normalized, run)
	}
	encoded := append([]Run(nil), normalized.Runs...)
	for i := range encoded {
		if encoded[i].Code && encoded[i].Math == "" && !encoded[i].Source {
			// GFM consumes one slash before every code-span pipe, even
			// when the code already contains a literal slash before it.
			encoded[i].Text = strings.ReplaceAll(encoded[i].Text, "|", `\|`)
		}
	}
	var value string
	for _, level := range []escapeLevel{escapeInline, escapeAll, escapeEntities, escapeReveal} {
		value = serializeRuns(encoded, level)
		if sameInlineSemantics(value, normalized.Runs) {
			break
		}
	}
	return escapeCellPipes(value)
}

// ParseInline interprets a native source fragment in its actual cell context.
// The artificial table keeps structural pipe escaping separate from TeX.
func ParseInline(source string, table bool) Document {
	if !table {
		return Parse(source)
	}
	doc := Parse("| " + escapeCellPipes(source) + " |\n| --- |")
	if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "table" || len(doc.Paragraphs[0].Rows) == 0 || len(doc.Paragraphs[0].Rows[0]) != 1 {
		return Document{}
	}
	return Document{Paragraphs: []Paragraph{{Kind: "body", Runs: doc.Paragraphs[0].Rows[0][0].Runs}}}
}

func escapeCellPipes(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '|' && (i == 0 || value[i-1] != '\\') {
			out.WriteByte('\\')
		}
		out.WriteByte(value[i])
	}
	return out.String()
}

// escapeLevel orders the escaping strategies from cleanest to safest.
type escapeLevel int

const (
	escapeNone     escapeLevel = iota // literal text
	escapeInline                      // inline delimiters only
	escapeAll                         // every Markdown punctuation character
	escapeEntities                    // numeric entities for boundary characters
	escapeReveal                      // entities keep styled boundary whitespace inside delimiters
)

func serializeRuns(runs []Run, level escapeLevel) string {
	return serializeRunsObserved(runs, level, nil)
}

func serializeRunsObserved(runs []Run, level escapeLevel, emitted func(int, int, int)) string {
	var text strings.Builder
	for i := 0; i < len(runs); i++ {
		run := runs[i]
		start := text.Len()
		italic := "_"
		if level == escapeEntities {
			italic = "*"
		}
		if i > 0 {
			last, _ := utf8.DecodeLastRuneInString(runs[i-1].Text)
			if unicode.IsLetter(last) || unicode.IsNumber(last) {
				italic = "*"
			}
		}
		if i+1 < len(runs) {
			first, _ := utf8.DecodeRuneInString(runs[i+1].Text)
			if unicode.IsLetter(first) || unicode.IsNumber(first) {
				italic = "*"
			}
		}
		// A reference attachment splits a native style range. Emit its shared
		// delimiters once so adjacent emphasis delimiters cannot become text.
		end, hasFootnote := i+1, run.Footnote != ""
		style := footnoteRunStyle(run)
		if !run.Source && !run.Code && !run.Break && run.Math == "" {
			for end < len(runs) && !runs[end].Source && !runs[end].Code && !runs[end].Break && runs[end].Math == "" && footnoteRunStyle(runs[end]) == style {
				hasFootnote = hasFootnote || runs[end].Footnote != ""
				end++
			}
		}
		if hasFootnote && end > i+1 {
			var value strings.Builder
			for _, part := range runs[i:end] {
				part.Bold, part.Italic, part.Strike, part.Underline, part.Highlight, part.Sup, part.Sub, part.Kbd = false, false, false, false, false, false, false, false
				part.Link, part.LinkTitle = "", ""
				value.WriteString(serializeRun(part, italic, level))
			}
			text.WriteString(formatInlineRun(style, value.String(), italic))
			i = end - 1
		} else {
			text.WriteString(serializeRun(run, italic, level))
			if emitted != nil {
				emitted(i, start, text.Len())
			}
		}
	}
	return text.String()
}

func footnoteRunStyle(run Run) Run { run.Text, run.Footnote = "", ""; return run }

func blockPrefix(p Paragraph, value string) string {
	switch p.Kind {
	case "heading":
		value = strings.Repeat("#", min(6, max(1, p.Level))) + " " + value
	case "bullet":
		value = listMarker(p) + " " + value
	case "ordered":
		value = fmt.Sprintf("%d%s %s", max(1, p.Ordinal), listMarker(p), value)
	case "checklist":
		mark := " "
		if p.Checked {
			mark = "x"
		}
		prefix := listMarker(p) + " "
		if p.Ordinal > 0 {
			prefix = fmt.Sprintf("%d%s ", p.Ordinal, listMarker(p))
		}
		value = prefix + "[" + mark + "] " + value
	case "quote":
		prefix := strings.Repeat("> ", max(1, p.Depth+1))
		value = prefix + strings.ReplaceAll(value, "\n", "\n"+prefix)
	}
	return value
}

// sameParagraphSemantics reports whether source re-parses as one paragraph of
// p's kind with the same styled characters.
func sameParagraphSemantics(source string, p Paragraph, runs []Run) bool {
	doc := Parse(source)
	if len(doc.Paragraphs) != 1 {
		return false
	}
	got := doc.Paragraphs[0]
	if got.Kind != p.Kind || (p.Kind == "heading" && got.Level != min(6, max(1, p.Level))) || (p.Kind == "checklist" && got.Checked != p.Checked) {
		return false
	}
	var expected []Run
	for _, run := range runs {
		if run.Source {
			for _, paragraph := range ParseInline(run.Text, false).Paragraphs {
				expected = append(expected, paragraph.Runs...)
			}
		} else {
			expected = append(expected, run)
		}
	}
	return sameStorageRuns(expected, got.Runs)
}

func sameInlineSemantics(value string, runs []Run) bool {
	doc := ParseInline(value, true)
	if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "body" {
		return false
	}
	var expected []Run
	for _, run := range runs {
		if run.Source {
			for _, p := range ParseInline(run.Text, true).Paragraphs {
				expected = append(expected, p.Runs...)
			}
		} else {
			expected = append(expected, run)
		}
	}
	return sameStorageRuns(expected, doc.Paragraphs[0].Runs)
}

// Plain boundary whitespace follows Markdown's existing normalization policy.
// Whitespace carrying inline formatting must retain those attributes on save.
func sameStorageRuns(expected, actual []Run) bool {
	meaningful := func(runs []Run) []Run {
		var out []Run
		for _, run := range runs {
			styled := run.Bold || run.Italic || run.Strike || run.Code || run.Link != "" || run.Underline || run.Highlight || run.Sup || run.Sub || run.Kbd
			if !styled && !run.Break && run.Math == "" {
				run.Text = strings.Map(func(r rune) rune {
					if unicode.IsSpace(r) {
						return -1
					}
					return r
				}, run.Text)
			}
			if run.Text != "" || run.Break {
				out = append(out, run)
			}
		}
		return out
	}
	return sameRevealRuns(meaningful(expected), meaningful(actual))
}

func sameRuns(runs, parsed []Run) bool {
	styles := func(rs []Run) []Run {
		var expanded []Run
		for _, run := range rs {
			if run.Source {
				for _, p := range Parse(run.Text).Paragraphs {
					expanded = append(expanded, p.Runs...)
				}
			} else {
				expanded = append(expanded, run)
			}
		}
		var out []Run
		for _, run := range expanded {
			if run.Math != "" {
				run.Text = run.Math
			}
			if run.Break {
				out = append(out, Run{Break: true})
				continue
			}
			for _, r := range run.Text {
				if unicode.IsSpace(r) {
					continue
				}
				run.Text = string(r)
				out = append(out, run)
			}
		}
		return out
	}
	want, got := styles(runs), styles(parsed)
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if want[i] != got[i] {
			return false
		}
	}
	return true
}

func serializeRun(run Run, italic string, level escapeLevel) string {
	if run.Break && run.Soft {
		return "\n"
	}
	if run.Break {
		return "  \n"
	}
	if run.Math != "" || run.Source {
		return run.Text // the exact $...$ source
	}
	if level == escapeNone && run.Link != "" && run.LinkTitle == "" && !run.Code && (run.Link == run.Text || run.Link == "http://"+run.Text || run.Link == "mailto:"+run.Text) {
		// A bare URL re-links itself (GFM autolink); the caller verifies.
		plain := run
		plain.Link = ""
		return serializeRun(plain, italic, level)
	}
	value := escapeText(run.Text, level)
	if run.Footnote != "" {
		value = run.Text
	}
	if (level == escapeEntities || level == escapeReveal) && !run.Code && run.Footnote == "" {
		var encoded strings.Builder
		for _, r := range run.Text {
			if level != escapeReveal && unicode.IsSpace(r) {
				encoded.WriteRune(r)
			} else {
				fmt.Fprintf(&encoded, "&#%d;", r)
			}
		}
		value = encoded.String()
	}
	if run.Code {
		longest, current := 0, 0
		for _, r := range run.Text {
			if r == '`' {
				current++
				longest = max(longest, current)
			} else {
				current = 0
			}
		}
		delim := strings.Repeat("`", longest+1)
		content := strings.ReplaceAll(run.Text, "\n", " ")
		if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") || (strings.HasPrefix(content, " ") && strings.HasSuffix(content, " ") && strings.TrimSpace(content) != "") {
			content = " " + content + " "
		}
		value = delim + content + delim
	}
	return formatInlineRun(run, value, italic)
}

func formatInlineRun(run Run, value, italic string) string {
	// Delimiters cannot enclose boundary whitespace in CommonMark.
	left := len(value) - len(strings.TrimLeftFunc(value, unicode.IsSpace))
	right := len(strings.TrimRightFunc(value, unicode.IsSpace))
	if right <= left {
		return value
	}
	before, after := value[:left], value[right:]
	value = value[left:right]
	if run.Kbd {
		value = "<kbd>" + value + "</kbd>"
	}
	if run.Sup {
		value = "^" + value + "^"
	}
	if run.Sub {
		value = "~" + value + "~"
	}
	if run.Highlight {
		value = "==" + value + "=="
	}
	if run.Underline {
		value = "<u>" + value + "</u>"
	}
	if run.Bold {
		value = "**" + value + "**"
	}
	if run.Italic {
		value = italic + value + italic
	}
	if run.Strike {
		value = "~~" + value + "~~"
	}
	value = before + value + after
	if run.Link != "" {
		destination := linkDestination(run.Link)
		title := ""
		if run.LinkTitle != "" {
			escaped := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "&", "\\&", "\n", "&#10;", "\r", "&#13;").Replace(run.LinkTitle)
			title = " \"" + escaped + "\""
		}
		// Edited reference links use resolved inline destinations and titles;
		// untouched paragraphs and definition blocks retain their exact Raw.
		value = "[" + value + "](" + destination + title + ")"
	}
	return value
}

var entityLike = regexp.MustCompile(`&[A-Za-z0-9#]+;`)

// linkDestination writes plain URLs bare, as people type them, and falls
// back to the <...> form when the bare one would not parse back exactly.
func linkDestination(link string) string {
	bare := link != "" && !strings.ContainsAny(link, " \t\n\r<>\\") && !entityLike.MatchString(link)
	depth := 0
	for _, r := range link {
		if r == '(' {
			depth++
		} else if r == ')' {
			if depth--; depth < 0 {
				break
			}
		}
	}
	if bare && depth == 0 {
		return link
	}
	return "<" + strings.NewReplacer("\\", "\\\\", "<", "\\<", ">", "\\>", "&", "\\&", "\n", "%0A", "\r", "%0D").Replace(link) + ">"
}

func escapeText(value string, level escapeLevel) string {
	if level == escapeNone {
		return value
	}
	// = and ^ only become markup as pairs (==mark==, ^sup^): escape them
	// only when the text could actually form a pair, so prose like
	// "a = b" or "2^10" stays clean.
	escapeEq := strings.Contains(value, "==")
	escapeCaret := hasCaretPair(value)
	// The inline level leaves block-only punctuation (. - + # ! | ( ) ...)
	// alone: inside a paragraph those characters cannot start markup.
	special := "\\`*_{}[]()<>#+-.!|~$&"
	if level == escapeInline {
		special = "\\`*_[]<~$&|"
	}
	var out strings.Builder
	for _, r := range value {
		if strings.ContainsRune(special, r) || (escapeEq && r == '=') || (escapeCaret && r == '^') {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func protectIndentation(value string) string {
	var out strings.Builder
	leading := true
	for _, r := range value {
		if leading && (r == ' ' || r == '\t') {
			out.WriteString(fmt.Sprintf("&#%d;", r))
			continue
		}
		leading = r == '\n'
		out.WriteRune(r)
	}
	return out.String()
}

func PlainText(p Paragraph) string {
	if p.Kind == "table" {
		rows := make([]string, len(p.Rows))
		for index, row := range p.Rows {
			cells := make([]string, len(row))
			for column, cell := range row {
				cells[column] = PlainText(Paragraph{Runs: cell.Runs})
			}
			rows[index] = strings.Join(cells, "\t")
		}
		return strings.Join(rows, "\n")
	}
	var out strings.Builder
	for _, run := range p.Runs {
		if run.Soft {
			out.WriteByte(' ') // a source line wrap reads as a space
		}
		out.WriteString(run.Text)
	}
	return out.String()
}

func UTF16Len(value string) int {
	length := 0
	for _, r := range value {
		length++
		if r > 0xffff {
			length++
		}
	}
	return length
}

// hasCaretPair reports whether two ^ in value could form a ^sup^ pair
// (no space between them), mirroring the inline parser's rule.
func hasCaretPair(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] != '^' {
			continue
		}
		for j := i + 1; j < len(value); j++ {
			switch value[j] {
			case '^':
				return true
			case ' ', '\t', '\n':
				j = len(value)
			}
		}
	}
	return false
}
