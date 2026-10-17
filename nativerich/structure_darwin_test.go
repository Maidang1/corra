//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func TestNativeTableStructuralCommands(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "before 😀\n\n| A | B |\n| :--- | ---: |\n| **one** | two |\n| three | four |\n\nafter"
		// Native table starts after the prose; the first body cell starts at 14.
		for _, tc := range []struct {
			command       string
			rows, columns int
			contains      string
		}{
			{"table:row-below", 4, 2, "|  |  |"},
			{"table:row-above", 4, 2, "|  |  |"},
			{"table:delete-row", 2, 2, "three"},
			{"table:column-right", 3, 3, "**one**"},
			{"table:column-left", 3, 3, "**one**"},
			{"table:delete-column", 3, 1, "two"},
			{"table:row-down", 3, 2, "three"},
			{"table:column-rightward", 3, 2, "| B | A |"},
			{"table:align-center", 3, 2, ":---:"},
		} {
			e.Load(original)
			nativeTestSelect(e, 14, 0)
			e.Format(tc.command)
			got, changed, err := e.Snapshot()
			if err != nil || !changed || !strings.HasPrefix(got, "before 😀\n\n") || !strings.HasSuffix(got, "\n\nafter") || !strings.Contains(got, tc.contains) {
				t.Errorf("%s: %q, changed=%v err=%v", tc.command, got, changed, err)
				continue
			}
			doc := richtext.Parse(got)
			p := doc.Paragraphs[1]
			if p.Kind != "table" || len(p.Rows) != tc.rows || len(p.Rows[0]) != tc.columns {
				t.Errorf("%s wrong shape: %+v", tc.command, p)
			}
			nativeTestAction(e, "undo", "")
			undone, _, _ := e.Snapshot()
			if undone != original {
				t.Errorf("%s undo: %q", tc.command, undone)
			}
			nativeTestAction(e, "redo", "")
			redone, _, _ := e.Snapshot()
			if redone != got {
				t.Errorf("%s redo: %q != %q", tc.command, redone, got)
			}
		}
	})
}

func TestNativeTableCommandsUseCurrentCellsAndSafeBoundaries(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("| A | B |\n| :--- | ---: |\n| one | two |")
		nativeTestSelect(e, 4, 3)
		nativeTestAction(e, "type", "changed")
		e.Format("table:column-rightward")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "| two | changed |") || !strings.Contains(got, "| ---: | :--- |") {
			t.Errorf("current text/alignment lost: %q", got)
		}
		original := "| A |\n| --- |\n| one |"
		for _, tc := range []struct {
			command  string
			location int
		}{
			{"table:delete-column", 2}, {"table:delete-row", 2},
			{"table:row-up", 2}, {"table:row-down", 0},
			{"table:column-leftward", 2}, {"table:column-rightward", 2},
		} {
			e.Load(original)
			nativeTestSelect(e, tc.location, 0)
			e.Format(tc.command)
			got, changed, err := e.Snapshot()
			if err != nil || changed || got != original {
				t.Errorf("boundary %s: %q %v %v", tc.command, got, changed, err)
			}
		}
		e.Load(original)
		nativeTestSelect(e, 2, 0)
		nativeTestAction(e, "mark", "zhong")
		e.Format("table:column-right")
		nativeTestAction(e, "commitIME", "中")
		got, _, _ = e.Snapshot()
		p := richtext.Parse(got).Paragraphs[0]
		if len(p.Rows[0]) != 1 || !strings.Contains(got, "中one") {
			t.Errorf("composition mutated table: %q", got)
		}
	})
}

func TestNativeTableKeyboardCommands(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "| A | B |\n| --- | --- |\n| one | two |"
		e.Load(original)
		nativeTestSelect(e, 4, 0)
		nativeTestAction(e, "key", "cmd-return")
		nativeTypeCharacters(e, "new")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "| new |  |") {
			t.Errorf("Cmd+Enter failed: %q", got)
		}
		nativeTestAction(e, "key", "cmd-shift-backspace")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("row keyboard deletion: %q", got)
		}
	})
}

func TestNativeEmptyTableStructureAndPaste(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, prefix := range []string{"", "before\n\n"} {
			original := prefix + "| | |\n| --- | --- |\n| | |"
			for _, command := range []string{"table:row-below", "table:column-right", "table:align-center", "paste"} {
				e.Load(original)
				location := 2
				if prefix != "" {
					location += 7
				}
				nativeTestSelect(e, location, 0)
				if command == "paste" {
					nativeTestAction(e, "paste", "x\ty")
				} else {
					e.Format(command)
				}
				got, changed, err := e.Snapshot()
				tableIndex := 0
				if prefix != "" {
					tableIndex = 1
				}
				if err != nil || !changed || richtext.Parse(got).Paragraphs[tableIndex].Kind != "table" {
					t.Errorf("empty grid %s: %q %v", command, got, err)
				}
				nativeTestAction(e, "undo", "")
				got, _, _ = e.Snapshot()
				if got != original {
					t.Errorf("empty grid undo %s: %q", command, got)
				}
			}
		}
	})
}

func TestNativeTableRectangularPaste(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "| A | B |\n| --- | --- |\n| **one** | two |"
		e.Load(original)
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "paste", "x\ty\r\nz\tpipe|value\r\n")
		got, _, _ := e.Snapshot()
		p := richtext.Parse(got).Paragraphs[0]
		if len(p.Rows) != 3 || len(p.Rows[0]) != 3 || !p.Rows[1][0].Runs[0].Bold ||
			richtext.PlainText(richtext.Paragraph{Runs: p.Rows[1][1].Runs}) != "x" ||
			richtext.PlainText(richtext.Paragraph{Runs: p.Rows[2][2].Runs}) != "pipe|value" {
			t.Errorf("rectangular paste lost grid or existing formatting: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("rectangular paste undo: %q", got)
		}
	})
}

func TestNativeSelectedListSiblingsAndComposition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "- 😀\n- one\n- two\n- three"
		e.Load(original)
		nativeTestSelect(e, 3, 7)
		nativeTestAction(e, "key", "cmd-right")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "\n  - one\n  - two\n- three") {
			t.Errorf("sibling selection indent: %q", got)
		}
		nativeTestAction(e, "key", "cmd-left")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("sibling selection outdent: %q", got)
		}
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "tab", "")
		got, changed, _ := e.Snapshot()
		if changed || got != original {
			t.Errorf("orphan indentation: %q", got)
		}
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "mark", "zhong")
		nativeTestAction(e, "tab", "")
		nativeTestAction(e, "commitIME", "中")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "\n- 中one\n- two") {
			t.Errorf("indent interfered with composition: %q", got)
		}
	})
}

func TestNativeOrderedTaskContinuation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "3. [x] first\n4. [ ] second"
		e.Load(original)
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "inserted")
		got, _, _ := e.Snapshot()
		if got != "3. [x] first\n4. [ ] inserted\n5. [ ] second" {
			t.Errorf("ordered task Return: %q", got)
		}
		nativeTestSelect(e, 6, 0)
		nativeTestAction(e, "tab", "")
		got, _, _ = e.Snapshot()
		p := richtext.Parse(got).Paragraphs
		if len(p) != 3 || p[1].Kind != "checklist" || p[1].Ordinal == 0 || p[2].Ordinal != 4 {
			t.Errorf("ordered task nesting: %q", got)
		}
	})
}

func TestNativeStructureContextMenuAndCaret(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		original := "| A | B |\n| --- | --- |\n| one | two |\n| three | four |"
		e.Load(original)
		nativeTestSelect(e, 4, 0)
		nativeTestAction(e, "context", "table:column-right")
		nativeTypeCharacters(e, "new")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "| one | new | two |") {
			t.Errorf("context menu insertion/caret: %q", got)
		}
		e.Load(original)
		nativeTestSelect(e, 4, 0)
		e.Format("table:delete-row")
		nativeTypeCharacters(e, "X")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "| Xthree | four |") {
			t.Errorf("delete row caret: %q", got)
		}
		e.Load(original)
		nativeTestSelect(e, 4, 0)
		e.Format("table:column-rightward")
		nativeTypeCharacters(e, "X")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "| two | Xone |") {
			t.Errorf("move column caret: %q", got)
		}
	})
}

func TestNativeNestedListBackspaceAndSourcePreservation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "+ parent\r\n  + child **bold**\r\n    + grand\r\n+ last\r\n\r\n<div>untouched</div>\r\n\r\n[ref]: https://example.com \"title\"\r\n"
		e.Load(original)
		nativeTestSelect(e, 7, 0)
		nativeTestAction(e, "backspace", "")
		got, _, err := e.Snapshot()
		if err != nil || !strings.Contains(got, "\r\n+ child **bold**\r\n  + grand") ||
			!strings.HasSuffix(got, "<div>untouched</div>\r\n\r\n[ref]: https://example.com \"title\"\r\n") {
			t.Errorf("nested Backspace/preservation: %q %v", got, err)
		}
		nativeTestAction(e, "undo", "")
		undone, _, _ := e.Snapshot()
		if undone != original {
			t.Errorf("nested Backspace undo: %q", undone)
		}
		nativeTestAction(e, "redo", "")
		redone, _, _ := e.Snapshot()
		if redone != got {
			t.Errorf("nested Backspace redo: %q", redone)
		}
		e.Load("- ")
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "body")
		got, _, _ = e.Snapshot()
		if got != "body" {
			t.Errorf("empty root Return: %q", got)
		}
	})
}

func TestNativeIndependentNestedOrderedLists(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "- parent\n\n  3. a\n  4. b\n\n  7) c\n  8) d\n- tail"
		e.Load(original)
		nativeTestSelect(e, 19, 0)
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "next")
		got, _, _ := e.Snapshot()
		if !strings.HasPrefix(got, strings.TrimSuffix(original, "tail")) || !strings.HasSuffix(got, "tail\n- next") {
			t.Errorf("unrelated Return changed independent nested list: %q", got)
		}
	})
}

func TestNativeCopyNestedSiblingsThenContinue(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("- parent\n  - one\n  - two")
		nativeTestSelect(e, 7, 7)
		nativeTestAction(e, "copy", "")
		e.Load("")
		nativeTestAction(e, "pasteboard", "")
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "inserted")
		got, _, _ := e.Snapshot()
		if got != "- one\n- inserted\n- two" {
			t.Errorf("pasted siblings changed hierarchy: %q", got)
		}
	})
}

func TestNativeNestedListIndentAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "- parent\n- child **bold**\n- last"
		e.Load(original)
		nativeTestSelect(e, 9, 5)
		nativeTestAction(e, "tab", "")
		got, changed, err := e.Snapshot()
		if err != nil || !changed || !strings.Contains(got, "\n  - child **bold**\n") {
			t.Errorf("indent: %q %v %v", got, changed, err)
		}
		if start, length := nativeTestSelection(e); start != 9 || length != 5 {
			t.Errorf("selection lost: %d %d", start, length)
		}
		nativeTestAction(e, "undo", "")
		undone, _, _ := e.Snapshot()
		if undone != original {
			t.Errorf("indent undo: %q", undone)
		}
		nativeTestAction(e, "redo", "")
		redone, _, _ := e.Snapshot()
		if redone != got {
			t.Errorf("indent redo: %q", redone)
		}
		nativeTestAction(e, "backtab", "")
		out, _, _ := e.Snapshot()
		if out != original {
			t.Errorf("outdent changed inline content: %q", out)
		}
	})
}

func TestNativeNestedListSubtreesAndReturn(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "- first\n- parent\n  - nested\n- last"
		e.Load(original)
		if nativeTestHasTraits(e, "source") {
			t.Error("nested list fell back to source")
			return
		}
		nativeTestSelect(e, 6, 0)
		nativeTestAction(e, "tab", "")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "\n  - parent\n    - nested\n- last") {
			t.Errorf("subtree indent: %q", got)
		}
		nativeTestAction(e, "backtab", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("subtree outdent: %q", got)
		}
		e.Load("3. parent\n\n   7. child\n4. sibling")
		nativeTestSelect(e, 12, 0) // end of child
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "next")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "8. next") || !strings.Contains(got, "\n4. sibling") {
			t.Errorf("nested numbering/Return: %q", got)
		}
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "newline", "")
		nativeTypeCharacters(e, "root")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "\n4. root\n5. sibling") {
			t.Errorf("empty nested item should outdent: %q", got)
		}
	})
}
