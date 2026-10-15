//go:build darwin && cgo

package nativerich

import "testing"

func TestNativeProbePair(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("")
		nativeTestAction(e, "type", "(")
		got, _, _ := e.Snapshot()
		start, length := nativeTestSelection(e)
		t.Logf("after (: got=%q len=%d caret=%d,%d", got, len(got), start, length)
		nativeTestAction(e, "type", ")")
		got, _, _ = e.Snapshot()
		start, length = nativeTestSelection(e)
		t.Logf("after ): got=%q len=%d caret=%d,%d", got, len(got), start, length)
		e.Load("hello")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "type", "*")
		got, _, _ = e.Snapshot()
		start, length = nativeTestSelection(e)
		t.Logf("wrap: got=%q len=%d caret=%d,%d", got, len(got), start, length)
		e.Load("")
		nativeTypeCharacters(e, ":smile:")
		nativeTestAction(e, "type", " ")
		got, _, _ = e.Snapshot()
		t.Logf("emoji: got=%q len=%d", got, len(got))
	})
}
