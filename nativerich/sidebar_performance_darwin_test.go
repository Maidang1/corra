//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func sidebarTableDocument() string {
	return "# Report\n\n| Date | Change | Problem | Result |\n| --- | --- | --- | --- |\n" +
		strings.Repeat("| 2026-10-26 | **Update** | Long table content 中文 | Preserve document formatting |\n", 120)
}

func TestNativeUnchangedSnapshotDoesNotRebuildDocument(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := sidebarTableDocument()
		e.Load(source)
		allocs := testing.AllocsPerRun(5, func() {
			got, changed, err := e.Snapshot()
			if got != source || changed || err != nil {
				t.Errorf("unchanged snapshot: changed=%v err=%v", changed, err)
			}
		})
		if allocs > 0 {
			t.Errorf("unchanged document rebuilt on animation frame: %.0f allocations per snapshot", allocs)
		}
	})
}

func BenchmarkNativeSidebarFrame(b *testing.B) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(280, 60, 980, 700)
		e.Load(sidebarTableDocument())
		nativeTestAction(e, "layout", "")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, err := e.Snapshot(); err != nil {
				b.Error(err)
			}
			x := float64(280 - i%15*18)
			e.SetFrame(x, 60, 1260-x, 700)
			nativeTestAction(e, "layout", "")
		}
		b.StopTimer()
	})
}
