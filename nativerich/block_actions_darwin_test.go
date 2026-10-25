//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func blockSnapshot(t *testing.T, e *Editor) string {
	t.Helper()
	source, _, err := e.Snapshot()
	if err != nil {
		t.Errorf("snapshot failed: %v", err)
		return ""
	}
	return source
}

func TestNativeBlockMove(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Moving a paragraph down swaps it with the next one, keeping the
		// blank-line rhythm of the note.
		e.Load("one\n\ntwo\n\nthree")
		nativeTestSelect(e, 0, 0)
		e.Format("block-down")
		if got := blockSnapshot(t, e); got != "two\n\none\n\nthree" {
			t.Errorf("move down = %q", got)
			return
		}
		// One undo restores the order.
		nativeTestAction(e, "undo", "")
		if got := blockSnapshot(t, e); got != "one\n\ntwo\n\nthree" {
			t.Errorf("undo after move = %q", got)
			return
		}

		// Moving up works from any paragraph in the pair.
		nativeTestSelect(e, 4, 0)
		e.Format("block-up")
		if got := blockSnapshot(t, e); got != "two\n\none\n\nthree" {
			t.Errorf("move up = %q", got)
			return
		}

		// The edges are no-ops.
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, 0)
		e.Format("block-up")
		if got := blockSnapshot(t, e); got != "one\n\ntwo" {
			t.Errorf("first block moved up: %q", got)
			return
		}
		nativeTestSelect(e, 4, 0)
		e.Format("block-down")
		if got := blockSnapshot(t, e); got != "one\n\ntwo" {
			t.Errorf("last block moved down: %q", got)
			return
		}

		// A list item moves as one item, and ordered numbering follows.
		e.Load("- a\n- b\n- c")
		nativeTestSelect(e, 0, 0)
		e.Format("block-down")
		if got := blockSnapshot(t, e); got != "- b\n- a\n- c" {
			t.Errorf("list move = %q", got)
			return
		}

		// A code block moves whole: its fences never split.
		e.Load("text\n\n```go\nx := 1\n```\n\nafter")
		nativeTestSelect(e, 11, 0)
		e.Format("block-up")
		got := blockSnapshot(t, e)
		if !strings.HasPrefix(got, "```go") || strings.Count(got, "```go") != 1 || !strings.Contains(got, "x := 1") || !strings.HasSuffix(got, "after") {
			t.Errorf("code move = %q", got)
			return
		}
		if !strings.Contains(got, "text") || strings.Index(got, "text") < strings.Index(got, "```go") {
			t.Errorf("code did not move above text: %q", got)
		}
	})
}

func TestNativeBlockDuplicateAndDelete(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Duplicating a paragraph leaves a fresh copy after it.
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, 0)
		e.Format("block-duplicate")
		if got := blockSnapshot(t, e); got != "one\n\none\n\ntwo" {
			t.Errorf("duplicate = %q", got)
			return
		}

		// Duplicating a list item copies the structure, not the identity.
		e.Load("- [ ] a\n- b")
		nativeTestSelect(e, 0, 0)
		e.Format("block-duplicate")
		if got := blockSnapshot(t, e); strings.Count(got, "- [ ] a") != 2 {
			t.Errorf("checklist duplicate = %q", got)
			return
		}

		// Duplicating a code block duplicates the whole fenced block.
		e.Load("```go\nx := 1\n```")
		nativeTestSelect(e, 7, 0)
		e.Format("block-duplicate")
		if got := blockSnapshot(t, e); strings.Count(got, "```go") != 2 || strings.Count(got, "x := 1") != 2 {
			t.Errorf("code duplicate = %q", got)
			return
		}

		// Delete removes the block and keeps the rest of the note.
		e.Load("one\n\ntwo\n\nthree")
		nativeTestSelect(e, 0, 0)
		e.Format("block-delete")
		if got := blockSnapshot(t, e); got != "two\n\nthree" {
			t.Errorf("delete first = %q", got)
			return
		}
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 4, 0)
		e.Format("block-delete")
		if got := blockSnapshot(t, e); got != "one" {
			t.Errorf("delete last = %q", got)
			return
		}
		e.Load("only")
		nativeTestSelect(e, 0, 0)
		e.Format("block-delete")
		if got := blockSnapshot(t, e); got != "" {
			t.Errorf("delete only = %q", got)
			return
		}
		// The cleared note still accepts typing as a plain paragraph.
		nativeTypeCharacters(e, "new")
		if got := blockSnapshot(t, e); got != "new" {
			t.Errorf("typing after delete = %q", got)
		}
	})
}

func TestNativeBlockDragReorder(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Dragging the handle down past two blocks moves the block to the end.
		e.Load("one\n\ntwo\n\nthree")
		if !nativeTestDragBlock(e, 0, 2, true) {
			t.Errorf("drag helper failed")
			return
		}
		if got, _, _ := e.Snapshot(); got != "two\n\nthree\n\none" {
			t.Errorf("drag down = %q", got)
			return
		}
		// One undo restores the drag as a single step.
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != "one\n\ntwo\n\nthree" {
			t.Errorf("undo after drag = %q", got)
			return
		}

		// Dragging up to the top works the same way.
		if !nativeTestDragBlock(e, 2, 0, false) {
			t.Errorf("drag helper failed")
			return
		}
		if got, _, _ := e.Snapshot(); got != "three\n\none\n\ntwo" {
			t.Errorf("drag up = %q", got)
			return
		}

		// A code block drags as one unit.
		e.Load("```go\nx := 1\n```\n\ntext")
		if !nativeTestDragBlock(e, 0, 1, true) {
			t.Errorf("code drag helper failed")
			return
		}
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "text") || !strings.Contains(got, "```go") {
			t.Errorf("code drag = %q", got)
			return
		}

		// Dropping a block onto itself (same gap) changes nothing.
		e.Load("one\n\ntwo")
		if !nativeTestDragBlock(e, 0, 0, false) {
			t.Errorf("no-op drag helper failed")
			return
		}
		if got, _, _ := e.Snapshot(); got != "one\n\ntwo" {
			t.Errorf("no-op drag changed the note: %q", got)
		}
	})
}

func TestNativeBlockShortcutsAndContextMenu(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Cmd+Shift+Down / Cmd+Shift+Up move the block like Notion.
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "key", "cmd-shift-down")
		if got := blockSnapshot(t, e); got != "two\n\none" {
			t.Errorf("cmd-shift-down = %q", got)
			return
		}
		// The caret moved with the block, so the opposite shortcut brings it
		// back to the top.
		nativeTestAction(e, "key", "cmd-shift-up")
		if got := blockSnapshot(t, e); got != "one\n\ntwo" {
			t.Errorf("cmd-shift-up = %q", got)
			return
		}

		// Cmd+D duplicates through the same transaction.
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "key", "cmd-d")
		if got := blockSnapshot(t, e); got != "one\n\none\n\ntwo" {
			t.Errorf("cmd-d = %q", got)
			return
		}

		// Cmd+Shift+Backspace deletes the block.
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "key", "cmd-shift-backspace")
		if got := blockSnapshot(t, e); got != "one\n\ntwo" {
			t.Errorf("cmd-shift-backspace = %q", got)
			return
		}

		// The context menu exposes the same commands.
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "context", "block-duplicate")
		if got := blockSnapshot(t, e); got != "one\n\none\n\ntwo" {
			t.Errorf("context duplicate = %q", got)
			return
		}
	})
}
