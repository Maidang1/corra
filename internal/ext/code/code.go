// Package code is fenced and indented code blocks; both read as fenced.
package code

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

// Language returns a code block's language.
func Language(n *doc.Node) string { return n.Attrs().String("language") }

func Extension() ext.Extension {
	return ext.Extension{
		Name: "code",
		Nodes: []doc.NodeSpec{{
			Name: "code_block", Group: "block", Content: doc.ContentText, Defining: true,
			Attrs: map[string]doc.AttrSpec{"language": {Default: ""}},
		}},
		Markdown: mdcodec.Rules{
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindFencedCodeBlock: func(p *mdcodec.Parser, n ast.Node) {
					lang := string(n.(*ast.FencedCodeBlock).Language(p.Source()))
					p.AddNode(doc.NewTextBlock(p.Node("code_block"), doc.A{"language": lang}, doc.Text(p.Lines(n))))
				},
				ast.KindCodeBlock: func(p *mdcodec.Parser, n ast.Node) {
					p.AddNode(doc.NewTextBlock(p.Node("code_block"), nil, doc.Text(p.Lines(n))))
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"code_block": func(w *mdcodec.Writer, n *doc.Node) {
					text := n.Inline().String()
					fence := Fence(w.Style.Fence, text)
					w.Raw(fence + Language(n))
					w.Newline()
					if text != "" {
						w.Raw(text)
						w.Newline()
					}
					w.Raw(fence)
				},
			},
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{editor.PrefixRule(regexp.MustCompile("^```([\\w+#.-]*) $"), func(m []string) doc.Command {
				return doc.SetBlockType(ext.Node(reg, "code_block"), doc.A{"language": m[1]})
			})}
		},
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			// Enter on a line of just "```lang" opens a code block.
			fence := regexp.MustCompile("^```([\\w+#.-]*)$")
			return map[string][]doc.Command{"Enter": {func(s doc.State, tx *doc.Tx) bool {
				sel, ok := s.TextSel()
				if !ok || !sel.Empty() {
					return false
				}
				leaf := s.Node(sel.Head.Leaf)
				m := fence.FindStringSubmatch(leaf.Inline().String())
				if m == nil || leaf.Type().Content() != doc.ContentInline || sel.Head.Offset != leaf.TextLen() {
					return false
				}
				tx.ReplaceText(leaf.ID(), 0, leaf.TextLen(), doc.Inline{})
				tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID()}))
				return doc.SetBlockType(ext.Node(reg, "code_block"), doc.A{"language": m[1]})(tx.State(), tx)
			}}}
		},
		Slash: []editor.SlashItem{{Title: "代码块", Hint: "```", Keywords: "code fence", Command: "codeBlock"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{"codeBlock": doc.SetBlockType(ext.Node(reg, "code_block"), nil)}
		},
	}
}

// Fence returns a fence of char longer than any run of it in text, and at
// least three long.
func Fence(char, text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if string(r) == char {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat(char, max(3, longest+1))
}
