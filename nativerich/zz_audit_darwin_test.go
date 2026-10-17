//go:build darwin && cgo

package nativerich

import (
	"os"
	"testing"
)

func TestAuditTypora(t *testing.T) {
	if os.Getenv("CORRA_PROBE") == "" {
		t.Skip()
	}
	cases := []struct {
		name, load string
		ops        []string
	}{
		{"top quote enter", "> note\n\nafter", []string{"sel:4,0", "⏎", "more"}},
		{"top list enter", "- a\n\nafter", []string{"sel:1,0", "⏎", "b"}},
	}
	for _, c := range cases {
		nativeTestMain(func() {
			window := nativeTestWindow()
			defer nativeTestCloseWindow(window)
			e := New(window, func() {}, func() {})
			defer e.Close()
			e.Load(c.load)
			nativeTestSelect(e, len([]rune(nativeTestText(e))), 0)
			probeRun(e, c.ops)
			got, _, _ := e.Snapshot()
			t.Logf("%-24s %q", c.name, got)
			if os.Getenv("CORRA_PROBE") == "text" {
				t.Logf("%-24s native=%q", "", nativeTestText(e))
			}
		})
	}
}
