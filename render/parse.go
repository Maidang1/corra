package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// markdown is the shared parser: GFM plus our math extension.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		MathExtender,
	),
)

// Document is a parsed markdown document.
type Document struct {
	Blocks []Block
	// Footnotes holds footnote definitions, rendered at the end.
	Footnotes []string
}

// Parse parses src into a Document.
func Parse(src string) *Document {
	reader := text.NewReader([]byte(src))
	parsed := markdown.Parser().Parse(reader)
	doc := &Document{}
	for n := parsed.FirstChild(); n != nil; n = n.NextSibling() {
		if fn, ok := n.(*east.FootnoteList); ok {
			for def := fn.FirstChild(); def != nil; def = def.NextSibling() {
				if d, ok := def.(*east.Footnote); ok {
					var buf bytes.Buffer
					for c := d.FirstChild(); c != nil; c = c.NextSibling() {
						buf.WriteString(PlainText(blockSource(c, []byte(src))))
						buf.WriteByte(' ')
					}
					doc.Footnotes = append(doc.Footnotes, strings.TrimSpace(buf.String()))
				}
			}
			continue
		}
		b, ok := buildBlock(n, []byte(src))
		if ok {
			b.Start, b.End = nodeRange(n)
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
				b.Items = append(b.Items, ListItem{Source: strings.TrimSpace(inlineSource(p, src))})
			}
		}
	case *gast.List:
		b.Kind = KindList
		b.Ordered = t.IsOrdered()
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			li, ok := c.(*gast.ListItem)
			if !ok {
				continue
			}
			item := ListItem{}
			// The item's checkbox, if the tasklist extension made one.
			if cb := li.FirstChild(); cb != nil {
				if box, ok := cb.FirstChild().(*east.TaskCheckBox); ok {
					item.Task = true
					item.Done = box.IsChecked
				}
			}
			var text bytes.Buffer
			collectListItemText(li, src, &text)
			item.Source = strings.TrimSpace(text.String())
			b.Items = append(b.Items, item)
		}
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
		b.Kind = KindHTML
	case *mathBlockNode:
		if t.TeX == "" {
			return Block{}, false // unclosed $$: not math
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
			break
		}
		b.Kind = KindParagraph
	default:
		return Block{}, false
	}
	return b, true
}

// collectListItemText gathers the text of a list item, skipping the checkbox.
func collectListItemText(n gast.Node, src []byte, buf *bytes.Buffer) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Kind() {
		case gast.KindParagraph, gast.KindTextBlock:
			for ic := c.FirstChild(); ic != nil; ic = ic.NextSibling() {
				if ic.Kind() == east.KindTaskCheckBox {
					continue
				}
				writeInline(buf, ic, src)
			}
		default:
			// Nested lists: keep their marker lines out, keep the text.
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
		buf.WriteByte('$')
		buf.WriteString(t.TeX)
		buf.WriteByte('$')
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(buf, c, src)
		}
	}
}

// ParseInlines parses inline markdown into an Inline tree.
func ParseInlines(src string) []Inline {
	reader := text.NewReader([]byte(src))
	parsed := markdown.Parser().Parse(reader)
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
		if t.HardLineBreak() {
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
	default:
		return inlinesOf(n, src)
	}
}
