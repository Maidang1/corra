//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func TestNativeMultiParagraphListEditing(t *testing.T) {
	var window uintptr
	var e *Editor
	nativeTestMain(func() {
		window = nativeTestWindow()
		e = New(window, nil, nil)
		original := "- anchor\n- first\n\n  second **bold**\n- next"
		e.Load(original)
		if nativeTestHasTraits(e, "source") {
			t.Error("multi-paragraph list became source")
			return
		}
		nativeTestSelect(e, 13, 6)
		nativeTestAction(e, "type", "edited")
	})
	defer nativeTestMain(func() { e.Close(); nativeTestCloseWindow(window) })
	if !nativeTestFinishEvent(e) {
		t.Fatal("AppKit did not close typing event")
	}
	nativeTestMain(func() {
		nativeTestAction(e, "tab", "")
		got, _, err := e.Snapshot()
		if err != nil || !strings.Contains(got, "\n  - first\n\n    edited **bold**\n- next") {
			t.Errorf("continuation indent: %q %v", got, err)
		}
	})
	if !nativeTestFinishEvent(e) {
		t.Fatal("AppKit did not close structural event")
	}
	nativeTestMain(func() {
		nativeTestAction(e, "undo", "")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "\n- first\n\n  edited **bold**\n- next") {
			t.Errorf("continuation indent undo: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != "- anchor\n- first\n\n  second **bold**\n- next" {
			t.Errorf("typing undo: %q", got)
		}
		nativeTestAction(e, "redo", "")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "\n- first\n\n  edited **bold**\n- next") {
			t.Errorf("typing redo: %q", got)
		}
		nativeTestAction(e, "redo", "")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "\n  - first\n\n    edited **bold**\n- next") {
			t.Errorf("indent redo: %q", got)
		}
		e.Load("3. first\n4. next")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "key", "alt-return")
		nativeTypeCharacters(e, "paragraph")
		got, _, _ = e.Snapshot()
		p := richtext.Parse(got).Paragraphs
		if len(p) != 3 || !p[1].ListContinuation || p[2].Ordinal != 4 {
			t.Errorf("new list paragraph: %q", got)
		}
	})
}

func TestNativeNestedQuotes(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "> outer\n>\n> > inner\n>\n> tail"
		e.Load(original)
		if nativeTestHasTraits(e, "source") {
			t.Error("nested quote became source")
			return
		}
		nativeTestSelect(e, 6, 0)
		nativeTestAction(e, "key", "cmd-right")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "> > > inner") {
			t.Errorf("quote increase depth: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("quote indent undo: %q", got)
		}
		nativeTestAction(e, "backspace", "")
		got, _, _ = e.Snapshot()
		p := richtext.Parse(got).Paragraphs
		if len(p) != 3 || p[1].Kind != "quote" || p[1].Depth != 0 {
			t.Errorf("quote Backspace should outdent: %q", got)
		}
	})
}
