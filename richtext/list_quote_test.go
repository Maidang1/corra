package richtext

import "testing"

func TestListQuoteRoundTrip(t *testing.T) {
	for _, source := range []string{
		"- a\n\n  > q\n- b",
		"1. step\n\n   > note one\n   >\n   > note two\n\n2. next",
		"- a\n  - b\n\n    > deep\n- c",
		"- a\n  > lazy",
	} {
		doc := Parse(source)
		kinds := ""
		for _, p := range doc.Paragraphs {
			kinds += p.Kind + " "
		}
		if got := Serialize(doc); got != source {
			t.Errorf("round trip %q -> %q (%s)", source, got, kinds)
		}
	}
}

func TestListQuoteEdit(t *testing.T) {
	doc := Parse("1. step\n\n   > note\n\n2. next")
	for i := range doc.Paragraphs {
		if doc.Paragraphs[i].Kind == "quote" {
			if doc.Paragraphs[i].CodeIndent == "" {
				t.Fatalf("quote not in item: %+v", doc.Paragraphs[i])
			}
			doc.Paragraphs[i].Runs = []Run{{Text: "edited"}}
		}
	}
	if got := Serialize(doc); got != "1. step\n\n   > edited\n\n2. next" {
		t.Fatalf("edit: %q", got)
	}
	doc = Parse("- a\n\n  > q")
	doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "quote", Runs: []Run{{Text: "more"}}, CodeIndent: "  ", ListItem: doc.Paragraphs[1].ListItem, ListContinuation: true})
	if got := Serialize(doc); got != "- a\n\n  > q\n  >\n  > more" {
		t.Fatalf("append: %q", got)
	}
}
