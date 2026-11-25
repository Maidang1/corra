// Package frontmatter is a leading YAML block between "---" lines, kept
// as written and edited as plain text.
package frontmatter

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
)

// firstOnly turns front matter anywhere but first in the document into a
// YAML code block, which is what it would read back as.
func firstOnly(n *doc.Node, fix *doc.Fixer) {
	s := fix.State()
	if p, ok := s.Parent(n.ID()); ok && p == s.Doc.ID() && s.Index(n.ID()) == 0 {
		return
	}
	if code := s.Registry().Node("code_block"); code != nil {
		fix.SetType(n.ID(), code, doc.A{"language": "yaml"})
	}
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "frontmatter",
		// The block's text is the YAML between the fences.
		Nodes: []doc.NodeSpec{{Name: "frontmatter", Group: "block", Content: doc.ContentText, Defining: true, Normalize: firstOnly}},
		Markdown: mdcodec.Rules{
			Preprocess: []func(p *mdcodec.Parser, src string) string{func(p *mdcodec.Parser, src string) string {
				fm, rest := mdext.SplitFrontMatter(src)
				if fm == "" {
					return src
				}
				lines := strings.Split(strings.TrimRight(strings.ReplaceAll(fm, "\r\n", "\n"), "\n"), "\n")
				yaml := strings.Join(lines[1:len(lines)-1], "\n")
				p.AddNode(doc.NewTextBlock(p.Node("frontmatter"), nil, doc.Text(yaml)))
				return rest
			}},
			Writes: map[string]mdcodec.WriteFunc{"frontmatter": func(w *mdcodec.Writer, n *doc.Node) {
				w.Raw("---")
				w.Newline()
				if yaml := n.Inline().String(); yaml != "" {
					w.Raw(yaml)
					w.Newline()
				}
				w.Raw("---")
			}},
		},
		Views: map[string]editor.ViewSpec{"frontmatter": {Text: func(n *doc.Node, env *editor.Env) editor.TextLook {
			return editor.TextLook{
				Font:  ui.Font{Family: env.Mono, Size: env.Font.Size * 0.8, LineHeight: 1.5},
				Color: env.Theme.TextMuted, Background: env.Tint(), Radius: 6, PadX: 14, PadY: 10,
				After: env.Font.Size,
			}
		}}},
	}
}
