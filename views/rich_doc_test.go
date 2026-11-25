package views

import (
	"strings"
	"testing"

	"github.com/Maidang1/corra/internal/doc"
)

func TestDocRichRoundTrip(t *testing.T) {
	n := 0
	d := NewDocRich(func() { n++ }, nil, nil)
	d.Load(mygoCorpus)
	if _, changed, _ := d.Snapshot(); changed {
		t.Fatal("unchanged document reported changed")
	}
	// Select the heading's first character and make it bold.
	s := d.ed.State()
	first := s.FirstLeaf(s.Doc.ID())
	d.ed.SetSelection(doc.TextSelection{Anchor: doc.Point{Leaf: first.ID()}, Head: doc.Point{Leaf: first.ID(), Offset: 1}})
	d.Format("bold")
	out, changed, _ := d.Snapshot()
	if !changed || n == 0 {
		t.Fatal("bold did not change the document")
	}
	if !strings.HasPrefix(out, "# **标**题") {
		t.Fatalf("bold not written: %q", out)
	}
	// Syntax the new editor has no extension for yet stays as source.
	for _, w := range []string{"| a | b |", "$$\nx^2\n$$", "[!NOTE]", "```go", "==高亮=="} {
		if !strings.Contains(out, w) {
			t.Errorf("lost %q in:\n%s", w, out)
		}
	}
	if _, changed, _ := d.Snapshot(); changed {
		t.Fatal("snapshot reported a change twice")
	}
}

func TestDocRichFormatCommands(t *testing.T) {
	d := NewDocRich(func() {}, nil, nil)
	d.Load("one\n")
	s := d.ed.State()
	p := s.FirstLeaf(s.Doc.ID())
	d.ed.SetSelection(doc.Caret(doc.Point{Leaf: p.ID(), Offset: 3}))
	d.Format("heading2")
	d.Format("link:http://x.y\nlink")
	out, _, _ := d.Snapshot()
	if out != "## one[link](http://x.y)\n" {
		t.Fatalf("got %q", out)
	}
	d.Format("undo")
	out, _, _ = d.Snapshot()
	if out != "## one\n" {
		t.Fatalf("undo got %q", out)
	}
}

func TestDocRichNoteLinkCompletion(t *testing.T) {
	d := NewDocRich(func() {}, nil, nil)
	d.OnLinkCandidates = func(q string) string { return `[{"title":"Other note","path":"other.md"}]` }
	d.Load("see it\n")
	s := d.ed.State()
	p := s.FirstLeaf(s.Doc.ID())
	d.ed.SetSelection(doc.Caret(doc.Point{Leaf: p.ID(), Offset: 4}))
	c := d.noteLinks()
	items := c.Items("oth")
	if len(items) != 1 || items[0].Title != "Other note" {
		t.Fatalf("items %+v", items)
	}
	d.ed.Run(c.Apply(items[0]), "input")
	out, _, _ := d.Snapshot()
	if out != "see [Other note](other.md)it\n" {
		t.Fatalf("got %q", out)
	}
}
