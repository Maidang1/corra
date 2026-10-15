// Package mdext holds the goldmark extensions shared by the renderer and
// the rich-text codec: ==highlight==, ^superscript^, ~subscript~ (single
// tilde; ~~ stays strikethrough, as in Typora), inline HTML tags
// (<u>/<ins>, <mark>, <kbd>, <sub>, <sup>, <br>), YAML front matter
// splitting and [TOC] detection.
package mdext

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	emojidef "github.com/yuin/goldmark-emoji/definition"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Node kinds for the custom inline nodes.
var (
	KindMark        = ast.NewNodeKind("Mark")
	KindUnderline   = ast.NewNodeKind("Underline")
	KindSuperscript = ast.NewNodeKind("Superscript")
	KindSubscript   = ast.NewNodeKind("Subscript")
	KindKbd         = ast.NewNodeKind("Kbd")
)

// Mark is ==highlighted== inline content.
type Mark struct{ ast.BaseInline }

// Dump implements ast.Node.
func (n *Mark) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// Kind implements ast.Node.
func (n *Mark) Kind() ast.NodeKind { return KindMark }

// Underline is <u>underlined</u> (or <ins>) inline content.
type Underline struct{ ast.BaseInline }

// Dump implements ast.Node.
func (n *Underline) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// Kind implements ast.Node.
func (n *Underline) Kind() ast.NodeKind { return KindUnderline }

// Superscript is ^raised^ inline content.
type Superscript struct{ ast.BaseInline }

// Dump implements ast.Node.
func (n *Superscript) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// Kind implements ast.Node.
func (n *Superscript) Kind() ast.NodeKind { return KindSuperscript }

// Subscript is ~lowered~ inline content (single tilde; ~~ is
// strikethrough).
type Subscript struct{ ast.BaseInline }

// Dump implements ast.Node.
func (n *Subscript) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// Kind implements ast.Node.
func (n *Subscript) Kind() ast.NodeKind { return KindSubscript }

// Kbd is <kbd>keystroke</kbd> inline content.
type Kbd struct{ ast.BaseInline }

// Dump implements ast.Node.
func (n *Kbd) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// Kind implements ast.Node.
func (n *Kbd) Kind() ast.NodeKind { return KindKbd }

// --- tilde: ~sub~ and ~~strike~~ share one processor ---------------------

type tildeProcessor struct{}

func (p *tildeProcessor) IsDelimiter(b byte) bool { return b == '~' }

// CanOpenCloser matches tildes of the same character and length only:
// ~ with ~, ~~ with ~~.
func (p *tildeProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char && opener.OriginalLength == closer.OriginalLength
}

func (p *tildeProcessor) OnMatch(consumes int) ast.Node {
	if consumes >= 2 {
		return east.NewStrikethrough()
	}
	return &Subscript{}
}

type tildeParser struct{}

func (s *tildeParser) Trigger() []byte { return []byte{'~'} }

func (s *tildeParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 1, tilde)
	if node == nil || node.OriginalLength > 2 || before == '~' {
		return nil
	}
	if node.OriginalLength == 1 && node.CanOpen && !spaceFreeCloser(line[1:], '~') {
		return nil // pandoc rule: ~sub~ contains no spaces
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *tildeParser) CloseBlock(parent ast.Node, pc parser.Context) {}

var tilde = &tildeProcessor{}

// --- caret: ^sup^ ---------------------------------------------------------

type caretProcessor struct{}

func (p *caretProcessor) IsDelimiter(b byte) bool { return b == '^' }
func (p *caretProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char && opener.OriginalLength == closer.OriginalLength
}
func (p *caretProcessor) OnMatch(consumes int) ast.Node { return &Superscript{} }

type caretParser struct{}

func (s *caretParser) Trigger() []byte { return []byte{'^'} }

func (s *caretParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 1, caret)
	if node == nil || node.OriginalLength != 1 || before == '^' {
		return nil
	}
	if node.CanOpen && !spaceFreeCloser(line[1:], '^') {
		return nil // pandoc rule: ^sup^ contains no spaces
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *caretParser) CloseBlock(parent ast.Node, pc parser.Context) {}

var caret = &caretProcessor{}

// --- equals: ==mark== ------------------------------------------------------

type equalsProcessor struct{}

func (p *equalsProcessor) IsDelimiter(b byte) bool { return b == '=' }
func (p *equalsProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char && opener.OriginalLength == closer.OriginalLength
}
func (p *equalsProcessor) OnMatch(consumes int) ast.Node { return &Mark{} }

type equalsParser struct{}

func (s *equalsParser) Trigger() []byte { return []byte{'='} }

func (s *equalsParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, equals)
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *equalsParser) CloseBlock(parent ast.Node, pc parser.Context) {}

var equals = &equalsProcessor{}

// --- inline HTML -----------------------------------------------------------

// htmlTag maps a raw HTML tag (lowercased, no attributes) to the node that
// wraps its content, or "" for a void tag replaced in place.
var htmlOpen = map[string]func() ast.Node{
	"<u>":    func() ast.Node { return &Underline{} },
	"<ins>":  func() ast.Node { return &Underline{} },
	"<mark>": func() ast.Node { return &Mark{} },
	"<kbd>":  func() ast.Node { return &Kbd{} },
	"<sub>":  func() ast.Node { return &Subscript{} },
	"<sup>":  func() ast.Node { return &Superscript{} },
}

var htmlClose = map[string]string{
	"</u>": "<u>", "</ins>": "<ins>", "</mark>": "<mark>",
	"</kbd>": "<kbd>", "</sub>": "<sub>", "</sup>": "<sup>",
}

// rawHTMLTag returns the lowercased bare tag when node is one of the inline
// HTML tags we interpret, e.g. "<u>" from `<u class="x">`.
func rawHTMLTag(node ast.Node, src []byte) (string, bool) {
	raw, ok := node.(*ast.RawHTML)
	if !ok || raw.Segments == nil {
		return "", false
	}
	var buf bytes.Buffer
	for i := 0; i < raw.Segments.Len(); i++ {
		seg := raw.Segments.At(i)
		buf.Write(seg.Value(src))
	}
	tag := buf.String()
	// Strip attributes: <u class="x"> -> <u>
	end := strings.IndexAny(tag, "> \t\n\r")
	name := tag
	if end >= 0 {
		name = tag[:end] + ">"
	}
	name = strings.ToLower(name)
	if strings.HasSuffix(name, "/>") && name != "<br/>" {
		name = strings.TrimSuffix(name, "/>") + ">"
	}
	if name == "<br/>" || name == "<br>" {
		return "<br>", true
	}
	if _, ok := htmlOpen[name]; ok {
		return name, true
	}
	if _, ok := htmlClose[name]; ok {
		return name, true
	}
	return "", false
}

// TransformHTML rewrites the inline tree under root: content between paired
// <u>/<ins>/<mark>/<kbd>/<sub>/<sup> tags is wrapped in the matching node,
// and <br> becomes a hard line break. Unknown tags stay raw HTML.
func TransformHTML(root ast.Node, src []byte) {
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
			transformInlineList(n, src)
		}
		return ast.WalkContinue, nil
	})
}

func transformInlineList(parent ast.Node, src []byte) {
	type openTag struct {
		node ast.Node
		tag  string
	}
	var stack []openTag
	for child := parent.FirstChild(); child != nil; {
		next := child.NextSibling() // captured before any mutation
		tag, ok := rawHTMLTag(child, src)
		if !ok {
			child = next
			continue
		}
		if !ok {
			continue
		}
		if tag == "<br>" {
			br := ast.NewTextSegment(text.NewSegment(0, 0))
			br.SetHardLineBreak(true)
			parent.ReplaceChild(parent, child, br)
			child = next
			continue
		}
		if _, isOpen := htmlOpen[tag]; isOpen {
			stack = append(stack, openTag{child, tag})
			child = next
			continue
		}
		// closing tag: find its opener (innermost first)
		want := htmlClose[tag]
		idx := -1
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].tag == want {
				idx = i
				break
			}
		}
		if idx < 0 {
			child = next
			continue
		}
		opener := stack[idx].node
		stack = stack[:idx]
		wrapper := htmlOpen[want]()
		// Move siblings between opener and child into the wrapper.
		var move []ast.Node
		for sib := opener.NextSibling(); sib != nil && sib != child; sib = sib.NextSibling() {
			move = append(move, sib)
		}
		parent.InsertAfter(parent, opener, wrapper)
		for _, m := range move {
			parent.RemoveChild(parent, m)
			wrapper.AppendChild(wrapper, m)
		}
		parent.RemoveChild(parent, opener)
		parent.RemoveChild(parent, child)
		child = next
	}
}

// --- front matter & TOC ----------------------------------------------------

// SplitFrontMatter splits a leading YAML front matter block (--- ... --- or
// ... ...) from the document. fm includes the fence lines; rest is the
// remaining source. fm is empty when the document has no front matter.
func SplitFrontMatter(src string) (fm, rest string) {
	if !strings.HasPrefix(src, "---\n") && !strings.HasPrefix(src, "---\r\n") {
		return "", src
	}
	lines := strings.SplitAfter(src, "\n")
	var buf strings.Builder
	buf.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimRight(lines[i], "\r\n")
		if trimmed == "---" || trimmed == "..." {
			buf.WriteString(lines[i])
			return buf.String(), src[len(buf.String()):]
		}
		buf.WriteString(lines[i])
	}
	return "", src // unclosed: not front matter
}

// IsTOC reports whether src is a paragraph whose only content is [TOC].
func IsTOC(src string) bool {
	return strings.EqualFold(strings.TrimSpace(src), "[TOC]")
}

// --- emoji ------------------------------------------------------------------

// EmojiFor returns the unicode emoji for a :shortcode:, or false.
func EmojiFor(name string) (string, bool) {
	e, ok := emojidef.Github().Get(name)
	if !ok {
		return "", false
	}
	return string(e.Unicode), true
}

// --- registration -----------------------------------------------------------

// InlineExtender registers the mdext inline parsers (== ^ ~) on m.
// Callers keep using extension.GFM's other parts; register this BEFORE
// extension.GFM so ~ runs before GFM's strikethrough.
type inlineExtender struct{}

// InlineExtender is the mdext inline extension.
var InlineExtender = &inlineExtender{}

// Extend implements goldmark.Extender. Priority: our tildes must run before
// GFM's strikethrough (500), so ~x~ means subscript; == and ^ have no
// competition. HTML transform runs as a post-parse step, not a renderer.
func (e *inlineExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&tildeParser{}, 499),
		util.Prioritized(&caretParser{}, 501),
		util.Prioritized(&equalsParser{}, 501),
	))
}

// spaceFreeCloser reports whether rest contains a closing delim before any
// space (pandoc's rule for ^sup^ and ~sub~).
func spaceFreeCloser(rest []byte, delim byte) bool {
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if c == '\\' {
			i++
			continue
		}
		if c == delim {
			return i > 0
		}
		if c == ' ' || c == '\t' || c == '\n' {
			return false
		}
	}
	return false
}
