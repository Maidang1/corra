// Package marks is the inline formatting beyond CommonMark that Typora
// writes: ==highlight==, ^superscript^, ~subscript~, <u>underline</u>,
// <kbd>keys</kbd> and colored <span>s.
package marks

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
)

func Extension() ext.Extension {
	return ext.Extension{
		Name: "marks",
		Marks: []doc.MarkSpec{
			{Name: "highlight", Rank: 30, Inclusive: true},
			{Name: "underline", Rank: 35, Inclusive: true},
			{Name: "color", Rank: 40, Inclusive: true, Attrs: map[string]doc.AttrSpec{
				"color": {Default: "", Validate: validColor}, "background": {Default: "", Validate: validColor},
			}},
			{Name: "sup", Rank: 45, Excludes: []string{"sub"}},
			{Name: "sub", Rank: 46, Excludes: []string{"sup"}},
			{Name: "kbd", Rank: 80},
		},
		Markdown: mdcodec.Rules{
			Extenders:  []goldmark.Extender{mdext.InlineExtender},
			Transforms: []func(ast.Node, []byte){mdext.TransformHTML},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				mdext.KindMark:        mark("highlight"),
				mdext.KindUnderline:   mark("underline"),
				mdext.KindSuperscript: mark("sup"),
				mdext.KindSubscript:   mark("sub"),
				mdext.KindKbd:         mark("kbd"),
				mdext.KindSpan: func(p *mdcodec.Parser, n ast.Node) {
					s := n.(*mdext.Span)
					m := doc.NewMark(p.Mark("color"), doc.A{"color": s.Color, "background": s.Background})
					p.WithMark(m, func() { p.Inlines(n) })
				},
			},
			Marks: map[string]mdcodec.MarkSyntax{
				"highlight": {
					Open:      func(doc.Mark, mdcodec.Style) string { return "==" },
					Close:     func(doc.Mark, mdcodec.Style) string { return "==" },
					Delimited: true,
				},
				"underline": tags("<u>", "</u>"),
				"color": {
					Open: func(m doc.Mark, _ mdcodec.Style) string {
						return mdext.SpanTag(m.Attrs.String("color"), m.Attrs.String("background"))
					},
					Close: func(doc.Mark, mdcodec.Style) string { return "</span>" },
				},
				"sup": short("^", "<sup>", "</sup>"),
				"sub": short("~", "<sub>", "</sub>"),
				"kbd": {PerSegment: true, WrapNear: func(_ doc.Mark, s string, _, _ rune) string { return "<kbd>" + s + "</kbd>" }},
			},
			Escape:     "~^",
			EscapeRuns: "=",
		},
		Views: map[string]editor.ViewSpec{
			"highlight": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) {
				s.Background, s.Radius = ui.Hex("#fde68a"), 3
				if env.Dark {
					s.Background = ui.Hex("#854d0e")
				}
			}},
			"underline": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) { s.Underline = true }},
			"color": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) {
				if c := m.Attrs.String("color"); strings.HasPrefix(c, "#") {
					s.Color = ui.Hex(c)
				}
				if b := m.Attrs.String("background"); strings.HasPrefix(b, "#") {
					s.Background, s.Radius = ui.Hex(b), 3
				}
			}},
			"sup": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) {
				s.Size, s.BaselineShift = env.Font.Size*0.72, env.Font.Size*0.36
			}},
			"sub": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) {
				s.Size, s.BaselineShift = env.Font.Size*0.72, -env.Font.Size*0.14
			}},
			"kbd": {Mark: func(m doc.Mark, env *editor.Env, s *ui.Span) {
				s.Font, s.Size = env.Mono, env.Font.Size*0.85
				s.Background, s.Radius, s.PaddingX = env.Tint(), 4, 4
			}},
		},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"highlight": doc.ToggleMark(ext.Mark(reg, "highlight")),
				"underline": doc.ToggleMark(ext.Mark(reg, "underline")),
				"sup":       doc.ToggleMark(ext.Mark(reg, "sup")),
				"sub":       doc.ToggleMark(ext.Mark(reg, "sub")),
				"kbd":       doc.ToggleMark(ext.Mark(reg, "kbd")),
			}
		},
		Keys: []editor.KeyBinding{
			{Mods: ui.Cmd, Key: ui.KeyU, Command: "underline"},
			{Mods: ui.Cmd | ui.Shift, Key: ui.KeyH, Command: "highlight"},
		},
	}
}

// validColor accepts what a <span style> can carry and read back: a hex
// value or a color name, or nothing.
func validColor(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("color %v", v)
	}
	if s == "" {
		return nil
	}
	if c, _, ok := mdext.SpanStyle(mdext.SpanTag(s, "")); !ok || c != strings.ToLower(s) {
		return fmt.Errorf("color %q", s)
	}
	return nil
}

func mark(name string) mdcodec.ParseFunc {
	return func(p *mdcodec.Parser, n ast.Node) {
		p.WithMark(doc.Mark{Type: p.Mark(name)}, func() { p.Inlines(n) })
	}
}

func tags(open, close string) mdcodec.MarkSyntax {
	return mdcodec.MarkSyntax{
		Open:  func(doc.Mark, mdcodec.Style) string { return open },
		Close: func(doc.Mark, mdcodec.Style) string { return close },
	}
}

// short writes delim+text+delim where it reads back for certain: no
// spaces (pandoc's rule), and letters or digits at both ends (the
// delimiters' flanking rules); the HTML tags otherwise.
func short(delim, open, close string) mdcodec.MarkSyntax {
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	d := []rune(delim)[0]
	return mdcodec.MarkSyntax{PerSegment: true, WrapNear: func(_ doc.Mark, s string, prev, next rune) string {
		rs := []rune(s)
		if len(rs) == 0 || strings.IndexFunc(s, unicode.IsSpace) >= 0 || strings.Contains(s, delim) ||
			!word(rs[0]) || !word(rs[len(rs)-1]) || prev == d || next == d || next == 0 {
			return open + s + close
		}
		return delim + s + delim
	}}
}

// Color returns a command coloring the selection's text and background;
// empty values remove them.
func Color(reg *doc.Registry, color, background string) doc.Command {
	t := ext.Mark(reg, "color")
	if color == "" && background == "" {
		return doc.RemoveMark(t)
	}
	return doc.AddMark(doc.NewMark(t, doc.A{"color": color, "background": background}))
}
