package doc_test

import (
	"strings"
	"testing"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/doc/doctest"
)

func TestRegistryErrors(t *testing.T) {
	r := doc.NewRegistry()
	if err := r.Build(); err == nil {
		t.Fatal("built without a doc type")
	}
	r = doc.NewRegistry()
	_ = r.AddNode(doc.NodeSpec{Name: "doc", Content: doc.ContentBlocks, Allow: []string{"nope"}})
	if err := r.Build(); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("unknown allow: %v", err)
	}
	if err := r.AddNode(doc.NodeSpec{Name: "x", Inline: true, Content: doc.ContentInline}); err == nil {
		t.Fatal("inline text block accepted")
	}
}

func TestAttrDefaultsAndValidation(t *testing.T) {
	b := doctest.NewBuilder()
	h := doc.NewTextBlock(b.Heading, nil, doc.Text("x"))
	if h.Attrs().Int("level") != 1 {
		t.Fatal("default level")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("invalid level accepted")
		}
	}()
	doc.NewTextBlock(b.Heading, doc.A{"level": 9}, doc.Text("x"))
}

func TestCloneFreshGivesNewIDs(t *testing.T) {
	b := doctest.NewBuilder()
	d := b.Doc(b.Quote(b.P("a", b.Math("x"))), b.Img("i"))
	c := doc.CloneFresh(d)
	if !doc.SameContent(d, c) || doc.Equal(d, c) {
		t.Fatal("clone differs in content or keeps IDs")
	}
	if err := doc.Validate(b.Doc(d.Child(0), doc.CloneFresh(d.Child(0)))); err != nil {
		t.Fatalf("clone collides: %v", err)
	}
}

func TestValidateFindsProblems(t *testing.T) {
	b := doctest.NewBuilder()
	p := b.P("a")
	if err := doc.Validate(b.Doc(p, p)); err == nil {
		t.Fatal("duplicate ID not found")
	}
	if err := doc.Validate(b.Doc(b.UL(b.P("x")))); err == nil {
		t.Fatal("paragraph in a list not found")
	}
	if err := doc.Validate(b.Doc(b.Quote())); err == nil {
		t.Fatal("empty quote not found")
	}
}

func TestNormalizationLimit(t *testing.T) {
	r := doc.NewRegistry()
	_ = r.AddNode(doc.NodeSpec{Name: "doc", Content: doc.ContentBlocks})
	// Two rules undoing each other: the attribute flips forever.
	_ = r.AddNode(doc.NodeSpec{Name: "p", Content: doc.ContentInline,
		Attrs: map[string]doc.AttrSpec{"on": {Default: false}},
		Normalize: func(n *doc.Node, fix *doc.Fixer) {
			fix.SetAttr(n.ID(), "on", !n.Attrs().Bool("on"))
		}})
	r.MustBuild()
	p := doc.NewTextBlock(r.Node("p"), nil, doc.Text("x"))
	s := doc.NewState(doc.NewBlock(r.Root(), nil, p))
	defer func() {
		if e := recover(); e == nil || !strings.Contains(e.(error).Error(), "did not settle") {
			t.Fatalf("no limit: %v", e)
		}
	}()
	doc.Run(s, doc.InsertText("y"), doc.Meta{})
}
