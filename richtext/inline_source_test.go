package richtext

import "testing"

func TestExpandedInlineSourcePreservesEditedMarkdown(t *testing.T) {
	for _, fragment := range []string{`$\unknown{中文}$`, `$x+1$`, `$unfinished`, `**strong**`, `[link](https://example.com)`} {
		doc := Document{Paragraphs: []Paragraph{{Kind: "body", Runs: []Run{{Text: "before "}, {Text: fragment, Source: true}, {Text: " after"}}}}}
		if got := Serialize(doc); got != "before "+fragment+" after" {
			t.Errorf("expanded fragment %q: %q", fragment, got)
		}
	}
}
