// Package blockquote is "> " quotations.
package blockquote

import (
	"regexp"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func Extension() ext.Extension {
	return ext.Extension{
		Name: "blockquote",
		Nodes: []doc.NodeSpec{{
			Name: "blockquote", Group: "block", Content: doc.ContentBlocks, Allow: []string{"block"},
		}},
		Markdown: mdcodec.Rules{
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindBlockquote: func(p *mdcodec.Parser, n ast.Node) {
					p.Open(p.Node("blockquote"), nil)
					p.Blocks(n)
					p.Close()
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"blockquote": func(w *mdcodec.Writer, n *doc.Node) {
					w.WithPrefix("> ", "> ", func() { w.Children(n, false) })
				},
			},
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{editor.PrefixRule(regexp.MustCompile(`^> $`), func([]string) doc.Command {
				return doc.Wrap(ext.Node(reg, "blockquote"), nil)
			})}
		},
		Slash: []editor.SlashItem{{Title: "引用", Hint: ">", Keywords: "quote blockquote", Command: "blockquote"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{"blockquote": doc.Wrap(ext.Node(reg, "blockquote"), nil)}
		},
	}
}
