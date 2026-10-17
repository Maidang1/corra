package richtext

import "testing"

// Shapes the editor can produce that Markdown only reads back correctly when
// written carefully; each must reload to the same document.
func TestSaveStability(t *testing.T) {
	stable := func(name string, doc Document, want string) {
		t.Helper()
		got := Serialize(doc)
		if want != "" && got != want {
			t.Errorf("%s: saved %q, want %q", name, got, want)
		}
		if again := Serialize(Parse(got)); again != got {
			t.Errorf("%s: reload changed %q -> %q", name, got, again)
		}
	}
	// A list that cannot interrupt a paragraph gets its blank line.
	doc := Parse("1. 有序\n2. 第二")
	doc.Paragraphs[0].Kind, doc.Paragraphs[0].Ordinal, doc.Paragraphs[0].ListID = "body", 0, ""
	stable("paragraph then 2.", doc, "有序\n\n2. 第二")
	// One bold span around differently styled runs, not "****".
	stable("bold across styles", Document{Paragraphs: []Paragraph{{Kind: "body", Runs: []Run{{Text: "H", Bold: true}, {Text: "2", Bold: true, Sub: true}, {Text: "O", Bold: true}}}}}, "**H~2~O**")
	// A fence inside code lengthens the block's own fence.
	doc = Parse("```\ncode\n```")
	doc.Paragraphs[0].Runs = []Run{{Text: "```"}}
	stable("fence in code", doc, "````\n```\n````")
	// Empty items, also before a nested list and inside a quote.
	for _, source := range []string{"- \n  - a\n- b", "> - a\n> - \n> - b"} {
		stable(source, Parse(source), source)
	}
}

func TestTableCellBreakSerializesAsBr(t *testing.T) {
	doc := Parse("| a | x |\n| --- | --- |\n| 1 | 2 |\n")
	cell := &doc.Paragraphs[0].Rows[0][1]
	cell.Runs = []Run{{Text: "a"}, {Text: "\n", Break: true}, {Text: "a"}}
	doc.Paragraphs[0].Raw, doc.Paragraphs[0].Baseline = "", ""
	saved := Serialize(doc)
	if want := "| a | a<br>a |\n| --- | --- |\n| 1 | 2 |\n"; saved != want {
		t.Fatalf("saved %q, want %q", saved, want)
	}
	if again := Serialize(Parse(saved)); again != saved {
		t.Fatalf("not a fixed point: %q", again)
	}
}

func TestTextAboveRuleKeepsBlankLine(t *testing.T) {
	doc := Parse("a\n\n---\n")
	doc.Paragraphs[0].Separator = "\n"
	doc.Paragraphs[0].Raw, doc.Paragraphs[0].Baseline = "", ""
	saved := Serialize(doc)
	if saved != "a\n\n---\n" {
		t.Fatalf("saved %q", saved)
	}
}

func TestCodeFormattedFootnoteStaysReference(t *testing.T) {
	doc := Parse("a[^1]\n\n[^1]: n\n")
	runs := doc.Paragraphs[0].Runs
	for i := range runs {
		runs[i].Code = true
	}
	doc.Paragraphs[0].Raw, doc.Paragraphs[0].Baseline = "", ""
	saved := Serialize(doc)
	if saved != "`a`[^1]\n\n[^1]: n\n" {
		t.Fatalf("saved %q", saved)
	}
}

func TestTableKeepsBlankLinesAroundIt(t *testing.T) {
	doc := Parse("- a\n\n| x |\n| --- |\n| 1 |\n\nb\n")
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Separator = "\n"
		doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline = "", ""
	}
	saved := Serialize(doc)
	if again := Serialize(Parse(saved)); again != saved || len(Parse(saved).Paragraphs) != 3 {
		t.Fatalf("saved %q, again %q", saved, again)
	}
}

func TestConsecutiveBreaksKeepEmptyLine(t *testing.T) {
	br := Run{Text: "\n", Break: true}
	doc := Document{Paragraphs: []Paragraph{{Kind: "body", Runs: []Run{{Text: "a"}, br, br, {Text: "# b"}}}}}
	saved := Serialize(doc)
	if saved != "a  \n\\\n\\# b" {
		t.Fatalf("saved %q", saved)
	}
	if got := Parse(saved).Paragraphs; len(got) != 1 || got[0].Kind != "body" {
		t.Fatalf("reparsed as %+v", got)
	}
}

func TestEmptyItemKeepsNestedList(t *testing.T) {
	doc := Parse("1. a\n   1. x\n      1. b\n")
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline = "", ""
		doc.Paragraphs[i].Ordinal, doc.Paragraphs[i].Separator = 2, "\n\n"
	}
	doc.Paragraphs[0].Ordinal, doc.Paragraphs[1].Runs = 1, nil
	doc.Paragraphs[2].Separator = ""
	saved := Serialize(doc)
	got := Parse(saved).Paragraphs
	if len(got) != 3 || got[2].Kind != "ordered" || got[2].Depth != 2 {
		t.Fatalf("saved %q reparsed as %d paragraphs", saved, len(got))
	}
}

func TestSuperscriptEdgeSpaceMovesOut(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{{Kind: "body", Runs: []Run{{Text: "x"}, {Text: "2a ", Sup: true}, {Text: "p"}}}}}
	if saved := Serialize(doc); saved != "x^2a^ p" {
		t.Fatalf("saved %q", saved)
	}
}

func TestItalicSpaceDoesNotCollideDelimiters(t *testing.T) {
	runs := []Run{{Text: "正文"}, {Text: " ", Italic: true}, {Text: "粗", Bold: true, Italic: true}, {Text: "体", Bold: true}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}})
	if saved != "正文 ***粗*体**" {
		t.Fatalf("saved %q", saved)
	}
}

func TestFootnoteDefinitionAfterQuoteKeepsBlankLine(t *testing.T) {
	doc := Parse("> a\n\n[^1]: n\n")
	doc.Paragraphs[0].Separator = "\n"
	doc.Paragraphs[0].Raw, doc.Paragraphs[0].Baseline = "", ""
	if saved := Serialize(doc); saved != "> a\n\n[^1]: n\n" {
		t.Fatalf("saved %q", saved)
	}
}

func TestItemBreakBeforeEqualsLine(t *testing.T) {
	br := Run{Text: "\n", Break: true}
	doc := Document{Paragraphs: []Paragraph{{Kind: "bullet", ListID: "l", ListItem: "i", Runs: []Run{{Text: "a"}, br, {Text: "=="}}}}}
	saved := Serialize(doc)
	if got := Parse(saved).Paragraphs; len(got) != 1 || got[0].Kind != "bullet" {
		t.Fatalf("saved %q reparsed as %+v", saved, got)
	}
}

func TestCodeOverInlineHTML(t *testing.T) {
	runs := []Run{{Text: "a "}, {Text: "<span>", Code: true, HTML: true}, {Text: "x", Code: true}, {Text: "</span>", Code: true, HTML: true}}
	if saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}}); saved != "a `<span>x</span>`" {
		t.Fatalf("saved %q", saved)
	}
}

func TestSoftBreakAfterBreakKeepsParagraph(t *testing.T) {
	runs := []Run{{Text: "a"}, {Text: "\n", Break: true}, {Text: "\n", Break: true, Soft: true}, {Text: "b"}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "quote", Runs: runs}}})
	if got := Parse(saved).Paragraphs; len(got) != 1 {
		t.Fatalf("saved %q reparsed as %d paragraphs", saved, len(got))
	}
}

func TestOverlappingEmphasisDoesNotCollide(t *testing.T) {
	runs := []Run{{Text: "a "}, {Text: "斜", Italic: true}, {Text: "体", Italic: true, Bold: true}, {Text: "，", Bold: true}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}})
	if saved != "a *斜**体***__，__" {
		t.Fatalf("saved %q", saved)
	}
}

func TestSubscriptBesideStrikeKeepsTildesApart(t *testing.T) {
	runs := []Run{{Text: "删", Strike: true}, {Text: "2", Sub: true}, {Text: "O"}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}})
	if saved != "~~删~~<sub>2</sub>O" {
		t.Fatalf("saved %q", saved)
	}
}

func TestOpenMathDraftSavesAsText(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Kind: "source", Runs: []Run{{Text: "$$ x"}}, Separator: "\n\n"},
		{Kind: "body", Runs: []Run{{Text: "after"}}},
	}}
	saved := Serialize(doc)
	if got := Parse(saved).Paragraphs; len(got) != 2 || got[1].Kind != "body" {
		t.Fatalf("saved %q swallows the rest", saved)
	}
}

func TestBrokenTagIsText(t *testing.T) {
	runs := []Run{{Text: "a "}, {Text: "<span\\>", HTML: true}, {Text: "x"}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}})
	if got := PlainText(Parse(saved).Paragraphs[0]); got != "a <span\\>x" {
		t.Fatalf("saved %q reads back %q", saved, got)
	}
}

func TestSuperscriptBeforeLetter(t *testing.T) {
	for _, source := range []string{"x^2^y", "H~2~O"} {
		runs := Parse(source).Paragraphs[0].Runs
		if len(runs) != 3 || !(runs[1].Sup || runs[1].Sub) {
			t.Errorf("%q parsed as %+v", source, runs)
		}
	}
}

func TestHeadingNewlineIsSpace(t *testing.T) {
	runs := []Run{{Text: "a <"}, {Text: "\nb", HTML: true}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "heading", Level: 2, Runs: runs}}})
	if got := Parse(saved).Paragraphs; len(got) != 1 || got[0].Kind != "heading" {
		t.Fatalf("saved %q", saved)
	}
}

func TestStruckSubscriptKeepsTildesApart(t *testing.T) {
	runs := []Run{{Text: "删 H", Strike: true}, {Text: "2", Strike: true, Sub: true}, {Text: "O"}}
	saved := Serialize(Document{Paragraphs: []Paragraph{{Kind: "body", Runs: runs}}})
	if saved != "~~删 H~~<sub>~~2~~</sub>O" {
		t.Fatalf("saved %q", saved)
	}
}
