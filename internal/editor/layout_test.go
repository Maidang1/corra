package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/doc/doctest"
)

func testEditor(t testing.TB, d *doc.Node, reg *doc.Registry) (*Editor, *ui.Tester) {
	e := New(Config{Registry: reg}, d)
	env := Env{Font: ui.Font{Size: 16}, Mono: "monospace"}
	tt := ui.NewTester(func(c *ui.Context) { e.View(c, env) }, 640, 480)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(100, 10)
	tt.Frame()
	return e, tt
}

// An edit lays out only the path to what changed: the other blocks keep
// their boxes, and their text is not shaped again.
func TestLayoutReusesUnchangedBoxes(t *testing.T) {
	b := doctest.NewBuilder()
	d := b.Doc(b.P("first"), b.Quote(b.P("second")), b.P("third"))
	e, tt := testEditor(t, d, b.Reg)
	before := map[doc.NodeID]*box{}
	for n, bx := range e.lay.cache {
		before[n.ID()] = bx
	}
	third := d.Child(2)
	e.SetSelection(doc.Caret(doc.Point{Leaf: third.ID(), Offset: 5}))
	tt.Type("!")
	tt.Frame()
	if got := e.State().Doc.Child(2).Inline().String(); got != "third!" {
		t.Fatalf("typed into %q", got)
	}
	reused, shaped := 0, 0
	for n, bx := range e.lay.cache {
		if before[n.ID()] == bx {
			reused++
		} else {
			shaped++
		}
	}
	// first, quote and its paragraph are reused; the root and the edited
	// paragraph are new.
	if reused != 3 || shaped != 2 {
		t.Fatalf("reused %d, laid out %d", reused, shaped)
	}
}

func BenchmarkTypeInLargeDocument(b *testing.B) {
	bd := doctest.NewBuilder()
	var ps []*doc.Node
	for i := 0; i < 1000; i++ {
		ps = append(ps, bd.P(strings.Repeat("字a ", 66)))
	}
	e, tt := testEditor(b, bd.Doc(ps...), bd.Reg)
	last := ps[len(ps)-1]
	e.SetSelection(doc.Caret(doc.Point{Leaf: last.ID()}))
	b.ResetTimer()
	for b.Loop() {
		tt.Type("x")
		tt.Frame()
	}
}
