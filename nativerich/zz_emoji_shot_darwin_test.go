//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes the emoji picker to PNGs for visual review.
// Run: CORRA_EMOJI_SHOT=/tmp/emoji.png go test ./nativerich -run TestEmojiShot
func TestEmojiShot(t *testing.T) {
	path := os.Getenv("CORRA_EMOJI_SHOT")
	if path == "" {
		t.Skip("set CORRA_EMOJI_SHOT=path.png")
	}
	filtered := strings.TrimSuffix(path, filepath.Ext(path)) + "-filtered" + filepath.Ext(path)
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
		e.SetFrame(0, 0, 760, 520)
		e.SetVisible(true)
		e.SetStyle(1, false, true)
		e.Load("")
		nativeTypeCharacters(e, ":")
		if !nativeTestEmojiShot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		nativeTypeCharacters(e, "smi")
		if !nativeTestEmojiShot(e, filtered) {
			t.Errorf("filtered screenshot failed")
		}
		nativeTestAction(e, "key", "plain-escape")
		e.Load("")
		e.SetStyle(1, true, true)
		nativeTypeCharacters(e, ":heart")
		if !nativeTestEmojiShot(e, dark) {
			t.Errorf("dark screenshot failed")
		}
	})
}
