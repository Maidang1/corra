package doc

import "testing"

func testMarks(t *testing.T) (bold, italic, code, link *MarkType) {
	t.Helper()
	r := NewRegistry()
	for _, s := range []MarkSpec{
		{Name: "link", Rank: 0, Attrs: map[string]AttrSpec{"href": {Default: ""}}},
		{Name: "bold", Rank: 10, Inclusive: true},
		{Name: "italic", Rank: 20, Inclusive: true},
		{Name: "code", Rank: 30, Excludes: []string{"_"}},
	} {
		if err := r.AddMark(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.AddNode(NodeSpec{Name: "doc", Content: ContentBlocks}); err != nil {
		t.Fatal(err)
	}
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}
	return r.Mark("bold"), r.Mark("italic"), r.Mark("code"), r.Mark("link")
}

func TestUTF16(t *testing.T) {
	if n := UTF16Len("a😀中"); n != 4 {
		t.Fatalf("len %d", n)
	}
	s := "a😀b"
	for off, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 5, 4: 6, 9: 6} {
		if got := byteOffset(s, off); got != want {
			t.Errorf("byteOffset(%d) = %d, want %d", off, got, want)
		}
	}
}

func TestInlineNormalizes(t *testing.T) {
	bold, _, _, _ := testMarks(t)
	b := Mark{Type: bold}
	in := NewInline(
		Segment{Text: "a"}, Segment{Text: ""}, Segment{Text: "b"},
		Segment{Text: "c", Marks: MarkSet{b}}, Segment{Text: "d", Marks: MarkSet{b}},
	)
	if in.NumSegments() != 2 || in.Segment(0).Text != "ab" || in.Segment(1).Text != "cd" || in.Len() != 4 {
		t.Fatalf("segments %+v", in.Segments())
	}
}

func TestInlineSliceAndReplace(t *testing.T) {
	bold, _, _, _ := testMarks(t)
	in := Text("he").Concat(Text("llo", Mark{Type: bold})).Concat(Text("😀!"))
	if got := in.Slice(1, 4).String(); got != "ell" {
		t.Fatalf("slice %q", got)
	}
	if !in.HasMark(bold, 2, 5) || in.HasMark(bold, 1, 3) {
		t.Fatal("HasMark")
	}
	// Inside the surrogate pair rounds down.
	if got := in.Slice(5, 6).String(); got != "" {
		t.Fatalf("half emoji %q", got)
	}
	if got := in.Slice(5, 7).String(); got != "😀" {
		t.Fatalf("emoji %q", got)
	}
	r := in.Replace(2, 5, Text("LLO"))
	if r.String() != "heLLO😀!" || r.HasMark(bold, 2, 5) {
		t.Fatalf("replace %q", r.String())
	}
	// The original is unchanged.
	if in.String() != "hello😀!" || !in.HasMark(bold, 2, 5) {
		t.Fatal("receiver changed")
	}
}

func TestConcatDoesNotAlias(t *testing.T) {
	a := Text("a")
	b := a.Concat(Text("b"))
	c := a.Concat(Text("c"))
	if b.String() != "ab" || c.String() != "ac" || a.String() != "a" {
		t.Fatalf("%q %q %q", a.String(), b.String(), c.String())
	}
}

func TestMarkSet(t *testing.T) {
	bold, italic, code, link := testMarks(t)
	var s MarkSet
	s = s.Add(Mark{Type: italic}).Add(Mark{Type: bold})
	if s[0].Type != bold || s[1].Type != italic {
		t.Fatalf("not sorted by rank: %v", s)
	}
	s = s.Add(NewMark(link, A{"href": "x"})).Add(NewMark(link, A{"href": "y"}))
	if len(s) != 3 || s[0].Attrs.String("href") != "y" {
		t.Fatalf("link not replaced: %v", s)
	}
	c := s.Add(Mark{Type: code})
	if len(c) != 1 || c[0].Type != code {
		t.Fatalf("code does not exclude: %v", c)
	}
	if got := c.Add(Mark{Type: bold}); len(got) != 1 || got[0].Type != code {
		t.Fatalf("bold added to code: %v", got)
	}
}

func TestMarksAt(t *testing.T) {
	bold, _, code, _ := testMarks(t)
	in := Text("a").Concat(Text("b", Mark{Type: bold})).Concat(Text("c", Mark{Type: code}))
	if m := in.MarksAt(2); !m.Has(bold) {
		t.Fatalf("end of bold: %v", m)
	}
	if m := in.MarksAt(3); m.Has(code) {
		t.Fatalf("end of code continues: %v", m)
	}
	if m := in.MarksAt(1); m.Has(bold) {
		t.Fatalf("start of bold: %v", m)
	}
}
