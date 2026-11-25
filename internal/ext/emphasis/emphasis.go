// Package emphasis is the basic inline marks: bold, italic, strikethrough,
// code and links.
package emphasis

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func Extension() ext.Extension {
	return ext.Extension{
		Name: "emphasis",
		Marks: []doc.MarkSpec{
			{Name: "link", Rank: 0, Attrs: map[string]doc.AttrSpec{"href": {Default: ""}, "title": {Default: ""}}},
			{Name: "bold", Rank: 10, Inclusive: true},
			{Name: "italic", Rank: 20, Inclusive: true},
			{Name: "strike", Rank: 25, Inclusive: true},
			// Code is innermost; Markdown allows any formatting around it.
			{Name: "code", Rank: 90},
		},
		Markdown: mdcodec.Rules{
			Extenders: []goldmark.Extender{extension.Strikethrough, extension.Linkify},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindEmphasis: func(p *mdcodec.Parser, n ast.Node) {
					name := "italic"
					if n.(*ast.Emphasis).Level == 2 {
						name = "bold"
					}
					p.WithMark(doc.Mark{Type: p.Mark(name)}, func() { p.Inlines(n) })
				},
				east.KindStrikethrough: func(p *mdcodec.Parser, n ast.Node) {
					p.WithMark(doc.Mark{Type: p.Mark("strike")}, func() { p.Inlines(n) })
				},
				ast.KindCodeSpan: func(p *mdcodec.Parser, n ast.Node) {
					var b strings.Builder
					for c := n.FirstChild(); c != nil; c = c.NextSibling() {
						switch t := c.(type) {
						case *ast.Text:
							b.Write(t.Segment.Value(p.Source()))
						case *ast.String:
							b.Write(t.Value)
						}
					}
					p.WithMark(doc.Mark{Type: p.Mark("code")}, func() { p.AddText(b.String()) })
				},
				ast.KindLink: func(p *mdcodec.Parser, n ast.Node) {
					l := n.(*ast.Link)
					m := doc.NewMark(p.Mark("link"), doc.A{"href": string(l.Destination), "title": string(l.Title)})
					p.WithMark(m, func() { p.Inlines(n) })
				},
				ast.KindAutoLink: func(p *mdcodec.Parser, n ast.Node) {
					l := n.(*ast.AutoLink)
					label := string(l.Label(p.Source()))
					href := string(l.URL(p.Source()))
					if l.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(href, "mailto:") {
						href = "mailto:" + href
					}
					p.WithMark(doc.NewMark(p.Mark("link"), doc.A{"href": href}), func() { p.AddText(label) })
				},
			},
			Marks: map[string]mdcodec.MarkSyntax{
				"bold":   alternated(delimited(func(st mdcodec.Style) string { return st.Strong }), "**", "__"),
				"italic": alternated(delimited(func(st mdcodec.Style) string { return st.Emphasis }), "*", "_"),
				"strike": delimited(func(mdcodec.Style) string { return "~~" }),
				"code":   {Verbatim: true, Wrap: func(_ doc.Mark, text string) string { return CodeSpan(text) }},
				"link": {
					Open: func(doc.Mark, mdcodec.Style) string { return "[" },
					Close: func(m doc.Mark, _ mdcodec.Style) string {
						return "]" + mdcodec.Destination(m.Attrs.String("href"), m.Attrs.String("title"))
					},
				},
			},
		},
		Keys: []editor.KeyBinding{
			{Mods: ui.Cmd, Key: ui.KeyB, Command: "bold"},
			{Mods: ui.Cmd, Key: ui.KeyI, Command: "italic"},
			{Mods: ui.Cmd | ui.Shift, Key: ui.KeyX, Command: "strikethrough"},
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			mark := func(name string) func() doc.Mark {
				return func() doc.Mark { return doc.Mark{Type: ext.Mark(reg, name)} }
			}
			return []editor.InputRule{
				editor.MarkRule(regexp.MustCompile(`(^|[^*\\])\*\*([^*\s](?:[^*]*[^*\s])?)\*\*$`), mark("bold")),
				editor.MarkRule(regexp.MustCompile(`(^|[^*\\])\*([^*\s](?:[^*]*[^*\s])?)\*$`), mark("italic")),
				editor.MarkRule(regexp.MustCompile(`(^|[^~\\])~~([^~\s](?:[^~]*[^~\s])?)~~$`), mark("strike")),
				editor.MarkRule(regexp.MustCompile("(^|[^`\\\\])`([^`]+)`$"), mark("code")),
			}
		},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"bold":          doc.ToggleMark(ext.Mark(reg, "bold")),
				"italic":        doc.ToggleMark(ext.Mark(reg, "italic")),
				"strikethrough": doc.ToggleMark(ext.Mark(reg, "strike")),
				"code":          doc.ToggleMark(ext.Mark(reg, "code")),
				"unlink":        doc.RemoveMark(ext.Mark(reg, "link")),
			}
		},
	}
}

func delimited(d func(mdcodec.Style) string) mdcodec.MarkSyntax {
	return mdcodec.MarkSyntax{
		Open:      func(_ doc.Mark, st mdcodec.Style) string { return d(st) },
		Close:     func(_ doc.Mark, st mdcodec.Style) string { return d(st) },
		Delimited: true,
	}
}

// alternated gives a delimiter the other of a and b as its alternative.
func alternated(syn mdcodec.MarkSyntax, a, b string) mdcodec.MarkSyntax {
	syn.Alt = func(m doc.Mark, st mdcodec.Style) (string, string) {
		d := syn.Open(m, st)
		if d == a {
			d = b
		} else {
			d = a
		}
		return d, d
	}
	return syn
}

// Link returns a command linking the selection to href.
func Link(reg *doc.Registry, href string) doc.Command {
	return doc.AddMark(doc.NewMark(ext.Mark(reg, "link"), doc.A{"href": href}))
}

// CodeSpan wraps text in the fewest backticks that occur in it as no run
// of the same length, padded with spaces where the text starts or ends
// with a backtick or is space at both ends.
func CodeSpan(text string) string {
	runs := map[int]bool{}
	run := 0
	for _, r := range text + "x" {
		if r == '`' {
			run++
			continue
		}
		if run > 0 {
			runs[run] = true
		}
		run = 0
	}
	n := 1
	for runs[n] {
		n++
	}
	ticks := strings.Repeat("`", n)
	if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") ||
		(strings.HasPrefix(text, " ") && strings.HasSuffix(text, " ") && strings.TrimSpace(text) != "") {
		text = " " + text + " "
	}
	return ticks + text + ticks
}
