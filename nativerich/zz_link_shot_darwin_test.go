//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes the "[[" link completion panel to PNGs for visual review.
// Run: CORRA_LINK_SHOT=/tmp/link.png go test ./nativerich -run TestLinkShot
func TestLinkShot(t *testing.T) {
	path := os.Getenv("CORRA_LINK_SHOT")
	if path == "" {
		t.Skip("set CORRA_LINK_SHOT=path.png")
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
		e.OnLinkCandidates = linkCandidates
		e.SetFrame(0, 0, 760, 520)
		e.SetVisible(true)
		e.SetStyle(1, false, true)
		e.Load("")
		nativeTypeCharacters(e, "[[")
		if !nativeTestLinkShot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		nativeTypeCharacters(e, "be")
		if !nativeTestLinkShot(e, filtered) {
			t.Errorf("filtered screenshot failed")
		}
		nativeTestAction(e, "key", "plain-escape")
		e.Load("")
		e.SetStyle(1, true, true)
		nativeTypeCharacters(e, "[[")
		if !nativeTestLinkShot(e, dark) {
			t.Errorf("dark screenshot failed")
		}
		headings := strings.TrimSuffix(path, filepath.Ext(path)) + "-headings" + filepath.Ext(path)
		e.SetStyle(1, false, true)
		e.Load("# Intro\n\n## Setup Notes\n\n## Release Plan\n\nbody")
		nativeTestSelect(e, len(nativeTestText(e)), 0)
		nativeTypeCharacters(e, "[[#")
		if !nativeTestLinkShot(e, headings) {
			t.Errorf("headings screenshot failed")
		}
	})
}
