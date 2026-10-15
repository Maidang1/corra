package render

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/mdext"
)

// mathExt is a small goldmark extension that lifts TeX math out of the
// Markdown grammar: inline $...$ and block $$...$$ (including $$ that opens
// mid-line). It replaces the heavier goldmark-katex dependency, whose
// renderer embeds a JavaScript engine. TeX is typeset separately by the offline
// document renderer, then painted as native attachments or exported images.

type mathInlineNode struct {
	gast.BaseInline
	TeX     string
	Raw     string
	Display bool
}

// KindMathInlineNode is the AST node kind of inline math.
var KindMathInlineNode = gast.NewNodeKind("MathInline")

func (n *mathInlineNode) Kind() gast.NodeKind { return KindMathInlineNode }

func (n *mathInlineNode) Text(_ []byte) []byte { return []byte(n.Raw) }

func (n *mathInlineNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"MathInline": n.TeX}, nil)
}

type mathBlockNode struct {
	gast.BaseBlock
	TeX    string
	lines  [][]byte // raw lines between the $$ fences
	open   bool     // still looking for the closing $$
	single bool     // TeX was captured completely by Open
	done   bool     // Close ran; goldmark may close a block again at EOF
}

// KindMathBlockNode is the AST node kind of block math.
var KindMathBlockNode = gast.NewNodeKind("MathBlock")

func (n *mathBlockNode) Kind() gast.NodeKind { return KindMathBlockNode }

func (n *mathBlockNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"MathBlock": n.TeX}, nil)
}

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$'} }

func (p mathInlineParser) Parse(parent gast.Node, r text.Reader, _ parser.Context) gast.Node {
	line, _ := r.PeekLine()
	teX, adv := mdext.MathSpan(line)
	if adv == 0 {
		return nil
	}
	r.Advance(adv)
	n := &mathInlineNode{TeX: mdext.MathText(parent, teX), Raw: string(line[:adv]), Display: bytes.HasPrefix(line, []byte("$$"))}
	// Goldmark extracts image alt text from Text/String descendants only.
	literal := gast.NewString([]byte(n.Raw))
	literal.SetCode(true)
	n.AppendChild(n, literal)
	return n
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (p mathBlockParser) Open(_ gast.Node, r text.Reader, _ parser.Context) (gast.Node, parser.State) {
	line, segment := r.PeekLine()
	if !bytes.HasPrefix(line, []byte("$$")) {
		return nil, parser.NoChildren
	}
	n := &mathBlockNode{}
	n.Lines().Append(segment)
	rest := line[2:]
	if i := bytes.Index(rest, []byte("$$")); i >= 0 {
		// Single-line $$...$$ block.
		n.TeX = string(bytes.TrimSpace(rest[:i]))
		n.open = false
		n.single = true
		r.AdvanceToEOL()
		return n, parser.NoChildren
	}
	// Multi-line: consume the opening line, gather in Continue.
	n.open = true
	if len(bytes.TrimSpace(rest)) > 0 {
		n.lines = append(n.lines, rest)
	}
	r.AdvanceToEOL()
	return n, parser.NoChildren
}

func (p mathBlockParser) Continue(node gast.Node, r text.Reader, _ parser.Context) parser.State {
	n := node.(*mathBlockNode)
	if !n.open {
		return parser.Close
	}
	l, segment := r.PeekLine()
	if l == nil {
		return parser.Close
	}
	n.Lines().Append(segment)
	t := bytes.TrimSpace(l)
	if bytes.HasSuffix(t, []byte("$$")) {
		body := bytes.TrimSuffix(t, []byte("$$"))
		if len(bytes.TrimSpace(body)) > 0 {
			n.lines = append(n.lines, body)
		}
		n.open = false
		r.AdvanceToEOL()
		return parser.Close
	}
	n.lines = append(n.lines, l)
	r.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p mathBlockParser) Close(node gast.Node, _ text.Reader, _ parser.Context) {
	n := node.(*mathBlockNode)
	if n.done {
		return
	}
	n.done = true
	if n.open {
		// Retain unclosed source as a literal block rather than losing it.
		n.lines = nil
		return
	}
	if n.single {
		return
	}
	var buf bytes.Buffer
	for i, l := range n.lines {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.Write(bytes.TrimSpace(l))
	}
	n.TeX = buf.String()
	n.lines = nil
}

func (p mathBlockParser) CanInterruptParagraph() bool { return false }

func (p mathBlockParser) CanAcceptIndentedLine() bool { return false }

type mathExtender struct{}

// MathExtender is the goldmark extension for $...$ / $$...$$ math.
var MathExtender = mathExtender{}

func (mathExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(mathInlineParser{}, 501),
		),
		parser.WithBlockParsers(
			util.Prioritized(mathBlockParser{}, 101),
		),
	)
}
