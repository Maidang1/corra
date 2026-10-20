//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func TestNativeCodeCopyButton(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		source := "```go\nx := 1\ny := 2\n```\n\ntext"
		e.Load(source)
		if !nativeTestHoverCode(e, false) {
			t.Errorf("copy button did not appear over the code chip")
			return
		}
		if !nativeTestHoverCode(e, true) {
			t.Errorf("copy hover did not copy")
			return
		}
		if got := nativeTestClipboardText(); got != "x := 1\ny := 2" {
			t.Errorf("copied %q, want code without fences", got)
		}
		// Copying never edits the document.
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("copy changed the source: %q", got)
		}

		// Clicking the code text (not the button) moves the caret instead of
		// copying: the button is a small target, not the whole block.
		e.Load(source)
		if caret := nativeTestClickCodeText(e); caret <= 0 {
			t.Errorf("clicking code text left the caret at %d, want past the block start", caret)
			return
		}

		// Prose has no copy affordance.
		e.Load("hello")
		if nativeTestHoverCode(e, false) {
			t.Error("copy button appeared over prose")
			return
		}

		// A code block with a single line copies that line.
		e.Load("```\nonly line\n```")
		hovered := nativeTestHoverCode(e, false)
		copyButton, _ := nativeTestHover(e)
		if !hovered || !copyButton {
			t.Errorf("single-line hover: result=%v state=%v", hovered, copyButton)
			return
		}
		if !nativeTestHoverCode(e, true) {
			t.Errorf("single-line copy failed")
			return
		}
		if got := nativeTestClipboardText(); got != "only line" {
			t.Errorf("single-line copy = %q", got)
		}
	})
}

func TestNativeBlockHandle(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("one\n\ntwo\n\nthree")
		if !nativeTestHoverBlock(e, 1, false) {
			t.Errorf("margin handle did not appear")
			return
		}
		// Clicking the handle selects its block, so the block menu and
		// shortcuts act on the hovered paragraph.
		if !nativeTestHoverBlock(e, 1, true) {
			t.Errorf("handle click did not prepare the block")
			return
		}
		if start, length := nativeTestSelection(e); start != 4 || length != 0 {
			t.Errorf("handle prepared %d,%d, want 4,0", start, length)
			return
		}
		e.Format("block-down")
		if got, _, _ := e.Snapshot(); got != "one\n\nthree\n\ntwo" {
			t.Errorf("block menu move = %q", got)
			return
		}

		// Typing dismisses the affordances: they belong to the hover only.
		e.Load("one\n\ntwo")
		if !nativeTestHoverBlock(e, 0, false) {
			t.Errorf("handle did not appear again")
			return
		}
		nativeTestAction(e, "type", "x")
		if copyButton, handle := nativeTestHover(e); copyButton || handle {
			t.Errorf("hover survived typing: copy=%v handle=%v", copyButton, handle)
			return
		}

		// The handle appears for a code block as one unit, not per line.
		e.Load("```go\nx := 1\ny := 2\n```")
		if !nativeTestHoverBlock(e, 0, true) {
			t.Errorf("code handle did not prepare")
			return
		}
		e.Format("block-duplicate")
		if got, _, _ := e.Snapshot(); strings.Count(got, "```go") != 2 {
			t.Errorf("handle duplicate = %q", got)
			return
		}
	})
}

func TestNativeCodeLanguagePicker(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("```go\nx := 1\n```\n\ntext")
		if !nativeTestHoverCode(e, false) {
			t.Errorf("code hover failed")
			return
		}
		if !nativeTestLanguageHover(e) {
			t.Errorf("language tag did not become a picker on hover")
			return
		}
		// Picking a language rewrites the hidden opening fence.
		if !nativeTestLanguagePick(e, "rust") {
			t.Errorf("language pick failed")
			return
		}
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "```rust") {
			t.Errorf("language pick wrote %q", got)
		}
		if !strings.Contains(nativeTestText(e), "x := 1") {
			t.Errorf("language pick lost the code: %q", nativeTestText(e))
		}
	})
}

func TestNativeImageSizePill(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		path := writeTestPNG(t, 480, 240, "pill")
		e.Load("![alt](" + path + ")")
		if !nativeTestHoverImage(e, -1, false) {
			t.Errorf("image hover did not raise the size pill")
			return
		}
		// Half width writes a 50% size suffix.
		if !nativeTestHoverImage(e, 2, true) {
			t.Errorf("50%% preset did not apply")
			return
		}
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "=50%") {
			t.Errorf("size pill wrote %q", got)
			return
		}
		// The original preset strips the suffix again.
		if !nativeTestHoverImage(e, 0, true) {
			t.Errorf("original preset did not apply")
			return
		}
		if got, _, _ := e.Snapshot(); strings.Contains(got, "=") {
			t.Errorf("original preset left %q", got)
			return
		}
		// Prose raises nothing.
		e.Load("just text")
		if nativeTestHoverImage(e, -1, false) {
			t.Error("image pill appeared over prose")
		}
	})
}

func TestNativeFootnoteHoverCard(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("正文引用[^1]\n\n[^1]: 脚注定义 **加粗**")
		if !nativeTestHoverFootnote(e) {
			t.Errorf("footnote reference not found")
			return
		}
		open, heading, body := nativeTestFootnoteCard(e)
		if !open || heading != "脚注 1" || !strings.Contains(body, "脚注定义") {
			t.Errorf("card: open=%v heading=%q body=%q", open, heading, body)
			return
		}
		// Leaving the reference dismisses the card.
		nativeTestHoverBlock(e, 0, false)
		if open, _, _ := nativeTestFootnoteCard(e); open {
			t.Errorf("card stayed open after leaving the reference")
		}
	})
}

func TestNativeImageActions(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		path := writeTestPNG(t, 240, 120, "actions")
		e.Load("![alt](" + path + ")")
		if !nativeTestHoverImage(e, -1, false) {
			t.Errorf("image hover did not raise the pill")
			return
		}
		// Copy puts the rendered image on the pasteboard.
		if !nativeTestImageCopy(e) {
			t.Errorf("copy image failed")
			return
		}
		if !nativeTestClipboardImage() {
			t.Errorf("pasteboard has no image")
			return
		}
		// Delete removes the whole figure, not just the attachment.
		if !nativeTestImageDelete(e) {
			t.Errorf("delete image failed")
			return
		}
		if got, _, _ := e.Snapshot(); strings.TrimSpace(got) != "" {
			t.Errorf("image left %q", got)
			return
		}
	})
}
