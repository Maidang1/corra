//go:build darwin && cgo

package nativerich

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exploratory Typora-parity probe (loop task): logs each scenario against
// what Typora would produce. Run: CORRA_PROBE=1 go test ./nativerich -run TestEditingProbe -v
func TestEditingProbe(t *testing.T) {
	if os.Getenv("CORRA_PROBE") == "" {
		t.Skip("set CORRA_PROBE=1")
	}
	type scenario struct {
		name, load string
		ops        []string
		want       string
	}
	end := func(s string) string { return "sel:" + strconv.Itoa(len([]rune(s))) + ",0" }
	_ = end
	cases := []scenario{
		{"ime in list", "", []string{"- ", "ime:zhong|中文", "⏎", "ime:wen|字"}, "- 中文\n- 字"},
		{"ime in heading", "", []string{"## ", "ime:biao|标题", "⏎", "x"}, "## 标题\n\nx"},
		{"ime bold rule", "", []string{"**", "ime:cu|粗", "**", " x"}, "**粗** x"},
		{"ime in table cell", "| a | b |\n|---|---|\n| 1 | 2 |", []string{"sel:5,0", "ime:x|中"}, "| a | b |\n| --- | --- |\n| 1中 | 2 |"},
		{"ime empty doc", "", []string{"ime:ni|你好"}, "你好"},
		{"ime then enter", "", []string{"ime:a|啊", "⏎", "ime:b|吧"}, "啊\n\n吧"},
		{"ime in quote", "", []string{"> ", "ime:yin|引用"}, "> 引用"},
		{"fullwidth paren pair", "", []string{"（"}, "（"},
		{"chinese quote pair", "", []string{"“"}, "“"},
		{"ime replace selection", "abc", []string{"sel:0,3", "ime:x|中"}, "中"},
		{"fmt bullet on ordered", "1. a\n2. b", []string{"sel:0,3", "fmt:bullet"}, "- a\n- b"},
		{"fmt ordered on bullets", "- a\n- b", []string{"sel:0,3", "fmt:ordered"}, "1. a\n2. b"},
		{"fmt checklist on bullet", "- a", []string{"sel:0,0", "fmt:checklist"}, "- [ ] a"},
		{"fmt quote on two paras", "a\n\nb", []string{"sel:0,3", "fmt:quote"}, "> a\n>\n> b"},
		{"fmt heading on bullet", "- a", []string{"sel:0,0", "fmt:heading1"}, "# a"},
		{"fmt body on quote", "> a", []string{"sel:0,0", "fmt:body"}, "a"},
		{"fmt bullet on 3 paras", "a\n\nb\n\nc", []string{"sel:0,5", "fmt:bullet"}, "- a\n- b\n- c"},
		{"fmt bullet toggles off", "- a\n- b", []string{"sel:0,3", "fmt:bullet"}, "a\n\nb"},
		{"fmt clear inline", "**a** b", []string{"sel:0,3", "fmt:clear"}, "a b"},
		{"fmt highlight sel", "abc", []string{"sel:0,3", "fmt:highlight"}, "==abc=="},
		{"fmt strike sel", "abc", []string{"sel:0,3", "fmt:strike"}, "~~abc~~"},
		{"table tab next cell", "| a | b |\n|---|---|\n| 1 | 2 |", []string{"sel:0,0", "⇥", "x"}, "| a | bx |\n| --- | --- |\n| 1 | 2 |"},
		{"table tab adds row", "| a | b |\n|---|---|\n| 1 | 2 |", []string{"sel:6,0", "⇥", "z"}, "| a | b |\n| --- | --- |\n| 1 | 2 |\n| z |  |"},
		{"table bs empty cell stays", "| a | b |\n|---|---|\n|  | 2 |", []string{"sel:4,0", "⌫", "⌫"}, "| a | b |\n| --- | --- |\n|  | 2 |?"},
		{"link via command", "see here", []string{"sel:4,4", "fmt:link:https://x.com"}, "see [here](https://x.com)"},
		{"italic wrap with star", "hello", []string{"sel:0,5", "*"}, "*hello*"},
		{"bold wrap selection twice", "hello", []string{"sel:0,5", "fmt:bold", "fmt:bold"}, "hello"},
		{"code wrap backtick", "hello", []string{"sel:0,5", "`"}, "`hello`"},
		{"enter in empty heading", "# ", []string{"sel:0,0", "⏎", "x"}, "?"},
		{"type after link no leak", "[l](http://x.com)", []string{"sel:1,0", " more"}, "[l](http://x.com) more"},
		{"paste url over selection makes link", "see here", []string{"sel:4,4", "paste:https://x.com"}, "see [here](https://x.com)"},
		{"bs after heading merge into list", "- a\n\n# b", []string{"sel:2,0", "⌫"}, "?"},
		{"checklist enter keeps unchecked", "- [x] done", []string{"sel:4,0", "⏎", "n"}, "- [x] done\n- [ ] n"},
		{"html loaded edit", "<kbd>x</kbd> y", []string{"sel:3,0", "z"}, "<kbd>x</kbd> yz"},
		{"autolink loaded edit", "see https://a.com now", []string{"sel:21,0", "!"}, "see https://a.com now!"},
		{"html typed", "", []string{"<kbd>x</kbd> y"}, "<kbd>x</kbd> y"},
		{"autolink typed", "", []string{"see https://a.com now"}, "see https://a.com now"},
		{"www typed", "", []string{"www.a.com x"}, "www.a.com x"},
		{"angle autolink typed", "", []string{"<https://a.com> x"}, "<https://a.com> x"},
		{"plain dots", "", []string{"v1.2.3 ok"}, "v1.2.3 ok"},
		{"plain lt", "", []string{"a < b > c"}, "a < b > c"},
		{"soft break list edit", "- a\n  b", []string{"sel:3,0", "x"}, "- a\n  bx"},
		{"soft break ordered edit", "1. a\n   b\n2. c", []string{"sel:3,0", "x"}, "1. a\n   bx\n2. c"},
		{"soft break alert edit", "> [!NOTE]\n> one\n> two", []string{"sel:15,0", "x"}, "> [!NOTE]\n> one\n> twox"},
		{"soft break enter split", "a\nb", []string{"sel:3,0", "⏎", "c"}, "a\nb\n\nc"},
		{"soft break bs join", "a\nb", []string{"sel:2,0", "⌫"}, "ab"},
		{"soft break copy paste", "a\nb\n\nz", []string{"sel:0,3", "copy", "sel:6,0", "pasteboard"}, "a\nb\n\nza\nb?"},
		{"soft break edit", "line1\nline2", []string{"sel:11,0", "x"}, "line1\nline2x"},
		{"soft break quote edit", "> a\n> b", []string{"sel:3,0", "x"}, "> a\n> bx"},
		{"bs body into list", "- a\n\nb", []string{"sel:2,0", "⌫"}, "- ab?"},
		{"enter mid item", "- abc", []string{"sel:1,0", "⏎"}, "- a\n- bc"},
		{"enter empty nested item", "- a\n  - b", []string{"sel:3,0", "⏎", "⏎", "c"}, "- a\n  - b\n- c"},
		{"tab first item", "- a", []string{"sel:1,0", "⇥"}, "- a"},
		{"bullet to task", "", []string{"- [ ] t"}, "- [ ] t"},
		{"paren ordered", "", []string{"1) a", "⏎", "b"}, "1) a\n2) b"},
		{"del forward merge", "a\n\nb", []string{"sel:1,0", "⌦"}, "ab"},
		{"del forward into list", "a\n\n- b", []string{"sel:1,0", "⌦"}, "ab?"},
		{"bold toggle off", "**ab** c", []string{"sel:0,2", "fmt:bold"}, "ab c"},
		{"code no leak", "", []string{"`c`", "x"}, "`c`x"},
		{"paste lines into item", "- a", []string{"sel:1,0", "paste:x\ny"}, "- ax\n- y?"},
		{"quote nested via >", "", []string{"> > n"}, "> > n"},
		{"enter in quote mid", "> abc", []string{"sel:1,0", "⏎"}, "> a\n>\n> bc"},
		{"bs empty quote line", "> a\n>\n> ", []string{"sel:2,0", "⏎", "⌫"}, "> a?"},
		{"heading then bs merge", "# T\n\nx", []string{"sel:2,0", "⌫", "⌫"}, "# T?"},
		{"emoji", "", []string{":smile: "}, "😄 "},
		{"html inline", "", []string{"<kbd>x</kbd> y"}, "<kbd>x</kbd> y"},
		{"autolink", "", []string{"https://a.com "}, "https://a.com "},
		{"type after mid math", "$$\nx\n$$\n\nb", []string{"sel:1,0", "a"}, "$$\nx\n$$\n\na\n\nb"},
		{"type before mid math", "a\n\n$$\nx\n$$", []string{"sel:2,0", "z"}, "a\n\nz\n\n$$\nx\n$$"},
		{"ime after math", "$$\nx\n$$", []string{"sel:1,0", "ime:zhong|中"}, "$$\nx\n$$\n\n中"},
		{"undo heading rule only", "", []string{"# ", "undo"}, "# "},
		{"undo bold rule only", "", []string{"**b**", "undo"}, "**b**"},
		{"undo list rule only", "", []string{"- ", "undo"}, "- "},
		{"after code at eof", "```go\nx\n```", []string{"sel:1,0", "down", "down", "after"}, "```go\nx\n```\n\nafter"},
		{"after table at eof", "| a |\n|---|\n| 1 |", []string{"sel:2,0", "down", "down", "after"}, "| a |\n| --- |\n| 1 |\n\nafter"},
		{"after math at eof", "$$\nx\n$$", []string{"sel:0,0", "down", "after"}, "$$\nx\n$$\n\nafter"},
		{"before code at bof", "```go\nx\n```", []string{"sel:0,0", "up", "before"}, "before\n\n```go\nx\n```"},
		{"after quote at eof", "> q", []string{"sel:3,0", "down", "after"}, "> q?"},
		{"heading then body", "", []string{"# Title", "⏎", "body"}, "# Title\n\nbody"},
		{"bullet list continue+exit", "", []string{"- a", "⏎", "b", "⏎", "⏎", "c"}, "- a\n- b\n\nc"},
		{"ordered continue", "", []string{"1. a", "⏎", "b"}, "1. a\n2. b"},
		{"star bullet", "", []string{"* a", "⏎", "b"}, "* a\n* b"},
		{"task continue", "", []string{"- [ ] a", "⏎", "b"}, "- [ ] a\n- [ ] b"},
		{"nested list tab", "", []string{"- a", "⏎", "⇥", "b", "⏎", "⇤", "c"}, "- a\n  - b\n- c"},
		{"quote continue", "", []string{"> q", "⏎", "r"}, "> q\n>\n> r"},
		{"quote exit", "", []string{"> q", "⏎", "⏎", "out"}, "> q\n\nout"},
		{"quote shift-enter", "", []string{"> q", "⇧⏎", "r"}, "> q\n> r"},
		{"code fence", "", []string{"```go", "⏎", "x", "⏎", "y"}, "```go\nx\ny\n```"},
		{"code fence exit", "", []string{"```", "⏎", "x", "⏎", "⏎", "⏎", "after"}, "```\nx\n```\n\nafter"},
		{"math block", "", []string{"$$", "⏎", "x^2"}, "$$\nx^2\n$$"},
		{"math block leave", "a", []string{"sel:1,0", "⏎", "$$", "⏎", "x^2", "sel:0,0"}, "a\n\n$$\nx^2\n$$"},
		{"table type", "", []string{"| a | b |", "⏎", "1", "⏎", "2"}, "| a | b |\n|---|---|\n| 1 | 2 |"},
		{"hr mid doc", "a\n\nb", []string{"sel:1,0", "⏎", "---", "⏎", "x"}, "a\n\n---\n\nx\n\nb"},
		{"paste inline", "ab", []string{"sel:1,0", "paste:**x**"}, "a**x**b"},
		{"paste heading into empty", "a\n\n", []string{"sel:3,0", "paste:## h"}, "a\n\n## h"},
		{"front matter stays", "---\ntitle: x\n---\n\nbody", []string{"sel:20,0", "y"}, "---\ntitle: x\n---\n\nbodyy"},
		{"hr", "", []string{"---", "⏎", "after"}, "---\n\nafter"},
		{"hr stars", "", []string{"***", "⏎", "after"}, "***\n\nafter"},
		{"table", "", []string{"| a | b |", "⏎", "1"}, "| a | b |\n|---|---|\n| 1 |  |"},
		{"bs heading", "# Title", []string{"sel:0,0", "⌫"}, "Title"},
		{"bs bullet", "- a", []string{"sel:0,0", "⌫"}, "a"},
		{"bs quote", "> a", []string{"sel:0,0", "⌫"}, "a"},
		{"bs merge para", "a\n\nb", []string{"sel:2,0", "⌫"}, "ab"},
		{"bs merge into heading", "# a\n\nb", []string{"sel:2,0", "⌫"}, "# ab"},
		{"bs empty item", "- a\n- b", []string{"sel:4,0", "⌫", "⌫", "⌫"}, "- a"},
		{"enter at heading start", "# Title", []string{"sel:0,0", "⏎"}, "\n# Title"},
		{"enter mid heading", "# Title", []string{"sel:2,0", "⏎"}, "# Ti\n\ntle"},
		{"enter end heading", "# Title", []string{"sel:5,0", "⏎", "x"}, "# Title\n\nx"},
		{"enter mid para", "hello", []string{"sel:2,0", "⏎"}, "he\n\nllo"},
		{"shift-enter para", "", []string{"a", "⇧⏎", "b"}, "a\nb"},
		{"bold inline", "", []string{"**b** x"}, "**b** x"},
		{"italic underscore", "", []string{"_i_ x"}, "*i* x"},
		{"code inline", "", []string{"`c` x"}, "`c` x"},
		{"link inline", "", []string{"[l](http://x.com) y"}, "[l](http://x.com) y"},
		{"image inline", "", []string{"![a](x.png)"}, "![a](x.png)"},
		{"image then enter", "", []string{"![a](x.png)", "⏎", "t"}, "![a](x.png)\n\nt"},
		{"rule then enter again", "---", []string{"sel:3,0", "⏎", "t"}, "---\n\nt"},
		{"cmd-b typing", "", []string{"fmt:bold", "b", "fmt:bold", " x"}, "**b** x"},
		{"cmd-b selection", "hello world", []string{"sel:0,5", "fmt:bold"}, "**hello** world"},
		{"select-all delete", "# a\n\n- b\n\n> c", []string{"sel:0,9", "⌫", "x"}, "x"},
		{"paste markdown", "", []string{"paste:# H\n\n- a\n- b"}, "# H\n\n- a\n- b"},
		{"heading cmd", "abc", []string{"sel:1,0", "fmt:heading2"}, "## abc"},
		{"heading again toggles", "## abc", []string{"sel:1,0", "fmt:heading2"}, "abc"},
		{"ordered bs1", "1. a\n2. b\n3. c", []string{"sel:5,0", "⌫"}, "?"},
		{"ordered bs2", "1. a\n2. b\n3. c", []string{"sel:5,0", "⌫", "⌫"}, "?"},
		{"ordered del item", "1. a\n2. b\n3. c", []string{"sel:2,2", "⌫"}, "1. a\n2. c?"},
		{"ordered renumber", "1. a\n2. b\n3. c", []string{"sel:2,0", "⌫", "⌫", "⌫"}, "1. b\n2. c"},
		{"undo heading rule", "", []string{"# T", "undo"}, "# T?"},
		{"enter after quote para in middle", "> a\n> b", []string{"sel:1,0", "⏎", "x"}, "> a\n>\n> x\n> b?"},
		{"ordered at 3", "", []string{"3. a", "⏎", "b"}, "3. a\n4. b"},
		{"footnote ref", "", []string{"a[^1]"}, "a[^1]"},
		{"highlight", "", []string{"==h== x"}, "==h== x"},
		{"strike", "", []string{"~~s~~ x"}, "~~s~~ x"},
		{"autopair bracket", "", []string{"("}, "()"},
		{"autopair star wrap", "hello", []string{"sel:0,5", "*"}, "*hello*"},
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range cases {
			e.Load(c.load)
			probeRun(e, c.ops)
			got, _, err := e.Snapshot()
			got = strings.TrimRight(got, "\n")
			mark := "ok  "
			if got != c.want {
				mark = "DIFF"
			}
			a, b := nativeTestSelection(e)
			t.Logf("%s %-32s got=%q want=%q caret=%d,%d err=%v", mark, c.name, got, c.want, a, b, err)
		}
	})
}

// Run: CORRA_TYPO_SHOT=/tmp/x.png go test ./nativerich -run TestEditingShot
func TestEditingShot(t *testing.T) {
	path := os.Getenv("CORRA_TYPO_SHOT")
	if path == "" {
		t.Skip("set CORRA_TYPO_SHOT=path.png")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 900, 900)
		e.SetVisible(true)
		e.Load("# 编辑测试\n\n上方段落\n\n---\n\n下方段落\n\n> [!NOTE]\n> callout 的第一行\n> callout 的第二行\n")
		e.SetStyle(1, os.Getenv("CORRA_DARK") != "", true)
		nativeTestSelect(e, len([]rune("编辑测试\n上方段落\n---\n下方段落")), 0)
		probeRun(e, []string{"⏎", "***", "⏎", "| 名称 | 值 |", "⏎", "a", "⏎", "1"})
		nativeTestSelect(e, 0, 0)
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		got, _, _ := e.Snapshot()
		t.Logf("%q", got)
	})
}

// Run: CORRA_PERF=1 go test ./nativerich -run TestTypingLatency -v
func TestTypingLatency(t *testing.T) {
	if os.Getenv("CORRA_PERF") == "" {
		t.Skip("set CORRA_PERF=1")
	}
	sizes := []int{25, 100, 300}
	if os.Getenv("CORRA_PERF") == "sample" {
		sizes = []int{100, 100, 100, 100}
	}
	for _, sections := range sizes {
		var b strings.Builder
		for i := 0; i < sections; i++ {
			formula := ""
			if os.Getenv("CORRA_PERF_TABLE") != "" {
				formula = "\n\n| 列 1 | 列 2 | 列 3 |\n|---|---|---|\n| a | b | c |\n| d | e | f |\n\n"
			}
			if os.Getenv("CORRA_PERF_MATH") != "" {
				formula = "行内公式 $x^2+y_" + strconv.Itoa(i) + "$ 与 "
			}
			b.WriteString("## 第 " + strconv.Itoa(i) + " 节\n\n这是一段比较长的正文，" + formula + "混合 English words 与 **粗体**、`code` 和 [链接](https://a.com)，用来模拟真实笔记。\n\n- 列表项一\n- 列表项二\n  - 嵌套项\n\n1. 有序\n2. 有序\n\n> 引用一行\n> 引用第二行\n\n```go\nfunc f() {}\n```\n\n")
		}
		doc := b.String()
		nativeTestMain(func() {
			window := nativeTestWindow()
			defer nativeTestCloseWindow(window)
			e := New(window, func() {}, func() {})
			defer e.Close()
			e.SetFrame(0, 0, 900, 900)
			e.SetVisible(true)
			start := time.Now()
			e.Load(doc)
			load := time.Since(start)
			// The app lays out in the background while the user reads; time
			// edits against a settled layout.
			nativeTestAction(e, "layout", "")
			mid := len([]rune(nativeTestText(e))) / 2
			nativeTestSelect(e, mid, 0)
			start = time.Now()
			n := 20
			if os.Getenv("CORRA_PERF") == "sample" {
				n = 40
			}
			for i := 0; i < n; i++ {
				nativeTestAction(e, "type", "字")
			}
			typing := time.Since(start) / time.Duration(n)
			start = time.Now()
			for i := 0; i < n; i++ {
				nativeTestAction(e, "type", "字")
				nativeTestAction(e, "display", "")
			}
			drawn := time.Since(start) / time.Duration(n)
			start = time.Now()
			for i := 0; i < 5; i++ {
				nativeTestAction(e, "newline", "")
			}
			enter := time.Since(start) / 5
			start = time.Now()
			for i := 0; i < 5; i++ {
				nativeTestAction(e, "backspace", "")
			}
			bs := time.Since(start) / 5
			start = time.Now()
			nativeTestAction(e, "undo", "")
			undo := time.Since(start)
			start = time.Now()
			e.Snapshot()
			snap := time.Since(start)
			t.Logf("paragraphs≈%d chars=%d load=%v type=%v/key type+draw=%v/key enter=%v backspace=%v undo=%v snapshot=%v", sections*13, len(doc), load, typing, drawn, enter, bs, undo, snap)
		})
	}
}

// Run: CORRA_TYPO_SHOT=/tmp/x.png go test ./nativerich -run TestScrolledShot
func TestScrolledShot(t *testing.T) {
	path := os.Getenv("CORRA_TYPO_SHOT")
	if path == "" {
		t.Skip("set CORRA_TYPO_SHOT=path.png")
	}
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("## 第 " + strconv.Itoa(i) + " 节\n\n正文段落，按 <kbd>⌘</kbd> <kbd>K</kbd> 插入`链接`与 ==高亮==。\n\n- 列表项\n- [ ] 任务\n\n> 引用一行\n> 引用第二行\n\n> [!TIP]\n> 提示内容\n\n> [!WARNING]\n> 警告内容\n\n---\n\n```go\nfunc a() {}\nfunc b() {}\nfunc c() {}\nfunc d() {}\n```\n\n| 名称 | 值 |\n|---|---|\n| a | 1 |\n\n")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 900, 700)
		e.SetVisible(true)
		e.Load(b.String())
		e.SetStyle(1, os.Getenv("CORRA_DARK") != "", true)
		text := []rune(nativeTestText(e))
		// Land the viewport inside a code block so its chip starts above.
		target := strings.LastIndex(string(text), "func c")
		nativeTestSelect(e, len([]rune(string(text)[:target])), 0)
		nativeTestAction(e, "display", "")
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
	})
}

func TestImageProbe(t *testing.T) {
	if os.Getenv("CORRA_PROBE") == "" {
		t.Skip("set CORRA_PROBE=1")
	}
	img := writeTestPNG(t, 40, 30, "probe")
	md := "![a](" + img + ")"
	cases := []struct {
		name, load string
		ops        []string
	}{
		{"bs after image", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫"}},
		{"bs at start of para after image", "x\n\n" + md + "\n\ny", []string{"sel:5,0", "⌫"}},
		{"enter after image", md, []string{"sel:1,0", "⏎", "t"}},
		{"type before image", md, []string{"sel:0,0", "t"}},
		{"delete fwd image", "x\n\n" + md, []string{"sel:1,0", "⌦", "⌦"}},
		{"bs after math", "x\n\n$$\nm\n$$\n\ny", []string{"sel:4,0", "⌫"}},
		{"bs after table", "| a |\n|---|\n| 1 |\n\ny", []string{"sel:4,0", "⌫"}},
		{"del before table", "x\n\n| a |\n|---|\n| 1 |", []string{"sel:1,0", "⌦"}},
		{"bs after code", "```\nc\n```\n\ny", []string{"sel:2,0", "⌫"}},
		{"bs twice after image", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫", "⌫"}},
		{"bs once selects image", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫", "z"}},
		{"del fwd once keeps", "x\n\n" + md, []string{"sel:1,0", "⌦"}},
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range cases {
			e.Load(c.load)
			probeRun(e, c.ops)
			got, _, err := e.Snapshot()
			got = strings.ReplaceAll(got, img, "IMG")
			t.Logf("%-34s got=%q text=%q err=%v", c.name, got, strings.ReplaceAll(nativeTestText(e), "￼", "▣"), err)
		}
	})
}

func TestInsertBlockProbe(t *testing.T) {
	if os.Getenv("CORRA_PROBE") == "" {
		t.Skip("set CORRA_PROBE=1")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range []struct{ name, load, insert string }{
			{"code block", "a", "\n```\n\n```\n"},
			{"math block", "a", "\n$$\n\n$$\n"},
			{"hr", "a", "\n---\n"},
			{"task", "a", "\n- [ ] "},
			{"code block empty doc", "", "\n```\n\n```\n"},
			{"table snippet", "a", "\n| 列 1 | 列 2 | 列 3 |\n| --- | --- | --- |\n|  |  |  |\n"},
			{"toc", "# h\n\na", "\n[TOC]\n"},
			{"table snippet mid", "a\n\nb", "\n| 列 1 | 列 2 |\n| --- | --- |\n|  |  |\n"},
		} {
			e.Load(c.load)
			nativeTestSelect(e, len([]rune(c.load)), 0)
			switch strings.TrimSpace(c.insert) {
			case "```\n\n```":
				e.Format("insert:```")
			case "$$\n\n$$":
				e.Format("insert:$$")
			case "---":
				e.Format("insert:---")
			default:
				e.InsertMarkdown(strings.TrimSpace(c.insert))
			}
			nativeTypeCharacters(e, "X")
			got, _, _ := e.Snapshot()
			t.Logf("%-22s got=%q", c.name, got)
		}
	})
}

func TestLoadLatency(t *testing.T) {
	if os.Getenv("CORRA_PERF") == "" {
		t.Skip("set CORRA_PERF=1")
	}
	var b strings.Builder
	for i := 0; i < 300; i++ {
		b.WriteString("## 第 " + strconv.Itoa(i) + " 节\n\n正文 $x^2+y_{" + strconv.Itoa(i) + "}$ 与 $\\frac{a}{b}$ 公式。\n\n$$\n\\int_0^1 f(x)\\,dx\n$$\n\n")
	}
	doc := b.String()
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 900, 900)
		e.SetVisible(true)
		n := 2
		if os.Getenv("CORRA_STYLE_LOG") != "" {
			n = 1
		}
		if os.Getenv("CORRA_PERF") == "sample" {
			n = 12
		}
		start := time.Now()
		for i := 0; i < n; i++ {
			e.Load(doc)
		}
		t.Logf("load=%v", time.Since(start)/time.Duration(n))
	})
}

func TestUndoLatency(t *testing.T) {
	if os.Getenv("CORRA_PERF") == "" {
		t.Skip("set CORRA_PERF=1")
	}
	var b strings.Builder
	for i := 0; i < 300; i++ {
		b.WriteString("## 第 " + strconv.Itoa(i) + " 节\n\n这是一段比较长的正文，混合 English words 与 **粗体**、`code` 和 [链接](https://a.com)。\n\n- 列表项一\n- 列表项二\n\n> 引用一行\n\n")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 900, 900)
		e.SetVisible(true)
		e.Load(b.String())
		nativeTestSelect(e, 5000, 0)
		n := 5
		if os.Getenv("CORRA_PERF") == "sample" {
			n = 60
		}
		var total time.Duration
		for i := 0; i < n; i++ {
			nativeTestAction(e, "type", "字")
			start := time.Now()
			nativeTestAction(e, "undo", "")
			total += time.Since(start)
		}
		t.Logf("undo=%v", total/time.Duration(n))
	})
}

// Run: CORRA_TYPO_SHOT=/tmp/x.png go test ./nativerich -run TestInlineShot
func TestInlineShot(t *testing.T) {
	path := os.Getenv("CORRA_TYPO_SHOT")
	if path == "" {
		t.Skip("set CORRA_TYPO_SHOT=path.png")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 760, 900)
		e.SetVisible(true)
		e.Load(os.Getenv("CORRA_SHOT_DOC"))
		e.SetStyle(1, os.Getenv("CORRA_DARK") != "", true)
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
	})
}
