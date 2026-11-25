// Package doctest helps test documents: a small schema, a compact notation
// for building trees with a selection, a dump in the same notation, and
// random documents and command sequences that check the model's invariants.
package doctest

import (
	"fmt"

	"github.com/Maidang1/corra/internal/doc"
)

func init() {
	// Rules that never settle are bugs; fail loudly in tests.
	doc.OnNormalizeLimit = func(err error) { panic(err) }
}

// Schema is the test schema:
//
//	doc         blocks: block
//	paragraph   inline                    group block
//	heading     inline {level}            group block
//	blockquote  blocks: block             group block
//	bullet_list blocks: list_item         group block
//	list_item   blocks: block
//	code_block  text {language}           group block
//	image       atom {src}                group block
//	hr          atom                      group block
//	math        inline atom {tex}
//
// with marks link (rank 0), bold (10), italic (20) and code (30, which
// excludes every other mark).
type Schema struct {
	Reg *doc.Registry

	Doc, Paragraph, Heading, Blockquote, BulletList, ListItem, CodeBlock, Image, HR, Math *doc.NodeType
	Link, Bold, Italic, Code                                                              *doc.MarkType
}

func levelValid(v any) error {
	if l, ok := v.(int); !ok || l < 1 || l > 6 {
		return fmt.Errorf("level %v", v)
	}
	return nil
}

// NewSchema builds the test schema.
func NewSchema() *Schema {
	r := doc.NewRegistry()
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	must(r.AddNode(doc.NodeSpec{Name: "doc", Content: doc.ContentBlocks, Allow: []string{"block"}}))
	must(r.AddNode(doc.NodeSpec{Name: "paragraph", Group: "block", Content: doc.ContentInline}))
	must(r.AddNode(doc.NodeSpec{Name: "heading", Group: "block", Content: doc.ContentInline,
		Attrs: map[string]doc.AttrSpec{"level": {Default: 1, Validate: levelValid}}}))
	must(r.AddNode(doc.NodeSpec{Name: "blockquote", Group: "block", Content: doc.ContentBlocks, Allow: []string{"block"}}))
	must(r.AddNode(doc.NodeSpec{Name: "bullet_list", Group: "block", Content: doc.ContentBlocks, Allow: []string{"list_item"}}))
	must(r.AddNode(doc.NodeSpec{Name: "list_item", Content: doc.ContentBlocks, Allow: []string{"block"}}))
	must(r.AddNode(doc.NodeSpec{Name: "code_block", Group: "block", Content: doc.ContentText, Defining: true,
		Attrs: map[string]doc.AttrSpec{"language": {Default: ""}}}))
	must(r.AddNode(doc.NodeSpec{Name: "image", Group: "block", Content: doc.ContentNone,
		Attrs: map[string]doc.AttrSpec{"src": {Default: ""}}}))
	must(r.AddNode(doc.NodeSpec{Name: "hr", Group: "block", Content: doc.ContentNone}))
	must(r.AddNode(doc.NodeSpec{Name: "math", Content: doc.ContentNone, Inline: true,
		Attrs: map[string]doc.AttrSpec{"tex": {Default: ""}}}))
	must(r.AddMark(doc.MarkSpec{Name: "link", Rank: 0, Attrs: map[string]doc.AttrSpec{"href": {Default: ""}}}))
	must(r.AddMark(doc.MarkSpec{Name: "bold", Rank: 10, Inclusive: true}))
	must(r.AddMark(doc.MarkSpec{Name: "italic", Rank: 20, Inclusive: true}))
	must(r.AddMark(doc.MarkSpec{Name: "code", Rank: 30, Excludes: []string{"_"}}))
	must(r.Build())
	return &Schema{
		Reg: r, Doc: r.Node("doc"), Paragraph: r.Node("paragraph"), Heading: r.Node("heading"),
		Blockquote: r.Node("blockquote"), BulletList: r.Node("bullet_list"), ListItem: r.Node("list_item"),
		CodeBlock: r.Node("code_block"), Image: r.Node("image"), HR: r.Node("hr"), Math: r.Node("math"),
		Link: r.Mark("link"), Bold: r.Mark("bold"), Italic: r.Mark("italic"), Code: r.Mark("code"),
	}
}
