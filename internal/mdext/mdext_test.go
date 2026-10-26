package mdext

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func parse(t *testing.T, src string) ast.Node {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, InlineExtender))
	root := md.Parser().Parse(text.NewReader([]byte(src)))
	TransformHTML(root, []byte(src))
	return root
}

func find(root ast.Node, kind ast.NodeKind) ast.Node {
	var found ast.Node
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == kind && found == nil {
			found = n
		}
		return ast.WalkContinue, nil
	})
	return found
}

func textOf(root ast.Node, src string) string {
	var b strings.Builder
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if t, ok := n.(*ast.Text); ok {
				b.Write(t.Segment.Value([]byte(src)))
			}
			if s, ok := n.(*ast.String); ok {
				b.Write(s.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func TestMark(t *testing.T) {
	root := parse(t, "a ==highlighted== b")
	n := find(root, KindMark)
	if n == nil {
		t.Fatal("no Mark node")
	}
	if got := textOf(n, "a ==highlighted== b"); got != "highlighted" {
		t.Fatalf("mark text %q", got)
	}
}

func TestMarkRejectsSingle(t *testing.T) {
	root := parse(t, "a = b and c = d")
	if find(root, KindMark) != nil {
		t.Fatal("single = should not mark")
	}
}

func TestSuperscript(t *testing.T) {
	root := parse(t, "x^2^ y")
	n := find(root, KindSuperscript)
	if n == nil {
		t.Fatal("no Superscript")
	}
}

func TestSubscript(t *testing.T) {
	root := parse(t, "H~2~O")
	n := find(root, KindSubscript)
	if n == nil {
		t.Fatal("no Subscript")
	}
	if got := textOf(root, "H~2~O"); got != "H2O" {
		t.Fatalf("text %q", got)
	}
}

func TestStrikethroughStillWorks(t *testing.T) {
	src := "~~gone~~"
	root := parse(t, src)
	var strike bool
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind().String() == "Strikethrough" {
			strike = true
		}
		return ast.WalkContinue, nil
	})
	if !strike {
		t.Fatal("~~ should be strikethrough")
	}
	if find(root, KindSubscript) != nil {
		t.Fatal("~~ should not be subscript")
	}
}

func TestUnderlineHTML(t *testing.T) {
	src := "a <u>under *it*</u> b"
	root := parse(t, src)
	n := find(root, KindUnderline)
	if n == nil {
		t.Fatal("no Underline")
	}
	if got := textOf(n, src); got != "under it" {
		t.Fatalf("underline text %q", got)
	}
	// emphasis inside underline must survive
	var italic bool
	ast.Walk(n, func(x ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && x.Kind().String() == "Emphasis" {
			italic = true
		}
		return ast.WalkContinue, nil
	})
	if !italic {
		t.Fatal("nested emphasis lost")
	}
}

func TestHTMLInsideInlineContainers(t *testing.T) {
	for _, src := range []string{
		"**<u>bold</u>**",
		"[***<u>label</u>***](https://example.test)",
		"~~<mark>strike</mark>~~",
		"| <u>cell</u> |\n| --- |",
	} {
		root := parse(t, src)
		if find(root, KindUnderline) == nil && find(root, KindMark) == nil {
			t.Errorf("HTML inside inline container was not transformed: %q", src)
		}
		ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && n.Kind() == ast.KindRawHTML {
				t.Errorf("supported raw HTML left behind: %q", src)
			}
			return ast.WalkContinue, nil
		})
	}
	code := "`<u>literal</u>`"
	if root := parse(t, code); find(root, KindUnderline) != nil || textOf(root, code) != "<u>literal</u>" {
		t.Fatalf("HTML in code must remain literal: %s", code)
	}
}

func TestKbdMarkSubSupHTML(t *testing.T) {
	for src, kind := range map[string]ast.NodeKind{
		"<kbd>Ctrl</kbd>":  KindKbd,
		"<mark>m</mark>":   KindMark,
		"<sub>s</sub>":     KindSubscript,
		"<sup>s</sup>":     KindSuperscript,
		"<ins>i</ins>":     KindUnderline,
		"<u class=x>u</u>": KindUnderline,
	} {
		if find(parse(t, src), kind) == nil {
			t.Fatalf("%s: no %s", src, kind)
		}
	}
}

func TestBrHTML(t *testing.T) {
	src := "a<br>b"
	root := parse(t, src)
	var hard bool
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if tn, ok := n.(*ast.Text); ok && tn.HardLineBreak() {
				hard = true
			}
		}
		return ast.WalkContinue, nil
	})
	if !hard {
		t.Fatal("<br> should become a hard break")
	}
}

func TestSplitFrontMatter(t *testing.T) {
	fm, rest := SplitFrontMatter("---\ntitle: hi\ntags: [a]\n---\n# body\n")
	if !strings.Contains(fm, "title: hi") || rest != "# body\n" {
		t.Fatalf("fm=%q rest=%q", fm, rest)
	}
	fm, rest = SplitFrontMatter("# no fm\n")
	if fm != "" || rest != "# no fm\n" {
		t.Fatalf("no fm: fm=%q rest=%q", fm, rest)
	}
	fm, rest = SplitFrontMatter("---\nunclosed\n")
	if fm != "" {
		t.Fatalf("unclosed fm should not split: %q", fm)
	}
	fm, rest = SplitFrontMatter("---\nt: 1\n...\nbody")
	if !strings.Contains(fm, "...") || rest != "body" {
		t.Fatalf("... fence: fm=%q rest=%q", fm, rest)
	}
}

func TestIsTOC(t *testing.T) {
	if !IsTOC("[TOC]") || !IsTOC("  [toc] \n") {
		t.Fatal("TOC not detected")
	}
	if IsTOC("[TOC] x") {
		t.Fatal("false TOC")
	}
}

func TestEmojiFor(t *testing.T) {
	if e, ok := EmojiFor("smile"); !ok || e == "" {
		t.Fatal("smile missing")
	}
	if _, ok := EmojiFor("nosuchemoji"); ok {
		t.Fatal("bogus emoji found")
	}
}

func TestSuperscriptNoSpaceRule(t *testing.T) {
	// pandoc rule: ^...^ content has no spaces
	if find(parse(t, "2^10 or v1.2^3"), KindSuperscript) != nil {
		t.Fatal("spaced carets should stay text")
	}
	if find(parse(t, "x^2^"), KindSuperscript) == nil {
		t.Fatal("x^2^ should be superscript")
	}
	if find(parse(t, "a~b c~d"), KindSubscript) != nil {
		t.Fatal("spaced tildes should stay text")
	}
	if find(parse(t, "H~2~O"), KindSubscript) == nil {
		t.Fatal("H~2~O should be subscript")
	}
}

func TestColoredSpan(t *testing.T) {
	src := `a <span style="color: #D44C47; background-color:#FBF3DB">red</span> b`
	n, ok := find(parse(t, src), KindSpan).(*Span)
	if !ok {
		t.Fatal("no Span node")
	}
	if n.Color != "#d44c47" || n.Background != "#fbf3db" {
		t.Fatalf("span colors %q %q", n.Color, n.Background)
	}
	if got := textOf(n, src); got != "red" {
		t.Fatalf("span text %q", got)
	}
}

func TestOtherSpansStayRaw(t *testing.T) {
	for _, src := range []string{
		`a <span class="x">b</span> c`,
		`a <span style="font-size:20px">b</span> c`,
		`a <span style="color:red" id="x">b</span> c`,
		`a <span style="color:url(x)">b</span> c`,
	} {
		if find(parse(t, src), KindSpan) != nil {
			t.Errorf("%s: styled as a color span", src)
		}
	}
}

func TestRawSpanKeepsItsClosingTag(t *testing.T) {
	// The inner raw span's </span> must not close the colored one.
	src := `<span style="color:red">a <span class="x">b</span> c</span>`
	n, ok := find(parse(t, src), KindSpan).(*Span)
	if !ok {
		t.Fatal("no Span node")
	}
	var raws int
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*ast.RawHTML); ok {
			raws++
		}
	}
	if raws != 2 {
		t.Fatalf("inner raw tags = %d, want 2", raws)
	}
}

func TestSpanTagRoundTrip(t *testing.T) {
	color, background, ok := SpanStyle(SpanTag("#337ea9", "#fbecdd"))
	if !ok || color != "#337ea9" || background != "#fbecdd" {
		t.Fatalf("round trip = %q %q %v", color, background, ok)
	}
}
