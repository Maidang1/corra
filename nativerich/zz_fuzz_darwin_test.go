//go:build darwin && cgo

package nativerich

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// TestEditingFuzz applies random edits and checks that what is saved is a
// fixed point: reloading the saved Markdown and saving again is identical.
// Run: CORRA_FUZZ=200 go test ./nativerich -run TestEditingFuzz -v
func TestEditingFuzz(t *testing.T) {
	rounds, _ := strconv.Atoi(os.Getenv("CORRA_FUZZ"))
	if rounds == 0 {
		t.Skip("set CORRA_FUZZ=rounds")
	}
	seed := int64(1)
	if s, err := strconv.ParseInt(os.Getenv("CORRA_FUZZ_SEED"), 10, 64); err == nil {
		seed = s
	}
	doc := "# 标题\n\n正文 **粗体** 与 *斜体*，`code` 和 [链接](https://a.com)。\n\n- 一\n- 二\n  - 嵌套\n\n1. 有序\n2. 第二\n\n> 引用\n> - 引用列表\n\n> [!NOTE]\n> 提示\n\n```go\nfunc f() {}\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n$$\nx^2\n$$\n\n脚注[^1]\n\n---\n\n- [ ] 任务\n\n[^1]: 内容\n"
	if os.Getenv("CORRA_FUZZ_DOC") == "2" {
		doc = "1. 安装\n\n   ```bash\n   go build\n   ```\n\n2. 运行 <kbd>⌘R</kbd>\n\n   > 注意\n   >\n   > 第二段\n\n3. 收尾\n\n> 引用\n> - 项一\n> - 项二\n\n> [!TIP]\n> 提示\n> - 列表\n\n文字 <span>html</span> 与 ![img](/tmp/imgt/badge.png) 和 [![b](/tmp/imgt/badge.png)](https://u.com)\n\n- [x] 完成\n  - 嵌套 ==高亮== ~~删~~ H~2~O x^2^\n\n[TOC]\n\n## 结尾 $a+b$\n"
	}
	if os.Getenv("CORRA_FUZZ_DOC") == "3" {
		// Footnotes, HTML, nested quotes, setext headings, math and links.
		doc = "Setext 标题\n===\n\n正文[^a]与[^b]，<sub>下</sub> <mark>标</mark> <u>线</u>。\n\n> 一层\n>\n> > 二层 **粗**\n>\n> 回到一层\n\n<details>\n<summary>摘要</summary>\n\n内容\n</details>\n\n$$\n\\int_0^1 x\\,dx\n$$\n\n* 星号列表\n* 第二项\n\n   1) 括号序号\n   2) 第二\n\n[链接](https://ref.example \"标题\") 和 <https://auto.link> 以及 https://bare.link\n\n***\n\n[^a]: 脚注一\n[^b]: 脚注二\n    第二段\n"
	}
	ops := []string{"type", "type", "type", "newline", "backspace", "delete", "select", "undo", "tab", "backtab", "linebreak", "markdown", "format", "format", "ime", "move", "paste", "copy", "cut", "pasteboard"}
	formats := []string{"bold", "italic", "strike", "code", "highlight", "heading2", "body", "bullet", "ordered", "checklist", "quote", "clear", "indent", "outdent"}
	pastes := []string{"- a\n- b", "| x | y |\n|---|---|\n| 1 | 2 |", "```\ncode\n```", "> q\n> - l", "# H\n\ntext", "1. one\n\n   ```\n   c\n   ```"}
	words := []string{"a", "字", " ", "*", "**", "`", "- ", "# ", "> ", "1. ", "[", "]", "(", ")", "$", "~~", "==", "|", "\\", "<", ">", "!", "^", "~", "_", "[^1]", "<br>", "<b>", "https://x.com ", "“", "（", ":smile:", "---", "[TOC]"}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		defer e.Close()
		e.SetFrame(0, 0, 800, 800)
		e.SetVisible(true)
		for round := 0; round < rounds; round++ {
			r := rand.New(rand.NewSource(seed + int64(round)))
			e.Load(doc)
			var log []string
			steps := 25
			if n, err := strconv.Atoi(os.Getenv("CORRA_FUZZ_STEPS")); err == nil && n > 0 {
				steps = n
			}
			for step := 0; step < steps; step++ {
				if os.Getenv("CORRA_FUZZ_TRACE") != "" {
					fmt.Fprintln(os.Stderr, "TRACE text?")
				}
				n := len([]rune(nativeTestText(e)))
				op := ops[r.Intn(len(ops))]
				if os.Getenv("CORRA_FUZZ_TRACE") != "" {
					fmt.Fprintln(os.Stderr, "TRACE begin", step, op)
				}
				switch op {
				case "type":
					w := words[r.Intn(len(words))]
					nativeTypeCharacters(e, w)
					op += " " + strconv.Quote(w)
				case "select":
					a := r.Intn(n + 1)
					b := 0
					if r.Intn(3) == 0 {
						b = r.Intn(n - a + 1)
					}
					nativeTestSelect(e, a, b)
					op += " " + strconv.Itoa(a) + "," + strconv.Itoa(b)
				case "markdown":
					nativeTestAction(e, "paste", "**p** `q`")
				case "format":
					f := formats[r.Intn(len(formats))]
					e.Format(f)
					op += " " + f
				case "paste":
					v := pastes[r.Intn(len(pastes))]
					nativeTestAction(e, "paste", v)
					op += " " + strconv.Quote(v)
				case "ime":
					nativeTestAction(e, "mark", "zhong")
					nativeTestAction(e, "commitIME", "中")
				case "move":
					if r.Intn(2) == 0 {
						nativeTestAction(e, "move-up", "")
					} else {
						nativeTestAction(e, "move-down", "")
					}
				default:
					nativeTestAction(e, op, "")
				}
				log = append(log, op)
				if os.Getenv("CORRA_FUZZ_TRACE") != "" {
					fmt.Fprintf(os.Stderr, "TRACE %d %s text=%q\n", step, op, nativeTestText(e))
				}
				trace := func(m string) {
					if os.Getenv("CORRA_FUZZ_TRACE") != "" {
						fmt.Fprintln(os.Stderr, "TRACE", m)
					}
				}
				trace("snapshot")
				saved, _, err := e.Snapshot()
				trace("snapshot done " + strconv.Quote(saved))
				if err != nil && strings.Contains(err.Error(), "输入法") {
					continue // saving waits for the input method by design
				}
				if err != nil {
					t.Errorf("seed %d: snapshot failed after %v: %v", seed+int64(round), log, err)
					return // t.Fatalf would deadlock inside the AppKit callback
				}
				probe := New(window, func() {}, func() {})
				trace("probe load")
				probe.Load(saved)
				if os.Getenv("CORRA_FUZZ_TEXT") != "" {
					// Revealed inline source shows its markers until the caret
					// leaves; compare rendered text on both sides.
					nativeTestAction(e, "blur", "")
					var blurErr error
					saved, _, blurErr = e.Snapshot()
					if blurErr != nil {
						probe.Close()
						continue // still composing: nothing saved to compare
					}
					probe.Load(saved)
					if a, b := visibleText(nativeTestText(e)), visibleText(nativeTestText(probe)); a != b {
						if os.Getenv("CORRA_FUZZ_TRACE") != "" {
							fmt.Fprint(os.Stderr, "PARAGRAPHS\n"+nativeTestParagraphs(e))
						}
						probe.Close()
						t.Errorf("seed %d: visible text changed by save after %v\neditor: %q\nreload: %q\nsaved: %q", seed+int64(round), log, a, b, saved)
						return
					}
				}
				trace("probe snapshot")
				again, _, _ := probe.Snapshot()
				trace("probe done")
				probe.Close()
				trace("probe closed")
				if again != saved && os.Getenv("CORRA_FUZZ_TRACE") != "" {
					fmt.Fprint(os.Stderr, "PARAGRAPHS\n"+nativeTestParagraphs(e))
				}
				if again != saved {
					t.Errorf("seed %d: not a fixed point after %v\nsaved: %q\nagain: %q", seed+int64(round), log, saved, again)
					return
				}
			}
		}
	})
}

// visibleText reduces editor text to what a reader sees, ignoring layout
// whitespace and attachment placeholders.
func visibleText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\uFFFC' || r == '\u2028' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
