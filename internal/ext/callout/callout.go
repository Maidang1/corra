// Package callout is GitHub's alerts: a quote whose first line is
// "[!NOTE]", "[!TIP]", "[!IMPORTANT]", "[!WARNING]" or "[!CAUTION]".
package callout

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
)

func validKind(v any) error {
	if s, ok := v.(string); !ok || !mdext.IsAlertLabel(s) {
		return fmt.Errorf("callout kind %v", v)
	}
	return nil
}

// Kind returns a callout's kind, upper case, as "NOTE".
func Kind(n *doc.Node) string { return strings.ToUpper(n.Attrs().String("kind")) }

// colors are the alerts' colors, as GitHub's.
var colors = map[string]string{
	"NOTE": "#0969da", "TIP": "#1a7f37", "IMPORTANT": "#8250df", "WARNING": "#9a6700", "CAUTION": "#cf222e",
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "callout",
		// Above blockquote: its reading of a quote runs first.
		Priority: 5,
		Nodes: []doc.NodeSpec{{
			Name: "callout", Group: "block", Content: doc.ContentBlocks, Allow: []string{"block"}, Isolating: true,
			Attrs: map[string]doc.AttrSpec{"kind": {Default: "NOTE", Validate: validKind}},
		}},
		Markdown: mdcodec.Rules{
			Nodes:  map[ast.NodeKind]mdcodec.ParseFunc{ast.KindBlockquote: parse},
			Writes: map[string]mdcodec.WriteFunc{"callout": write},
		},
		Views: map[string]editor.ViewSpec{"callout": {Container: func(n *doc.Node, env *editor.Env) editor.ContainerLook {
			c := ui.Hex(colors[Kind(n)])
			return editor.ContainerLook{
				Indent: env.Font.Size,
				After:  env.Font.Size * 0.6,
				Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
					p.Fill(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}, c.Alpha(0.08), 6)
					p.Fill(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: 3, H: r.H + 8}, c, 1.5)
				},
			}
		}}},
		Slash: []editor.SlashItem{
			{Title: "提示块", Keywords: "callout note alert", Command: "callout:NOTE"},
			{Title: "警告块", Keywords: "callout warning alert", Command: "callout:WARNING"},
		},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			cmds := map[string]doc.Command{}
			for _, k := range mdext.AlertLabels {
				cmds["callout:"+k] = doc.Wrap(ext.Node(reg, "callout"), doc.A{"kind": k})
			}
			return cmds
		},
	}
}

// parse reads a quote whose first line is an alert marker as a callout,
// and any other quote as a blockquote.
func parse(p *mdcodec.Parser, n ast.Node) {
	first := n.FirstChild()
	label := ""
	if first != nil && (first.Kind() == ast.KindParagraph || first.Kind() == ast.KindTextBlock) {
		label = mdext.AlertLabel(firstLine(p.Lines(first)))
	}
	if label == "" {
		p.Open(p.Node("blockquote"), nil)
		p.Blocks(n)
		p.Close()
		return
	}
	p.Open(p.Node("callout"), doc.A{"kind": label})
	// The first paragraph's first line is the marker: keep what follows,
	// if anything.
	var rest []ast.Node
	skipping := true
	for c := first.FirstChild(); c != nil; c = c.NextSibling() {
		if skipping {
			if t, ok := c.(*ast.Text); ok && (t.SoftLineBreak() || t.HardLineBreak()) {
				skipping = false
			}
			continue
		}
		rest = append(rest, c)
	}
	if len(rest) > 0 {
		p.Open(p.Node("paragraph"), nil)
		for _, c := range rest {
			p.Walk(c)
		}
		p.Close()
	}
	for c := first.NextSibling(); c != nil; c = c.NextSibling() {
		p.Walk(c)
	}
	p.Close()
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func write(w *mdcodec.Writer, n *doc.Node) {
	w.WithPrefix("> ", "> ", func() {
		w.Raw("[!" + Kind(n) + "]")
		w.Newline()
		// A paragraph continues the marker's line, as GitHub writes it;
		// anything else needs a blank line, or "---" would underline the
		// marker into a heading.
		if n.NumChildren() > 0 && n.Child(0).Type().Name() != "paragraph" {
			w.Newline()
		}
		w.Children(n, false)
	})
}
