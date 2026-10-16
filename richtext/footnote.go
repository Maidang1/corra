package richtext

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var kindFootnoteBlock = ast.NewNodeKind("RichTextFootnoteDefinition")
var kindFootnoteInline = ast.NewNodeKind("RichTextFootnoteReference")

type footnoteBlock struct {
	ast.BaseBlock
	label string
}

func (n *footnoteBlock) Kind() ast.NodeKind         { return kindFootnoteBlock }
func (n *footnoteBlock) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

// Footnotes remain at their source position rather than being collected into
// the synthetic trailing list produced by Goldmark's HTML footnote extension.
type footnoteBlockParser struct{}

func (footnoteBlockParser) Trigger() []byte { return []byte{'['} }
func (footnoteBlockParser) Open(_ ast.Node, r text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, segment := r.PeekLine()
	trimmed := strings.TrimLeft(string(line), " ")
	label, end := footnoteToken(trimmed)
	if end != 0 {
		end += len(line) - len(trimmed)
	}
	if end == 0 || end >= len(line) || line[end] != ':' {
		return nil, parser.NoChildren
	}
	n := &footnoteBlock{label: label}
	n.SetPos(segment.Start)
	n.Lines().Append(segment)
	r.AdvanceToEOL()
	return n, parser.NoChildren
}
func (footnoteBlockParser) Continue(node ast.Node, r text.Reader, _ parser.Context) parser.State {
	line, segment := r.PeekLine()
	if len(line) == 0 || (strings.TrimSpace(string(line)) != "" && footnoteIndentWidth(string(line)) == 0) {
		return parser.Close
	}
	node.Lines().Append(segment)
	r.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}
func (footnoteBlockParser) Close(ast.Node, text.Reader, parser.Context) {}
func (footnoteBlockParser) CanInterruptParagraph() bool                 { return true }
func (footnoteBlockParser) CanAcceptIndentedLine() bool                 { return false }

func footnoteToken(line string) (string, int) {
	if !strings.HasPrefix(line, "[^") {
		return "", 0
	}
	for i := 2; i < len(line); i++ {
		switch line[i] {
		case ']':
			if i == 2 {
				return "", 0
			}
			return line[2:i], i + 1
		case '[', '\\', '\r', '\n', ' ', '\t':
			return "", 0
		}
	}
	return "", 0
}
func footnoteIndentWidth(line string) int {
	column := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			column++
		case '\t':
			column += 4 - column%4
		default:
			return 0
		}
		if column >= 4 {
			return i + 1
		}
	}
	return 0
}

func footnoteParagraphs(n *footnoteBlock, raw string) []Paragraph {
	lines := strings.SplitAfter(raw, "\n")
	first := strings.TrimLeft(lines[0], " ")
	_, end := footnoteToken(first)
	end += len(lines[0]) - len(first) + 1
	for end < len(lines[0]) && (lines[0][end] == ' ' || lines[0][end] == '\t') {
		end++
	}
	marker := lines[0][:end]
	lines[0] = lines[0][end:]
	indent := "    "
	for i := 1; i < len(lines); i++ {
		if width := footnoteIndentWidth(lines[i]); width != 0 {
			indent = lines[i][:width]
			lines[i] = lines[i][width:]
		}
	}
	inner := Parse(strings.Join(lines, ""))
	// The native model has one footnote container per paragraph. Preserve a
	// nested definition's complete Markdown before assigning the outer identity.
	var children []Paragraph
	for i := 0; i < len(inner.Paragraphs); {
		p := inner.Paragraphs[i]
		if p.FootnoteID == "" {
			children = append(children, p)
			i++
			continue
		}
		end := i + 1
		for end < len(inner.Paragraphs) && inner.Paragraphs[end].FootnoteGroup == p.FootnoteGroup {
			end++
		}
		nested := sourceLines(p.FootnoteRaw)
		nested[0].Prefix = p.Prefix
		nested[len(nested)-1].Separator = inner.Paragraphs[end-1].Separator
		children = append(children, nested...)
		i = end
	}
	inner.Paragraphs = children
	for i := range inner.Paragraphs {
		p := &inner.Paragraphs[i]
		p.FootnoteID, p.FootnoteGroup = n.label, fmt.Sprintf("footnote%d", n.Pos())
		p.FootnoteContinuation, p.FootnoteMarker, p.FootnoteIndent = i != 0, marker, indent
	}
	inner.Paragraphs[0].FootnoteRaw = raw
	inner.Paragraphs[0].FootnoteBaseline = footnoteFingerprint(inner.Paragraphs)
	return inner.Paragraphs
}

func footnoteFingerprint(ps []Paragraph) string {
	var value strings.Builder
	for _, p := range ps {
		value.WriteString(fingerprint(p))
		value.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(value.String()))
	return hex.EncodeToString(sum[:])
}

// Serialize collapses definition containers before the ordinary block writer,
// letting lists, tables, code fences and rich inline editing reuse their codecs.
func Serialize(doc Document) string {
	out := Document{}
	for i := 0; i < len(doc.Paragraphs); {
		p := doc.Paragraphs[i]
		if p.FootnoteID == "" {
			out.Paragraphs = append(out.Paragraphs, p)
			i++
			continue
		}
		end := i + 1
		for end < len(doc.Paragraphs) && doc.Paragraphs[end].FootnoteID == p.FootnoteID && doc.Paragraphs[end].FootnoteGroup == p.FootnoteGroup {
			end++
		}
		ps := doc.Paragraphs[i:end]
		raw := p.FootnoteRaw
		if p.FootnoteBaseline == "" || p.FootnoteBaseline != footnoteFingerprint(ps) {
			inner := Document{Paragraphs: append([]Paragraph(nil), ps...)}
			for j := range inner.Paragraphs {
				q := &inner.Paragraphs[j]
				q.FootnoteID, q.FootnoteGroup, q.FootnoteMarker, q.FootnoteIndent, q.FootnoteRaw, q.FootnoteBaseline = "", "", "", "", "", ""
				q.FootnoteContinuation = false
				// Removing the container must not invalidate the inner raw baseline.
				if fingerprint(ps[j]) == ps[j].Baseline {
					q.Baseline = fingerprint(*q)
				}
			}
			inner.Paragraphs[len(inner.Paragraphs)-1].Separator = ""
			if inner.Paragraphs[len(inner.Paragraphs)-1].Kind == "code" {
				for j := len(inner.Paragraphs) - 1; j >= 0 && inner.Paragraphs[j].Kind == "code"; j-- {
					inner.Paragraphs[j].CodeSeparator = ""
				}
			}
			inner.Paragraphs[0].Prefix = ""
			marker, indent := p.FootnoteMarker, p.FootnoteIndent
			if marker == "" {
				marker = "[^" + p.FootnoteID + "]: "
			}
			if indent == "" {
				indent = "    "
			}
			value := serializeDocument(inner)
			if strings.Contains(p.FootnoteRaw+ps[len(ps)-1].Separator, "\r\n") {
				value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\n", "\r\n")
			}
			raw = marker + strings.ReplaceAll(value, "\n", "\n"+indent)
		}
		last := ps[len(ps)-1]
		q := Paragraph{ID: p.ID, Kind: "source", Runs: []Run{{Text: raw}}, Raw: raw, Prefix: p.Prefix, Separator: last.Separator, NextID: last.NextID}
		q.Baseline = fingerprint(q)
		out.Paragraphs = append(out.Paragraphs, q)
		i = end
	}
	return serializeDocument(out)
}

type footnoteInline struct {
	ast.BaseInline
	source, label string
}

func (n *footnoteInline) Kind() ast.NodeKind         { return kindFootnoteInline }
func (n *footnoteInline) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type footnoteInlineParser struct{}

func (footnoteInlineParser) Trigger() []byte { return []byte{'['} }
func (footnoteInlineParser) Parse(_ ast.Node, r text.Reader, _ parser.Context) ast.Node {
	line, segment := r.PeekLine()
	label, end := footnoteToken(string(line))
	if end == 0 {
		return nil
	}
	n := &footnoteInline{source: string(line[:end]), label: label}
	n.SetPos(segment.Start)
	r.Advance(end)
	return n
}
