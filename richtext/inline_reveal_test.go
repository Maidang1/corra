package richtext

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestRevealInlineContentBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		run        Run
		want       string
		start, end int
	}{
		{"bold", Run{Text: "ab", Bold: true}, "**ab**", 2, 4},
		{"italic", Run{Text: "ab", Italic: true}, "*ab*", 1, 3},
		{"underline", Run{Text: "ab", Underline: true}, "<u>ab</u>", 3, 5},
		{"link", Run{Text: "same", Link: "https://same.test/same", LinkTitle: "same"}, `[same](https://same.test/same "same")`, 1, 5},
		{"code padded", Run{Text: " ab ", Code: true}, "`  ab  `", 2, 6},
		{"code backticks", Run{Text: "`a`", Code: true}, "`` `a` ``", 3, 6},
		{"math", Run{Text: `$x+1$`, Math: "x+1"}, `$x+1$`, 1, 4},
		{"literal source", Run{Text: "**ab**", Source: true}, "**ab**", 0, 6},
		{"bare link", Run{Text: "https://same.test", Link: "https://same.test"}, "https://same.test", 0, 17},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RevealInline([]Run{tc.run}, false)
			if err != nil {
				t.Fatal(err)
			}
			if got.Source != tc.want {
				t.Fatalf("source = %q, want %q", got.Source, tc.want)
			}
			if got.Positions[0] != tc.start || got.Positions[len(got.Positions)-1] != tc.end {
				t.Fatalf("content boundaries = %v, want %d..%d", got.Positions, tc.start, tc.end)
			}
			assertRevealPositions(t, []Run{tc.run}, got)
		})
	}
}

func TestRevealInlineNestedRunsAndUnicode(t *testing.T) {
	runs := []Run{{Text: "a", Bold: true}, {Text: "中😀", Bold: true, Italic: true, Underline: true, Link: "https://中.test", LinkTitle: "中😀"}, {Text: "z", Highlight: true, Sup: true}}
	got, err := RevealInline(runs, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRevealPositions(t, runs, got)
	if !sameRuns(runs, ParseInline(got.Source, false).Paragraphs[0].Runs) {
		t.Fatalf("semantics changed: %q", got.Source)
	}
	chars := utf16.Encode([]rune(got.Source))
	if chars[got.Positions[1]] != '中' || chars[got.Positions[2]] != 0xd83d || got.Positions[3] != got.Positions[2] || chars[got.Positions[4]] != 'z' {
		t.Fatalf("Unicode boundaries mapped into markup or surrogate: %q %v", got.Source, got.Positions)
	}
}

func TestRevealInlineTablePipes(t *testing.T) {
	for _, run := range []Run{{Text: `a\|b`, Code: true}, {Text: "a|b", Bold: true}, {Text: `$\lvert x\rvert$`, Math: `\lvert x\rvert`}, {Text: `$a\|b$`, Math: `a|b`}} {
		got, err := RevealInline([]Run{run}, true)
		if err != nil {
			t.Fatalf("%+v: %v", run, err)
		}
		assertRevealPositions(t, []Run{run}, got)
		doc := ParseInline(got.Source, true)
		if len(doc.Paragraphs) != 1 || !sameRuns([]Run{run}, doc.Paragraphs[0].Runs) {
			t.Fatalf("table semantics changed: %q %+v", got.Source, doc)
		}
	}
}

func TestRevealInlineEscapesAndBreaks(t *testing.T) {
	runs := []Run{{Text: " **x** & <x> ", Bold: true}, {Break: true}, {Text: "next😀", Italic: true}}
	got, err := RevealInline(runs, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRevealPositions(t, runs, got)
	if !strings.Contains(got.Source, "  \n") {
		t.Fatalf("missing hard break: %q", got.Source)
	}
	if !sameRuns(runs, ParseInline(got.Source, false).Paragraphs[0].Runs) {
		t.Fatalf("semantics changed: %q", got.Source)
	}
	before := append([]Run(nil), runs...)
	_, _ = RevealInline(runs, true)
	if !reflect.DeepEqual(before, runs) {
		t.Fatal("mutated input runs")
	}
}

func TestRevealInlineCanonicalCombinedUnderline(t *testing.T) {
	runs := []Run{{Text: "mixed", Bold: true, Italic: true, Underline: true, Link: "https://example.test"}}
	source := serializeParagraph(Paragraph{Kind: "body", Runs: runs})
	doc := ParseInline(source, false)
	if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "body" || !sameRuns(runs, doc.Paragraphs[0].Runs) {
		t.Fatalf("canonical serializer lost combined underline: %q -> %+v", source, doc)
	}
}

func TestInlineCaretMapActualSource(t *testing.T) {
	for _, tc := range []struct {
		source     string
		table      bool
		boundaries map[int]int
	}{
		{"**ab**", false, map[int]int{0: 0, 1: 0, 2: 0, 3: 1, 4: 2, 5: 2, 6: 2}},
		{"_😀中_", false, map[int]int{0: 0, 1: 0, 2: 0, 3: 2, 4: 3, 5: 3}},
		{`[ab](<https://ab.test> "ab")`, false, map[int]int{0: 0, 1: 0, 2: 1, 3: 2, 10: 2, 20: 2, 25: 2}},
		{"`  ab  `", false, map[int]int{0: 0, 1: 0, 2: 0, 3: 1, 4: 2, 5: 3, 6: 4, 7: 4, 8: 4}},
		{"**<u>ab</u>**", false, map[int]int{0: 0, 4: 0, 5: 0, 6: 1, 7: 2, 12: 2, 13: 2}},
		{"a\\*b", false, map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 4: 3}},
		{"`a\\\\|b`", true, map[int]int{0: 0, 1: 0, 2: 1, 3: 2, 4: 2, 5: 3, 6: 4, 7: 4}},
		{"**a|b**", true, map[int]int{0: 0, 2: 0, 3: 1, 4: 2, 5: 3, 7: 3}},
		{"`a|b`", true, map[int]int{0: 0, 1: 0, 2: 1, 3: 2, 4: 3, 5: 3}},
		{"$x+1$", false, map[int]int{0: 0, 1: 0, 2: 0, 3: 1, 4: 1, 5: 1}},
		{"a<br>b", false, map[int]int{0: 0, 1: 1, 5: 2, 6: 3}},
		{"longtext<br>b", false, map[int]int{8: 8, 9: 8, 10: 8, 11: 9, 12: 9, 13: 10}},
		{"a https://same.test b", false, map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 18: 18, 19: 19, 20: 20}},
		{"`a\r\nb`", false, map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 4: 2, 5: 3, 6: 3}},
	} {
		t.Run(tc.source, func(t *testing.T) {
			positions, err := InlineCaretMap(tc.source, tc.table)
			if err != nil {
				t.Fatal(err)
			}
			if len(positions) != UTF16Len(tc.source)+1 {
				t.Fatalf("wrong source boundary count: %v", positions)
			}
			for source, expected := range tc.boundaries {
				if positions[source] != expected {
					t.Errorf("source caret %d -> %d, want %d; %v", source, positions[source], expected, positions)
				}
			}
		})
	}
	if _, err := InlineCaretMap("**broken", false); err != nil {
		t.Fatalf("unfinished syntax must remain ordinary editable text: %v", err)
	}
	if _, err := InlineCaretMap("a\n\nb", false); err == nil {
		t.Fatal("multiple paragraphs cannot fold into one inline cluster")
	}
}

func TestInlineCaretMapInvalidUTF8(t *testing.T) {
	invalid := string([]byte{'a', 0xff, 'b'})
	if _, err := InlineCaretMap(invalid, false); err == nil {
		t.Fatal("invalid UTF-8 cannot have valid source positions")
	}
	if _, err := RevealInline([]Run{{Text: invalid, Bold: true}}, false); err == nil {
		t.Fatal("invalid UTF-8 cannot be revealed")
	}
}

func TestRevealInlineLargeRegion(t *testing.T) {
	content := strings.Repeat("a😀中", 16000)
	got, err := RevealInline([]Run{{Text: content, Bold: true}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Positions) != UTF16Len(content)+1 || got.Positions[len(got.Positions)-1] != UTF16Len(content)+2 {
		t.Fatal("large styled region mapping incomplete")
	}
	positions, err := InlineCaretMap(got.Source, false)
	if err != nil || positions[len(positions)-1] != UTF16Len(content) {
		t.Fatalf("large source fold map incomplete: %v", err)
	}
}

func TestRevealInlinePreservesBoundaryWhitespace(t *testing.T) {
	for _, table := range []bool{false, true} {
		for _, run := range []Run{{Text: " ab ", Bold: true}, {Text: " ", Italic: true}, {Text: "  raw  ", Source: true}} {
			got, err := RevealInline([]Run{run}, table)
			if err != nil {
				t.Fatalf("table=%v run=%+v: %v", table, run, err)
			}
			assertRevealPositions(t, []Run{run}, got)
		}
	}
}

func TestRevealInlinePreservesWhitespaceStyles(t *testing.T) {
	styles := []Run{{Bold: true}, {Italic: true}, {Strike: true}, {Underline: true}, {Highlight: true}, {Sup: true}, {Sub: true}, {Link: "https://example.test", LinkTitle: "title"}, {Code: true, Bold: true}, {Bold: true, Italic: true, Underline: true, Link: "https://example.test"}}
	for _, table := range []bool{false, true} {
		for _, space := range []string{" ", "\u00a0", "\u2003", "\t"} {
			for _, content := range []string{space + "a", "a" + space, space + "a" + space, space} {
				for _, style := range styles {
					run := style
					run.Text = content
					got, err := RevealInline([]Run{run}, table)
					if err != nil {
						t.Errorf("table=%v run=%+v: %v", table, run, err)
						continue
					}
					doc := ParseInline(got.Source, table)
					if len(doc.Paragraphs) != 1 || !reflect.DeepEqual(doc.Paragraphs[0].Runs, []Run{run}) {
						t.Errorf("styled boundary whitespace changed: table=%v want=%+v source=%q got=%+v", table, run, got.Source, doc)
						continue
					}
					assertRevealPositions(t, []Run{run}, got)
					reverse, err := InlineCaretMap(got.Source, table)
					if err != nil {
						t.Error(err)
						continue
					}
					for native, source := range got.Positions {
						if reverse[source] != native {
							t.Errorf("whitespace caret changed: native=%d source=%d reverse=%d (%q)", native, source, reverse[source], got.Source)
						}
					}
				}
			}
		}
	}
}

func TestRevealInlinePreservesAdjacentWhitespaceStyles(t *testing.T) {
	runs := []Run{{Text: "prefix "}, {Text: " \u00a0a\u2003 ", Bold: true}, {Text: " \u2003b\u00a0 ", Italic: true}, {Text: " suffix"}}
	for _, table := range []bool{false, true} {
		got, err := RevealInline(runs, table)
		if err != nil {
			t.Fatal(err)
		}
		doc := ParseInline(got.Source, table)
		if len(doc.Paragraphs) != 1 || !reflect.DeepEqual(doc.Paragraphs[0].Runs, runs) {
			t.Fatalf("adjacent whitespace style changed: table=%v source=%q got=%+v", table, got.Source, doc)
		}
	}
}

func assertRevealPositions(t *testing.T, runs []Run, got InlineReveal) {
	t.Helper()
	length := 0
	for _, run := range runs {
		if run.Math != "" || run.Break {
			length++
		} else {
			length += UTF16Len(run.Text)
		}
	}
	if len(got.Positions) != length+1 {
		t.Fatalf("positions len = %d, want %d: %q %v", len(got.Positions), length+1, got.Source, got.Positions)
	}
	source := utf16.Encode([]rune(got.Source))
	last := -1
	for _, p := range got.Positions {
		if p < last || p < 0 || p > len(source) {
			t.Fatalf("invalid positions: %v source=%q", got.Positions, got.Source)
		}
		if p < len(source) && source[p] >= 0xdc00 && source[p] <= 0xdfff {
			t.Fatalf("position splits surrogate: %v", got.Positions)
		}
		last = p
	}
}
