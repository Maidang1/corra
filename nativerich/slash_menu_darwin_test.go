//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

// The slash menu drives a real child NSPanel; these tests run inside the
// AppKit harness like the other editing tests.

// slashState reports the menu, failing the test for an unreadable state.
func slashState(t *testing.T, e *Editor) (open bool, filter string, selected int, items []string, ok bool) {
	t.Helper()
	open, filter, selected, _, items = nativeTestSlash(e)
	if open && items == nil {
		t.Errorf("slash state could not be read")
		return false, "", 0, nil, false
	}
	return open, filter, selected, items, true
}

func expectSlash(t *testing.T, e *Editor, wantOpen bool, wantFilter string) bool {
	t.Helper()
	open, filter, _, items, ok := slashState(t, e)
	if !ok {
		return false
	}
	if open != wantOpen {
		text := nativeTestText(e)
		t.Errorf("menu open=%v, want %v (filter=%q text=%q items=%v)", open, wantOpen, filter, text, items)
		return false
	}
	if wantOpen && filter != wantFilter {
		t.Errorf("menu filter=%q, want %q", filter, wantFilter)
		return false
	}
	return true
}

// slashApply loads an empty note, types /filter and confirms the highlighted
// entry with Return, the way a writer would.
func slashApply(t *testing.T, e *Editor, filter string) (string, bool) {
	t.Helper()
	e.Load("")
	nativeTypeCharacters(e, "/"+filter)
	if !expectSlash(t, e, true, filter) {
		return "", false
	}
	nativeTestAction(e, "key", "plain-return")
	if !expectSlash(t, e, false, "") {
		return "", false
	}
	source, _, err := e.Snapshot()
	if err != nil {
		t.Errorf("/%s: snapshot failed: %v", filter, err)
		return "", false
	}
	return source, true
}

func newSlashEditor(t *testing.T) (uintptr, *Editor) {
	t.Helper()
	window := nativeTestWindow()
	e := New(window, nil, nil)
	if e == nil {
		t.Error("native editor unavailable")
		return 0, nil
	}
	e.SetFrame(0, 0, 720, 500)
	e.SetVisible(true)
	return window, e
}

func TestNativeSlashMenuOpensAndFilters(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// A lone "/" opens the full palette with the first row highlighted.
		e.Load("")
		nativeTypeCharacters(e, "/")
		open, filter, selected, items, ok := slashState(t, e)
		if !ok {
			return
		}
		if !open || filter != "" || selected != 0 {
			t.Errorf("open=%v filter=%q selected=%d", open, filter, selected)
			return
		}
		if len(items) != 23 {
			t.Errorf("palette has %d entries: %v", len(items), items)
			return
		}
		if items[0] != "一级标题" {
			t.Errorf("first entry = %q", items[0])
			return
		}

		// Filtering narrows by title, by CJK title and by keyword.
		nativeTypeCharacters(e, "h1")
		open, filter, _, items, _ = slashState(t, e)
		if !open || filter != "h1" || len(items) != 1 || items[0] != "一级标题" {
			t.Errorf("h1 filter: open=%v filter=%q items=%v", open, filter, items)
			return
		}
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "backspace", "")
		if !expectSlash(t, e, true, "") {
			return
		}
		// Keywords match regardless of case.
		nativeTypeCharacters(e, "H1")
		if open, _, _, items, _ = slashState(t, e); !open || len(items) != 1 || items[0] != "一级标题" {
			t.Errorf("H1 filter: open=%v items=%v", open, items)
			return
		}
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "backspace", "")
		nativeTypeCharacters(e, "标题")
		open, filter, _, items, _ = slashState(t, e)
		if !open || filter != "标题" || len(items) != 6 || items[5] != "六级标题" {
			t.Errorf("标题 filter: open=%v filter=%q items=%v", open, filter, items)
			return
		}
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "backspace", "")
		nativeTypeCharacters(e, "todo")
		open, _, _, items, _ = slashState(t, e)
		if !open || len(items) != 1 || items[0] != "任务列表" {
			t.Errorf("todo filter: open=%v items=%v", open, items)
			return
		}

		// An unmatched query keeps the menu open on the empty state.
		for i := 0; i < 4; i++ {
			nativeTestAction(e, "backspace", "")
		}
		nativeTypeCharacters(e, "zzz")
		open, filter, _, items, _ = slashState(t, e)
		if !open || filter != "zzz" || len(items) != 0 {
			t.Errorf("zzz filter: open=%v filter=%q items=%v", open, filter, items)
		}
	})
}

func TestNativeSlashMenuKeyboard(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Arrows move the highlight and wrap at the ends.
		e.Load("")
		nativeTypeCharacters(e, "/")
		nativeTestAction(e, "key", "plain-down")
		if _, _, selected, _, _ := slashState(t, e); selected != 1 {
			t.Errorf("down selected=%d, want 1", selected)
			return
		}
		nativeTestAction(e, "key", "plain-up")
		nativeTestAction(e, "key", "plain-up")
		_, _, selected, items, _ := slashState(t, e)
		if selected != len(items)-1 {
			t.Errorf("wrap-up selected=%d, want %d", selected, len(items)-1)
			return
		}

		// The highlight scrolls the palette: walking down past the ninth row
		// moves the visible window instead of hiding the selection.
		e.Load("")
		nativeTypeCharacters(e, "/")
		for i := 0; i < 10; i++ {
			nativeTestAction(e, "key", "plain-down")
		}
		open2, _, selected, offset, items2 := nativeTestSlash(e)
		if !open2 || items2 == nil {
			t.Errorf("slash state could not be read")
			return
		}
		if selected != 10 || offset < 2 {
			t.Errorf("scrolled selection=%d offset=%d, want selected=10 offset>=2", selected, offset)
		}

		// Escape dismisses and leaves the typed text alone.
		nativeTestAction(e, "key", "plain-escape")
		if !expectSlash(t, e, false, "") {
			return
		}
		if got, _, _ := e.Snapshot(); got != "/" {
			t.Errorf("escape changed the line: %q", got)
			return
		}

		// Backspacing past the slash closes without deleting anything else.
		nativeTypeCharacters(e, "h")
		if !expectSlash(t, e, true, "h") {
			return
		}
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "backspace", "")
		if !expectSlash(t, e, false, "") {
			return
		}
		if got, _, _ := e.Snapshot(); got != "" {
			t.Errorf("backspace left %q", got)
			return
		}

		// Clicking a row follows the same path as Return (slash-pick is the
		// click handler without a synthetic mouse event).
		e.Load("")
		nativeTypeCharacters(e, "/math")
		nativeTestAction(e, "slash-pick", "0")
		if !expectSlash(t, e, false, "") {
			return
		}
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "$$") {
			t.Errorf("clicked entry wrote %q", got)
			return
		}

		// A space ends the query like Notion does.
		e.Load("")
		nativeTypeCharacters(e, "/h")
		nativeTestAction(e, "type", " ")
		if !expectSlash(t, e, false, "") {
			return
		}
		if got, _, _ := e.Snapshot(); got != "/h " {
			t.Errorf("space closed menu but wrote %q", got)
			return
		}

		// Moving the caret off the end of the query dismisses the menu.
		e.Load("")
		nativeTypeCharacters(e, "/h")
		nativeTestAction(e, "move-left", "")
		if !expectSlash(t, e, false, "") {
			return
		}

		// A "/" typed mid-sentence never opens.
		e.Load("")
		nativeTypeCharacters(e, "ab/")
		if !expectSlash(t, e, false, "") {
			return
		}

		// Nor does one typed in a heading.
		e.Load("# heading")
		nativeTestSelect(e, 10, 0)
		nativeTestAction(e, "type", "/")
		if !expectSlash(t, e, false, "") {
			return
		}
	})
}

func TestNativeSlashMenuAppliesBlocks(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		cases := []struct {
			filter string
			trait  string
			prefix string
		}{
			{"h1", "heading", "# "},
			{"h3", "heading", "### "},
			{"bullet", "bullet", "- "},
			{"ordered", "ordered", "1. "},
			{"todo", "checklist", "- [ ] "},
			{"quote", "quote", "> "},
		}
		for _, tc := range cases {
			source, ok := slashApply(t, e, tc.filter)
			if !ok {
				return
			}
			if !strings.HasPrefix(source, tc.prefix) {
				t.Errorf("/%s: source %q does not start with %q", tc.filter, source, tc.prefix)
				return
			}
			// On an empty note the block only lives in the typing state, so
			// type into it before reading its paragraph kind.
			nativeTypeCharacters(e, "x")
			if !nativeTestHasTraits(e, tc.trait) {
				t.Errorf("/%s: block kind %q missing (text=%q)", tc.filter, tc.trait, nativeTestText(e))
				return
			}
		}

		// Divider, code, math, table, toc, footnote and image go through the
		// insert transactions and land as their own blocks.
		if source, ok := slashApply(t, e, "divider"); !ok {
			return
		} else if !strings.Contains(source, "---") {
			t.Errorf("divider source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "code"); !ok {
			return
		} else if !strings.Contains(source, "```") {
			t.Errorf("code source=%q", source)
			return
		} else {
			// The empty block only becomes a real paragraph once typed into.
			nativeTypeCharacters(e, "x")
			if !nativeTestHasTraits(e, "code") {
				t.Errorf("code block missing after typing (text=%q)", nativeTestText(e))
				return
			}
		}
		if source, ok := slashApply(t, e, "math"); !ok {
			return
		} else if !strings.Contains(source, "$$") {
			t.Errorf("math source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "table"); !ok {
			return
		} else if !strings.Contains(source, "|") || !nativeTestHasTraits(e, "table") {
			t.Errorf("table source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "toc"); !ok {
			return
		} else if !strings.Contains(strings.ToUpper(source), "[TOC]") || !nativeTestHasTraits(e, "toc") {
			t.Errorf("toc source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "脚注"); !ok {
			return
		} else if !strings.Contains(source, "[^") {
			t.Errorf("footnote source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "图片"); !ok {
			return
		} else if !strings.Contains(source, "![") {
			t.Errorf("image source=%q", source)
			return
		}
		if source, ok := slashApply(t, e, "分页符"); !ok {
			return
		} else if !strings.Contains(source, "page-break-after") {
			t.Errorf("page break source=%q", source)
		}

		// Callouts write the GitHub marker on a quote.
		source, ok := slashApply(t, e, "提示")
		if !ok {
			return
		}
		if !nativeTestHasTraits(e, "quote", "alert:NOTE") || !strings.HasPrefix(source, "> [!NOTE]") {
			t.Errorf("callout source=%q", source)
			return
		}

		// A callout picked on an empty line mid-document keeps the next line
		// separate instead of swallowing its line break.
		e.Load("a\n\n\nb")
		nativeTestSelect(e, 2, 0)
		nativeTypeCharacters(e, "/提示")
		if !expectSlash(t, e, true, "提示") {
			return
		}
		nativeTestAction(e, "key", "plain-return")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "[!NOTE]") || strings.Contains(got, "[!NOTE] b") || !strings.HasSuffix(got, "b") {
			t.Errorf("mid-document callout merged lines: %q", got)
		}
	})
}

func TestNativeSlashMenuUndoAndIME(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// One undo reverts both the deleted query and the applied block.
		e.Load("")
		nativeTypeCharacters(e, "/h1")
		nativeTestAction(e, "key", "plain-return")
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "# ") {
			t.Errorf("heading not applied: %q", got)
			return
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != "" {
			t.Errorf("one undo left %q", got)
			return
		}
		if nativeTestHasTraits(e, "heading") {
			t.Error("undo left the heading behind")
			return
		}

		// A committed IME composition filters the palette.
		e.Load("")
		nativeTypeCharacters(e, "/")
		nativeTestAction(e, "mark", "biao")
		if open, _, _, _, _ := slashState(t, e); !open {
			t.Error("composition closed the menu")
			return
		}
		nativeTestAction(e, "commitIME", "标")
		open, filter, _, items, _ := slashState(t, e)
		if !open || filter != "标" || len(items) != 6 {
			t.Errorf("IME filter: open=%v filter=%q items=%v", open, filter, items)
			return
		}

		// Switching document dismisses the panel.
		e.Load("")
		nativeTypeCharacters(e, "/")
		if !expectSlash(t, e, true, "") {
			return
		}
		e.Load("other")
		expectSlash(t, e, false, "")
	})
}

// TestNativeSlashMenuRenders paints the palette offscreen, so the drawing
// code (icons, rows, empty state, dark colors) runs without a visible window.
func TestNativeSlashMenuRenders(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		dir := t.TempDir()
		render := func(name string) {
			t.Helper()
			if !nativeTestSlashShot(e, dir+"/"+name) {
				t.Errorf("render %s failed", name)
			}
		}

		e.Load("")
		nativeTypeCharacters(e, "/")
		render("light.png")
		nativeTypeCharacters(e, "zzz")
		render("empty.png")
		nativeTestAction(e, "key", "plain-escape")

		e.SetStyle(1, true, true)
		nativeTypeCharacters(e, "//")
		render("dark.png")
	})
}
