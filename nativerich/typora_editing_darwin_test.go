//go:build darwin && cgo

package nativerich

import (
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
		case strings.HasPrefix(op, "sel:"):
			parts := strings.Split(op[4:], ",")
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[1])
			nativeTestSelect(e, a, b)
		case strings.HasPrefix(op, "fmt:"):
			e.Format(op[4:])
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
		{"paste heading into empty line", "a\n\n", []string{"sel:3,0", "paste:## h"}, "a\n\n## h"},
		{"rule on enter", "", []string{"---", "⏎", "after"}, "---\n\nafter"},
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
		{"ime after math", "$$\nx\n$$", []string{"sel:1,0", "ime:zhong|中"}, "$$\nx\n$$\n\n中"},
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
			if got = strings.TrimRight(got, "\n"); err != nil || got != c.want {
				t.Errorf("%s: got %q, want %q (%v)", c.name, got, c.want, err)
			}
		}
	})
}
