package richtext

import "testing"

func TestQuotedListRoundTrip(t *testing.T) {
	for _, source := range []string{
		"> - a\n> - b",
		"> intro\n>\n> - a\n>   - nested\n> - b\n>\n> outro",
		"> 1. one\n> 2. two",
		"> - [ ] task\n> - [x] done",
		"> > - deep",
	} {
		doc := Parse(source)
		for _, p := range doc.Paragraphs {
			if p.Kind == "source" {
				t.Fatalf("%q fell back to source lines", source)
			}
		}
		if got := Serialize(doc); got != source {
			t.Fatalf("round trip %q -> %q", source, got)
		}
	}
}

func TestQuotedListEdits(t *testing.T) {
	doc := Parse("> - a\n> - b")
	doc.Paragraphs[1].Runs = []Run{{Text: "b2"}}
	if got := Serialize(doc); got != "> - a\n> - b2" {
		t.Fatalf("edit: %q", got)
	}
	// A new item appended by the editor carries only kind, quote and runs.
	doc = Parse("> - a")
	doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "bullet", Quote: 1, Runs: []Run{{Text: "c"}}})
	if got := Serialize(doc); got != "> - a\n> - c" {
		t.Fatalf("append: %q", got)
	}
	// A quote line after the list stays in the quote.
	doc = Parse("> - a")
	doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "quote", Runs: []Run{{Text: "after"}}})
	if got := Serialize(doc); got != "> - a\n>\n> after" {
		t.Fatalf("quote after list: %q", got)
	}
}

// An empty quote line and a paragraph that starts with a line break have no
// Markdown spelling; saving drops them so a reload gives the same file.
func TestUnrepresentableQuoteStateSavesStably(t *testing.T) {
	doc := Parse("> a\n>\n> b")
	doc.Paragraphs = append(doc.Paragraphs[:1], append([]Paragraph{{Kind: "quote", Runs: []Run{}}}, doc.Paragraphs[1:]...)...)
	doc.Paragraphs[2].Runs = append([]Run{{Break: true}}, doc.Paragraphs[2].Runs...)
	saved := Serialize(doc)
	if again := Serialize(Parse(saved)); again != saved {
		t.Fatalf("not stable: %q -> %q", saved, again)
	}
	if saved != "> a\n>\n> b" {
		t.Fatalf("saved %q", saved)
	}
}

func TestSeparateQuotesAroundQuotedListsStaySeparate(t *testing.T) {
	for _, source := range []string{
		"> - l\n\n> b\n",
		"> a\n\n> - b\n",
		"> - l\n>\n> b\n",
		"> a\n>\n> - b\n",
	} {
		if got := Serialize(Parse(source)); got != source {
			t.Errorf("%q saved as %q", source, got)
		}
	}
}

func TestQuotedEmptyItemHoldingNestedList(t *testing.T) {
	source := "> - \n>   - a\n> - b\n"
	doc := Parse(source)
	if len(doc.Paragraphs) != 3 || doc.Paragraphs[1].Depth != 1 || doc.Paragraphs[1].Quote != 1 {
		t.Fatalf("parsed %+v", doc.Paragraphs)
	}
	if got := Serialize(doc); got != source {
		t.Fatalf("saved %q", got)
	}
}

func TestQuotedListItemsKeepTheirQuoteAcrossBlankLines(t *testing.T) {
	doc := Parse("> - a\n> - b\n")
	doc.Paragraphs[0].Separator = "\n\n"
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline, doc.Paragraphs[i].ListBaseline = "", "", ""
	}
	saved := Serialize(doc)
	if got := Parse(saved).Paragraphs; len(got) != 2 || got[1].Quote != 1 || got[0].ListID != got[1].ListID {
		t.Fatalf("saved %q", saved)
	}
}
