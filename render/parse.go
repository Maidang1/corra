package render

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	emojiast "github.com/yuin/goldmark-emoji/ast"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/Maidang1/corra/internal/mdext"
)

// markdown is the shared parser: GFM plus math, mdext (highlight, super-
// and subscript, inline HTML tags) and emoji short codes.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		MathExtender,
		mdext.InlineExtender,
		emoji.New(),
	),
)

// markdownSmart adds Typographer smart punctuation for readers that prefer
// it (preview and export); the source on disk never changes.
var markdownSmart = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		extension.NewTypographer(extension.WithTypographicSubstitutions(map[extension.TypographicPunctuation]string{
			extension.LeftSingleQuote:  "\u2018",
			extension.RightSingleQuote: "\u2019",
			extension.LeftDoubleQuote:  "\u201c",
			extension.RightDoubleQuote: "\u201d",
			extension.EnDash:           "\u2013",
			extension.EmDash:           "\u2014",
			extension.Ellipsis:         "\u2026",
			extension.LeftAngleQuote:   "\u00ab",
			extension.RightAngleQuote:  "\u00bb",
			extension.Apostrophe:       "\u2019",
		})),
		MathExtender,
		mdext.InlineExtender,
		emoji.New(),
	),
)

// Document is a parsed markdown document.
type Document struct {
	Blocks          []Block
	MathExpressions []MathExpression
	MathOptions     MathOptions
	mathBlocks      map[int]int
	// Footnotes holds footnote definitions, rendered at the end.
	Footnotes []string
}

// Parse parses src into a Document.
func Parse(src string) *Document { return ParseOptions(src, false) }

// ParseOptions parses src, with smart punctuation when smartPunct holds.
func ParseOptions(src string, smartPunct bool) *Document {
	md := markdown
	if smartPunct {
		md = markdownSmart
	}
	fm, body := mdext.SplitFrontMatter(src)
	reader := text.NewReader([]byte(body))
	parsed := md.Parser().Parse(reader)
	mdext.TransformHTML(parsed, []byte(body))
	plan := collectMath(parsed)
	doc := &Document{MathExpressions: plan.Expressions, mathBlocks: make(map[int]int)}
	if fm != "" {
		doc.Blocks = append(doc.Blocks, Block{Kind: KindFrontMatter, Source: strings.TrimRight(fm, "\r\n"), Start: 0, End: len(fm)})
	}
	offset := len(fm)
	for n := parsed.FirstChild(); n != nil; n = n.NextSibling() {
		if fn, ok := n.(*east.FootnoteList); ok {
			for def := fn.FirstChild(); def != nil; def = def.NextSibling() {
				if d, ok := def.(*east.Footnote); ok {
					var buf bytes.Buffer
					for c := d.FirstChild(); c != nil; c = c.NextSibling() {
						buf.WriteString(plainTextBody(blockSource(c, []byte(body)), smartPunct))
						buf.WriteByte(' ')
					}
					doc.Footnotes = append(doc.Footnotes, strings.TrimSpace(buf.String()))
				}
			}
			continue
		}
		b, ok := buildBlock(n, []byte(body))
		if ok {
			if slot, exists := plan.Nodes[n]; exists {
				doc.mathBlocks[len(doc.Blocks)] = slot
			}
			b.Start, b.End = nodeRange(n)
			b.Start += offset
			b.End += offset
			doc.Blocks = append(doc.Blocks, b)
		}
	}
	return doc
}

// nodeRange returns the byte range a node spans in the source.
func nodeRange(n gast.Node) (int, int) {
	lines := n.Lines()
	if lines.Len() == 0 {
		return 0, 0
	}
	return lines.At(0).Start, lines.At(lines.Len() - 1).Stop
}

// blockSource reconstructs a block's exact source from its line segments.
func blockSource(n gast.Node, src []byte) string {
	var buf bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(src))
	}
	return buf.String()
}

func buildBlock(n gast.Node, src []byte) (Block, bool) {
	b := Block{Source: blockSource(n, src)}
	switch t := n.(type) {
	case *gast.Heading:
		b.Kind = KindHeading
		b.Level = t.Level
	case *gast.FencedCodeBlock:
		info := strings.ToLower(strings.TrimSpace(string(t.Info.Segment.Value(src))))
		switch info {
		case "mermaid", "d2", "goat":
			b.Kind = KindDiagram
			b.DiagramS = blockSource(t, src)
			b.Diagram = map[string]DiagramKind{"mermaid": DiagramMermaid, "d2": DiagramD2, "goat": DiagramGoAT}[info]
		case "chart":
			b.Kind = KindChart
			b.ChartS = blockSource(t, src)
		default:
			b.Kind = KindCodeBlock
			b.Language = info
		}
	case *gast.CodeBlock:
		b.Kind = KindCodeBlock
	case *gast.Blockquote:
		b.Kind = KindQuote
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if p, ok := c.(*gast.Paragraph); ok {
				b.Items = append(b.Items, ListItem{Source: strings.TrimSpace(inlineSource(p, src)), CheckboxOffset: -1})
			} else if list, ok := c.(*gast.List); ok {
				appendListItems(list, src, 0, &b.Items)
			}
		}
	case *gast.List:
		b.Kind = KindList
		b.Ordered = t.IsOrdered()
		appendListItems(t, src, 0, &b.Items)
	case *east.Table:
		b.Kind = KindTable
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			var cells []TableCell
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				tc, ok := cell.(*east.TableCell)
				if !ok {
					continue
				}
				c := TableCell{Text: PlainText(inlineSource(cell, src))}
				switch tc.Alignment {
				case east.AlignCenter:
					c.Align = 1
				case east.AlignRight:
					c.Align = 2
				}
				cells = append(cells, c)
			}
			if row.Kind() == east.KindTableHeader {
				b.Header = cells
			} else {
				b.Rows = append(b.Rows, cells)
			}
		}
	case *gast.ThematicBreak:
		b.Kind = KindHR
	case *gast.HTMLBlock:
		if img, ok := parseImgTag(b.Source); ok {
			b = img
			b.Source = blockSource(n, src)
			break
		}
		b.Kind = KindHTML
	case *mathBlockNode:
		if t.open {
			b.Kind = KindHTML // preserve malformed math as visible editable source
			break
		}
		b.Kind = KindMathBlock
		b.Math = t.TeX
	case *gast.Paragraph:
		if img := singleImage(n); img != nil {
			b.Kind = KindImage
			b.ImageURL = string(img.Destination)
			b.ImageTitle = string(img.Title)
			if txt, ok := img.FirstChild().(*gast.Text); ok {
				b.ImageAlt = string(txt.Segment.Value(src))
			}
			parseImageSize(&b)
			break
		}
		if mdext.IsTOC(b.Source) {
			b.Kind = KindTOC
			break
		}
		b.Kind = KindParagraph
	default:
		return Block{}, false
	}
	return b, true
}

func appendListItems(list *gast.List, src []byte, depth int, items *[]ListItem) {
	for li := list.FirstChild(); li != nil; li = li.NextSibling() {
		item := ListItem{Depth: depth, CheckboxOffset: -1}
		if first := li.FirstChild(); first != nil {
			if box, ok := first.FirstChild().(*east.TaskCheckBox); ok {
				item.Task, item.Done = true, box.IsChecked
				// Goldmark's checkbox inline has no segment of its own. Its
				// containing paragraph retains the exact original first line.
				if lines := first.Lines(); lines.Len() > 0 {
					line := lines.At(0)
					start := line.Start
					for start < line.Stop && (src[start] == ' ' || src[start] == '\t') {
						start++
					}
					if start+2 < line.Stop && src[start] == '[' && src[start+2] == ']' {
						item.CheckboxOffset = start + 1
					}
				}
			}
		}
		var content bytes.Buffer
		collectListItemText(li, src, &content)
		item.Source = strings.TrimSpace(content.String())
		*items = append(*items, item)
		for child := li.FirstChild(); child != nil; child = child.NextSibling() {
			if nested, ok := child.(*gast.List); ok {
				appendListItems(nested, src, depth+1, items)
			}
		}
	}
}

// collectListItemText preserves the item's own lines, excluding nested lists.
func collectListItemText(n gast.Node, src []byte, buf *bytes.Buffer) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Kind() {
		case gast.KindParagraph, gast.KindTextBlock:
			own := strings.TrimSpace(blockSource(c, src))
			if _, ok := c.FirstChild().(*east.TaskCheckBox); ok && len(own) >= 3 {
				own = strings.TrimLeft(own[3:], " \t")
			}
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(own)
		case gast.KindList:
			continue
		default:
			collectListItemText(c, src, buf)
		}
	}
}

// singleImage returns the image if the paragraph consists of one image only.
func singleImage(n gast.Node) *gast.Image {
	p, ok := n.(*gast.Paragraph)
	if !ok || p.ChildCount() != 1 {
		return nil
	}
	img, ok := p.FirstChild().(*gast.Image)
	if !ok {
		return nil
	}
	return img
}

// inlineSource renders a node's inline children back to markdown-ish source.
func inlineSource(n gast.Node, src []byte) string {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		writeInline(&buf, c, src)
	}
	return buf.String()
}

func writeInline(buf *bytes.Buffer, n gast.Node, src []byte) {
	switch t := n.(type) {
	case *gast.Text:
		buf.Write(t.Segment.Value(src))
		if t.HardLineBreak() {
			buf.WriteByte('\n')
		}
	case *gast.String:
		buf.Write(t.Value)
	case *gast.CodeSpan:
		buf.WriteByte('`')
		var inner bytes.Buffer
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(&inner, c, src)
		}
		buf.Write(inner.Bytes())
		buf.WriteByte('`')
	case *gast.Emphasis:
		marker := "*"
		if t.Level == 2 {
			marker = "**"
		}
		buf.WriteString(marker)
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(buf, c, src)
		}
		buf.WriteString(marker)
	case *east.Strikethrough:
		buf.WriteString("~~")
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(buf, c, src)
		}
		buf.WriteString("~~")
	case *gast.Link:
		buf.WriteByte('[')
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(buf, c, src)
		}
		buf.WriteString("](")
		buf.Write(t.Destination)
		buf.WriteByte(')')
	case *gast.AutoLink:
		buf.Write(t.URL(src))
	case *east.FootnoteLink:
		fmt.Fprintf(buf, "[%d]", t.Index)
	case *mathInlineNode:
		buf.WriteString(t.Raw)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(buf, c, src)
		}
	}
}

// plainTextBody is PlainText with the smart-punctuation option.
func plainTextBody(src string, smartPunct bool) string {
	ins := parseInlines(src, smartPunct)
	var s string
	for _, in := range ins {
		s += inlinePlain(in)
	}
	return s
}

// ParseInlines parses inline markdown into an Inline tree.
func ParseInlines(src string) []Inline { return parseInlines(src, false) }

func parseInlines(src string, smartPunct bool) []Inline {
	md := markdown
	if smartPunct {
		md = markdownSmart
	}
	reader := text.NewReader([]byte(src))
	parsed := md.Parser().Parse(reader)
	mdext.TransformHTML(parsed, []byte(src))
	var out []Inline
	for n := parsed.FirstChild(); n != nil; n = n.NextSibling() {
		p, ok := n.(*gast.Paragraph)
		if !ok {
			continue
		}
		out = append(out, inlinesOf(p, []byte(src))...)
	}
	return out
}

func inlinesOf(n gast.Node, src []byte) []Inline {
	var out []Inline
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, inlineOf(c, src)...)
	}
	return out
}

func inlinePlainAll(ins []Inline) string {
	var s string
	for _, in := range ins {
		s += inlinePlain(in)
	}
	return s
}

func inlineOf(n gast.Node, src []byte) []Inline {
	switch t := n.(type) {
	case *gast.Text:
		ins := []Inline{{Kind: InlineText, Text: string(t.Segment.Value(src))}}
		if t.HardLineBreak() || t.SoftLineBreak() {
			// Soft breaks render as line breaks (GitHub-style), like Typora's
			// default non-strict mode.
			ins = append(ins, Inline{Kind: InlineHardBreak})
		}
		return ins
	case *gast.String:
		return []Inline{{Kind: InlineText, Text: string(t.Value)}}
	case *gast.CodeSpan:
		return []Inline{{Kind: InlineCode, Text: inlinePlainAll(inlinesOf(n, src))}}
	case *gast.Emphasis:
		kind := InlineItalic
		if t.Level == 2 {
			kind = InlineBold
		}
		return []Inline{{Kind: kind, Children: inlinesOf(n, src)}}
	case *east.Strikethrough:
		return []Inline{{Kind: InlineStrikethrough, Children: inlinesOf(n, src)}}
	case *gast.Link:
		return []Inline{{Kind: InlineLink, Text: inlinePlainAll(inlinesOf(n, src)), URL: string(t.Destination), Title: string(t.Title)}}
	case *gast.Image:
		return []Inline{{Kind: InlineImage, Text: inlinePlainAll(inlinesOf(n, src)), URL: string(t.Destination), Title: string(t.Title)}}
	case *gast.AutoLink:
		return []Inline{{Kind: InlineLink, Text: string(t.URL(src)), URL: string(t.URL(src))}}
	case *east.FootnoteLink:
		return []Inline{{Kind: InlineText, Text: fmt.Sprintf("[%d]", t.Index)}}
	case *mathInlineNode:
		return []Inline{{Kind: InlineMath, Text: t.TeX}}
	case *mdext.Mark:
		return []Inline{{Kind: InlineHighlight, Children: inlinesOf(n, src)}}
	case *mdext.Underline:
		return []Inline{{Kind: InlineUnderline, Children: inlinesOf(n, src)}}
	case *mdext.Superscript:
		return []Inline{{Kind: InlineSuperscript, Children: inlinesOf(n, src)}}
	case *mdext.Subscript:
		return []Inline{{Kind: InlineSubscript, Children: inlinesOf(n, src)}}
	case *mdext.Kbd:
		return []Inline{{Kind: InlineKbd, Text: inlinePlainAll(inlinesOf(n, src))}}
	case *emojiast.Emoji:
		if t.Value != nil && t.Value.IsUnicode() {
			return []Inline{{Kind: InlineText, Text: string(t.Value.Unicode)}}
		}
		return []Inline{{Kind: InlineText, Text: ":" + string(t.ShortName) + ":"}}
	default:
		return inlinesOf(n, src)
	}
}

// parseImageSize reads a Typora/Pandoc-style size suffix from the image
// destination (=300x200, =300x, =x200, =50%) and strips it from the URL.
func parseImageSize(b *Block) {
	url := b.ImageURL
	i := strings.LastIndexByte(url, ' ')
	if i < 0 || i+1 >= len(url) || url[i+1] != '=' {
		return
	}
	spec := url[i+2:]
	b.ImageURL = strings.TrimRight(url[:i], " ")
	if strings.HasSuffix(spec, "%") {
		if v, err := strconv.Atoi(strings.TrimSuffix(spec, "%")); err == nil && v > 0 && v <= 400 {
			b.ImageZoom = v
		}
		return
	}
	w, h, _ := strings.Cut(spec, "x")
	if v, err := strconv.Atoi(w); err == nil && v > 0 {
		b.ImageWidth = v
	}
	if v, err := strconv.Atoi(h); err == nil && v > 0 {
		b.ImageHeight = v
	}
}

// parseImgTag parses an HTML block consisting of a single <img> tag into an
// image block, as Typora sizes images with <img src="..." width="300">.
func parseImgTag(src string) (Block, bool) {
	s := strings.TrimSpace(src)
	if !strings.HasPrefix(strings.ToLower(s), "<img") || !strings.HasSuffix(s, ">") {
		return Block{}, false
	}
	attr := func(name string) string {
		for _, q := range []byte{'"', '\''} {
			key := name + "=" + string(q)
			i := strings.Index(s, key)
			if i < 0 {
				continue
			}
			j := strings.IndexByte(s[i+len(key):], q)
			if j < 0 {
				return ""
			}
			return s[i+len(key) : i+len(key)+j]
		}
		return ""
	}
	url := attr("src")
	if url == "" {
		return Block{}, false
	}
	b := Block{Kind: KindImage, ImageURL: url, ImageAlt: attr("alt"), ImageTitle: attr("title")}
	if v, err := strconv.Atoi(strings.TrimSuffix(attr("width"), "px")); err == nil && v > 0 {
		b.ImageWidth = v
	}
	if v, err := strconv.Atoi(strings.TrimSuffix(attr("height"), "px")); err == nil && v > 0 {
		b.ImageHeight = v
	}
	if strings.HasSuffix(attr("width"), "%") {
		if v, err := strconv.Atoi(strings.TrimSuffix(attr("width"), "%")); err == nil && v > 0 && v <= 400 {
			b.ImageZoom = v
			b.ImageWidth = 0
		}
	}
	return b, true
}
