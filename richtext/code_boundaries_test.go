package richtext

import (
	"strings"
	"testing"
)

func TestAdjacentCodeBlocksKeepIndependentFences(t *testing.T) {
	for _, separator := range []string{"\n", "\r\n", "\n\n"} {
		source := "````go\nfirst\n````" + separator + "~~~python\nsecond\n~~~"
		doc := Parse(source)
		if got := Serialize(doc); got != source {
			t.Errorf("untouched adjacent fences merged: %q", got)
		}
		if len(doc.Paragraphs) != 2 {
			t.Fatalf("fixture did not create two editable code lines: %+v", doc)
		}
		doc.Paragraphs[0].Runs[0].Text = "edited"
		if got := Serialize(doc); got != strings.Replace(source, "first", "edited", 1) {
			t.Errorf("editing one code block changed adjacent fences: %q", got)
		}
	}
}
