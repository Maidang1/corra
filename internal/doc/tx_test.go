package doc_test

import (
	"testing"
	"time"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/doc/doctest"
)

// run applies cmd and checks the dumped result and the op log.
func run(t *testing.T, s doc.State, cmd doc.Command, want string) doc.State {
	t.Helper()
	before := doctest.DumpIDs(s.Doc)
	after, c, ok := doc.Run(s, cmd, doc.Meta{Kind: "structure"})
	if !ok {
		if want != "" {
			t.Fatalf("command did not apply; want %s", want)
		}
		return s
	}
	if want == "" {
		t.Fatalf("command applied: %s", doctest.DumpState(after))
	}
	if got := doctest.DumpState(after); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if got := doctest.DumpIDs(s.Doc); got != before {
		t.Fatalf("the old document changed:\n%s\n%s", before, got)
	}
	if err := doc.Validate(after.Doc); err != nil {
		t.Fatalf("invalid result: %v", err)
	}
	doctest.CheckLog(t, c)
	return after
}

func TestInsertText(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("ab|c")))
	run(t, s, doc.InsertText("X"), `doc(p("abX|c"))`)
}

func TestInsertTextTakesMarks(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a", b.B("bold|"), "c")))
	run(t, s, doc.InsertText("X"), `doc(p("a", b("boldX"), "|c"))`)
	s = b.State(b.Doc(b.P("a", b.C("code|"), "c")))
	run(t, s, doc.InsertText("X"), `doc(p("a", c("code"), "X|c"))`)
}

func TestInsertTextWithNewlines(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a|b")))
	run(t, s, doc.InsertText("1\n2"), `doc(p("a1"), p("2|b"))`)
	s = b.State(b.Doc(b.Pre("go", "a|b")))
	run(t, s, doc.InsertText("1\n2"), `doc(pre(language="go", "a1\n2|b"))`)
}

func TestDeleteAcrossBlocks(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("ab[c"), b.Img("x"), b.UL(b.LI(b.P("de]f")), b.LI(b.P("g")))))
	run(t, s, doc.DeleteSelection, `doc(p("ab|f"), ul(li(p("g"))))`)
}

func TestDeleteIntoList(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.UL(b.LI(b.P("a[b")), b.LI(b.P("c"))), b.P("d]e")))
	run(t, s, doc.DeleteSelection, `doc(ul(li(p("a|e"))))`)
}

func TestSplitBlock(t *testing.T) {
	b := doctest.NewBuilder()
	run(t, b.State(b.Doc(b.P("a|b"))), doc.SplitBlock, `doc(p("a"), p("|b"))`)
	run(t, b.State(b.Doc(b.H(2, "title|"))), doc.SplitBlock, `doc(h(level=2, "title"), p("|"))`)
	run(t, b.State(b.Doc(b.H(2, "ti|tle"))), doc.SplitBlock, `doc(h(level=2, "ti"), h(level=2, "|tle"))`)
}

func TestJoinBackward(t *testing.T) {
	b := doctest.NewBuilder()
	run(t, b.State(b.Doc(b.P("a"), b.P("|b"))), doc.JoinBackward, `doc(p("a|b"))`)
	run(t, b.State(b.Doc(b.P("a"), b.Img("x"), b.P("|b"))), doc.JoinBackward, `doc(p("a"), img(src="x"), p("b")) sel(image)`)
	// First block of a quote lifts out of it.
	run(t, b.State(b.Doc(b.P("a"), b.Quote(b.P("|b"), b.P("c")))), doc.JoinBackward, `doc(p("a"), p("|b"), quote(p("c")))`)
	run(t, b.State(b.Doc(b.P("|a"))), doc.JoinBackward, "")
}

func TestJoinForward(t *testing.T) {
	b := doctest.NewBuilder()
	run(t, b.State(b.Doc(b.P("a|"), b.Quote(b.P("b")))), doc.JoinForward, `doc(p("a|b"))`)
}

func TestToggleMark(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a[b"), b.P("c]d")))
	s = run(t, s, doc.ToggleMark(b.Bold), `doc(p("a", b("[b")), p(b("c"), "]d"))`)
	run(t, s, doc.ToggleMark(b.Bold), `doc(p("a[b"), p("c]d"))`)
	// Code text does not take bold.
	s = b.State(b.Doc(b.P(b.C("[co]de"))))
	run(t, s, doc.AddMark(doc.Mark{Type: b.Bold}), `doc(p(c("[co]de")))`)
}

func TestToggleMarkAtCaretStores(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a|")))
	s, _, _ = doc.Run(s, doc.ToggleMark(b.Bold), doc.Meta{})
	if s.Stored == nil || !s.Stored.Has(b.Bold) {
		t.Fatal("bold not stored")
	}
	s = run(t, s, doc.InsertText("X"), `doc(p("a", b("X|")))`)
	if s.Stored != nil {
		t.Fatal("stored marks survive typing")
	}
	run(t, s, doc.InsertText("Y"), `doc(p("a", b("XY|")))`)
}

func TestSetBlockType(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a", b.B("[b")), b.P("c]")))
	run(t, s, doc.SetBlockType(b.Heading, doc.A{"level": 3}), `doc(h(level=3, "a", b("[b")), h(level=3, "c]"))`)
	run(t, s, doc.SetBlockType(b.CodeBlock, nil), `doc(pre(language="", "a[b"), pre(language="", "c]"))`)
}

func TestWrapAndLift(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("[a"), b.P("b]"), b.P("c")))
	s = run(t, s, doc.Wrap(b.Blockquote, nil), `doc(quote(p("[a"), p("b]")), p("c"))`)
	run(t, s, doc.Lift, `doc(p("[a"), p("b]"), p("c"))`)
	// Lifting the middle block splits the quote.
	s = b.State(b.Doc(b.Quote(b.P("a"), b.P("|b"), b.P("c"))))
	run(t, s, doc.Lift, `doc(quote(p("a")), p("|b"), quote(p("c")))`)
}

func TestWrapGoesUpUntilAllowed(t *testing.T) {
	b := doctest.NewBuilder()
	// A list item may not be a quote's direct child's sibling inside a
	// list, so the quote wraps the whole list.
	s := b.State(b.Doc(b.UL(b.LI(b.P("[a")), b.LI(b.P("b]")))))
	run(t, s, doc.Wrap(b.Blockquote, nil), `doc(quote(ul(li(p("[a")), li(p("b]")))))`)
}

func TestMoveBlockKeepsID(t *testing.T) {
	b := doctest.NewBuilder()
	img := b.Img("x")
	d := b.Doc(b.P("|a"), img, b.P("b"))
	s := b.State(d)
	after, _, ok := doc.Run(s, doc.MoveBlock(img.ID(), d.ID(), 3), doc.Meta{})
	if !ok || doctest.Dump(after.Doc) != `doc(p("a"), p("b"), img(src="x"))` {
		t.Fatalf("got %s", doctest.Dump(after.Doc))
	}
	if after.Node(img.ID()) != img {
		t.Fatal("moved node is not the same node")
	}
}

func TestInsertFragment(t *testing.T) {
	b := doctest.NewBuilder()
	frag := []*doc.Node{b.P("X"), b.Img("i"), b.P("Y")}
	s := b.State(b.Doc(b.P("a|b")))
	run(t, s, doc.InsertFragment(frag), `doc(p("aX"), img(src="i"), p("Y|b"))`)
	s = b.State(b.Doc(b.P("a|b")))
	run(t, s, doc.InsertFragment([]*doc.Node{b.P(b.B("X"))}), `doc(p("a", b("X"), "|b"))`)
}

func TestNormalizationFillsAndWraps(t *testing.T) {
	b := doctest.NewBuilder()
	li := b.LI(b.P("x"))
	d := b.Doc(b.UL(li), b.P("|"))
	s := b.State(d)
	// Removing the only list item leaves an empty list, which is filled.
	ul := d.Child(0)
	after, _, _ := doc.Run(s, func(s doc.State, tx *doc.Tx) bool {
		tx.ReplaceChildren(ul.ID(), 0, 1)
		return true
	}, doc.Meta{})
	if got := doctest.Dump(after.Doc); got != `doc(ul(li(p())), p())` {
		t.Fatalf("got %s", got)
	}
	// A paragraph put straight into a list is wrapped in an item.
	after, _, _ = doc.Run(after, func(s doc.State, tx *doc.Tx) bool {
		tx.ReplaceChildren(s.Doc.Child(0).ID(), 1, 1, b.P("y"))
		return true
	}, doc.Meta{})
	if got := doctest.Dump(after.Doc); got != `doc(ul(li(p()), li(p("y"))), p())` {
		t.Fatalf("got %s", got)
	}
}

func TestDuplicateIDPanics(t *testing.T) {
	b := doctest.NewBuilder()
	p := b.P("a")
	s := b.State(b.Doc(p))
	defer func() {
		if recover() == nil {
			t.Fatal("inserting a node twice did not panic")
		}
	}()
	doc.Run(s, func(s doc.State, tx *doc.Tx) bool {
		tx.ReplaceChildren(s.Doc.ID(), 1, 1, p)
		return true
	}, doc.Meta{})
}

func TestCopyOnce(t *testing.T) {
	b := doctest.NewBuilder()
	var ps []*doc.Node
	for i := 0; i < 50; i++ {
		ps = append(ps, b.P("text"))
	}
	s := b.State(b.Doc(ps...))
	tx := s.Begin()
	for _, p := range ps {
		tx.ReplaceText(p.ID(), 0, 0, doc.Text("x"))
	}
	root := tx.State().Doc
	tx.ReplaceText(ps[0].ID(), 0, 0, doc.Text("y"))
	if tx.State().Doc != root {
		t.Fatal("the root was copied again")
	}
	after, _ := tx.Commit()
	if s.Doc.Child(0).Inline().String() != "text" || after.Doc.Child(0).Inline().String() != "yxtext" {
		t.Fatal("wrong texts")
	}
}

func TestFrozenNodesKeepLoggedValues(t *testing.T) {
	b := doctest.NewBuilder()
	q := b.Quote(b.P("a"))
	s := b.State(b.Doc(b.P("|x"), q))
	tx := s.Begin()
	// Copy the quote into the transaction, move it, then change it.
	tx.ReplaceText(q.Child(0).ID(), 1, 1, doc.Text("b"))
	cur := tx.State().Node(q.ID())
	tx.ReplaceChildren(s.Doc.ID(), 1, 2)
	tx.ReplaceChildren(s.Doc.ID(), 0, 0, cur)
	tx.ReplaceText(q.Child(0).ID(), 2, 2, doc.Text("c"))
	_, c := tx.Commit()
	doctest.CheckLog(t, c)
	if got := doctest.Dump(c.After.Doc); got != `doc(quote(p("abc")), p("x"))` {
		t.Fatalf("got %s", got)
	}
}

func TestFirstOfRollsBack(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("a|")))
	failing := func(s doc.State, tx *doc.Tx) bool {
		tx.ReplaceText(s.Doc.Child(0).ID(), 0, 0, doc.Text("junk"))
		return false
	}
	run(t, s, doc.FirstOf(failing, doc.InsertText("b")), `doc(p("ab|"))`)
}

func TestPositions(t *testing.T) {
	b := doctest.NewBuilder()
	p1, p2 := b.P("ab"), b.P("cde")
	d := b.Doc(p1, b.Img("x"), b.Quote(p2), b.P(""))
	s := doc.NewState(d)
	cases := []struct {
		p doc.Point
		g int
	}{{doc.Point{Leaf: p1.ID(), Offset: 0}, 0}, {doc.Point{Leaf: p1.ID(), Offset: 2}, 2}, {doc.Point{Leaf: p2.ID(), Offset: 0}, 3}, {doc.Point{Leaf: p2.ID(), Offset: 3}, 6}}
	for _, c := range cases {
		if g := s.GlobalOffset(c.p); g != c.g {
			t.Errorf("GlobalOffset(%v) = %d, want %d", c.p, g, c.g)
		}
		if p := s.PointAt(c.g); p != c.p {
			t.Errorf("PointAt(%d) = %v, want %v", c.g, p, c.p)
		}
	}
	if n := s.NextLeaf(p1.ID()); n != p2 {
		t.Fatal("NextLeaf skips the image to the quote's paragraph")
	}
	rs := s.LeafRanges(doc.TextSelection{Anchor: doc.Point{Leaf: p2.ID(), Offset: 1}, Head: doc.Point{Leaf: p1.ID(), Offset: 1}})
	if len(rs) != 2 || rs[0].From != 1 || rs[0].To != 2 || rs[1].From != 0 || rs[1].To != 1 {
		t.Fatalf("ranges %+v", rs)
	}
}

func TestHistoryMergesTyping(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("|")))
	var h doc.History
	start := s
	now := time.Now()
	for i, ch := range []string{"a", "b", "c"} {
		var c doc.Commit
		s, c, _ = doc.Run(s, doc.InsertText(ch), doc.Meta{Kind: "input", Time: now.Add(time.Duration(i) * 100 * time.Millisecond)})
		h.Push(c)
	}
	p := s.Doc.Child(0).ID()
	s.Sel = doc.TextSelection{Anchor: doc.Point{Leaf: p}, Head: doc.Point{Leaf: p, Offset: 3}}
	s, c, _ := doc.Run(s, doc.ToggleMark(b.Bold), doc.Meta{Kind: "format", Time: now})
	h.Push(c)
	if st, _ := h.Undo(); doctest.Dump(st.Doc) != `doc(p("abc"))` {
		t.Fatalf("first undo %s", doctest.Dump(st.Doc))
	}
	st, _ := h.Undo()
	if st.Doc != start.Doc {
		t.Fatal("typing was not one undo step back to the same document")
	}
	if st, _ := h.Redo(); doctest.Dump(st.Doc) != `doc(p("abc"))` {
		t.Fatalf("redo %s", doctest.Dump(st.Doc))
	}
	_ = s
}

func TestHistoryDoesNotMergeAfterPause(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("|")))
	var h doc.History
	now := time.Now()
	for i, ch := range []string{"a", "b"} {
		var c doc.Commit
		s, c, _ = doc.Run(s, doc.InsertText(ch), doc.Meta{Kind: "input", Time: now.Add(time.Duration(i) * 2 * time.Second)})
		h.Push(c)
	}
	if st, _ := h.Undo(); doctest.Dump(st.Doc) != `doc(p("a"))` {
		t.Fatalf("undo %s", doctest.Dump(st.Doc))
	}
}

func TestFragment(t *testing.T) {
	b := doctest.NewBuilder()
	s := b.State(b.Doc(b.P("ab[c"), b.UL(b.LI(b.P("d"), b.P("ef")), b.LI(b.P("g]h"))), b.P("z")))
	sel := s.Sel.(doc.TextSelection)
	frag := s.Fragment(sel)
	got := ""
	for _, n := range frag {
		got += doctest.Dump(n) + " "
	}
	if want := `p("c") ul(li(p("d"), p("ef")), li(p("g"))) `; got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if doc.PlainText(frag) != "c\nd\nef\ng" {
		t.Fatalf("plain %q", doc.PlainText(frag))
	}
	// Pasting it back elsewhere keeps the structure.
	s2 := b.State(b.Doc(b.P("x|y")))
	run(t, s2, doc.InsertFragment(frag), `doc(p("xc"), ul(li(p("d"), p("ef")), li(p("g|"))), p("y"))`)
}

func TestReconcileKeepsUnchangedNodes(t *testing.T) {
	b := doctest.NewBuilder()
	keep := b.P("same")
	q := b.Quote(b.P("x"), b.P("old"))
	prev := b.Doc(keep, q, b.P("gone"))
	next := b.Doc(b.P("new"), b.P("same"), b.Quote(b.P("x"), b.P("changed")))
	r := doc.Reconcile(prev, next)
	if !doc.SameContent(r, next) {
		t.Fatalf("content changed: %s", doctest.Dump(r))
	}
	if r.Child(1) != keep {
		t.Fatal("unchanged paragraph not reused")
	}
	if r.Child(2).Child(0) != q.Child(0) {
		t.Fatal("unchanged paragraph inside a changed quote not reused")
	}
	if err := doc.Validate(r); err != nil {
		t.Fatal(err)
	}
}
