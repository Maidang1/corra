package richtext

import "testing"

func TestEditedInlineStylesKeepBoundaryWhitespaceWhenSaved(t *testing.T) {
	for _, whitespace := range []string{" ", "\u00a0", "\u2003", "\t"} {
		for _, style := range []Run{{Bold: true}, {Italic: true}, {Underline: true}, {Highlight: true}, {Link: "https://example.com"}, {Code: true, Bold: true}} {
			style.Text = whitespace + "edited" + whitespace
			runs := []Run{{Text: "before "}, style, {Text: " after"}}
			for _, table := range []bool{false, true} {
				paragraph := Paragraph{Kind: "body", Runs: runs}
				if table {
					paragraph = Paragraph{Kind: "table", Rows: [][]Cell{{{Runs: []Run{{Text: "Header"}}}}, {{Runs: runs}}}}
				}
				source := Serialize(Document{Paragraphs: []Paragraph{paragraph}})
				parsed := Parse(source)
				if len(parsed.Paragraphs) != 1 {
					t.Errorf("save changed paragraph structure: %q", source)
					continue
				}
				actual := parsed.Paragraphs[0].Runs
				if table {
					actual = parsed.Paragraphs[0].Rows[1][0].Runs
				}
				if !sameRevealRuns(runs, actual) {
					t.Errorf("save lost whitespace style: table=%v style=%+v source=%q actual=%+v", table, style, source, actual)
				}
			}
		}
	}
}
