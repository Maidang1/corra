// Package image is inline images, "![alt](src "title")".
package image

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
	"github.com/Maidang1/corra/internal/mdext"
)

func Extension() ext.Extension {
	return ext.Extension{
		Name: "image",
		Nodes: []doc.NodeSpec{{
			Name: "image", Content: doc.ContentNone, Inline: true,
			// size is Typora's suffix: "300x200", "300", "x200" or "50%".
			Attrs: map[string]doc.AttrSpec{"src": {Default: ""}, "alt": {Default: ""}, "title": {Default: ""}, "size": {Default: "", Validate: validSize}},
		}},
		Markdown: mdcodec.Rules{
			Transforms: []func(ast.Node, []byte){sizedImages},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindImage: func(p *mdcodec.Parser, n ast.Node) {
					img := n.(*ast.Image)
					p.AddAtom(doc.NewAtom(p.Node("image"), doc.A{
						"src": string(img.Destination), "title": string(img.Title), "alt": p.Text(n), "size": sizeOf(img),
					}))
				},
			},
			Writes: map[string]mdcodec.WriteFunc{
				"image": func(w *mdcodec.Writer, n *doc.Node) {
					a := n.Attrs()
					if size := a.String("size"); size != "" && a.String("title") == "" && !strings.ContainsAny(a.String("src"), " ()<>") {
						w.Raw("![" + w.Escape(a.String("alt")) + "](" + a.String("src") + " =" + size + ")")
						return
					}
					w.Raw("![" + w.Escape(a.String("alt")) + "]" + mdcodec.Destination(a.String("src"), a.String("title")))
				},
			},
		},
	}
}

// sizedImages reads Typora's "![a](p.png =300x200)" with mdext's
// transform, keeping the titles it clears from images goldmark read.
func sizedImages(root ast.Node, src []byte) {
	titles := map[*ast.Image][]byte{}
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if img, ok := n.(*ast.Image); ok && entering && len(img.Title) > 0 {
			titles[img] = img.Title
		}
		return ast.WalkContinue, nil
	})
	mdext.TransformSizedImages(root, src)
	for img, title := range titles {
		if len(img.Title) == 0 {
			img.Title = title
		}
	}
}

func validSize(v any) error {
	if s, ok := v.(string); !ok || (s != "" && !mdext.ImageSizeValid(s)) {
		return fmt.Errorf("image size %v", v)
	}
	return nil
}

// sizeOf reads the size the sized-image transform left as attributes.
func sizeOf(img *ast.Image) string {
	attr := func(name string) string {
		if v, ok := img.AttributeString(name); ok {
			if b, ok := v.([]byte); ok {
				return string(b)
			}
		}
		return ""
	}
	if style := attr("style"); strings.HasPrefix(style, "width:") && strings.HasSuffix(style, "%") {
		return strings.TrimPrefix(style, "width:")
	}
	w, h := attr("width"), attr("height")
	switch {
	case w != "" && h != "":
		return w + "x" + h
	case w != "":
		return w
	case h != "":
		return "x" + h
	}
	return ""
}

// Insert returns a command inserting an image at the caret.
func Insert(reg *doc.Registry, src, alt string) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !s.Node(sel.Head.Leaf).Type().Allows(ext.Node(reg, "image")) {
			return false
		}
		atom := doc.NewAtom(ext.Node(reg, "image"), doc.A{"src": src, "alt": alt})
		return doc.InsertFragment([]*doc.Node{doc.NewTextBlock(ext.Node(reg, "paragraph"), nil, doc.AtomInline(atom))})(s, tx)
	}
}
