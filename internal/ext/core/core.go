// Package core is the extension every schema starts from: the document,
// paragraphs, line breaks, and raw source for syntax nothing else reads.
package core

import (
	"strings"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

// Extension returns the core extension. Its priority is the lowest, so
// every other extension's key handlers run before its defaults.
func Extension() ext.Extension {
	return ext.Extension{
		Name:     "core",
		Priority: -100,
		Nodes: []doc.NodeSpec{
			{Name: "doc", Content: doc.ContentBlocks, Allow: []string{"block"}},
			{Name: "paragraph", Group: "block", Content: doc.ContentInline},
			// raw keeps source no rule reads, verbatim and editable as text.
			{Name: "raw", Group: "block", Content: doc.ContentText, Defining: true},
			// hard_break is a line break inside a paragraph; soft ones were
			// plain newlines in the source and are written back as such.
			{Name: "hard_break", Content: doc.ContentNone, Inline: true,
				Attrs: map[string]doc.AttrSpec{"soft": {Default: false}}},
		},
		Marks: []doc.MarkSpec{
			// raw_inline is inline source no rule reads, as unknown HTML.
			{Name: "raw_inline", Rank: 100, Excludes: []string{"_"}},
		},
		Markdown: mdcodec.Rules{
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindParagraph: paragraph,
				ast.KindTextBlock: paragraph,
				ast.KindText:      text,
				ast.KindString: func(p *mdcodec.Parser, n ast.Node) {
					p.AddText(string(n.(*ast.String).Value))
				},
				ast.KindRawHTML: rawHTML,
			},
			Writes: map[string]mdcodec.WriteFunc{
				"doc":        func(w *mdcodec.Writer, n *doc.Node) { w.Children(n, false) },
				"paragraph":  func(w *mdcodec.Writer, n *doc.Node) { w.Inline(n.Inline()) },
				"raw":        func(w *mdcodec.Writer, n *doc.Node) { w.Raw(n.Inline().String()) },
				"hard_break": func(w *mdcodec.Writer, n *doc.Node) { w.Break(n.Attrs().Bool("soft")) },
			},
			Marks: map[string]mdcodec.MarkSyntax{
				"raw_inline": {Verbatim: true, Wrap: func(_ doc.Mark, text string) string { return text }},
			},
		},
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			return map[string][]doc.Command{
				"Enter":     {doc.SplitBlock},
				"Backspace": {doc.DeleteSelection, doc.JoinBackward},
				"Delete":    {doc.DeleteSelection, doc.JoinForward},
			}
		},
		Slash: []editor.SlashItem{{Title: "正文", Keywords: "paragraph text body", Command: "paragraph"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"selectAll": doc.SelectAll,
				"paragraph": doc.SetBlockType(reg.Node("paragraph"), nil),
			}
		},
	}
}

func paragraph(p *mdcodec.Parser, n ast.Node) {
	p.Open(p.Node("paragraph"), nil)
	p.Inlines(n)
	p.Close()
}

func text(p *mdcodec.Parser, n ast.Node) {
	t := n.(*ast.Text)
	v := t.Segment.Value(p.Source())
	if !t.IsRaw() {
		v = mdcodec.Decode(v)
	}
	p.AddText(string(v))
	switch {
	case t.HardLineBreak():
		p.AddAtom(doc.NewAtom(p.Node("hard_break"), nil))
	case t.SoftLineBreak():
		p.AddAtom(doc.NewAtom(p.Node("hard_break"), doc.A{"soft": true}))
	}
}

func rawHTML(p *mdcodec.Parser, n ast.Node) {
	var b strings.Builder
	segs := n.(*ast.RawHTML).Segments
	for i := 0; i < segs.Len(); i++ {
		s := segs.At(i)
		b.Write(s.Value(p.Source()))
	}
	p.WithMark(doc.Mark{Type: p.Mark("raw_inline")}, func() { p.AddText(b.String()) })
}
