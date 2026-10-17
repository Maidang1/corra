package richtext

import "testing"

func TestListCodeBlockRoundTrip(t *testing.T) {
	for _, source := range []string{
		"1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next",
		"- a\n\n  ```go\n  func f() {\n\n      return\n  }\n  ```\n- b",
		"- a\n  - nested\n\n    ```\n    x\n    ```",
		"1. step\n   ```\n   cmd\n   ```\n2. next",
	} {
		doc := Parse(source)
		kinds := ""
		for _, p := range doc.Paragraphs {
			kinds += p.Kind + " "
			if p.Kind == "source" {
				t.Fatalf("%q fell back to source: %s", source, kinds)
			}
		}
		if got := Serialize(doc); got != source {
			t.Fatalf("round trip %q -> %q (%s)", source, got, kinds)
		}
	}
}

func TestListCodeBlockEdit(t *testing.T) {
	doc := Parse("1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next")
	for i := range doc.Paragraphs {
		if doc.Paragraphs[i].Kind == "code" {
			doc.Paragraphs[i].Runs = []Run{{Text: "cmd --flag"}}
		}
	}
	if got := Serialize(doc); got != "1. step\n\n   ```bash\n   cmd --flag\n   ```\n\n2. next" {
		t.Fatalf("edit: %q", got)
	}
}

func TestListCodeBlockIrregularStaysSource(t *testing.T) {
	doc := Parse("- a\n\n  ```\n x\n  ```")
	if Serialize(doc) != "- a\n\n  ```\n x\n  ```" {
		t.Fatal("irregular block did not round-trip")
	}
}

func TestEmptyCodeBlockInItem(t *testing.T) {
	source := "1. one\n\n   ```\n   ```\n"
	doc := Parse(source)
	if len(doc.Paragraphs) != 2 || doc.Paragraphs[1].Kind != "code" || doc.Paragraphs[1].CodeIndent != "   " {
		t.Fatalf("parsed %+v", doc.Paragraphs)
	}
	if got := Serialize(doc); got != source {
		t.Fatalf("saved %q", got)
	}
}

func TestItemCodeAndTopLevelCodeStaySeparate(t *testing.T) {
	doc := Parse("1. one\n\n   ```\n   c\n   ```\n\n```\nx\n```\n")
	if len(doc.Paragraphs) != 3 {
		t.Fatalf("parsed %d paragraphs", len(doc.Paragraphs))
	}
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline = "", ""
		doc.Paragraphs[i].Open, doc.Paragraphs[i].Close, doc.Paragraphs[i].CodeClose = "", "", ""
	}
	doc.Paragraphs[1].Separator = "\n"
	got := Parse(Serialize(doc)).Paragraphs
	if len(got) != 3 || got[1].CodeIndent != "   " || got[2].CodeIndent != "" || PlainText(got[2]) != "x" {
		t.Fatalf("saved %q", Serialize(doc))
	}
}

func TestFenceOnItemLine(t *testing.T) {
	doc := Parse("- a\n  - ```js\n    x()\n    ```\n")
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Raw, doc.Paragraphs[i].Baseline, doc.Paragraphs[i].ListBaseline = "", "", ""
	}
	got := Parse(Serialize(doc)).Paragraphs
	if len(got) != 3 || got[2].Kind != "code" || PlainText(got[2]) != "x()" {
		t.Fatalf("saved %q", Serialize(doc))
	}
}
