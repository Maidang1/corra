// Package hr is thematic breaks.
package hr

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func Extension() ext.Extension {
	return ext.Extension{
		Name:  "hr",
		Nodes: []doc.NodeSpec{{Name: "horizontal_rule", Group: "block", Content: doc.ContentNone}},
		Markdown: mdcodec.Rules{
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindThematicBreak: func(p *mdcodec.Parser, n ast.Node) {
					p.AddNode(doc.NewAtom(p.Node("horizontal_rule"), nil))
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"horizontal_rule": func(w *mdcodec.Writer, n *doc.Node) {
					rule := w.Style.Rule
					// "- ---" is one rule, not a list item holding one: use
					// another character than the line's list marker.
					// Nor "---" first in the document, which opens front matter.
					if strings.ContainsRune(w.LinePrefix(), rune(rule[0])) || (!w.Started() && rule == "---") {
						rule = map[byte]string{'-': "***", '*': "---", '_': "***"}[rule[0]]
					}
					w.Raw(rule)
				},
			},
			// "---" right under a paragraph line would underline it into a
			// heading.
			Loose: []string{"horizontal_rule"},
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{editor.PrefixRule(regexp.MustCompile(`^(---|\*\*\*|___)$`), func([]string) doc.Command {
				return func(s doc.State, tx *doc.Tx) bool {
					return doc.InsertBlocks(doc.NewAtom(ext.Node(reg, "horizontal_rule"), nil), doc.Empty(ext.Node(reg, "paragraph")))(s, tx)
				}
			})}
		},
		Slash: []editor.SlashItem{{Title: "分隔线", Hint: "---", Keywords: "rule divider hr", Command: "horizontalRule"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{"horizontalRule": func(s doc.State, tx *doc.Tx) bool {
				return doc.InsertBlocks(doc.NewAtom(ext.Node(reg, "horizontal_rule"), nil), doc.Empty(ext.Node(reg, "paragraph")))(s, tx)
			}}
		},
	}
}
