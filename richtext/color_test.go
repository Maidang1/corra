package richtext

import "testing"

func TestColorSpanRoundTrip(t *testing.T) {
	for _, source := range []string{
		`a <span style="color:#d44c47">red</span> b`,
		`<span style="background-color:#fbf3db">tinted</span>`,
		`<span style="color:#337ea9;background-color:#fbecdd">both</span>`,
		`<span style="color:#d44c47">red **bold** red</span>`,
		`[<span style="color:#448361">green link</span>](note.md)`,
	} {
		doc := Parse(source)
		if got := Serialize(doc); got != source {
			t.Errorf("untouched %q saved as %q", source, got)
		}
		// Drop the raw source so the runs themselves are written.
		for i := range doc.Paragraphs {
			doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline = "", ""
		}
		if got := Serialize(doc); got != source {
			t.Errorf("edited %q saved as %q", source, got)
		}
	}
}

func TestColorRuns(t *testing.T) {
	doc := Parse(`x <span style="color: #D44C47">r **b**</span> y`)
	runs := doc.Paragraphs[0].Runs
	want := []Run{
		{Text: "x "},
		{Text: "r ", Color: "#d44c47"},
		{Text: "b", Color: "#d44c47", Bold: true},
		{Text: " y"},
	}
	if len(runs) != len(want) {
		t.Fatalf("runs = %+v", runs)
	}
	for i := range want {
		if runs[i] != want[i] {
			t.Errorf("run %d = %+v, want %+v", i, runs[i], want[i])
		}
	}
}

func TestColorSerializesFromRuns(t *testing.T) {
	cases := []struct {
		runs []Run
		want string
	}{
		{[]Run{{Text: "a "}, {Text: "red", Color: "#d44c47"}, {Text: " b"}}, `a <span style="color:#d44c47">red</span> b`},
		// Boundary whitespace keeps its background inside the tag.
		{[]Run{{Text: "a"}, {Text: " hi ", Background: "#fbf3db"}, {Text: "b"}}, `a<span style="background-color:#fbf3db"> hi </span>b`},
		// Bold inside a colored word: the tag sits outside the delimiters.
		{[]Run{{Text: "x"}, {Text: "y", Bold: true, Color: "#d44c47"}, {Text: "z"}}, `x<span style="color:#d44c47">**y**</span>z`},
		{[]Run{{Text: "a", Color: "#d44c47"}, {Text: "b", Color: "#d44c47", Italic: true}}, `<span style="color:#d44c47">a*b*</span>`},
		{[]Run{{Text: "a", Color: "#d44c47"}, {Text: "b", Background: "#fbf3db"}}, `<span style="color:#d44c47">a</span><span style="background-color:#fbf3db">b</span>`},
	}
	for _, c := range cases {
		doc := Document{Paragraphs: []Paragraph{{Kind: "body", Runs: c.runs}}}
		if got := Serialize(doc); got != c.want {
			t.Errorf("runs %+v saved as %q, want %q", c.runs, got, c.want)
		}
	}
}

func TestColorRevealsSource(t *testing.T) {
	runs := []Run{{Text: "a "}, {Text: "red", Color: "#d44c47"}}
	reveal, err := RevealInline(runs, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := `a <span style="color:#d44c47">red</span>`; reveal.Source != want {
		t.Fatalf("reveal source %q, want %q", reveal.Source, want)
	}
}

func TestPageCardParagraph(t *testing.T) {
	for source, kind := range map[string]string{
		"![计划](计划.md)":                "page",
		"![x](<dir/my note.md#part>)": "page",
		"![logo](logo.png)":           "image",
		"![x](https://a.com/b.md)":    "image",
		"see ![x](a.md)":              "body",
	} {
		doc := Parse(source)
		if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != kind {
			t.Errorf("%q parsed as %+v, want kind %s", source, doc.Paragraphs, kind)
			continue
		}
		if got := Serialize(doc); got != source {
			t.Errorf("%q saved as %q", source, got)
		}
	}
}
