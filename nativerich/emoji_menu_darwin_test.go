//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func TestNativeEmojiPicker(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// A colon at a word start opens the picker with a default list.
		e.Load("")
		nativeTypeCharacters(e, ":")
		open, filter, selected, items := nativeTestEmoji(e)
		if !open || filter != "" || selected != 0 || len(items) == 0 {
			t.Errorf("colon open=%v filter=%q selected=%d items=%d", open, filter, selected, len(items))
			return
		}
		// Typing filters the shortcodes.
		nativeTypeCharacters(e, "smi")
		open, filter, _, items = nativeTestEmoji(e)
		if !open || filter != "smi" || len(items) == 0 || items[0] != "smile" {
			t.Errorf("filter=%q items=%v", filter, items)
			return
		}
		// Return inserts the emoji and closes the picker.
		nativeTestAction(e, "key", "plain-return")
		if open, _, _, _ := nativeTestEmoji(e); open {
			t.Errorf("picker stayed open after Return")
			return
		}
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "😄") {
			t.Errorf("return inserted %q", got)
			return
		}
		// Undo removes the inserted emoji (the harness runs the typing and the
		// insert in one undo group; a real key-by-key session splits them).
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); strings.Contains(got, "😄") {
			t.Errorf("undo kept the emoji: %q", got)
			return
		}

		// Escape dismisses and keeps the text.
		e.Load("")
		nativeTypeCharacters(e, ":s")
		nativeTestAction(e, "key", "plain-escape")
		if open, _, _, _ := nativeTestEmoji(e); open {
			t.Errorf("escape kept the picker open")
			return
		}
		if got, _, _ := e.Snapshot(); got != ":s" {
			t.Errorf("escape left %q", got)
			return
		}

		// A space ends the query, like the block picker.
		e.Load("")
		nativeTypeCharacters(e, ":s")
		nativeTestAction(e, "type", " ")
		if open, _, _, _ := nativeTestEmoji(e); open {
			t.Errorf("space kept the picker open")
			return
		}
		if got, _, _ := e.Snapshot(); got != ":s " {
			t.Errorf("space left %q", got)
			return
		}

		// A colon after a word is not a shortcode start.
		e.Load("")
		nativeTypeCharacters(e, "time:")
		if open, _, _, _ := nativeTestEmoji(e); open {
			t.Errorf("mid-word colon opened the picker")
			return
		}

		// Backspacing past the colon closes it.
		e.Load("")
		nativeTypeCharacters(e, ":sm")
		nativeTestAction(e, "backspace", "")
		if open, filter, _, _ := nativeTestEmoji(e); !open || filter != "s" {
			t.Errorf("backspace: open=%v filter=%q", open, filter)
			return
		}
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "backspace", "")
		if open, _, _, _ := nativeTestEmoji(e); open {
			t.Errorf("backspacing past the colon kept the picker open")
			return
		}
		if got, _, _ := e.Snapshot(); got != "" {
			t.Errorf("backspace left %q", got)
			return
		}

		// Clicking a row inserts it too.
		e.Load("")
		nativeTypeCharacters(e, ":heart")
		nativeTestAction(e, "emoji-pick", "0")
		if got, _, _ := e.Snapshot(); got == "" || strings.Contains(got, ":") {
			t.Errorf("click inserted %q", got)
		}
	})
}
