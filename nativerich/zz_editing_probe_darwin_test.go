//go:build darwin && cgo

package nativerich

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// Exploratory Typora-parity probe (loop task). Each scenario is a list of
// ops; the resulting Markdown is compared with what Typora would produce.
// Run: CORRA_PROBE=1 go test ./nativerich -run TestEditingProbe -v
func probeRun(e *Editor, ops []string) {
	for _, op := range ops {
		switch {
		case op == "⏎":
			nativeTestAction(e, "newline", "")
		case op == "⇧⏎":
			nativeTestAction(e, "linebreak", "")
		case op == "⌫":
			nativeTestAction(e, "backspace", "")
		case op == "⌦":
			nativeTestAction(e, "delete", "")
		case op == "⇥":
			nativeTestAction(e, "tab", "")
		case op == "⇤":
			nativeTestAction(e, "backtab", "")
		case op == "undo":
			nativeTestAction(e, "undo", "")
		case strings.HasPrefix(op, "sel:"):
			parts := strings.Split(op[4:], ",")
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[1])
			nativeTestSelect(e, a, b)
		case strings.HasPrefix(op, "fmt:"):
			e.Format(op[4:])
		case strings.HasPrefix(op, "paste:"):
			nativeTestAction(e, "paste", op[6:])
		default:
			nativeTypeCharacters(e, op)
		}
	}
}

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
		e.SetStyle(1, false, true)
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
