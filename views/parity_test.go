package views

import (
	"strings"
	"testing"

	"github.com/Maidang1/corra/render"
)

func TestLeavesOf(t *testing.T) {
	var leaves []inlineLeaf
	leavesOf(render.ParseInlines("a **b *i*** ==h== <u>u</u> ^s^ ~b~ `c` [l](https://x) :smile:"), inlineLeaf{}, &leaves)
	joined := ""
	var bold, italic, hl, ul, sup, sub, code bool
	var link string
	for _, lf := range leaves {
		joined += lf.Text
		bold = bold || lf.Bold
		italic = italic || lf.Italic
		hl = hl || lf.Highlight
		ul = ul || lf.Underline
		sup = sup || lf.Sup
		sub = sub || lf.Sub
		code = code || lf.Code
		if lf.Link != "" {
			link = lf.Link
		}
	}
	if joined != "a b i h u s b c l \U0001F604" {
		t.Fatalf("leaves text: %q", joined)
	}
	if !bold || !italic || !hl || !ul || !sup || !sub || !code || link != "https://x" {
		t.Fatalf("styles: %+v", leaves)
	}
}

func TestCountWords(t *testing.T) {
	wc := countWords("你好 world\n第二行 test")
	if wc.Words != 5 { // 你 好 world 第 二 行 test -> 2 + 1 + 3 + 1 = 7? count: CJK 5 + latin 2 = 7
		_ = wc
	}
	if wc.Words != 7 {
		t.Fatalf("words: %+v", wc)
	}
	if wc.Lines != 2 {
		t.Fatalf("lines: %+v", wc)
	}
	if wc.Minutes != 1 {
		t.Fatalf("minutes: %+v", wc)
	}
	if countWords("").Words != 0 {
		t.Fatal("empty")
	}
}

func TestAnchorSlug(t *testing.T) {
	if got := anchorSlug("格式 一览"); got != "格式-一览" {
		t.Fatalf("slug: %q", got)
	}
	if got := anchorSlug("Hello World!"); got != "hello-world" {
		t.Fatalf("slug: %q", got)
	}
}

func TestHeadingNumber(t *testing.T) {
	doc := render.Parse("# A\n\n## A.1\n\n## A.2\n\n### A.2.1\n\n# B\n")
	var nums []string
	for i := range doc.Blocks {
		if doc.Blocks[i].Kind == render.KindHeading {
			nums = append(nums, headingNumber(doc, i))
		}
	}
	want := []string{"1.", "1.1.", "1.2.", "1.2.1.", "2."}
	if strings.Join(nums, ",") != strings.Join(want, ",") {
		t.Fatalf("numbers: %v", nums)
	}
}
