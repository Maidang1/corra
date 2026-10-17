//go:build darwin && cgo

package nativerich

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// probeRun replays editing ops: plain strings are typed character by
// character; ⏎ ⇧⏎ ⌫ ⌦ ⇥ ⇤ up down undo, sel:start,len, fmt:command,
// paste:text and ime:marked|committed drive the matching actions.
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
		case op == "down":
			nativeTestAction(e, "move-down", "")
		case op == "up":
			nativeTestAction(e, "move-up", "")
		case op == "undo":
			nativeTestAction(e, "undo", "")
		case op == "⌘⏎":
			nativeTestAction(e, "key", "cmd-return")
		case strings.HasPrefix(op, "sel:"):
			parts := strings.Split(op[4:], ",")
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[1])
			nativeTestSelect(e, a, b)
		case strings.HasPrefix(op, "fmt:"):
			e.Format(op[4:])
		case strings.HasPrefix(op, "insert:"):
			e.Format(op)
		case op == "click-below":
			nativeTestAction(e, op, "")
		case op == "copy" || op == "pasteboard":
			nativeTestAction(e, op, "")
		case strings.HasPrefix(op, "ime:"):
			parts := strings.SplitN(op[4:], "|", 2)
			nativeTestAction(e, "mark", parts[0])
			nativeTestAction(e, "commitIME", parts[1])
		case strings.HasPrefix(op, "paste:"):
			nativeTestAction(e, "paste", op[6:])
		default:
			nativeTypeCharacters(e, op)
		}
	}
}

// Typora behaviors that a Markdown writer reaches for without thinking.
func TestNativeTyporaEditing(t *testing.T) {
	cases := []struct {
		name, load string
		ops        []string
		want       string
	}{
		{"paste markdown blocks", "", []string{"paste:# H\n\n- a\n- b"}, "# H\n\n- a\n- b"},
		{"paste inline markdown", "ab", []string{"sel:1,0", "paste:**x**"}, "a**x**b"},
		{"paste lines into revealed bold", "a **b** c", []string{"sel:3,0", "paste:x\n\n# y"}, "a **b**x\n\n# y c"},
		{"paste heading into empty line", "a\n\n", []string{"sel:3,0", "paste:## h"}, "a\n\n## h"},
		{"rule on enter", "", []string{"---", "⏎", "after"}, "---\n\nafter"},
		{"typed br is a line break", "", []string{"a<br>b"}, "a  \nb"},
		{"cjk paren pairs", "", []string{"（注", "）", "后"}, "（注）后"},
		{"cjk quote pairs", "", []string{"“引", "⌫", "⌫", "x"}, "x"},
		{"wrap selection keeps rest", "a bc d", []string{"sel:2,2", "【"}, "a 【bc】 d"},
		{"enter keeps quote together", "> note\n\nafter", []string{"sel:4,0", "⏎", "more"}, "> note\n>\n> more\n\nafter"},
		{"enter keeps list tight", "- a\n\nafter", []string{"sel:1,0", "⏎", "b"}, "- a\n- b\n\nafter"},
		{"quote in item: list marker stays text", "1. step\n\n   > note\n\n2. next", []string{"sel:5,0", "1. "}, "1. step\n\n   > 1\\. note\n\n2. next"},
		{"quote in item to bullet", "1. step\n\n   > note\n\n2. next", []string{"sel:5,0", "fmt:bullet"}, "1. step\n\n- note\n\n2. next"},
		{"quote in item: type", "1. step\n\n   > note\n\n2. next", []string{"sel:9,0", "!"}, "1. step\n\n   > note!\n\n2. next"},
		{"quote in item: enter", "1. step\n\n   > note\n\n2. next", []string{"sel:9,0", "⏎", "more"}, "1. step\n\n   > note\n   >\n   > more\n\n2. next"},
		{"quote in item: leave", "1. step\n\n   > note\n\n2. next", []string{"sel:9,0", "⏎", "⏎", "after"}, "1. step\n\n   > note\n\n   after\n\n2. next"},
		{"cmd-return leaves code", "```\na\nb\n```\n\nnext", []string{"sel:0,0", "⌘⏎", "x"}, "```\na\nb\n```\n\nx\n\nnext"},
		{"cmd-return in code in item", "1. step\n\n   ```\n   cmd\n   ```", []string{"sel:6,0", "⌘⏎", "x"}, "1. step\n\n   ```\n   cmd\n   ```\n\n   x"},
		{"code in item: type", "1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next", []string{"sel:8,0", "x"}, "1. step\n\n   ```bash\n   cmdx\n   ```\n\n2. next"},
		{"code in item: enter", "1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next", []string{"sel:8,0", "⏎", "ls"}, "1. step\n\n   ```bash\n   cmd\n   ls\n   ```\n\n2. next"},
		{"code in item: leave", "1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next", []string{"sel:8,0", "⏎", "⏎", "after"}, "1. step\n\n   ```bash\n   cmd\n   ```\n\n   after\n\n2. next"},
		{"paste code mid-line keeps tail", "ab", []string{"sel:1,0", "paste:x\n\n```\nc\n```"}, "ax\n\n```\nc\n```\n\nb"},
		{"shortened rule is text", "a\n\n---\n\nb", []string{"sel:5,0", "⌫", "⌫"}, "a\n\n\\-\n\nb"},
		{"rest of deleted formula is text", "$$\nx\n$$\n\n---", []string{"sel:0,4", "⌫"}, "\\-"},
		{"paste code over lines keeps tail", "ab\n\ncd", []string{"sel:1,3", "paste:x\n\n```\nc\n```"}, "ax\n\n```\nc\n```\n\nd"},
		{"typing into rule makes text", "a\n\n---\n\nb", []string{"sel:2,0", "x", "sel:5,0", "⇧⏎"}, "a\n\nx\\-\\-  \n\\-\n\nb"},
		{"delete from cell out of table", "| a | b |\n| --- | --- |\n| 1 | 2 |\n\nafter here", []string{"sel:4,8", "⌫", "x"}, "| a | b |\n| --- | --- |\n| x |  |\n\nr here"},
		{"type over cells and text after", "| a | b |\n| --- | --- |\n| 1 | 2 |\n\nafter here", []string{"sel:1,10", "x"}, "| ax |  |\n| --- | --- |\n|  |  |\n\ner here"},
		{"delete from before table into cell", "before\n\n| a | b |\n| --- | --- |\n| 1 | 2 |", []string{"sel:3,5", "⌫", "x"}, "befx\n\n|  | b |\n| --- | --- |\n| 1 | 2 |"},
		{"undo delete across table", "| a | b |\n| --- | --- |\n| 1 | 2 |\n\nafter here", []string{"sel:4,8", "⌫", "undo"}, "| a | b |\n| --- | --- |\n| 1 | 2 |\n\nafter here"},
		{"nested quote to list", "> > a", []string{"sel:0,0", "fmt:ordered"}, "1. a"},
		{"typing over toc is text", "a\n\n[TOC]\n\nb", []string{"sel:2,1", "-"}, "a\n\n\\-\n\nb"},
		{"heading on raw html stays raw", "<details>\n<summary>s</summary>\n\nbody\n</details>", []string{"sel:11,0", "fmt:heading2"}, "<details>\n<summary>s</summary>\n\nbody\n</details>"},
		{"paste into raw html is text", "<details>\n<summary>s</summary>\n\nbody\n</details>", []string{"sel:13,0", "paste:# x"}, "<details>\n<su# xmmary>s</summary>\n\nbody\n</details>"},
		{"tag spanning lines round-trips", "> q\n\n<d\netails>\n<summary>s</summary>\n\nx\n</details>", []string{}, "> q\n\n<d\netails>\n<summary>s</summary>\n\nx\n</details>"},
		{"tag spanning lines edits", "a <span\nclass=\"x\">b</span>", []string{"sel:1,0", "z"}, "az <span\nclass=\"x\">b</span>"},
		{"line break inside tag", "a <span>x</span>", []string{"sel:4,0", "⇧⏎"}, "a <s\npan>x</span>"},
		{"unclosed display math stays text", "a", []string{"sel:1,0", "⏎", "$$ x", "up"}, "a\n\n\\$\\$ x"},
		{"enter inside rule", "a\n\n---\n\nb", []string{"sel:3,0", "⏎", "x"}, "a\n\n---\n\nx\n\nb"},
		{"type after deleted footnote", "a[^1]\n\n[^1]: n", []string{"sel:1,0", "⌦", "x"}, "ax\n\n[^1]: n"},
		{"type over selected footnote", "a[^1] b\n\n[^1]: n", []string{"sel:1,2", "x"}, "axb\n\n[^1]: n"},
		{"type after deleted formula", "a $x$", []string{"sel:2,0", "⌦", "]"}, "a ]"},
		{"typed heading in item text", "1. step\n\n   ```\n   cmd\n   ```", []string{"sel:5,0", "fmt:body", "# "}, "1. step\n\n# cmd"},
		{"heading on code in item", "1. step\n\n   ```\n   cmd\n   ```", []string{"sel:6,0", "fmt:heading2"}, "1. step\n\n## cmd"},
		{"code in item: renumber", "1. step\n\n   ```bash\n   cmd\n   ```\n\n2. next", []string{"sel:4,0", "⏎", "new"}, "1. step\n2. new\n\n   ```bash\n   cmd\n   ```\n\n3. next"},
		{"inline html stays verbatim", "1. a <name> b\n2. c", []string{"sel:10,0", "z"}, "1. a <name> bz\n2. c"},
		{"edit inside html tag", "a <name> b", []string{"sel:7,0", "s"}, "a <names> b"},
		{"typing after html is text", "a <span>", []string{"sel:8,0", "*x*"}, "a <span>*x*"},
		{"html comment kept", "a <!-- c --> d", []string{"sel:1,0", "!"}, "a! <!-- c --> d"},
		{"quoted list continues", "> - a", []string{"sel:1,0", "⏎", "b"}, "> - a\n> - b"},
		{"quoted list exits to quote", "> - a", []string{"sel:1,0", "⏎", "⏎", "q"}, "> - a\n>\n> q"},
		{"dash in quote starts list", "> q", []string{"sel:1,0", "⏎", "- x", "⏎", "y"}, "> q\n>\n> - x\n> - y"},
		{"backspace quoted item", "> - a", []string{"sel:0,0", "⌫"}, "> a"},
		{"tab nests quoted item", "> - a", []string{"sel:1,0", "⏎", "⇥", "b"}, "> - a\n>   - b"},
		{"typed quote then list", "", []string{"> ", "- a", "⏎", "b"}, "> - a\n> - b"},
		{"toc on enter", "", []string{"[toc]", "⏎", "# A"}, "[TOC]\n\n# A"},
		{"backspace joins into list", "- a\n\nb", []string{"sel:2,0", "⌫"}, "- ab"},
		{"backspace joins into heading", "# a\n\nb", []string{"sel:2,0", "⌫"}, "# ab"},
		{"delete joins paragraphs", "a\n\nb", []string{"sel:1,0", "⌦"}, "ab"},
		{"enter inside bold", "**bold text**", []string{"sel:4,0", "⏎", "x"}, "**bold**\n\nx **text**"},
		{"loaded escape stays literal", `\*a`, []string{"sel:2,0", "b*"}, `\*ab\*`},
		{"loaded escaped heading", `\#x`, []string{"sel:1,0", " "}, `\# x`},
		{"escape stays literal", "", []string{`a \*b\* c`}, `a \*b\* c`},
		{"escape is not a delimiter", "", []string{`\*a* **b** c`}, `\*a\* **b** c`},
		{"escaped backslash", "", []string{`a \\*b* c`}, `a \\*b* c`},
		{"escaped heading marker", "", []string{`\# not heading`}, `\# not heading`},
		{"code enter keeps indent", "```\n    x\n```", []string{"sel:5,0", "⏎", "y"}, "```\n    x\n    y\n```"},
		{"footnote definition typed", "", []string{"a[^1]", "⏎", "[^1]: note"}, "a[^1]\n\n[^1]: note"},
		{"rule mid document", "a\n\nb", []string{"sel:1,0", "⏎", "***", "⏎", "x"}, "a\n\n***\n\nx\n\nb"},
		{"enter after loaded rule", "---", []string{"sel:3,0", "⏎", "t"}, "---\n\nt"},
		{"front matter is not a rule", "---\ntitle: x\n---\n\nbody", []string{"sel:20,0", "y"}, "---\ntitle: x\n---\n\nbodyy"},
		{"math block on enter", "a", []string{"sel:1,0", "⏎", "$$", "⏎", "x^2", "sel:0,0"}, "a\n\n$$\nx^2\n$$"},
		{"table on enter", "", []string{"| a | b |", "⏎", "1", "⏎", "2"}, "| a | b |\n| --- | --- |\n| 1 | 2 |"},
		{"typed image line", "", []string{"![a](x.png)", "⏎", "t"}, "![a](x.png)\n\nt"},
		{"link without brackets", "", []string{"[l](http://x.com) y"}, "[l](http://x.com) y"},
		{"heading command toggles", "## abc", []string{"sel:1,0", "fmt:heading2"}, "abc"},
		{"renumber after delete", "1. a\n2. b\n3. c", []string{"sel:2,2", "⌫"}, "1. a\n2. c"},
		{"select all delete", "# a\n\n- b\n\n> c", []string{"sel:0,5", "⌫", "x"}, "x"},
		{"down out of trailing code", "```go\nx\n```", []string{"sel:1,0", "down", "after"}, "```go\nx\n```\n\nafter"},
		{"up out of leading code", "```go\nx\n```", []string{"sel:0,0", "up", "before"}, "before\n\n```go\nx\n```"},
		{"down out of trailing math", "$$\nx\n$$", []string{"sel:0,0", "down", "after"}, "$$\nx\n$$\n\nafter"},
		{"type after math", "$$\nx\n$$\n\nb", []string{"sel:1,0", "a"}, "$$\nx\n$$\n\na\n\nb"},
		{"type before math", "a\n\n$$\nx\n$$", []string{"sel:2,0", "z"}, "a\n\nz\n\n$$\nx\n$$"},
		{"soft break survives edit", "line1\nline2", []string{"sel:11,0", "x"}, "line1\nline2x"},
		{"soft break in quote", "> a\n> b", []string{"sel:3,0", "x"}, "> a\n> bx"},
		{"soft break in list", "- a\n  b", []string{"sel:3,0", "x"}, "- a\n  bx"},
		{"soft break in ordered list", "1. a\n   b\n2. c", []string{"sel:3,0", "x"}, "1. a\n   bx\n2. c"},
		{"soft break in alert", "> [!NOTE]\n> one\n> two", []string{"sel:15,0", "x"}, "> [!NOTE]\n> one\n> twox"},
		{"soft break copy paste", "a\nb\n\nz", []string{"sel:0,3", "copy", "sel:6,0", "pasteboard"}, "a\nb\n\nza\nb"},
		{"shift enter is a hard break", "", []string{"a", "⇧⏎", "b"}, "a  \nb"},
		{"kbd survives edit", "<kbd>x</kbd> y", []string{"sel:3,0", "z"}, "<kbd>x</kbd> yz"},
		{"kbd typed", "", []string{"<kbd>x</kbd> y"}, "<kbd>x</kbd> y"},
		{"url autolinks", "", []string{"see https://a.com now"}, "see https://a.com now"},
		{"www autolinks", "", []string{"www.a.com x"}, "www.a.com x"},
		{"version stays plain", "", []string{"v1.2.3 ok"}, "v1.2.3 ok"},
		{"cross-block delete keeps heading", "# ab\n\ncd", []string{"sel:1,3", "⌫"}, "# ad"},
		{"cross-block delete keeps body", "ab\n\n# cd", []string{"sel:1,3", "⌫"}, "ad"},
		{"italic uses asterisks", "hello", []string{"sel:0,5", "*"}, "*hello*"},
		{"url paste links selection", "see here", []string{"sel:4,4", "paste:https://x.com"}, "see [here](https://x.com)"},
		{"tab lands after cell text", "| a | b |\n|---|---|\n| 1 | 2 |", []string{"sel:0,0", "⇥", "x"}, "| a | bx |\n| --- | --- |\n| 1 | 2 |"},
		{"quote joins paragraphs", "a\n\nb", []string{"sel:0,3", "fmt:quote"}, "> a\n>\n> b"},
		{"bullet joins paragraphs", "a\n\nb\n\nc", []string{"sel:0,5", "fmt:bullet"}, "- a\n- b\n- c"},
		{"bullet toggles off", "- a\n- b", []string{"sel:0,3", "fmt:bullet"}, "a\n\nb"},
		{"click below trailing code", "```go\nx\n```", []string{"click-below", "after"}, "```go\nx\n```\n\nafter"},
		{"click below prose appends", "a", []string{"click-below", "b"}, "ab"},
		{"star bullet keeps marker", "", []string{"* a", "⏎", "b"}, "* a\n* b"},
		{"paren ordered keeps marker", "", []string{"1) a", "⏎", "b"}, "1) a\n2) b"},
		{"insert code block", "a", []string{"sel:1,0", "insert:```", "x"}, "a\n\n```\nx\n```"},
		{"insert code into empty line", "", []string{"insert:```", "x"}, "```\nx\n```"},
		{"insert rule", "a", []string{"sel:1,0", "insert:---", "x"}, "a\n\n---\n\nx"},
		{"insert math block", "a", []string{"sel:1,0", "insert:$$", "x", "sel:0,0"}, "a\n\n$$\nx\n$$"},
		{"link command inserts at caret", "a b", []string{"sel:2,0", "fmt:link:https://x.com\nL"}, "a [L](https://x.com)b"},
		{"link command uses url as text", "", []string{"fmt:link:https://x.com\n"}, "https://x.com"},
		{"ime after math", "$$\nx\n$$", []string{"sel:1,0", "ime:zhong|中"}, "$$\nx\n$$\n\n中"},
	}
	var long, renumbered strings.Builder
	for i := 1; i <= 30; i++ {
		long.WriteString(strconv.Itoa(i) + ". item\n")
		if i > 1 {
			renumbered.WriteString(strconv.Itoa(i-1) + ". item\n")
		}
	}
	cases = append(cases,
		struct {
			name, load string
			ops        []string
			want       string
		}{"renumber far down a long list", strings.TrimSuffix(long.String(), "\n"), []string{"sel:0,5", "⌫"}, strings.TrimSuffix(renumbered.String(), "\n")},
		struct {
			name, load string
			ops        []string
			want       string
		}{"edit deep in callout keeps it", "> [!WARNING]\n> a\n>\n> b\n>\n> c", []string{"sel:20,0", "x"}, "> [!WARNING]\n> a\n>\n> b\n>\n> cx"},
	)
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range cases {
			e.Load(c.load)
			probeRun(e, c.ops)
			got, _, err := e.Snapshot()
			if got = strings.TrimRight(got, "\n"); err != nil || got != c.want {
				t.Errorf("%s: got %q, want %q (%v)", c.name, got, c.want, err)
			}
		}
	})
}

// Figures are blocks: the keys beside them select first, then remove whole.
func TestNativeTyporaFigureKeys(t *testing.T) {
	img := writeTestPNG(t, 40, 30, "typora-keys")
	md := "![a](" + img + ")"
	cases := []struct {
		name, load string
		ops        []string
		want       string
	}{
		{"backspace after image keeps it", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫"}, "x\n\nIMG\n\ny"},
		{"second backspace removes it", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫", "⌫"}, "x\n\ny"},
		{"delete before image keeps it", "x\n\n" + md, []string{"sel:1,0", "⌦"}, "x\n\nIMG"},
		{"second delete removes it", "x\n\n" + md, []string{"sel:1,0", "⌦", "⌦"}, "x"},
		{"typing replaces selected image", "x\n\n" + md + "\n\ny", []string{"sel:4,0", "⌫", "z"}, "x\n\nz\n\ny"},
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
			got = strings.TrimRight(strings.ReplaceAll(got, md, "IMG"), "\n")
			if err != nil || got != c.want {
				t.Errorf("%s: got %q, want %q (%v)", c.name, got, c.want, err)
			}
		}
	})
}

func TestNativeAnchorLinks(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("intro\n\n## 格式 一览\n\ntext\n\n## Hello World!\n\nend")
		for _, c := range []struct {
			anchor string
			want   int
		}{{"#格式-一览", 6}, {"#hello-world", 17}, {"#%E6%A0%BC%E5%BC%8F-%E4%B8%80%E8%A7%88", 6}} {
			nativeTestSelect(e, 0, 0)
			nativeTestAction(e, "anchor", c.anchor)
			if start, _ := nativeTestSelection(e); start != c.want {
				t.Errorf("%s jumped to %d, want %d", c.anchor, start, c.want)
			}
		}
	})
}

func TestNativeInsertSnippets(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range []struct{ name, load, insert, want string }{
			{"table selects first header", "a", "| 列 1 | 列 2 |\n| --- | --- |\n|  |  |", "a\n\n| X | 列 2 |\n| --- | --- |\n|  |  |"},
			{"toc on its own block", "# h\n\na", "[TOC]", "# h\n\na\n\n[TOC]\n\nX"},
		} {
			e.Load(c.load)
			nativeTestSelect(e, len([]rune(nativeTestText(e))), 0)
			e.InsertMarkdown(c.insert)
			nativeTypeCharacters(e, "X")
			if got, _, err := e.Snapshot(); err != nil || strings.TrimRight(got, "\n") != c.want {
				t.Errorf("%s: got %q, want %q (%v)", c.name, got, c.want, err)
			}
		}
	})
}

// Undo swaps only the paragraphs it changes; neighbouring tables must come
// back whole and keep working as tables.
func TestNativeUndoBesideTables(t *testing.T) {
	source := "| a | b |\n| --- | --- |\n| 1 | 2 |\n\nmid\n\n| c |\n| --- |\n| 3 |"
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.Load(source)
		for _, word := range []string{"mid", "1", "c"} {
			at := strings.Index(nativeTestText(e), word)
			if at < 0 {
				t.Fatalf("%q missing from %q", word, nativeTestText(e))
			}
			nativeTestSelect(e, len([]rune(nativeTestText(e)[:at]))+1, 0)
			probeRun(e, []string{"xy", "⏎", "z", "undo", "undo", "undo"})
			if got, _, _ := e.Snapshot(); got != source {
				t.Fatalf("after undo near %q got %q", word, got)
			}
		}
	})
}

// Images inside a paragraph render inline and keep their source.
func TestNativeInlineImages(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 40, 12))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	path := filepath.Join(dir, "badge.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	source := "a ![badge](" + path + ") c"
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.Load(source)
		if got := nativeTestText(e); got != "a ￼ c" {
			t.Fatalf("not rendered inline: %q", got)
		}
		nativeTestSelect(e, 3, 0)
		probeRun(e, []string{"x"})
		if got, _, _ := e.Snapshot(); got != "a ![badge]("+path+")x c" {
			t.Fatalf("typing after image: %q", got)
		}
		e.Load("[" + source + "](https://u.com) z")
		probeRun(e, []string{"sel:7,0", "!"})
		if got, _, _ := e.Snapshot(); got != "[a ![badge]("+path+") c](https://u.com) z!" && got != "[a ](https://u.com)[![badge]("+path+")](https://u.com)[ c](https://u.com) z!" {
			t.Fatalf("linked image: %q", got)
		}
		e.Load(source)
		nativeTestSelect(e, 3, 0)
		probeRun(e, []string{"⌫"})
		if got, _, _ := e.Snapshot(); got != "a  c" {
			t.Fatalf("deleting image: %q", got)
		}
		probeRun(e, []string{"undo"})
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "![badge]") {
			t.Fatalf("undo did not restore image: %q", got)
		}
	})
}
