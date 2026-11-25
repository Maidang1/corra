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
	"github.com/Maidang1/corra/render"
)

func validKind(v any) error {
	if s, ok := v.(string); !ok || !mdext.IsAlertLabel(s) {
		return fmt.Errorf("callout kind %v", v)
	}
	return nil
}

// Kind returns a callout's kind, upper case, as "NOTE".
func Kind(n *doc.Node) string { return strings.ToUpper(n.Attrs().String("kind")) }

// colors are the alerts' colors, as GitHub's, on light and dark pages.
var colors = map[string][2]string{
	"NOTE":      {"#0969da", "#4493f8"},
	"TIP":       {"#1a7f37", "#3fb950"},
	"IMPORTANT": {"#8250df", "#ab7df8"},
	"WARNING":   {"#9a6700", "#d29922"},
	"CAUTION":   {"#cf222e", "#f85149"},
}

// look is a callout's card: a tinted panel with a bar down its left edge
// and the kind's title above the content.
func look(n *doc.Node, env *editor.Env) editor.ContainerLook {
	kind := Kind(n)
	shade := 0
	if env.Dark {
		shade = 1
	}
	c := ui.Hex(colors[kind][shade])
	size := env.Font.Size
	pad := size * 0.9
	title := env.Font
	title.Weight = 600
	return editor.ContainerLook{
		Indent: pad + 3, IndentRight: pad,
		PadTop: size * 2, PadBottom: size * 0.6,
		Before: size * 0.4, After: size,
		Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
			p.Fill(r, c.Alpha(0.1), 6)
			p.FillRadii(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, [4]float32{6, 0, 0, 6}, c)
			editor.DrawLabel(p, r.X+pad+3, r.Y+size*1.35, render.AlertTitle(kind), title, c)
		},
	}
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
		Views: map[string]editor.ViewSpec{"callout": {Container: look}},
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
