package render

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"testing"
)

func kinds(doc *Document) []Kind {
	out := make([]Kind, len(doc.Blocks))
	for i, b := range doc.Blocks {
		out[i] = b.Kind
	}
	return out
}

func TestParseGFM(t *testing.T) {
	doc := Parse(`# H1

Paragraph with **bold**, *italic*, ~~strike~~ and ` + "`code`" + `.

## H2

- a
- b

1. one
2. two

| A | B |
|---|--:|
| 1 | 2 |

> quote

---

` + "```go\nfmt.Println()\n```" + `
`)
	want := []Kind{KindHeading, KindParagraph, KindHeading, KindList, KindList,
		KindTable, KindQuote, KindHR, KindCodeBlock}
	got := kinds(doc)
	if len(got) != len(want) {
		t.Fatalf("block count = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d kind = %d, want %d", i, got[i], want[i])
		}
	}
	if doc.Blocks[0].Level != 1 || doc.Blocks[2].Level != 2 {
		t.Errorf("heading levels = %d, %d", doc.Blocks[0].Level, doc.Blocks[2].Level)
	}
	if doc.Blocks[5].Header[0].Text != "A" || doc.Blocks[5].Rows[0][1].Text != "2" {
		t.Errorf("table parse wrong: %+v", doc.Blocks[5])
	}
	if doc.Blocks[5].Rows[0][1].Align != 2 {
		t.Errorf("column alignment = %d, want right(2)", doc.Blocks[5].Rows[0][1].Align)
	}
}

func TestParseTaskList(t *testing.T) {
	t.Run("absolute checkbox offsets", func(t *testing.T) {
		for _, src := range []string{
			"# Before\n\n- [ ] parent\n  - [X] child **bold**\n- plain [x] text\n",
			"1. [x] first\n   1. [ ] second\n2. ordinary\n",
			"> intro\n>\n> - [ ] quoted\n>   - [x] nested\n",
			"- [ ] first\r\n  - [x] second\r\n",
		} {
			t.Run(src, func(t *testing.T) {
				doc := Parse(src)
				var offsets []int
				for _, block := range doc.Blocks {
					for _, item := range block.Items {
						if !item.Task {
							if item.CheckboxOffset != -1 {
								t.Errorf("non-task offset = %d, want -1", item.CheckboxOffset)
							}
							continue
						}
						i := item.CheckboxOffset
						if i <= 0 || i+1 >= len(src) || src[i-1] != '[' || src[i+1] != ']' {
							t.Fatalf("invalid checkbox offset %d in %q", i, src)
						}
						if item.Done != (src[i] == 'x' || src[i] == 'X') {
							t.Errorf("checkbox state doesn't match original source at %d", i)
						}
						offsets = append(offsets, i)
					}
				}
				if len(offsets) != 2 {
					t.Fatalf("task offsets = %v, want exactly two", offsets)
				}
				wantFirst := strings.Index(src, "[") + 1
				wantSecond := strings.Index(src[wantFirst+2:], "[") + wantFirst + 3
				if offsets[0] != wantFirst || offsets[1] != wantSecond {
					t.Fatalf("task offsets = %v, want %d %d", offsets, wantFirst, wantSecond)
				}
			})
		}
	})
	t.Run("loose continuation paragraphs", func(t *testing.T) {
		doc := Parse("- [ ] first\n\n  next paragraph\n\n  - [ ] child\n")
		if len(doc.Blocks) != 1 || len(doc.Blocks[0].Items) != 2 || doc.Blocks[0].Items[0].Source != "first\n\nnext paragraph" {
			t.Fatalf("continuation paragraph lost: %+v", doc.Blocks)
		}
	})
	t.Run("nested items and continuation lines", func(t *testing.T) {
		src := "- [ ] parent\n  continuation\n  - [x] child\n    - plain grandchild\n- sibling\n"
		doc := Parse(src)
		if len(doc.Blocks) != 1 || len(doc.Blocks[0].Items) != 4 {
			t.Fatalf("nested list not retained: %+v", doc.Blocks)
		}
		wantText := []string{"parent\ncontinuation", "child", "plain grandchild", "sibling"}
		wantDepth := []int{0, 1, 2, 0}
		for i, item := range doc.Blocks[0].Items {
			if item.Source != wantText[i] || item.Depth != wantDepth[i] {
				t.Errorf("item %d = %+v, want text %q depth %d", i, item, wantText[i], wantDepth[i])
			}
		}
	})
	doc := Parse("- [ ] todo\n- [x] done\n")
	if len(doc.Blocks) != 1 || len(doc.Blocks[0].Items) != 2 {
		t.Fatalf("blocks = %+v", doc.Blocks)
	}
	if !doc.Blocks[0].Items[0].Task || doc.Blocks[0].Items[0].Done {
		t.Errorf("item0 = %+v", doc.Blocks[0].Items[0])
	}
	if !doc.Blocks[0].Items[1].Done {
		t.Errorf("item1 = %+v", doc.Blocks[0].Items[1])
	}
	if doc.Blocks[0].Items[0].Source != "todo" {
		t.Errorf("item0 source = %q", doc.Blocks[0].Items[0].Source)
	}
}

func TestParseMath(t *testing.T) {
	doc := Parse("inline $E=mc^2$ here\n\n$$\na^2+b^2=c^2\n$$\n")
	if len(doc.Blocks) != 2 {
		t.Fatalf("blocks = %v", kinds(doc))
	}
	if doc.Blocks[0].Kind != KindParagraph {
		t.Fatalf("block0 kind = %d", doc.Blocks[0].Kind)
	}
	ins := ParseInlines(strings.TrimSpace(doc.Blocks[0].Source))
	found := false
	for _, in := range ins {
		if in.Kind == InlineMath && in.Text == "E=mc^2" {
			found = true
		}
	}
	if !found {
		t.Errorf("inline math not found: %+v", ins)
	}
	if doc.Blocks[1].Kind != KindMathBlock || doc.Blocks[1].Math != "a^2+b^2=c^2" {
		t.Errorf("block1 = %+v", doc.Blocks[1])
	}
}

func TestParseDiagramsAndChart(t *testing.T) {
	doc := Parse("```mermaid\nflowchart TD\n A-->B\n```\n\n```d2\na -> b\n```\n\n```goat\n.-.\n```\n\n```chart\ntype: line\nlabels: [一, 二]\nseries:\n  - name: s\n    data: [1, 2]\n```\n")
	if len(doc.Blocks) != 4 {
		t.Fatalf("blocks = %v", kinds(doc))
	}
	if doc.Blocks[0].Kind != KindDiagram || doc.Blocks[0].Diagram != DiagramMermaid {
		t.Errorf("block0 = %+v", doc.Blocks[0])
	}
	if doc.Blocks[1].Diagram != DiagramD2 {
		t.Errorf("block1 = %+v", doc.Blocks[1])
	}
	if doc.Blocks[2].Diagram != DiagramGoAT {
		t.Errorf("block2 = %+v", doc.Blocks[2])
	}
	if doc.Blocks[3].Kind != KindChart {
		t.Errorf("block3 kind = %d", doc.Blocks[3].Kind)
	}
	spec, err := ParseChart(doc.Blocks[3].ChartS)
	if err != nil {
		t.Fatalf("ParseChart: %v", err)
	}
	if spec.Type != "line" || len(spec.Series) != 1 || spec.Series[0].Data[1] != 2 {
		t.Errorf("spec = %+v", spec)
	}
	if spec.Labels[0] != "一" {
		t.Errorf("CJK labels = %v", spec.Labels)
	}
}

func TestParseFootnote(t *testing.T) {
	doc := Parse("text[^1]\n\n[^1]: the note\n")
	if len(doc.Footnotes) != 1 || doc.Footnotes[0] != "the note" {
		t.Errorf("footnotes = %v", doc.Footnotes)
	}
}

func TestHighlight(t *testing.T) {
	spans := Highlight("package main\nfunc main() {}", "go")
	joined := ""
	for _, s := range spans {
		joined += s.Text
	}
	if joined != "package main\nfunc main() {}" {
		t.Errorf("highlight roundtrip = %q", joined)
	}
	classes := map[string]bool{}
	for _, s := range spans {
		classes[s.Class] = true
	}
	if !classes[ClassKeyword] || !classes[ClassText] {
		t.Errorf("classes = %v", classes)
	}
	spans = Highlight("whatever", "nosuchlang")
	if len(spans) != 1 || spans[0].Class != ClassText {
		t.Errorf("fallback = %+v", spans)
	}
}

func TestDiagramMermaidRenders(t *testing.T) {
	ctx := context.Background()
	svg, err := RenderDiagram(ctx, DiagramMermaid, "flowchart TD\n  A[Start] --> B[Done]")
	if err != nil {
		t.Skipf("mermaid wasm unavailable in this environment: %v", err)
	}
	if !strings.Contains(string(svg), "<svg") {
		t.Fatalf("not an svg: %.80q", svg)
	}
	img, err := RasterizeSVG(svg, 1)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}
	if !imageHasInk(img) {
		t.Error("rasterized diagram has no content")
	}
}

func TestDiagramGoAT(t *testing.T) {
	svg, err := RenderDiagram(context.Background(), DiagramGoAT, " A --> B\n")
	if err != nil {
		t.Skipf("goat unavailable: %v", err)
	}
	if !strings.Contains(string(svg), "<svg") {
		t.Fatalf("not an svg: %.80q", svg)
	}
}

func TestDiagramD2(t *testing.T) {
	svg, err := RenderDiagram(context.Background(), DiagramD2, "a -> b")
	if err != nil {
		t.Skipf("d2 unavailable: %v", err)
	}
	if !strings.Contains(string(svg), "<svg") {
		t.Fatalf("not an svg: %.80q", svg)
	}
	img, err := RasterizeSVG(svg, 1)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}
	if !imageHasInk(img) {
		t.Error("rasterized d2 has no content")
	}
}

func TestChartImage(t *testing.T) {
	spec, err := ParseChart("type: line\nlabels: [一, 二, 三]\nseries:\n  - name: 销量\n    data: [3, 7, 5]\n")
	if err != nil {
		t.Fatal(err)
	}
	img, err := RenderChartImage(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !imageHasInk(img) {
		t.Error("chart image is blank")
	}
}

func TestMathRender(t *testing.T) {
	// Scripts, Greek letters and big operators must all typeset: they are the
	// reason mtex and tex are vendored (upstream go-latex lacks Sup/Sub and
	// Latin Modern lacks the symbol glyphs).
	for _, expr := range []string{"E = mc^2", "x^2 + y^2 = z^2", "a_i + b_{ij}", "\\alpha + \\beta", "\\int_0^1 x^2\\,dx"} {
		img, err := RenderMath(expr, false, 14, darkInk)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if !imageHasInk(img) {
			t.Errorf("%s: math image is blank", expr)
		}
	}
}

// imageHasInk reports whether the image has non-white pixels.
func imageHasInk(img image.Image) bool {
	b := img.Bounds()
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 0xf000 || g < 0xf000 || bl < 0xf000 {
				n++
				if n > 50 {
					return true
				}
			}
		}
	}
	return false
}

func TestInlineExtensions(t *testing.T) {
	src := "a ==hl== b <u>u</u> x^2^ H~2~O <kbd>⌘</kbd> :smile: ~~s~~"
	ins := ParseInlines(src)
	var kinds []InlineKind
	for _, in := range ins {
		kinds = append(kinds, in.Kind)
	}
	for _, want := range []InlineKind{InlineHighlight, InlineUnderline, InlineSuperscript, InlineSubscript, InlineKbd, InlineStrikethrough} {
		found := false
		for _, k := range kinds {
			if k == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inline kind %v in %v", want, kinds)
		}
	}
	// emoji expands to unicode text
	var hasEmoji bool
	for _, in := range ins {
		if in.Kind == InlineText && strings.ContainsRune(in.Text, '\U0001F604') {
			hasEmoji = true
		}
	}
	if !hasEmoji {
		t.Fatalf("emoji not expanded: %+v", ins)
	}
}

func TestFrontMatterAndTOC(t *testing.T) {
	doc := Parse("---\ntitle: T\n---\n\n[TOC]\n\n# A\n")
	if len(doc.Blocks) != 3 {
		t.Fatalf("blocks: %d", len(doc.Blocks))
	}
	if doc.Blocks[0].Kind != KindFrontMatter {
		t.Fatalf("first block: %v", doc.Blocks[0].Kind)
	}
	if doc.Blocks[1].Kind != KindTOC {
		t.Fatalf("second block: %v", doc.Blocks[1].Kind)
	}
	if doc.Blocks[2].Kind != KindHeading {
		t.Fatalf("third block: %v", doc.Blocks[2].Kind)
	}
	// offsets are relative to the original source
	if doc.Blocks[2].Start == 0 {
		t.Fatal("heading offset not adjusted for front matter")
	}
}

func TestImageSizeSuffix(t *testing.T) {
	doc := Parse("![a](<img.png =320x200>)\n")
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != KindImage {
		t.Fatalf("blocks: %+v", doc.Blocks)
	}
	b := doc.Blocks[0]
	if b.ImageURL != "img.png" || b.ImageWidth != 320 || b.ImageHeight != 200 {
		t.Fatalf("size: %+v", b)
	}
	doc = Parse("![a](<img.png =50%>)")
	if doc.Blocks[0].ImageZoom != 50 {
		t.Fatalf("zoom: %+v", doc.Blocks[0])
	}
}

func TestImgTagBlock(t *testing.T) {
	doc := Parse(`<img src="pic.png" width="240" alt="p">`)
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != KindImage || doc.Blocks[0].ImageWidth != 240 {
		t.Fatalf("img tag: %+v", doc.Blocks)
	}
	doc = Parse("<div>stay html</div>")
	if doc.Blocks[0].Kind != KindHTML {
		t.Fatalf("non-img html: %+v", doc.Blocks[0])
	}
}

func TestSmartPunct(t *testing.T) {
	doc := ParseOptions(`"quoted" -- dash`, true)
	if doc.Blocks[0].Source == "" {
		t.Fatal("empty")
	}
	ins := ParseInlines(`"quoted"`)
	_ = ins // source-faithful default parse; smart variant exercises below
	b := plainTextBody(`"a" -- b`, true)
	if !strings.ContainsRune(b, '\u201c') {
		t.Fatalf("smart punct missing: %q", b)
	}
}

func TestExportHTML(t *testing.T) {
	src := "---\ntitle: T\n---\n\n# Hi *there*\n\n==mark== <u>u</u> ^s^ ~b~ :smile: ~~x~~\n\n[TOC]\n\n## Two\n\n```mermaid\nA-->B\n```\n\n$E=mc^2$\n"
	out, err := ExportHTML(src, "T", false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"<title>T</title>", "<h1", "<em>there</em>", "<mark>mark</mark>", "<u>u</u>",
		"<sup>s</sup>", "<sub>b</sub>", "<del>x</del>", `<pre class="mermaid">`, "A--&gt;B",
		`class="toc"`, `href="#two"`, "katex", "$E=mc^2$"} {
		if !strings.Contains(s, want) {
			t.Fatalf("export missing %q:\n%s", want, s)
		}
	}
}

func TestSoftBreakRenders(t *testing.T) {
	ins := ParseInlines("第一行\n第二行")
	var breaks int
	for _, in := range ins {
		if in.Kind == InlineHardBreak {
			breaks++
		}
	}
	if breaks != 1 {
		t.Fatalf("soft break lost: %+v", ins)
	}
}

func TestExportPDFWithImageAndBreaks(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	doc := Parse("# T\n\n![p](pic.png =40x)\n\n<div style=\"page-break-after: always\"></div>\n\n[TOC]\n\nsecond\n")
	out, err := ExportPDFOpts(nil, doc, "T", func(url string) ([]byte, error) {
		if url != "pic.png" {
			t.Fatalf("url %q", url)
		}
		return buf.Bytes(), nil
	})
	if err != nil || len(out) < 1000 {
		t.Fatalf("pdf: %v %d", err, len(out))
	}
}
