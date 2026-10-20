//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes the slash palette to PNGs for visual review: the light palette, its
// scrolled tail and the dark palette. The panel is its own window, so the
// editor screenshot cannot capture it.
// Run: CORRA_SLASH_SHOT=/tmp/slash.png go test ./nativerich -run TestSlashMenuShot
func TestSlashMenuShot(t *testing.T) {
	path := os.Getenv("CORRA_SLASH_SHOT")
	if path == "" {
		t.Skip("set CORRA_SLASH_SHOT=path.png")
	}
	variant := func(suffix string) string {
		ext := filepath.Ext(path)
		return strings.TrimSuffix(path, ext) + suffix + ext
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 900, 700)
		e.SetVisible(true)
		e.SetStyle(1, false, true)
		e.Load("")
		nativeTypeCharacters(e, "/")
		if !nativeTestSlashShot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		// A filtered palette shows the matched part of the title in accent.
		filterPath := strings.TrimSuffix(path, filepath.Ext(path)) + "-filter" + filepath.Ext(path)
		nativeTypeCharacters(e, "列表")
		if !nativeTestSlashShot(e, filterPath) {
			t.Errorf("filtered screenshot failed")
		}
		for i := 0; i < 2; i++ {
			nativeTestAction(e, "backspace", "")
		}
		// Walk the highlight to the end so the lower palette is on screen.
		for i := 0; i < 20; i++ {
			nativeTestAction(e, "key", "plain-down")
		}
		if !nativeTestSlashShot(e, variant("-tail")) {
			t.Errorf("tail screenshot failed")
		}
		nativeTestAction(e, "key", "plain-escape")
		e.Load("")
		e.SetStyle(1, true, true)
		nativeTypeCharacters(e, "/")
		if !nativeTestSlashShot(e, variant("-dark")) {
			t.Errorf("dark screenshot failed")
		}
	})
}
