// Package toc is "[TOC]", a paragraph that stands for the table of
// contents.
package toc

import (
	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
)

func paragraph(p *mdcodec.Parser, n ast.Node) {
	if mdext.IsTOC(p.Lines(n)) {
		p.AddNode(doc.NewAtom(p.Node("toc"), nil))
		return
	}
	p.Open(p.Node("paragraph"), nil)
	p.Inlines(n)
	p.Close()
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "toc",
		// Above core: it reads paragraphs first.
		Priority: 5,
		Nodes:    []doc.NodeSpec{{Name: "toc", Group: "block", Content: doc.ContentNone}},
		Markdown: mdcodec.Rules{
			// Paragraphs in tight lists are text blocks: both are read.
			Nodes:  map[ast.NodeKind]mdcodec.ParseFunc{ast.KindParagraph: paragraph, ast.KindTextBlock: paragraph},
			Writes: map[string]mdcodec.WriteFunc{"toc": func(w *mdcodec.Writer, n *doc.Node) { w.Raw("[TOC]") }},
		},
		Views: map[string]editor.ViewSpec{"toc": {Atom: func(n *doc.Node, env *editor.Env) editor.AtomLook {
			return editor.AtomLook{
				Height: func(float32) float32 { return env.Font.Size * 2 },
				Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
					p.Fill(r, env.Theme.Surface, 6)
					p.Text(r.X+10, r.Y+r.H*0.65, "目录 [TOC]", env.Font.Size*0.85, env.Theme.TextMuted)
				},
			}
		}}},
		Slash: []editor.SlashItem{{Title: "目录", Hint: "[TOC]", Keywords: "toc contents outline", Command: "toc"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{"toc": doc.InsertBlocks(doc.NewAtom(ext.Node(reg, "toc"), nil))}
		},
	}
}
