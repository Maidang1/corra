package render

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mathExt is a small goldmark extension that lifts TeX math out of the
// Markdown grammar: inline $...$ and block $$...$$ (including $$ that opens
// mid-line). It replaces the heavier goldmark-katex dependency, whose
// renderer embeds a JavaScript engine; we only ever need the parsing, and
// render the TeX ourselves with go-latex (see views/math.go).

type mathInlineNode struct {
	gast.BaseInline
	TeX string
}

// KindMathInlineNode is the AST node kind of inline math.
var KindMathInlineNode = gast.NewNodeKind("MathInline")

func (n *mathInlineNode) Kind() gast.NodeKind { return KindMathInlineNode }

func (n *mathInlineNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"MathInline": n.TeX}, nil)
}

type mathBlockNode struct {
	gast.BaseBlock
	TeX   string
	lines [][]byte // raw lines between the $$ fences
	open  bool     // still looking for the closing $$
	done  bool     // Close ran; goldmark may close a block again at EOF
}

// KindMathBlockNode is the AST node kind of block math.
var KindMathBlockNode = gast.NewNodeKind("MathBlock")

func (n *mathBlockNode) Kind() gast.NodeKind { return KindMathBlockNode }

func (n *mathBlockNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{"MathBlock": n.TeX}, nil)
}

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$'} }

func (p mathInlineParser) Parse(_ gast.Node, r text.Reader, _ parser.Context) gast.Node {
	line, _ := r.PeekLine()
	if len(line) == 0 || line[0] != '$' {
		return nil
	}
	rest := line[1:]
	var teX []byte
	var adv int
	if len(rest) > 0 && rest[0] == '$' {
		// $$ display math in place: scan for the closing $$.
		body := rest[1:]
		closeIdx := bytes.Index(body, []byte("$$"))
		if closeIdx < 0 {
			return nil
		}
		teX = body[:closeIdx]
		adv = 2 + closeIdx + 2
	} else {
		// $ inline math: scan for the next $.
		body := rest
		closeIdx := bytes.IndexByte(body, '$')
		if closeIdx < 0 {
			return nil
		}
		teX = body[:closeIdx]
		adv = 1 + closeIdx + 1
	}
	if len(bytes.TrimSpace(teX)) == 0 {
		return nil
	}
	r.Advance(adv)
	return &mathInlineNode{TeX: string(teX)}
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (p mathBlockParser) Open(_ gast.Node, r text.Reader, _ parser.Context) (gast.Node, parser.State) {
	line, _ := r.PeekLine()
	if !bytes.HasPrefix(line, []byte("$$")) {
		return nil, parser.NoChildren
	}
	n := &mathBlockNode{}
	rest := line[2:]
	if i := bytes.Index(rest, []byte("$$")); i >= 0 {
		// Single-line $$...$$ block.
		n.TeX = string(bytes.TrimSpace(rest[:i]))
		n.open = false
		r.AdvanceToEOL()
		return n, parser.NoChildren
	}
	// Multi-line: consume the opening line, gather in Continue.
	n.open = true
	r.AdvanceToEOL()
	return n, parser.NoChildren
}

func (p mathBlockParser) Continue(node gast.Node, r text.Reader, _ parser.Context) parser.State {
	n := node.(*mathBlockNode)
	l, _ := r.PeekLine()
	if l == nil {
		return parser.Close
	}
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
		// Never closed: not math after all. Leave TeX empty; buildBlock
		// drops empty math blocks.
		n.lines = nil
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
