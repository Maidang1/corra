package richtext

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var kindMath = ast.NewNodeKind("RichTextProtectedMath")

type mathBlock struct {
	ast.BaseBlock
	closed bool
}

func (n *mathBlock) Kind() ast.NodeKind         { return kindMath }
func (n *mathBlock) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

// Keep display math indivisible even when its body contains blank lines or
// Markdown-looking text. Unknown math stays editable only in source mode.
type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }
func (mathBlockParser) Open(_ ast.Node, r text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, segment := r.PeekLine()
	if !bytes.HasPrefix(line, []byte("$$")) {
		return nil, parser.NoChildren
	}
	n := &mathBlock{closed: bytes.Contains(line[2:], []byte("$$"))}
	n.SetPos(segment.Start)
	n.Lines().Append(segment)
	r.AdvanceToEOL()
	return n, parser.NoChildren
}
func (mathBlockParser) Continue(node ast.Node, r text.Reader, _ parser.Context) parser.State {
	n := node.(*mathBlock)
	if n.closed {
		return parser.Close
	}
	line, segment := r.PeekLine()
	if line == nil {
		return parser.Close
	}
	n.Lines().Append(segment)
	n.closed = bytes.HasSuffix(bytes.TrimSpace(line), []byte("$$"))
	r.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}
func (mathBlockParser) Close(ast.Node, text.Reader, parser.Context) {}
func (mathBlockParser) CanInterruptParagraph() bool                 { return false }
func (mathBlockParser) CanAcceptIndentedLine() bool                 { return false }
