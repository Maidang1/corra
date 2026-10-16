package richtext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFootnoteDefinitionsAreEditableAtTheirOriginalPosition(t *testing.T) {
	source := "before[^n]\r\n\r\n[^n]: **first**\r\n\r\n    second\r\n\r\nafter"
	doc := Parse(source)
	if len(doc.Paragraphs) != 4 || doc.Paragraphs[1].Kind != "body" || doc.Paragraphs[2].Kind != "body" {
		t.Fatalf("expected editable body, two definition paragraphs, body: %+v", doc.Paragraphs)
	}
	data, _ := json.Marshal(doc.Paragraphs[1])
	if !strings.Contains(string(data), `"footnoteId":"n"`) || !doc.Paragraphs[1].Runs[0].Bold {
		t.Fatalf("missing definition identity or inline formatting: %s", data)
	}
	if got := Serialize(doc); got != source {
		t.Fatalf("untouched definition changed: %q", got)
	}
}

func TestFootnoteReferencesRetainFormattingAndTableIdentity(t *testing.T) {
	for _, source := range []string{
		"**ref[^n]** and [^missing][^missing]",
		"**_ref[^n]_** and [^missing][^missing]",
		"[**ref[^n]**](/url) and [^missing][^missing]",
		"| ref[^n] |\n| --- |",
	} {
		doc := Parse(source)
		runs := doc.Paragraphs[0].Runs
		if doc.Paragraphs[0].Kind == "table" {
			runs = doc.Paragraphs[0].Rows[0][0].Runs
		}
		found := 0
		for _, run := range runs {
			if run.Footnote != "" {
				found++
				if run.Text != "[^"+run.Footnote+"]" {
					t.Fatalf("lost source token: %+v", run)
				}
			}
		}
		if found == 0 || (doc.Paragraphs[0].Kind == "body" && (!runs[1].Bold || found != 3)) {
			t.Fatalf("missing styled references: %+v", runs)
		}
		if got := Serialize(doc); got != source {
			t.Fatalf("roundtrip: %q", got)
		}
		if _, err := RevealInline(runs, doc.Paragraphs[0].Kind == "table"); err != nil {
			t.Fatalf("%q: %v; canonical=%q parsed=%+v runs=%+v", source, err, serializeRuns(runs, escapeNone), ParseInline(serializeRuns(runs, escapeNone), false), runs)
		}
	}
}

func TestFootnoteEditsKeepDefinitionContainersAndCRLF(t *testing.T) {
	source := "before[^n]\r\n\r\n[^n]: **first**\r\n\r\n    second\r\n\r\nafter"
	doc := Parse(source)
	doc.Paragraphs[1].Runs[0].Text = "edited"
	doc.Paragraphs[2].Runs[0].Text = "continued"
	got := Serialize(doc)
	want := "before[^n]\r\n\r\n[^n]: **edited**\r\n    \r\n    continued\r\n\r\nafter"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	reparsed := Parse(got)
	if len(reparsed.Paragraphs) != 4 || reparsed.Paragraphs[2].FootnoteID != "n" {
		t.Fatalf("edited definition lost continuation: %+v", reparsed)
	}
	doc.Paragraphs = append(doc.Paragraphs[:2], doc.Paragraphs[3:]...)
	got = Serialize(doc)
	if !strings.Contains(got, "[^n]: **edited**\r\n\r\nafter") || strings.Contains(got, "second") {
		t.Fatalf("last continuation deletion: %q", got)
	}
}

func TestFootnoteDuplicateDefinitionsAndRichChildrenRoundTrip(t *testing.T) {
	for _, source := range []string{
		"[^x]: one\n\nbody\n\n[^x]: two\n",
		"[^x]: # Heading\n\n    - **list**\n    - next\n\n    | one | two |\n    | --- | --- |\n    | [^x] | $x$ |\n\n    ```go\n    line\n    ```\n\nend",
		"  [^x]:   one\r\n\tcontinued\r\n\r\nend",
		"[^x]: one\n \tcontinued\n\nend",
	} {
		doc := Parse(source)
		if got := Serialize(doc); got != source {
			t.Fatalf("got %q, want %q", got, source)
		}
		groups := map[string]bool{}
		for _, p := range doc.Paragraphs {
			if p.FootnoteID != "" {
				if p.Kind == "source" {
					t.Fatalf("rich footnote protected: %+v", p)
				}
				groups[p.FootnoteGroup] = true
			}
		}
		if len(groups) == 0 || (strings.Contains(source, "continued") && PlainText(doc.Paragraphs[0]) != "one continued") {
			t.Fatalf("missing definition: %+v", doc)
		}
		if strings.Contains(source, "[^x]: two") && len(groups) != 2 {
			t.Fatalf("duplicates merged: %+v", doc)
		}
	}
}

func TestFootnoteReferencesRespectCodeEscapesAndLinks(t *testing.T) {
	doc := Parse("`[^code]` \\[^escape] [label[^ref]](/url) [^orphan]")
	refs := []string{}
	for _, run := range doc.Paragraphs[0].Runs {
		if run.Footnote != "" {
			refs = append(refs, run.Footnote)
		}
	}
	if strings.Join(refs, ",") != "ref,orphan" {
		t.Fatalf("wrong reference recognition: %+v", doc)
	}
}

func TestFootnoteInlineCaretMapUsesOneNativeAttachment(t *testing.T) {
	runs := []Run{{Text: "a"}, {Text: "[^long]", Footnote: "long"}, {Text: "b"}}
	reveal, err := RevealInline(runs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reveal.Positions) != 4 {
		t.Fatalf("expected 3 native characters: %+v", reveal)
	}
	positions, err := InlineCaretMap(reveal.Source, false)
	if err != nil {
		t.Fatal(err)
	}
	if positions[len(positions)-1] != 3 {
		t.Fatalf("footnote expanded native length: %v", positions)
	}
}

func TestFootnoteNativeFragmentationKeepsOriginalDefinitionBytes(t *testing.T) {
	source := "[^n]: __first__\r\n\r\n    second\r\n"
	doc := Parse(source)
	doc.Paragraphs[0].Runs = []Run{{Text: "fi", Bold: true}, {Text: "rst", Bold: true}}
	if got := Serialize(doc); got != source {
		t.Fatalf("fragmentation normalized untouched definition: %q", got)
	}
}

func TestFootnoteFirstParagraphDeletionRecoversMarker(t *testing.T) {
	doc := Parse("  [^n]:   first\n\n    __second__\n\nend")
	doc.Paragraphs = doc.Paragraphs[1:]
	doc.Paragraphs[0].FootnoteContinuation = false
	got := Serialize(doc)
	if !strings.HasPrefix(got, "  [^n]:   ") || !strings.Contains(got, "second") || strings.Contains(got, "first") {
		t.Fatalf("lost surviving definition: %q", got)
	}
	if Parse(got).Paragraphs[0].FootnoteID != "n" {
		t.Fatalf("survivor no longer a definition: %q", got)
	}
}

func TestFootnoteExpandedSourceAndStyledReferenceReveal(t *testing.T) {
	runs := []Run{{Text: "**live**", Source: true}, {Text: " "}, {Text: "ref", Bold: true}, {Text: "[^n]", Footnote: "n", Bold: true}}
	reveal, err := RevealInline(runs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reveal.Positions) != len("**live** ")+3+1+1 {
		t.Fatalf("lost source or attachment positions: %+v", reveal)
	}
}

func TestFootnoteEditedTableAndCodeKeepContainerBoundaries(t *testing.T) {
	doc := Parse("[^n]: text\r\n\r\n    | one |\r\n    | --- |\r\n\r\n    ```go\r\n    first\r\n    second\r\n    ```\r\n\r\nend")
	doc.Paragraphs[1].Rows[0][0].Runs[0].Text = "edited"
	doc.Paragraphs = append(doc.Paragraphs[:3], doc.Paragraphs[4:]...)
	got := Serialize(doc)
	reparsed := Parse(got)
	if len(reparsed.Paragraphs) != 4 || reparsed.Paragraphs[1].Kind != "table" || reparsed.Paragraphs[2].Kind != "code" || reparsed.Paragraphs[2].FootnoteID != "n" || reparsed.Paragraphs[3].FootnoteID != "" {
		t.Fatalf("edited table/code escaped container: %q; %+v", got, reparsed)
	}
	if strings.Contains(got, "second") || strings.Count(got, "```") != 2 {
		t.Fatalf("deleted last code line lost fences: %q", got)
	}
}

func TestFootnoteOuterEditPreservesNestedDefinitionSource(t *testing.T) {
	source := "[^n]: a\n\n    [^m]: **nested**\n\n        continued\n\nafter"
	doc := Parse(source)
	if got := Serialize(doc); got != source {
		t.Fatalf("untouched nested definition: %q", got)
	}
	doc.Paragraphs[0].Runs[0].Text = "a!"
	got := Serialize(doc)
	if !strings.Contains(got, "    [^m]: **nested**") || !strings.Contains(got, "        continued") {
		t.Fatalf("outer edit discarded nested definition source: %q", got)
	}
	if !strings.HasSuffix(got, "\n\nafter") {
		t.Fatalf("nested definition swallowed following body: %q", got)
	}
	for _, p := range doc.Paragraphs[1 : len(doc.Paragraphs)-1] {
		if p.FootnoteID != "n" || p.Kind != "source" {
			t.Fatalf("nested definition should retain editable source in outer container: %+v", p)
		}
	}
}
