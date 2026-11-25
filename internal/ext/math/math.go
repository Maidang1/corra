// Package math is TeX math: "$inline$" and "$$" display blocks, typeset
// by the embedded MathJax.
package math

import (
	"bytes"
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
	"github.com/Maidang1/corra/render"
)

// TeX returns a formula's source: an inline atom's attribute, a block's
// text.
func TeX(n *doc.Node) string {
	if n.Type().IsTextBlock() {
		return n.Inline().String()
	}
	return n.Attrs().String("tex")
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "math",
		Nodes: []doc.NodeSpec{
			// An inline formula always has TeX: "$$" has no meaning inline.
			{Name: "math_inline", Content: doc.ContentNone, Inline: true, Attrs: map[string]doc.AttrSpec{"tex": {Validate: nonEmpty}}},
			// A display formula's text is its TeX, edited as plain text, with
			// the typeset formula under it.
			{Name: "math_block", Group: "block", Content: doc.ContentText, Defining: true},
		},
		Markdown: mdcodec.Rules{
			BlockParsers:  []util.PrioritizedValue{util.Prioritized(blockParser{}, 50)},
			InlineParsers: []util.PrioritizedValue{util.Prioritized(inlineParser{}, 50)},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				kindBlock: func(p *mdcodec.Parser, n ast.Node) {
					p.AddNode(doc.NewTextBlock(p.Node("math_block"), nil, doc.Text(n.(*blockNode).tex())))
				},
				kindInline: func(p *mdcodec.Parser, n ast.Node) {
					p.AddAtom(doc.NewAtom(p.Node("math_inline"), doc.A{"tex": n.(*inlineNode).tex}))
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"math_inline": func(w *mdcodec.Writer, n *doc.Node) {
					// "$…$" holds one line without spaces at its ends; an empty
					// formula has no spelling and is dropped.
					tex := strings.TrimSpace(strings.ReplaceAll(TeX(n), "\n", " "))
					if tex != "" {
						w.Raw("$" + tex + "$")
						// A digit right after the closing "$" keeps it from
						// closing.
						w.EncodeNext()
					}
				},
				"math_block": func(w *mdcodec.Writer, n *doc.Node) {
					w.Raw("$$")
					w.Newline()
					if tex := TeX(n); tex != "" {
						w.Raw(tex)
						w.Newline()
					}
					w.Raw("$$")
				},
			},
			Escape: "$",
		},
		Views: map[string]editor.ViewSpec{
			"math_inline": {Inline: inlineView},
			"math_block": {Text: func(n *doc.Node, env *editor.Env) editor.TextLook {
				l := editor.TextLook{
					Font: ui.Font{Family: env.Mono, Size: env.Font.Size * 0.85, LineHeight: 1.5}, Color: env.Theme.TextMuted,
					Background: env.Theme.Surface, Radius: 6, PadX: 12, PadY: 8, After: env.Font.Size * 0.6,
				}
				tex := TeX(n)
				if env.MathNumbering == "all" && env.Ordinals != nil && strings.TrimSpace(tex) != "" && !strings.Contains(tex, `\tag`) {
					tex += fmt.Sprintf(`\tag{%d}`, env.Ordinals.Ordinal(n))
				}
				l.Below = func(width float32) (float32, func(*ui.Painter, ui.Rect, *editor.Env)) {
					img := typeset(tex, true, env)
					if img.bmp == nil {
						return 0, nil
					}
					w, h := img.size()
					return h + 16, func(p *ui.Painter, r ui.Rect, env *editor.Env) {
						fw, fh := w, h
						if fw > r.W-16 {
							fh, fw = fh*(r.W-16)/fw, r.W-16
						}
						p.Image(img.bmp, ui.Rect{X: r.X + (r.W-fw)/2, Y: r.Y + 8, W: fw, H: fh}, ui.Contain)
					}
				}
				return l
			}},
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{
				editor.PrefixRule(regexp.MustCompile(`^\$\$ $`), func([]string) doc.Command {
					return doc.SetBlockType(ext.Node(reg, "math_block"), nil)
				}),
				{
					Match: regexp.MustCompile(`(^|[^$\\])\$([^$\s](?:[^$]*[^$\s])?)\$$`).FindStringSubmatch,
					Handle: func(s doc.State, tx *doc.Tx, m []string) bool {
						sel, ok := s.TextSel()
						if !ok || !sel.Empty() {
							return false
						}
						leaf := s.Node(sel.Head.Leaf)
						t := ext.Node(reg, "math_inline")
						if !leaf.Type().Allows(t) {
							return false
						}
						end := sel.Head.Offset
						start := end - doc.UTF16Len(m[0]) + doc.UTF16Len(m[1])
						tx.ReplaceText(leaf.ID(), start, end, doc.AtomInline(doc.NewAtom(t, doc.A{"tex": m[2]})))
						tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID(), Offset: start + 1}))
						return true
					},
				},
			}
		},
		Slash: []editor.SlashItem{{Title: "公式块", Hint: "$$", Keywords: "math formula latex tex equation", Command: "mathBlock"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"mathBlock": doc.SetBlockType(ext.Node(reg, "math_block"), nil),
				"editMath":  EditInline(reg),
			}
		},
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			// Backspace right after an inline formula opens it as "$tex$"
			// text to edit, rather than deleting it.
			return map[string][]doc.Command{"Backspace": {EditInline(reg)}}
		},
	}
}

func nonEmpty(v any) error {
	if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
		return fmt.Errorf("empty formula")
	}
	return nil
}

// EditInline turns the inline formula before the caret back into its
// "$tex$" source.
func EditInline(reg *doc.Registry) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !sel.Empty() || sel.Head.Offset == 0 {
			return false
		}
		leaf := s.Node(sel.Head.Leaf)
		at := leaf.Inline().Slice(sel.Head.Offset-1, sel.Head.Offset)
		if at.NumSegments() != 1 || at.Segment(0).Atom == nil || at.Segment(0).Atom.Type().Name() != "math_inline" {
			return false
		}
		src := "$" + TeX(at.Segment(0).Atom) + "$"
		tx.ReplaceText(leaf.ID(), sel.Head.Offset-1, sel.Head.Offset, doc.Text(src, at.Segment(0).Marks...))
		tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID(), Offset: sel.Head.Offset - 1 + doc.UTF16Len(src) - 1}))
		return true
	}
}

func inlineView(n *doc.Node, env *editor.Env) editor.InlineLook {
	tex := TeX(n)
	img := typeset(tex, false, env)
	if img.bmp == nil {
		// Unreadable TeX shows as its source.
		src := "$" + tex + "$"
		w := float32(len([]rune(src))) * env.Font.Size * 0.55
		return editor.InlineLook{Width: w, Height: env.Font.Size, Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
			p.Text(r.X, r.Y+r.H*0.85, src, env.Font.Size*0.9, env.Theme.Danger)
		}}
	}
	w, h := img.size()
	// Formulas sit on the baseline with about a third below it.
	return editor.InlineLook{Width: w, Height: h * 0.7, Descent: h * 0.3, Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
		p.Image(img.bmp, r, ui.Contain)
	}}
}

// formulas caches typeset formulas by source, display, size and color,
// not by node: undoing a deletion finds its formula ready.
var formulas = struct {
	sync.Mutex
	m map[string]*formula
}{m: map[string]*formula{}}

type formula struct {
	bmp  *ui.Bitmap
	w, h float32
}

func (f *formula) size() (float32, float32) { return f.w, f.h }

func typeset(tex string, display bool, env *editor.Env) *formula {
	fg := color.RGBA{0x1d, 0x1d, 0x1f, 0xff}
	if env.Dark {
		fg = color.RGBA{0xee, 0xee, 0xf0, 0xff}
	}
	key := fmt.Sprintf("%t|%v|%v|%s", display, env.Font.Size, env.Dark, tex)
	formulas.Lock()
	f, ok := formulas.m[key]
	formulas.Unlock()
	if ok {
		return f
	}
	f = &formula{}
	if strings.TrimSpace(tex) != "" {
		res, err := render.RenderMathBatch([]render.MathExpression{{TeX: tex, Display: display}}, render.MathOptions{}, float64(env.Font.Size), fg)
		if err == nil && len(res) == 1 && res[0].Error == nil && res[0].Image != nil {
			b := res[0].Image.Bounds()
			f.bmp, f.w, f.h = ui.NewBitmap(res[0].Image), float32(b.Dx()), float32(b.Dy())
		}
	}
	formulas.Lock()
	formulas.m[key] = f
	formulas.Unlock()
	return f
}

var (
	kindBlock  = ast.NewNodeKind("Math")
	kindInline = ast.NewNodeKind("InlineMath")
)

type blockNode struct {
	ast.BaseBlock
	lines  []string
	closed bool
}

func (n *blockNode) tex() string                { return strings.Join(n.lines, "\n") }
func (n *blockNode) Kind() ast.NodeKind         { return kindBlock }
func (n *blockNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

// blockParser reads "$$" ... "$$" as one formula, blank lines and all.
type blockParser struct{}

func (blockParser) Trigger() []byte { return []byte{'$'} }
func (blockParser) Open(_ ast.Node, r text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, seg := r.PeekLine()
	trimmed := bytes.TrimRight(line, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("$$")) {
		return nil, parser.NoChildren
	}
	n := &blockNode{}
	n.SetPos(seg.Start)
	n.Lines().Append(seg)
	rest := trimmed[2:]
	if i := bytes.Index(rest, []byte("$$")); i >= 0 {
		// "$$x$$" on one line.
		n.lines, n.closed = []string{string(bytes.TrimSpace(rest[:i]))}, true
	} else if len(bytes.TrimSpace(rest)) > 0 {
		n.lines = []string{string(bytes.TrimSpace(rest))}
	}
	r.AdvanceToEOL()
	return n, parser.NoChildren
}
func (blockParser) Continue(node ast.Node, r text.Reader, _ parser.Context) parser.State {
	n := node.(*blockNode)
	if n.closed {
		return parser.Close
	}
	line, seg := r.PeekLine()
	if line == nil {
		return parser.Close
	}
	n.Lines().Append(seg)
	trimmed := bytes.TrimRight(line, " \t\r\n")
	r.AdvanceToEOL()
	if bytes.HasSuffix(trimmed, []byte("$$")) {
		n.closed = true
		if body := bytes.TrimRight(trimmed[:len(trimmed)-2], " \t"); len(body) > 0 {
			n.lines = append(n.lines, string(body))
		}
		return parser.Close
	}
	n.lines = append(n.lines, string(trimmed))
	return parser.Continue | parser.NoChildren
}
func (blockParser) Close(ast.Node, text.Reader, parser.Context) {}
func (blockParser) CanInterruptParagraph() bool                 { return false }
func (blockParser) CanAcceptIndentedLine() bool                 { return false }

type inlineNode struct {
	ast.BaseInline
	tex string
}

func (n *inlineNode) Kind() ast.NodeKind         { return kindInline }
func (n *inlineNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type inlineParser struct{}

func (inlineParser) Trigger() []byte { return []byte{'$'} }
func (inlineParser) Parse(parent ast.Node, reader text.Reader, _ parser.Context) ast.Node {
	line, _ := reader.PeekLine()
	tex, length := mdext.MathSpan(line)
	if length == 0 {
		return nil
	}
	reader.Advance(length)
	return &inlineNode{tex: mdext.MathText(parent, tex)}
}
