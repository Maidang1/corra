//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func TestNativeTableControls(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		source := "| a | b |\n| --- | --- |\n| c | d |"
		// Hovering a cell raises the add-row and add-column buttons.
		e.Load(source)
		if !nativeTestHoverTable(e) {
			t.Errorf("table controls did not appear on hover")
			return
		}
		// Adding a column widens the header separator row.
		if !nativeTestTableButton(e, 1) {
			t.Errorf("add column failed")
			return
		}
		after := blockSnapshot(t, e)
		if strings.Count(after, "---") != 3 {
			t.Errorf("add column = %q", after)
			return
		}

		// Adding a row appends one more table line.
		e.Load(source)
		if !nativeTestHoverTable(e) {
			t.Errorf("table controls did not return")
			return
		}
		if !nativeTestTableButton(e, 0) {
			t.Errorf("add row failed")
			return
		}
		after = blockSnapshot(t, e)
		lines := 0
		for _, line := range strings.Split(after, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "|") {
				lines++
			}
		}
		if lines != 4 {
			t.Errorf("add row = %q", after)
		}

		// Prose raises no table controls.
		e.Load("just text")
		if nativeTestHoverTable(e) {
			t.Error("table controls appeared over prose")
		}
	})
}
