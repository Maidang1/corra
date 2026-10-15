//go:build darwin && cgo

package nativerich

import (
	"os"
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func TestNativeInlineSourceRevealEditFold(t *testing.T) {
	for _, fragment := range []string{"**ab**", "__ab__", "*ab*", "_ab_", "~~ab~~", "`ab`", "<u>ab</u>", "==ab==", "<mark>ab</mark>", "^ab^", "~ab~", `[ab](https://example.com "Title")`, "**a*b***"} {
		t.Run(fragment, func(t *testing.T) {
			nativeTestMain(func() {
				window := nativeTestWindow()
				defer nativeTestCloseWindow(window)
				e := New(window, nil, nil)
				defer e.Close()
				source := "before " + fragment + " after"
				e.Load(source)
				nativeTestSelect(e, 8, 0)
				nativeTestAction(e, "reveal", "")
				if !nativeTestHasTraits(e, "inline-source") {
					t.Errorf("styled text did not reveal editable syntax: %q", nativeTestText(e))
					return
				}
				if got, _, _ := e.Snapshot(); got != source {
					t.Errorf("revealing unchanged syntax changed stored source: %q", got)
				}
				caret, length := nativeTestSelection(e)
				visible := nativeTestText(e)
				if length != 0 || caret >= len(visible) || visible[caret] != 'b' {
					t.Errorf("caret lost its text boundary: %d %d %q", caret, length, visible)
					return
				}
				nativeTestSelect(e, caret, 1)
				nativeTestAction(e, "type", "Z")
				nativeTestAction(e, "key", "cmd-return")
				if nativeTestHasTraits(e, "inline-source") || nativeTestText(e) != "before aZ after" {
					t.Errorf("edited source did not fold: %q", nativeTestText(e))
				}
				got, _, _ := e.Snapshot()
				want := richtext.Parse("before " + strings.Replace(fragment, "b", "Z", 1) + " after")
				actual := richtext.Parse(got)
				if len(actual.Paragraphs) != 1 || richtext.PlainText(actual.Paragraphs[0]) != "before aZ after" || !sameNativeRunStyles(want.Paragraphs[0].Runs, actual.Paragraphs[0].Runs) {
					t.Errorf("edited syntax lost style/content: %q", got)
				}
			})
		})
	}
}

func TestNativeInlineSourceUneditedReference(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "before [ab][ref] after\r\n\r\n[ref]: https://example.com 'Title'\r\n"
		e.Load(source)
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		if got, _, _ := e.Snapshot(); got != source || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("reference reveal changed original Markdown: %q", got)
		}
		nativeTestAction(e, "blur", "")
		if got, _, _ := e.Snapshot(); got != source || nativeTestHasTraits(e, "inline-source") {
			t.Errorf("unedited reference fold changed source: %q", got)
		}
	})
}

func TestNativeInlineSourceFoldMapsSelection(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("before **ab** after")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 9, 1) // only a, excluding the opening delimiter
		nativeTestAction(e, "key", "cmd-return")
		if caret, length := nativeTestSelection(e); caret != 7 || length != 1 {
			t.Errorf("fold moved the selected character: %d %d", caret, length)
		}
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 10, 0) // between a and b
		nativeTestAction(e, "key", "cmd-return")
		if caret, length := nativeTestSelection(e); caret != 8 || length != 0 {
			t.Errorf("fold moved the interior caret: %d %d", caret, length)
		}
		nativeTestAction(e, "type", "Y")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "**aYb**") {
			t.Errorf("typing after fold lost active style: %q", got)
		}
	})
}

func TestNativeInlineSourcePaste(t *testing.T) {
	for _, table := range []bool{false, true} {
		t.Run(map[bool]string{false: "paragraph", true: "table"}[table], func(t *testing.T) {
			nativeTestMain(func() {
				window := nativeTestWindow()
				defer nativeTestCloseWindow(window)
				e := New(window, nil, nil)
				defer e.Close()
				source := "before **ab** after"
				if table {
					source = "| A | B |\n| --- | --- |\n| **ab** | untouched |"
				}
				e.Load(source)
				index := strings.Index(nativeTestText(e), "ab")
				nativeTestSelect(e, index+1, 0)
				nativeTestAction(e, "reveal", "")
				caret, _ := nativeTestSelection(e)
				nativeTestSelect(e, caret, 1)
				nativeTestAction(e, "paste", "Z")
				nativeTestAction(e, "key", "cmd-return")
				got, _, _ := e.Snapshot()
				if !strings.Contains(got, "**aZ**") || nativeTestHasTraits(e, "inline-source") {
					t.Errorf("paste split or lost the inline source: %q", got)
				}
			})
		})
	}
}

func TestNativeInlineSourceFormattingCommand(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("before **ab** after")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 9, 1)
		e.Format("italic")
		got, _, _ := e.Snapshot()
		want := richtext.Parse("before ***a*b** after")
		actual := richtext.Parse(got)
		if nativeTestHasTraits(e, "inline-source") || len(actual.Paragraphs) != 1 || !sameNativeRunStyles(want.Paragraphs[0].Runs, actual.Paragraphs[0].Runs) {
			t.Errorf("format command did not target source content: %q", got)
		}
	})
}

func sameNativeRunStyles(a, b []richtext.Run) bool {
	// The serializer may normalize adjacent equal runs; compare each character.
	flatten := func(runs []richtext.Run) []richtext.Run {
		var out []richtext.Run
		for _, run := range runs {
			for _, r := range run.Text {
				run.Text = string(r)
				out = append(out, run)
			}
		}
		return out
	}
	a, b = flatten(a), flatten(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNativeInlineSourceUnicodeTableAndComposition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("前 **😀中文** 后")
		nativeTestSelect(e, 4, 0) // after the emoji, before the CJK text
		nativeTestAction(e, "reveal", "")
		if caret, _ := nativeTestSelection(e); caret != 6 || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("UTF-16 caret mapping failed: %d %q", caret, nativeTestText(e))
		}
		nativeTestAction(e, "mark", "zhong")
		nativeTestAction(e, "key", "cmd-return")
		if _, marked, _ := e.Snapshot(); !marked || !nativeTestHasTraits(e, "inline-source") {
			t.Error("finishing source disturbed composition")
		}
		nativeTestAction(e, "commitIME", "中")
		nativeTestAction(e, "blur", "")
		if nativeTestText(e) != "前 😀中中文 后" || nativeTestHasTraits(e, "inline-source") {
			t.Errorf("committed inline composition did not fold: %q", nativeTestText(e))
		}
		e.Load("| A | B |\n| --- | --- |\n| **ab** | untouched |\n\nafter")
		index := strings.Index(nativeTestText(e), "ab")
		nativeTestSelect(e, index+1, 0)
		nativeTestAction(e, "reveal", "")
		if !nativeTestHasTraits(e, "inline-source") {
			t.Error("table inline syntax did not reveal")
			return
		}
		caret, _ := nativeTestSelection(e)
		nativeTestSelect(e, caret, 1)
		nativeTestAction(e, "type", "Z")
		nativeTestAction(e, "blur", "")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "**aZ**") || !strings.Contains(got, "untouched") || !strings.HasSuffix(got, "after") || nativeTestHasTraits(e, "inline-source") || !nativeTestHasTraits(e, "table") {
			t.Errorf("table source fold lost native table/content: %q", got)
		}
	})
}

func TestNativeInlineSourceNavigationAndSelections(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		e.Load("a **bc** d")
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "move-right", "")
		if nativeTestText(e) != "a **bc** d" || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("keyboard navigation did not reveal syntax: %q", nativeTestText(e))
		}
		nativeTestSelect(e, 0, 8)
		if start, length := nativeTestSelection(e); start != 0 || length != 8 || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("crossing selection folded or moved: %d %d", start, length)
		}
		nativeTestSelect(e, len(nativeTestText(e)), 0)
		if nativeTestText(e) != "a bc d" || nativeTestHasTraits(e, "inline-source") {
			t.Errorf("leaving source did not fold: %q", nativeTestText(e))
		}
		if caret, _ := nativeTestSelection(e); caret != 6 {
			t.Errorf("fold did not map following caret: %d", caret)
		}
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "click", "")
		if !nativeTestHasTraits(e, "inline-source") {
			caret, length := nativeTestSelection(e)
			t.Errorf("native single click did not reveal syntax: %d %d %q", caret, length, nativeTestText(e))
		}
		e.Load("a **bc** d")
		nativeTestSelect(e, 1, 4)
		if nativeTestHasTraits(e, "inline-source") {
			t.Error("range selection unexpectedly revealed syntax")
		}
		e.Load("`code`\n\n```go\nx := 1\n```\n\nend")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "move-right", "")
		if nativeTestHasTraits(e, "inline-source") {
			t.Error("code block navigation incorrectly revealed inline syntax")
		}
	})
}

func TestNativeInlineSourceSingleCharacterNavigation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		e.Load("a **b** c")
		nativeTestSelect(e, 2, 0)
		nativeTestAction(e, "move-right", "")
		if nativeTestText(e) != "a **b** c" || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("rightward navigation skipped single styled character: %q", nativeTestText(e))
		}
		e.Load("a **b** c")
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "move-left", "")
		if nativeTestText(e) != "a **b** c" || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("leftward navigation skipped single styled character: %q", nativeTestText(e))
		}
		e.Load("a **b** c")
		nativeTestSelect(e, 2, 0)
		nativeTestAction(e, "click-trailing", "")
		if !nativeTestHasTraits(e, "inline-source") {
			t.Error("clicking the trailing half of a styled glyph skipped reveal")
		}
		e.Load("a **bc** d")
		nativeTestSelect(e, 2, 0)
		nativeTestAction(e, "drag", "")
		if nativeTestHasTraits(e, "inline-source") {
			t.Error("dragging over styled text unexpectedly replaced the selection with source")
		}
	})
}

func TestNativeInlineSourceIncompleteSyntaxAndPresentation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		e.Load("before __ab__ after")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 11, 2)
		nativeTestAction(e, "type", "")
		nativeTestAction(e, "blur", "")
		if got, _, _ := e.Snapshot(); got != "before **ab after" {
			t.Errorf("unfinished delimiter did not remain literal Markdown: %q", got)
		}
		nativeTestSelect(e, 11, 0)
		nativeTestAction(e, "type", "**")
		if got := nativeTestText(e); got != "before ab after" || !nativeTestHasTraits(e, "bold") {
			t.Errorf("completing literal syntax did not recover formatting: %q", got)
		}
		if path := os.Getenv("CORRA_INLINE_SCREENSHOT"); path != "" {
			e.Load("# Native inline editing\n\nClick **bold text**, ==highlight== or [a link](https://example.com).\n\nEditing **中文与 emoji 😀** keeps the caret beside its character.\n\n| Format | Edit here |\n| --- | --- |\n| Inline | **table text** |")
			text := nativeTestText(e)
			index := strings.Index(text, "中文")
			nativeTestSelect(e, richtext.UTF16Len(text[:index])+1, 0)
			nativeTestAction(e, "reveal", "")
			if !nativeTestScreenshot(e, path) {
				t.Error("could not capture native inline editor")
			}
		}
	})
}

func TestNativeInlineSourceBackspaceDoesNotPairDelete(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("before `()` after")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 9, 0)
		nativeTestAction(e, "backspace", "")
		if got := nativeTestText(e); got != "before `)` after" {
			t.Errorf("backspace in literal source deleted an auto-pair: %q", got)
		}
	})
}

func TestNativeInlineSourceCrossBoundaryReplacement(t *testing.T) {
	for _, action := range []string{"type", "paste", "backspace", "delete"} {
		t.Run(action, func(t *testing.T) {
			nativeTestMain(func() {
				window := nativeTestWindow()
				defer nativeTestCloseWindow(window)
				e := New(window, nil, nil)
				defer e.Close()
				e.Load("before **ab** after")
				nativeTestSelect(e, 8, 0)
				nativeTestAction(e, "reveal", "")
				nativeTestSelect(e, 5, 5) // the end of 'before', opener and 'a'
				nativeTestAction(e, action, "Z")
				nativeTestAction(e, "blur", "")
				want := "beforZb after"
				if action == "backspace" || action == "delete" {
					want = "beforb after"
				}
				if got := nativeTestText(e); got != want || nativeTestHasTraits(e, "inline-source") {
					t.Errorf("replacement split the remaining source: %q", got)
				}
				got, _, _ := e.Snapshot()
				doc := richtext.Parse(got)
				if len(doc.Paragraphs) != 1 || richtext.PlainText(doc.Paragraphs[0]) != want {
					t.Errorf("cross-boundary edit corrupted Markdown: %q", got)
				}
			})
		})
	}
}

func TestNativeInlineSourceUndo(t *testing.T) {
	var window uintptr
	var e *Editor
	nativeTestMain(func() {
		window = nativeTestWindow()
		e = New(window, nil, nil)
		e.Load("before **ab** after")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
	})
	defer nativeTestMain(func() { e.Close(); nativeTestCloseWindow(window) })
	if !nativeTestFinishEvent(e) {
		t.Fatal("reveal event did not finish")
	}
	nativeTestMain(func() {
		nativeTestSelect(e, 10, 1)
		nativeTestAction(e, "type", "Z")
	})
	if !nativeTestFinishEvent(e) {
		t.Fatal("source typing event did not finish")
	}
	nativeTestMain(func() { nativeTestAction(e, "key", "cmd-return") })
	if !nativeTestFinishEvent(e) {
		t.Fatal("fold event did not finish")
	}
	nativeTestMain(func() {
		nativeTestAction(e, "undo", "")
		if got := nativeTestText(e); got != "before **aZ** after" || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("undo fold lost edited source: %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got := nativeTestText(e); got != "before **ab** after" {
			t.Errorf("undo typing lost source: %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got := nativeTestText(e); got != "before ab after" || nativeTestHasTraits(e, "inline-source") {
			t.Errorf("undo reveal lost rich text: %q", got)
		}
		nativeTestAction(e, "redo", "")
		nativeTestAction(e, "redo", "")
		nativeTestAction(e, "redo", "")
		if got := nativeTestText(e); got != "before aZ after" || nativeTestHasTraits(e, "inline-source") {
			t.Errorf("redo did not restore rendered edit: %q", got)
		}
	})
}
