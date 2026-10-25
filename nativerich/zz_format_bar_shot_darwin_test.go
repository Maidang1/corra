//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes the floating format bar to PNGs for visual review: the light pill
// with bold+italic active and the dark pill.
// Run: CORRA_FORMAT_BAR_SHOT=/tmp/bar.png go test ./nativerich -run TestFormatBarShot
func TestFormatBarShot(t *testing.T) {
	path := os.Getenv("CORRA_FORMAT_BAR_SHOT")
	if path == "" {
		t.Skip("set CORRA_FORMAT_BAR_SHOT=path.png")
	}
	dark := strings.TrimSuffix(path, filepath.Ext(path)) + "-dark" + filepath.Ext(path)
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
		e.Load("hello world")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "bold")
		nativeTestAction(e, "formatbar-pick", "italic")
		if !nativeTestFormatBarShot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		e.SetStyle(1, true, true)
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "highlight")
		if !nativeTestFormatBarShot(e, dark) {
			t.Errorf("dark screenshot failed: %s", dark)
		}
	})
}
