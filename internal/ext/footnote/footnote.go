// Package footnote is footnotes: references "[^label]" in text, and
// definitions "[^label]: text" kept where they are written.
package footnote

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func validLabel(v any) error {
	s, ok := v.(string)
	if !ok || s == "" || strings.ContainsAny(s, "[]\\ \t\r\n") {
		return fmt.Errorf("footnote label %q", v)
	}
	return nil
}

// Label returns a reference's or a definition's label.
func Label(n *doc.Node) string { return n.Attrs().String("label") }

func Extension() ext.Extension {
	return ext.Extension{
		Name: "footnote",
		Nodes: []doc.NodeSpec{
			{Name: "footnote_ref", Content: doc.ContentNone, Inline: true, Attrs: map[string]doc.AttrSpec{"label": {Validate: validLabel}}},
			{Name: "footnote_def", Group: "block", Content: doc.ContentBlocks, Allow: []string{"block"}, Isolating: true,
				Attrs: map[string]doc.AttrSpec{"label": {Validate: validLabel}}},
		},
		Markdown: mdcodec.Rules{
			BlockParsers:  []util.PrioritizedValue{util.Prioritized(defParser{}, 49)},
			InlineParsers: []util.PrioritizedValue{util.Prioritized(refParser{}, 49)},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				kindRef: func(p *mdcodec.Parser, n ast.Node) {
					p.AddAtom(doc.NewAtom(p.Node("footnote_ref"), doc.A{"label": n.(*refNode).label}))
				},
				kindDef: func(p *mdcodec.Parser, n ast.Node) {
					d := n.(*defNode)
					p.Open(p.Node("footnote_def"), doc.A{"label": d.label})
					for _, b := range p.Sub(defBody(p.Lines(n))) {
						p.AddNode(b)
					}
					p.Close()
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"footnote_ref": func(w *mdcodec.Writer, n *doc.Node) { w.Raw("[^" + Label(n) + "]") },
				"footnote_def": func(w *mdcodec.Writer, n *doc.Node) {
					w.WithPrefix("[^"+Label(n)+"]: ", "    ", func() { w.Children(n, false) })
				},
			},
		},
		Views: map[string]editor.ViewSpec{
			"footnote_ref": {Inline: func(n *doc.Node, env *editor.Env) editor.InlineLook {
				label := Label(n)
				f := env.Font
				f.Size *= 0.7
				w, _ := ui.ShapeText(label, f, 0).Size()
				// A superscript: its baseline raised off the line's.
				return editor.InlineLook{Width: w + 4, Height: env.Font.Size, Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
					editor.DrawLabel(p, r.X+2, r.Y+r.H-env.Font.Size*0.38, label, f, env.Theme.Accent)
				}}
			}},
			"footnote_def": {Container: func(n *doc.Node, env *editor.Env) editor.ContainerLook {
				label := Label(n)
				return editor.ContainerLook{
					Indent: env.Font.Size * 2.2,
					After:  env.Font.Size * 0.6,
					// A note reads as a note, not as body text.
					Color: env.Theme.TextMuted,
					Marker: func(i int, _ *doc.Node) string {
						if i == 0 {
							return label + "."
						}
						return ""
					},
				}
			}},
		},
		Slash: []editor.SlashItem{{Title: "脚注", Hint: "[^1]", Keywords: "footnote note", Command: "footnote"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{"footnote": Insert(reg)}
		},
	}
}

// Insert puts a reference with the next free number at the caret and its
// empty definition at the end of the document.
func Insert(reg *doc.Registry) doc.Command {
	ref, def := ext.Node(reg, "footnote_ref"), ext.Node(reg, "footnote_def")
	return func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !s.Node(sel.Head.Leaf).Type().Allows(ref) {
			return false
		}
		used := map[string]bool{}
		s.Doc.Walk(func(n *doc.Node) bool {
			if n.Type() == def {
				used[Label(n)] = true
			}
			return true
		})
		n := 1
		for used[strconv.Itoa(n)] {
			n++
		}
		label := strconv.Itoa(n)
		if !sel.Empty() {
			doc.DeleteSelection(s, tx)
		}
		at := tx.State().Sel.(doc.TextSelection).Head
		tx.ReplaceText(at.Leaf, at.Offset, at.Offset, doc.AtomInline(doc.NewAtom(ref, doc.A{"label": label})))
		d := doc.NewBlock(def, doc.A{"label": label}, doc.Empty(ext.Node(reg, "paragraph")))
		root := tx.State().Doc
		tx.ReplaceChildren(root.ID(), root.NumChildren(), root.NumChildren(), d)
		tx.SetSelection(doc.Caret(doc.Point{Leaf: at.Leaf, Offset: at.Offset + 1}))
		return true
	}
}

// defBody removes a definition's marker and its continuation lines'
// indentation, leaving its Markdown.
func defBody(raw string) string {
	lines := strings.Split(raw, "\n")
	first := strings.TrimLeft(lines[0], " ")
	_, end := token(first)
	lines[0] = strings.TrimLeft(first[end+1:], " \t")
	for i := 1; i < len(lines); i++ {
		if w := indentWidth(lines[i]); w > 0 {
			lines[i] = lines[i][w:]
		}
	}
	return strings.Join(lines, "\n")
}

var (
	kindDef = ast.NewNodeKind("FootnoteDefinition")
	kindRef = ast.NewNodeKind("FootnoteReference")
)

type defNode struct {
	ast.BaseBlock
	label string
}

func (n *defNode) Kind() ast.NodeKind         { return kindDef }
func (n *defNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

// defParser keeps a definition where it is, its lines raw; they are read
// again as the definition's blocks.
type defParser struct{}

func (defParser) Trigger() []byte { return []byte{'['} }
func (defParser) Open(_ ast.Node, r text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, seg := r.PeekLine()
	trimmed := strings.TrimLeft(string(line), " ")
	label, end := token(trimmed)
	if end != 0 {
		end += len(line) - len(trimmed)
	}
	if end == 0 || end >= len(line) || line[end] != ':' {
		return nil, parser.NoChildren
	}
	n := &defNode{label: label}
	n.SetPos(seg.Start)
	n.Lines().Append(seg)
	r.AdvanceToEOL()
	return n, parser.NoChildren
}
func (defParser) Continue(node ast.Node, r text.Reader, _ parser.Context) parser.State {
	line, seg := r.PeekLine()
	if len(line) == 0 || (strings.TrimSpace(string(line)) != "" && indentWidth(string(line)) == 0) {
		return parser.Close
	}
	node.Lines().Append(seg)
	r.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}
func (defParser) Close(ast.Node, text.Reader, parser.Context) {}
func (defParser) CanInterruptParagraph() bool                 { return true }
func (defParser) CanAcceptIndentedLine() bool                 { return false }

type refNode struct {
	ast.BaseInline
	label string
}

func (n *refNode) Kind() ast.NodeKind         { return kindRef }
func (n *refNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type refParser struct{}

func (refParser) Trigger() []byte { return []byte{'['} }
func (refParser) Parse(_ ast.Node, r text.Reader, _ parser.Context) ast.Node {
	line, seg := r.PeekLine()
	label, end := token(string(line))
	if end == 0 {
		return nil
	}
	n := &refNode{label: label}
	n.SetPos(seg.Start)
	r.Advance(end)
	return n
}

// token reads "[^label]" at the start of line: the label and the length.
func token(line string) (string, int) {
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

// indentWidth returns the bytes of a continuation line's four-column
// indentation, or 0 when it has less.
func indentWidth(line string) int {
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
