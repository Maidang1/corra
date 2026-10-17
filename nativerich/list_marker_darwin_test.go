//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

// List markers must sit on their item's first line while typing: computed
// from lazy layout, as drawRect does, and unmoved when an item gains a
// following sibling.
func TestListMarkersFollowTheirLines(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 700, 800)
		e.SetVisible(true)
		e.Load("## hello\n\n今天是一个好天气\n")
		e.SetStyle(1, false, true)
		nativeTestSelect(e, len([]rune(nativeTestText(e))), 0)
		nativeTestAction(e, "display", "")
		markers := func(step string) []string {
			lazy, full, _ := strings.Cut(nativeTestMarkers(e), "|")
			if lazy != full {
				t.Errorf("%s: markers drawn at %s, lines at %s", step, lazy, full)
			}
			return strings.Split(full, ",")
		}
		for _, op := range [][2]string{{"newline", ""}, {"newline", ""}, {"type", "- "}} {
			nativeTestAction(e, op[0], op[1])
		}
		markers("after list rule")
		nativeTestAction(e, "mark", "hell")
		nativeTestAction(e, "commitIME", "hell")
		nativeTestAction(e, "display", "")
		first := markers("after IME commit")[0]
		nativeTestAction(e, "newline", "")
		if got := markers("after newline"); got[0] != first {
			t.Errorf("first marker moved from %s to %s when a sibling item was added", first, got[0])
		}
		nativeTestAction(e, "type", "wd")
		if got := markers("after typing"); got[0] != first {
			t.Errorf("first marker moved from %s to %s", first, got[0])
		}
	})
}
