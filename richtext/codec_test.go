package richtext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseSimpleRichParagraphs(t *testing.T) {
	doc := Parse("# Heading\n\nA **bold** and *italic* and ~~gone~~ and `code` [link](https://example.com).\n\n- item\n- [x] done\n\n3. third\n\n> quoted\n")
	wantKinds := []string{"heading", "body", "bullet", "checklist", "ordered", "quote"}
	if len(doc.Paragraphs) != len(wantKinds) {
		t.Fatalf("got %d paragraphs: %+v", len(doc.Paragraphs), doc)
	}
	ids := map[string]bool{}
	for i, kind := range wantKinds {
		p := doc.Paragraphs[i]
		if p.Kind != kind || p.ID == "" || ids[p.ID] {
			t.Errorf("paragraph %d = %+v, want unique ID and kind %s", i, p, kind)
		}
		ids[p.ID] = true
	}
	if doc.Paragraphs[0].Level != 1 || !doc.Paragraphs[3].Checked || doc.Paragraphs[4].Ordinal != 3 {
		t.Fatalf("missing block metadata: %+v", doc)
	}
	checks := map[string]func(Run) bool{
		"bold": func(r Run) bool { return r.Bold }, "italic": func(r Run) bool { return r.Italic },
		"gone": func(r Run) bool { return r.Strike }, "code": func(r Run) bool { return r.Code },
		"link": func(r Run) bool { return r.Link == "https://example.com" },
	}
	for _, run := range doc.Paragraphs[1].Runs {
		if check, ok := checks[run.Text]; ok && check(run) {
			delete(checks, run.Text)
		}
	}
	if len(checks) != 0 {
		t.Errorf("missing formatted runs %v: %+v", checks, doc.Paragraphs[1].Runs)
	}
}

func TestUntouchedDocumentRoundTripsExactlyThroughJSON(t *testing.T) {
	for _, source := range []string{
		"", "\r\n \t\r\n", "\n\n# Title  \r\n\r\nbody\t \r\n\r\n", "one\ntwo\n\nthree", "- one\n- two\n",
		"## Setext?\n\nTitle\n=====\n\n~~~go\n\ncode\n~~~\n\n", "[ref]: /url \"title\"\n\n[ref]\n",
		"<div>\n\ntext\n</div>\n", "| a | b |\n|---|---|\n| 1 | 2 |\n", "- outer\n  - inner\n\nend\n",
	} {
		t.Run(source, func(t *testing.T) {
			encoded, err := json.Marshal(Parse(source))
			if err != nil {
				t.Fatal(err)
			}
			var doc Document
			if err := json.Unmarshal(encoded, &doc); err != nil {
				t.Fatal(err)
			}
			if got := Serialize(doc); got != source {
				t.Errorf("round trip:\n got %q\nwant %q", got, source)
			}
		})
	}
}

func TestEditOneParagraphPreservesEveryOtherByte(t *testing.T) {
	source := " \r\n# old title  \r\n\r\n**Keep** this\t\r\n\r\n```go\r\nvar x = 1\r\n```\r\n\r\nlast  "
	doc := Parse(source)
	for i := range doc.Paragraphs {
		if doc.Paragraphs[i].Kind == "heading" {
			doc.Paragraphs[i].Runs = []Run{{Text: "new title"}}
		}
	}
	if got, want := Serialize(doc), strings.Replace(source, "# old title  ", "# new title", 1); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUnsupportedMarkdownIsEditableSource(t *testing.T) {
	for name, source := range map[string]string{
		"reference_definition": "[ref]: /url", "html": "hello <b>world</b>", "image": "before ![alt](image.png)",
		"complex_table": "| a | b |\n|---|---|\n| ![image](x.png) | 2 |",
		"indent_code":   "    code", "nested_list": "- outer\n  - inner\n\n        indented code",
		"complex_quote": "> - nested", "image_link": "[![alt](image.png)](/url)",
	} {
		t.Run(name, func(t *testing.T) {
			// Untouched source lines round-trip byte-exact.
			doc := Parse(source + "\n\nEditable")
			for i := range doc.Paragraphs[:len(doc.Paragraphs)-1] {
				if p := &doc.Paragraphs[i]; p.Kind != "source" {
					t.Fatalf("not an editable source line: %+v", *p)
				}
			}
			doc.Paragraphs[len(doc.Paragraphs)-1].Runs = []Run{{Text: "Edited"}}
			if got, want := Serialize(doc), source+"\n\nEdited"; got != want {
				t.Errorf("untouched round trip: got %q, want %q", got, want)
			}
			// Editing a source line changes the output, line for line.
			doc = Parse(source + "\n\nEditable")
			doc.Paragraphs[0].Runs = []Run{{Text: "changed line"}}
			first := strings.SplitN(source, "\n", 2)[0]
			if got, want := Serialize(doc), strings.Replace(source, first, "changed line", 1)+"\n\nEditable"; got != want {
				t.Errorf("edited source line: got %q, want %q", got, want)
			}
		})
	}
}

func TestHardLineBreakIsEditableRich(t *testing.T) {
	// Every hard-break spelling becomes one editable rich paragraph with a
	// break run, not a read-only source block.
	for name, source := range map[string]string{
		"spaces":    "line one  \nline two",
		"backslash": "line one\\\nline two",
		"br":        "line one<br>line two",
		"br_spaced": "line one <br/> line two",
		"quote":     "> quoted a  \n> quoted b",
	} {
		t.Run(name, func(t *testing.T) {
			doc := Parse(source)
			if len(doc.Paragraphs) != 1 {
				t.Fatalf("%q split into %d paragraphs", source, len(doc.Paragraphs))
			}
			p := doc.Paragraphs[0]
			if p.Kind == "source" {
				t.Fatalf("%q fell back to read-only source", source)
			}
			breaks := 0
			for _, r := range p.Runs {
				if r.Break {
					breaks++
				}
			}
			if breaks != 1 {
				t.Fatalf("%q: want 1 break run, got %d (%+v)", source, breaks, p.Runs)
			}
			if got := Serialize(doc); got != source {
				t.Errorf("round trip: got %q, want %q", got, source)
			}
		})
	}

	// An edited break paragraph (fresh runs, no Raw) serializes to a hard break
	// and re-parses to the same shape.
	for _, tc := range []struct {
		kind, want string
	}{
		{"body", "first  \nsecond"},
		{"quote", "> first  \n> second"},
	} {
		doc := Document{Paragraphs: []Paragraph{{
			Kind: tc.kind,
			Runs: []Run{{Text: "first"}, {Break: true}, {Text: "second"}},
		}}}
		if got := Serialize(doc); got != tc.want {
			t.Fatalf("%s edited: got %q, want %q", tc.kind, got, tc.want)
		}
		re := Parse(tc.want)
		if len(re.Paragraphs) != 1 || re.Paragraphs[0].Kind != tc.kind {
			t.Fatalf("%s reparse: %+v", tc.kind, re.Paragraphs)
		}
		breaks := 0
		for _, r := range re.Paragraphs[0].Runs {
			if r.Break {
				breaks++
			}
		}
		if breaks != 1 {
			t.Fatalf("%s reparse lost the break: %+v", tc.kind, re.Paragraphs[0].Runs)
		}
	}
}

func TestEscapedDelimitersAndUnicode(t *testing.T) {
	source := "\\*literal\\* &amp; &#x1F600; **中文😀**"
	doc := Parse(source)
	if got, want := PlainText(doc.Paragraphs[0]), "*literal* & 😀 中文😀"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	for text, want := range map[string]int{"": 0, "ascii": 5, "中文": 2, "😀": 2, "a中文😀": 5, "👩‍💻": 5} {
		if got := UTF16Len(text); got != want {
			t.Errorf("UTF16Len(%q) = %d, want %d", text, got, want)
		}
	}
	doc.Paragraphs[0].Runs = []Run{{Text: "# not a heading *nor emphasis* [nor a link] &copy; 中文😀"}}
	changed := Serialize(doc)
	again := Parse(changed)
	if got := PlainText(again.Paragraphs[0]); got != doc.Paragraphs[0].Runs[0].Text || again.Paragraphs[0].Kind != "body" {
		t.Fatalf("escaped serialization changed content: %q -> %+v", changed, again)
	}
}

func TestSerializeNewParagraphsAndChecklistChange(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Kind: "heading", Level: 2, Runs: []Run{{Text: "Title"}}},
		{Kind: "body", Runs: []Run{{Text: "bold", Bold: true}, {Text: " and "}, {Text: "italic", Italic: true}}},
		{Kind: "checklist", Checked: true, Runs: []Run{{Text: "done"}}},
		{Kind: "quote", Runs: []Run{{Text: "quoted"}}},
	}}
	if got, want := Serialize(doc), "## Title\n\n**bold** and _italic_\n\n- [x] done\n\n> quoted"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	doc = Parse("- [ ] todo\r\n- [X] keep\r\n")
	doc.Paragraphs[0].Checked = true
	if got, want := Serialize(doc), "- [x] todo\r\n- [X] keep\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSerializeFormattingPreservesRunSemantics(t *testing.T) {
	for name, runs := range map[string][]Run{
		"adjacent_bold":      {{Text: "one", Bold: true}, {Text: "two", Bold: true}},
		"intraword_italic":   {{Text: "un"}, {Text: "believ", Italic: true}, {Text: "able"}},
		"nested":             {{Text: "bold ", Bold: true}, {Text: "both", Bold: true, Italic: true}, {Text: " end", Bold: true}},
		"styled_code":        {{Text: "some`code", Code: true, Bold: true, Strike: true}},
		"code_spaces":        {{Text: " code ", Code: true}},
		"code_backticks":     {{Text: "``code`", Code: true}},
		"leading_spaces":     {{Text: "    not code"}},
		"entity_escape":      {{Text: `\&copy; &amp;`}},
		"partial_emoji":      {{Text: "a"}, {Text: "😀", Bold: true}},
		"compound_partial":   {{Text: "a"}, {Text: "b", Bold: true, Strike: true}},
		"punctuation_middle": {{Text: "a"}, {Text: "😀!", Bold: true, Italic: true}, {Text: "z"}},
	} {
		t.Run(name, func(t *testing.T) {
			p := Paragraph{Kind: "body", Runs: runs}
			encoded := Serialize(Document{Paragraphs: []Paragraph{p}})
			got := Parse(encoded)
			if len(got.Paragraphs) != 1 || got.Paragraphs[0].Kind != "body" || PlainText(got.Paragraphs[0]) != PlainText(p) {
				t.Fatalf("text changed: %q -> %+v; want %+v", encoded, got, p)
			}
			// Whitespace at a formatting boundary can move outside Markdown
			// delimiters; all non-space characters must retain their attributes.
			styles := func(rs []Run) []Run {
				var out []Run
				for _, r := range rs {
					for _, ch := range r.Text {
						if ch != ' ' {
							r.Text = string(ch)
							out = append(out, r)
						}
					}
				}
				return out
			}
			wantJSON, _ := json.Marshal(styles(runs))
			gotJSON, _ := json.Marshal(styles(got.Paragraphs[0].Runs))
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("styles changed: %q\n got %s\nwant %s", encoded, gotJSON, wantJSON)
			}
		})
	}
}

func TestDollarInCodeAndLinkDestinationIsEditable(t *testing.T) {
	for _, source := range []string{"price `$5` today", "[link](https://example.com/$path)", "[link](https://example.com/?cost=$5)"} {
		doc := Parse(source)
		if doc.Paragraphs[0].Kind != "body" {
			t.Errorf("safe dollars protected: %q -> %+v", source, doc)
		}
		if got := Serialize(doc); got != source {
			t.Errorf("round trip: %q", got)
		}
	}
}

func TestInvalidUTF8SurvivesJSONAndUnrelatedEdit(t *testing.T) {
	source := "before\n\n\xff\n\nafter"
	doc := Parse(source)
	if len(doc.Paragraphs) != 3 || doc.Paragraphs[1].Kind != "opaque" {
		t.Fatalf("invalid byte block not protected: %+v", doc)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Paragraphs[2].Runs = []Run{{Text: "edited"}}
	if got, want := Serialize(doc), "before\n\n\xff\n\nedited"; got != want {
		t.Errorf("bytes lost through JSON: got %q, want %q", got, want)
	}
}

func TestLinkDestinationSurvivesLabelEdit(t *testing.T) {
	for _, source := range []string{`[label](<C:\foo\bar>)`, `[label](<https://example.com/\<literal\>>)`, `[label](<https://example.com/%5Ckeep%3Cescaped%3E>)`} {
		doc := Parse(source)
		if doc.Paragraphs[0].Kind != "body" || len(doc.Paragraphs[0].Runs) != 1 {
			t.Fatalf("unexpected parse: %q -> %+v", source, doc)
		}
		want := doc.Paragraphs[0].Runs[0].Link
		doc.Paragraphs[0].Runs[0].Text = "edited"
		encoded := Serialize(doc)
		got := Parse(encoded)
		if got.Paragraphs[0].Kind != "body" || len(got.Paragraphs[0].Runs) != 1 || got.Paragraphs[0].Runs[0].Link != want {
			t.Errorf("destination changed: %q -> %q -> %+v; want %q", source, encoded, got, want)
		}
	}
}

func TestLinkDestinationEntitiesDecodeOnce(t *testing.T) {
	for source, want := range map[string]string{
		`[label](<https://example.com/?a=1&amp;b=2>)`: "https://example.com/?a=1&b=2",
		`[label](<https://example.com/\&copy;>)`:      "https://example.com/&copy;",
	} {
		doc := Parse(source)
		if got := doc.Paragraphs[0].Runs[0].Link; got != want {
			t.Errorf("parse %q: link %q, want %q", source, got, want)
		}
		doc.Paragraphs[0].Runs[0].Text = "edited"
		encoded := Serialize(doc)
		if got := Parse(encoded).Paragraphs[0].Runs[0].Link; got != want {
			t.Errorf("edit %q -> %q: link %q, want %q", source, encoded, got, want)
		}
	}
}

func TestBlankParagraphsSurviveSerializationAndReopen(t *testing.T) {
	for _, texts := range [][]string{{"", "x"}, {"x", ""}, {"", "", "x"}, {"x", "", "y"}, {"x", "", "", "y"}, {"x", "", ""}, {"", ""}, {"", "", ""}} {
		doc := Document{}
		for _, value := range texts {
			doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "body", Runs: []Run{{Text: value}}})
		}
		encoded := Serialize(doc)
		got := Parse(encoded)
		if len(got.Paragraphs) != len(texts) {
			t.Errorf("%q: %q -> %d paragraphs, want %d", texts, encoded, len(got.Paragraphs), len(texts))
			continue
		}
		for i, value := range texts {
			if PlainText(got.Paragraphs[i]) != value {
				t.Errorf("paragraph %d = %q, want %q", i, PlainText(got.Paragraphs[i]), value)
			}
		}
		if twice := Serialize(got); twice != encoded {
			t.Errorf("extra separators appeared: %q -> %q", encoded, twice)
		}
	}
}

func TestBlankParagraphInsertionIntoExistingSource(t *testing.T) {
	for _, source := range []string{"x", "x\n", "x\r\n", "x\n\ny", "x\n# heading"} {
		for insert := 0; insert <= len(Parse(source).Paragraphs); insert++ {
			doc := Parse(source)
			blank := Paragraph{Kind: "body", Runs: []Run{}}
			doc.Paragraphs = append(doc.Paragraphs[:insert], append([]Paragraph{blank}, doc.Paragraphs[insert:]...)...)
			encoded := Serialize(doc)
			got := Parse(encoded)
			if len(got.Paragraphs) != len(doc.Paragraphs) {
				t.Errorf("insert %d into %q: %q -> %d paragraphs, want %d", insert, source, encoded, len(got.Paragraphs), len(doc.Paragraphs))
			}
		}
	}
}

func TestClearedParagraphStaysAnEmptyParagraph(t *testing.T) {
	doc := Parse("x\n\ny")
	doc.Paragraphs[0].Runs = []Run{}
	encoded := Serialize(doc)
	got := Parse(encoded)
	if len(got.Paragraphs) != 2 || PlainText(got.Paragraphs[0]) != "" || PlainText(got.Paragraphs[1]) != "y" {
		t.Errorf("cleared paragraph changed boundaries: %q -> %+v", encoded, got)
	}
}

func TestPartialFormattingCombinations(t *testing.T) {
	for mask := 1; mask < 16; mask++ {
		for _, value := range []string{"b", "😀", "!", "中文", "a`b"} {
			run := Run{Text: value, Bold: mask&1 != 0, Italic: mask&2 != 0, Strike: mask&4 != 0, Code: mask&8 != 0}
			p := Paragraph{Kind: "body", Runs: []Run{{Text: "a"}, run, {Text: "z"}}}
			encoded := Serialize(Document{Paragraphs: []Paragraph{p}})
			if !sameInlineSemantics(encoded, p.Runs) {
				t.Errorf("formatting lost: mask=%d value=%q Markdown=%q", mask, value, encoded)
			}
		}
	}
}

func TestDisplayMathSpanningBlankLinesIsOneBlock(t *testing.T) {
	source := "before\n\n$$\nfirst\n\nsecond\n$$\n\nafter"
	doc := Parse(source)
	if len(doc.Paragraphs) != 3 || doc.Paragraphs[1].Kind != "math" {
		t.Fatalf("math was split: %+v", doc)
	}
	doc.Paragraphs[2].Runs = []Run{{Text: "changed"}}
	if got, want := Serialize(doc), strings.Replace(source, "after", "changed", 1); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineMathRuns(t *testing.T) {
	doc := Parse("公式 $E=mc^2$ 和 $a+b$\n\n价格 $5 和 $6 不变")
	if len(doc.Paragraphs) != 2 {
		t.Fatalf("paragraphs: %+v", doc.Paragraphs)
	}
	runs := doc.Paragraphs[0].Runs
	if len(runs) != 4 || runs[1].Math != "E=mc^2" || runs[3].Math != "a+b" {
		t.Fatalf("math runs: %+v", runs)
	}
	for _, r := range runs {
		if r.Math != "" && r.Text != "$"+r.Math+"$" {
			t.Fatalf("math run lost its source: %+v", r)
		}
	}
	// $5 ... $6 around digits is not math; the paragraph stays plain text.
	if doc.Paragraphs[1].Kind != "body" || len(doc.Paragraphs[1].Runs) != 1 {
		t.Fatalf("currency must stay text: %+v", doc.Paragraphs[1])
	}
	if got := Serialize(doc); got != "公式 $E=mc^2$ 和 $a+b$\n\n价格 $5 和 $6 不变" {
		t.Errorf("round trip: %q", got)
	}
	// Edits around a math run keep its exact source.
	doc.Paragraphs[0].Runs[0].Text = "改过的"
	if got := Serialize(doc); got != "改过的$E=mc^2$ 和 $a+b$\n\n价格 $5 和 $6 不变" {
		t.Errorf("edited round trip: %q", got)
	}
}

func TestCodeBlockIsEditableLines(t *testing.T) {
	for _, source := range []string{
		"```go\nx := 1\n```", "```\n```", "```python\nprint(1)\n\nprint(2)\n```",
		"~~~\nno fences inside\n~~~", "```go\nx := 1\n```\n\nnext",
	} {
		doc := Parse(source)
		for _, p := range doc.Paragraphs {
			if p.Kind != "code" && p.Kind != "body" {
				t.Fatalf("%q: unexpected kind %q", source, p.Kind)
			}
		}
		if got := Serialize(doc); got != source {
			t.Errorf("round trip: got %q, want %q", got, source)
		}
	}
	// Editing a line keeps the hidden fences.
	doc := Parse("```go\nx := 1\n```")
	doc.Paragraphs[0].Runs = []Run{{Text: "x := 2"}}
	if got, want := Serialize(doc), "```go\nx := 2\n```"; got != want {
		t.Errorf("edited line: got %q, want %q", got, want)
	}
	// A fresh line (no metadata, as a native split produces) stays in the
	// block and the closing fence regenerates.
	doc = Parse("```go\nx := 1\n```")
	doc.Paragraphs = append(doc.Paragraphs, Paragraph{Kind: "code", Runs: []Run{{Text: "y := 2"}}})
	if got, want := Serialize(doc), "```go\nx := 1\ny := 2\n```"; got != want {
		t.Errorf("added line: got %q, want %q", got, want)
	}
	// An unclosed block stays plain source lines.
	doc = Parse("```go\nx := 1")
	if doc.Paragraphs[0].Kind != "source" {
		t.Fatalf("unclosed fence: %+v", doc.Paragraphs[0])
	}
	if got := Serialize(doc); got != "```go\nx := 1" {
		t.Errorf("unclosed round trip: %q", got)
	}
}

func TestChangingParagraphKindKeepsFollowingBlockSeparate(t *testing.T) {
	doc := Parse("# heading\nbody")
	doc.Paragraphs[0].Kind = "body"
	doc.Paragraphs[0].Level = 0
	got := Parse(Serialize(doc))
	if len(got.Paragraphs) != 2 {
		t.Fatalf("block boundaries collapsed: %+v", got)
	}
}

func TestDeletingOrReorderingParagraphPreservesBlockBoundaries(t *testing.T) {
	for name, edit := range map[string]func([]Paragraph) []Paragraph{
		"delete":  func(ps []Paragraph) []Paragraph { return []Paragraph{ps[0], ps[2]} },
		"reorder": func(ps []Paragraph) []Paragraph { return []Paragraph{ps[2], ps[0], ps[1]} },
	} {
		t.Run(name, func(t *testing.T) {
			doc := Parse("body\n# heading\n\nother")
			doc.Paragraphs = edit(doc.Paragraphs)
			encoded := Serialize(doc)
			got := Parse(encoded)
			if len(got.Paragraphs) != len(doc.Paragraphs) {
				t.Fatalf("paragraph boundary collapsed: %q -> %+v", encoded, got)
			}
			for i, want := range doc.Paragraphs {
				if PlainText(got.Paragraphs[i]) != PlainText(want) {
					t.Errorf("paragraph %d changed: %+v", i, got.Paragraphs[i])
				}
			}
		})
	}
}

func TestDeletingParagraphRetainsCRLFBoundaryAfterJSON(t *testing.T) {
	doc := Parse("body\r\n# heading\r\n\r\nother")
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Paragraphs = []Paragraph{doc.Paragraphs[0], doc.Paragraphs[2]}
	if got, want := Serialize(doc), "body\r\n\r\nother"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFootnotesAreProtectedIncludingDefinitionContinuations(t *testing.T) {
	doc := Parse("text[^1]\n\n[^1]: first\n\n    second\n\nafter")
	for _, p := range doc.Paragraphs[:len(doc.Paragraphs)-1] {
		if p.Kind != "source" {
			t.Fatalf("footnote not kept as source: %+v", p)
		}
	}
}

func TestNativeRunFragmentationDoesNotNormalizeUnchangedSource(t *testing.T) {
	source := "__bold__ and  text  \r\n"
	doc := Parse(source)
	doc.Paragraphs[0].Runs = []Run{{Text: "bo", Bold: true}, {Text: "ld", Bold: true}, {Text: " and  text"}}
	if got := Serialize(doc); got != source {
		t.Errorf("native run split changed source: %q", got)
	}
}

func FuzzUntouchedRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "\n\t\r\n", "# title\n- [x] done\n", "$$\na\n\nb\n$$\n", "```go\n\n", "a\r\nb", "-\n-\n", "[x]: /url\n\nx"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if got := Serialize(Parse(source)); got != source {
			t.Fatalf("got %q, want %q", got, source)
		}
	})
}

func TestNewInlineStylesRoundTrip(t *testing.T) {
	src := "plain ==hl== <u>u</u> ^sup^ ~sub~ **b**"
	doc := Parse(src)
	if len(doc.Paragraphs) != 1 {
		t.Fatalf("paragraphs: %+v", doc.Paragraphs)
	}
	runs := doc.Paragraphs[0].Runs
	var hl, ul, sup, sub bool
	for _, r := range runs {
		hl = hl || r.Highlight
		ul = ul || r.Underline
		sup = sup || r.Sup
		sub = sub || r.Sub
	}
	if !hl || !ul || !sup || !sub {
		t.Fatalf("styles lost: %+v", runs)
	}
	// Unedited paragraph round-trips byte-exact via Raw/Baseline.
	if got := Serialize(doc); got != src {
		t.Fatalf("unedited round trip: %q", got)
	}
	// Edited: replace the plain run, styles must re-serialize.
	doc.Paragraphs[0].Runs[0].Text = "changed"
	got := Serialize(doc)
	for _, want := range []string{"changed", "==hl==", "<u>u</u>", "^sup^", "~sub~", "**b**"} {
		if !strings.Contains(got, want) {
			t.Fatalf("edited serialize missing %q: %q", want, got)
		}
	}
	// And the edited result re-parses to the same styles.
	reparsed := Parse(got)
	runs = reparsed.Paragraphs[0].Runs
	hl, ul, sup, sub = false, false, false, false
	for _, r := range runs {
		hl = hl || r.Highlight
		ul = ul || r.Underline
		sup = sup || r.Sup
		sub = sub || r.Sub
	}
	if !hl || !ul || !sup || !sub {
		t.Fatalf("re-parse lost styles: %q -> %+v", got, runs)
	}
}

func TestFrontMatterRoundTrip(t *testing.T) {
	src := "---\ntitle: 标题\ndate: 2026-10-14\n---\n\n# 正文\n\n内容\n"
	doc := Parse(src)
	if got := Serialize(doc); got != src {
		t.Fatalf("front matter round trip: %q", got)
	}
	// Body edits keep front matter intact.
	for i, p := range doc.Paragraphs {
		if p.Kind == "heading" {
			doc.Paragraphs[i].Runs[0].Text = "改动"
		}
	}
	got := Serialize(doc)
	if !strings.HasPrefix(got, "---\ntitle: 标题\n") || !strings.Contains(got, "# 改动") {
		t.Fatalf("edited with fm: %q", got)
	}
}

func TestKbdAndMarkHTML(t *testing.T) {
	doc := Parse("按 <kbd>⌘</kbd> 键和 <mark>重点</mark>")
	runs := doc.Paragraphs[0].Runs
	var kbd, mark bool
	for _, r := range runs {
		if r.Code && r.Text == "⌘" {
			kbd = true
		}
		if r.Highlight && r.Text == "重点" {
			mark = true
		}
	}
	if !kbd || !mark {
		t.Fatalf("kbd/mark: %+v", runs)
	}
}

func TestPlainEqualsAndCaretStayClean(t *testing.T) {
	doc := Parse("x")
	doc.Paragraphs[0].Runs = []Run{{Text: "a = b and 2^10 or v1.2^3"}}
	got := Serialize(doc)
	// '.' was already escaped before (= ordered-list protection); = and ^
	// must stay clean here.
	if strings.Contains(got, "\\=") || strings.Contains(got, "\\^") {
		t.Fatalf("over-escaped: %q", got)
	}
	// A real pair must be escaped to stay text (a==b==c would render as
	// a<mark>b</mark>c; x^y^z as x<sup>y</sup>z).
	doc.Paragraphs[0].Runs = []Run{{Text: "a==b==c and x^y^z"}}
	got = Serialize(doc)
	if !strings.Contains(got, "\\=") || !strings.Contains(got, "\\^") {
		t.Fatalf("pair not escaped: %q", got)
	}
	if !sameInlineSemantics(got, doc.Paragraphs[0].Runs) {
		t.Fatalf("escaped text does not round-trip: %q", got)
	}
}

// Edited prose stays readable Markdown: punctuation is only escaped where a
// re-parse would otherwise change the paragraph (Typora writes clean source).
func TestSerializeEscapesMinimally(t *testing.T) {
	cases := []struct {
		kind, text, want string
	}{
		{"body", "Hi. (test) end-of-line! a+b #1 x|y 3.14", "Hi. (test) end-of-line! a+b #1 x|y 3.14"},
		{"body", "foo_bar_baz a*b price: 2.5", "foo_bar_baz a*b price: 2.5"},
		{"body", "*not italic*", `\*not italic\*`},
		{"body", "1. not a list", `1\. not a list`},
		{"body", "# not a heading", `\# not a heading`},
		{"body", "- not a bullet", `\- not a bullet`},
		{"heading", "Title (draft)", "# Title (draft)"},
		{"bullet", "item - with dash", "- item - with dash"},
	}
	for _, c := range cases {
		p := Paragraph{Kind: c.kind, Level: 1, Runs: []Run{{Text: c.text}}}
		got := Serialize(Document{Paragraphs: []Paragraph{p}})
		if got != c.want {
			t.Errorf("%q: got %q, want %q", c.text, got, c.want)
		}
		again := Parse(got)
		if len(again.Paragraphs) != 1 || again.Paragraphs[0].Kind != c.kind || PlainText(again.Paragraphs[0]) != c.text {
			t.Errorf("%q does not round-trip: %q -> %+v", c.text, got, again.Paragraphs)
		}
	}
}

func TestBareURLStaysRichText(t *testing.T) {
	for _, source := range []string{"visit https://example.com today", "see <https://example.com>"} {
		doc := Parse(source)
		if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "body" {
			t.Fatalf("%q became %+v", source, doc.Paragraphs)
		}
		if Serialize(doc) != source {
			t.Errorf("unchanged %q changed: %q", source, Serialize(doc))
		}
		var linked bool
		for _, run := range doc.Paragraphs[0].Runs {
			linked = linked || run.Link == "https://example.com"
		}
		if !linked {
			t.Errorf("%q lost its link: %+v", source, doc.Paragraphs[0].Runs)
		}
		doc.Paragraphs[0].Runs = append(doc.Paragraphs[0].Runs, Run{Text: "!"})
		got := Serialize(doc)
		if again := Parse(got); len(again.Paragraphs) != 1 || again.Paragraphs[0].Kind != "body" || PlainText(again.Paragraphs[0]) != PlainText(doc.Paragraphs[0]) {
			t.Errorf("edited %q -> %q does not round-trip", source, got)
		}
	}
}

func TestTyporaBlockShapes(t *testing.T) {
	// Each shape loads as editable rich paragraphs and round-trips exactly.
	for source, kinds := range map[string][]string{
		"> a\n>\n> b":         {"quote", "quote"},
		"- \n- a":             {"bullet", "bullet"},
		"mail a@b.com now":    {"body"},
		"> a\r\n>\r\n> b\r\n": {"quote", "quote"},
	} {
		doc := Parse(source)
		var got []string
		for _, p := range doc.Paragraphs {
			got = append(got, p.Kind)
		}
		if strings.Join(got, ",") != strings.Join(kinds, ",") {
			t.Errorf("%q kinds %v != %v", source, got, kinds)
		}
		if out := Serialize(doc); out != source {
			t.Errorf("%q round trip: %q", source, out)
		}
	}
	// New items and quote paragraphs are written the way Typora writes them.
	doc := Document{Paragraphs: []Paragraph{
		{Kind: "quote", Runs: []Run{{Text: "a"}}}, {Kind: "quote", Runs: []Run{{Text: "b"}}},
		{Kind: "bullet", Runs: []Run{{Text: "c"}}}, {Kind: "checklist", Runs: []Run{{Text: "d"}}},
		{Kind: "ordered", Ordinal: 1, Runs: []Run{{Text: "e"}}}, {Kind: "ordered", Ordinal: 2, Runs: []Run{{Text: "f"}}},
	}}
	if got, want := Serialize(doc), "> a\n>\n> b\n\n- c\n- [ ] d\n\n1. e\n2. f"; got != want {
		t.Errorf("new blocks: %q != %q", got, want)
	}
	// Changing a kind never lets the old separator merge blocks.
	for _, c := range []struct {
		source string
		index  int
		kind   string
		want   string
	}{
		{"- a\n- b", 1, "body", "- a\n\nb"},
		{"> a\n>\n> b", 0, "body", "a\n\n> b"},
		{"1. a\n2. b", 1, "bullet", "1. a\n\n- b"},
	} {
		doc := Parse(c.source)
		doc.Paragraphs[c.index].Kind = c.kind
		if got := Serialize(doc); got != c.want {
			t.Errorf("%q kind change: %q != %q", c.source, got, c.want)
		}
	}
}
