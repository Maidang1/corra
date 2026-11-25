// Package heading is ATX headings, "#" to "######".
package heading

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func validLevel(v any) error {
	if l, ok := v.(int); !ok || l < 1 || l > 6 {
		return fmt.Errorf("heading level %v", v)
	}
	return nil
}

// Level returns a heading's level.
func Level(n *doc.Node) int { return n.Attrs().Int("level") }

// noBreaks turns line breaks in a heading into spaces: a Markdown heading
// is one line.
func noBreaks(n *doc.Node, fix *doc.Fixer) {
	in := n.Inline()
	for i, off := 0, 0; i < in.NumSegments(); i++ {
		seg := in.Segment(i)
		if seg.Atom != nil && seg.Atom.Type().Name() == "hard_break" {
			fix.ReplaceText(n.ID(), off, off+1, doc.Text(" ", seg.Marks...))
			return
		}
		if seg.Atom != nil {
			off++
		} else {
			off += doc.UTF16Len(seg.Text)
		}
	}
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "heading",
		Nodes: []doc.NodeSpec{{
			Name: "heading", Group: "block", Content: doc.ContentInline, Defining: true,
			Attrs:     map[string]doc.AttrSpec{"level": {Default: 1, Validate: validLevel}},
			Normalize: noBreaks,
		}},
		Markdown: mdcodec.Rules{
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindHeading: func(p *mdcodec.Parser, n ast.Node) {
					p.Open(p.Node("heading"), doc.A{"level": n.(*ast.Heading).Level})
					p.Inlines(n)
					p.Close()
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"heading": func(w *mdcodec.Writer, n *doc.Node) {
					w.Raw(strings.Repeat("#", Level(n)))
					if n.TextLen() > 0 {
						w.Raw(" ")
						w.Inline(n.Inline())
						// A final "#" would read as an optional closing
						// sequence and vanish.
						if w.LastRune() == '#' {
							w.EncodeLast()
						}
					}
				},
			},
		},
		Slash: []editor.SlashItem{
			{Title: "一级标题", Hint: "#", Keywords: "h1 heading", Command: "heading1"},
			{Title: "二级标题", Hint: "##", Keywords: "h2 heading", Command: "heading2"},
			{Title: "三级标题", Hint: "###", Keywords: "h3 heading", Command: "heading3"},
		},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			cmds := map[string]doc.Command{}
			h := ext.Node(reg, "heading")
			for l := 1; l <= 6; l++ {
				cmds[fmt.Sprintf("heading%d", l)] = doc.SetBlockType(h, doc.A{"level": l})
			}
			return cmds
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{editor.PrefixRule(regexp.MustCompile(`^(#{1,6}) $`), func(m []string) doc.Command {
				return doc.SetBlockType(ext.Node(reg, "heading"), doc.A{"level": len(m[1])})
			})}
		},
		Keys: headingKeys(),
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			// Backspace at the start of a heading makes it a paragraph.
			return map[string][]doc.Command{"Backspace": {func(s doc.State, tx *doc.Tx) bool {
				sel, ok := s.TextSel()
				if !ok || !sel.Empty() || sel.Head.Offset != 0 {
					return false
				}
				if n := s.Node(sel.Head.Leaf); n.Type().Name() != "heading" {
					return false
				}
				return doc.SetBlockType(ext.Node(reg, "paragraph"), nil)(s, tx)
			}}}
		},
	}
}

// headingKeys binds Cmd+Alt+1 to Cmd+Alt+6 to the heading levels, and
// Cmd+Alt+0 to a paragraph.
func headingKeys() []editor.KeyBinding {
	digits := []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6}
	keys := []editor.KeyBinding{{Mods: ui.Cmd | ui.Alt, Key: digits[0], Command: "paragraph"}}
	for l := 1; l <= 6; l++ {
		keys = append(keys, editor.KeyBinding{Mods: ui.Cmd | ui.Alt, Key: digits[l], Command: fmt.Sprintf("heading%d", l)})
	}
	return keys
}
