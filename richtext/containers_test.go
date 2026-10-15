package richtext

import (
	"strings"
	"testing"
)

func TestMultiParagraphListEditing(t *testing.T) {
	for _, source := range []string{
		"- first\n\n  second **bold**\n- next",
		"3. first\n\n   second\n\n   third\n4. next",
		"+ first\r\n\r\n  second\r\n  - child\r\n\r\n  after child\r\n+ next\r\n",
	} {
		doc := Parse(source)
		if len(doc.Paragraphs) < 3 || doc.Paragraphs[0].Kind == "source" {
			t.Errorf("multi-paragraph list is not editable: %+v", doc)
			continue
		}
		if got := Serialize(doc); got != source {
			t.Errorf("no-edit roundtrip: %q != %q", got, source)
		}
		for i := range doc.Paragraphs {
			if strings.HasPrefix(PlainText(doc.Paragraphs[i]), "second") {
				doc.Paragraphs[i].Runs[0].Text = "edited"
			}
		}
		got := Serialize(doc)
		parsed := Parse(got)
		if len(parsed.Paragraphs) != len(doc.Paragraphs) || !strings.Contains(got, "edited") {
			t.Errorf("continuation changed structure: %q %+v", got, parsed)
		}
	}
}

func TestNestedQuoteEditing(t *testing.T) {
	for _, source := range []string{
		"> outer\n>\n> > inner\n> >\n> > deeper paragraph\n>\n> tail",
		"> > > 深层 **bold**\r\n> >\r\n> > next\r\n",
	} {
		doc := Parse(source)
		for _, p := range doc.Paragraphs {
			if p.Kind != "quote" {
				t.Errorf("nested quote is not editable: %+v", doc)
				break
			}
		}
		if got := Serialize(doc); got != source {
			t.Errorf("quote roundtrip: %q", got)
		}
		if len(doc.Paragraphs) == 0 {
			continue
		}
		doc.Paragraphs[0].Runs[0].Text = "edited"
		got := Serialize(doc)
		if !strings.Contains(got, "edited") || len(Parse(got).Paragraphs) != len(doc.Paragraphs) {
			t.Errorf("quote edit changed hierarchy: %q", got)
		}
	}
}

func TestEditableWrappedListParagraphsKeepCommonMarkMeaning(t *testing.T) {
	for _, source := range []string{"- parent\n  - child\n    continuation\n- sibling", "3. parent\n   7. child\n4. end"} {
		doc := Parse(source)
		if got := Serialize(doc); got != source {
			t.Errorf("wrapped source roundtrip: %q", got)
		}
		for _, p := range doc.Paragraphs {
			if p.Kind == "source" {
				t.Errorf("wrapped paragraph is not editable: %+v", doc)
				break
			}
		}
		doc.Paragraphs[0].Runs[0].Text += " edited"
		if got := Serialize(doc); len(Parse(got).Paragraphs) != len(doc.Paragraphs) {
			t.Errorf("wrapped paragraph edit changed structure: %q", got)
		}
	}
}
