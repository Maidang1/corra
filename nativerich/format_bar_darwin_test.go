//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

func formatBarState(t *testing.T, e *Editor) (open bool, active []string) {
	t.Helper()
	open, active, _, _, _ = nativeTestFormatBar(e)
	return open, active
}

func hasCommand(active []string, command string) bool {
	for _, item := range active {
		if item == command {
			return true
		}
	}
	return false
}

func TestNativeFormatBarVisibility(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		// Selecting prose shows the pill with no traits on.
		e.Load("hello world")
		nativeTestSelect(e, 0, 5)
		open, active := formatBarState(t, e)
		if !open || len(active) != 0 {
			t.Errorf("prose selection: open=%v active=%v", open, active)
			return
		}

		// A collapsed caret hides it again.
		nativeTestSelect(e, 5, 0)
		if open, _ := formatBarState(t, e); open {
			t.Error("collapsed caret kept the bar open")
			return
		}

		// Headings are prose; code blocks are not.
		e.Load("# heading")
		nativeTestSelect(e, 2, 5)
		if open, _ := formatBarState(t, e); !open {
			t.Error("heading selection did not show the bar")
			return
		}
		e.Load("```\ncode\n```")
		nativeTestSelect(e, 5, 3)
		if open, _ := formatBarState(t, e); open {
			t.Error("code selection showed the bar")
			return
		}

		// Table cells support inline formatting.
		e.Load("| a | b |\n| --- | --- |\n| c | d |")
		nativeTestSelect(e, 2, 1)
		if open, _ := formatBarState(t, e); !open {
			t.Error("table cell selection did not show the bar")
			return
		}

		// Read-only documents do not offer formatting.
		e.SetStyle(1, false, false)
		nativeTestSelect(e, 0, 3)
		if open, _ := formatBarState(t, e); open {
			t.Error("read-only document showed the bar")
		}
	})
}

func TestNativeFormatBarAppliesTraits(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("hello")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "bold")
		open, active := formatBarState(t, e)
		if !open || !hasCommand(active, "bold") {
			t.Fatalf("bold not active: open=%v active=%v", open, active)
		}
		if got, _, _ := e.Snapshot(); got != "**hello**" {
			t.Errorf("bold wrote %q", got)
			return
		}

		// A second press toggles the trait off.
		nativeTestAction(e, "formatbar-pick", "bold")
		if _, active = formatBarState(t, e); hasCommand(active, "bold") {
			t.Errorf("second bold press did not toggle: %v", active)
			return
		}

		// Italic and highlight stack, and the active set reports both.
		nativeTestAction(e, "formatbar-pick", "italic")
		nativeTestAction(e, "formatbar-pick", "highlight")
		_, active = formatBarState(t, e)
		if !hasCommand(active, "italic") || !hasCommand(active, "highlight") {
			t.Errorf("stacked traits not reported: %v", active)
			return
		}
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "*") || !strings.Contains(got, "==") {
			t.Errorf("stacked source %q", got)
			return
		}

		// Clear strips every inline trait again.
		nativeTestAction(e, "formatbar-pick", "clear")
		_, active = formatBarState(t, e)
		if len(active) != 0 {
			t.Errorf("clear left traits: %v", active)
			return
		}
		if got, _, _ := e.Snapshot(); got != "hello" {
			t.Errorf("clear wrote %q", got)
		}

		// Loading a document with bold on shows it as active.
		e.Load("**strong**")
		nativeTestSelect(e, 2, 6)
		if _, active = formatBarState(t, e); !hasCommand(active, "bold") {
			t.Errorf("existing bold not reported: %v", active)
		}
	})
}

func TestNativeFormatBarLinkAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("hello")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-link", "https://example.com")
		open, active := formatBarState(t, e)
		if !open || !hasCommand(active, "link") {
			t.Fatalf("link not active: open=%v active=%v", open, active)
		}
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "[hello](https://example.com)") {
			t.Errorf("link wrote %q", got)
		}

		// An empty URL removes the link.
		nativeTestAction(e, "formatbar-link", "")
		if _, active = formatBarState(t, e); hasCommand(active, "link") {
			t.Errorf("empty URL did not remove the link: %v", active)
			return
		}
		if got, _, _ := e.Snapshot(); got != "hello" {
			t.Errorf("link removal wrote %q", got)
			return
		}

		// One undo reverts a bar format.
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "bold")
		if got, _, _ := e.Snapshot(); got != "**hello**" {
			t.Fatalf("bold wrote %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != "hello" {
			t.Errorf("undo left %q", got)
		}
	})
}

// TestNativeFormatBarRenders paints the pill, including the active state and
// the dark palette, so the drawing code runs without a visible window.
func TestNativeFormatBarRenders(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		dir := t.TempDir()

		e.Load("hello")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "bold")
		if !nativeTestFormatBarShot(e, dir+"/light.png") {
			t.Error("light render failed")
		}
		nativeTestAction(e, "formatbar-pick", "clear")
		e.SetStyle(1, true, true)
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "italic")
		if !nativeTestFormatBarShot(e, dir+"/dark.png") {
			t.Error("dark render failed")
		}
	})
}

func TestNativeFormatBarPosition(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("first line\n\nsecond paragraph here")
		// The pill sits above the selection's line and stays inside the window.
		nativeTestSelect(e, 14, 5)
		open, _, frame, win, anchor := nativeTestFormatBar(e)
		if !open {
			t.Fatal("bar not shown")
		}
		if frame.X < win.X || frame.X+frame.W > win.X+win.W || frame.Y < win.Y || frame.Y+frame.H > win.Y+win.H {
			t.Errorf("bar outside window: bar=%+v window=%+v", frame, win)
		}
		if anchor.W == 0 || anchor.H == 0 {
			t.Fatalf("no anchor line: %+v", anchor)
		}
		if frame.Y < anchor.Y+anchor.H {
			t.Errorf("bar not above line: bar=%+v anchor=%+v", frame, anchor)
		}

		// The first line is close to the window's top edge; whether it flips or
		// not, the pill must stay inside the window and never cover the line.
		nativeTestSelect(e, 0, 5)
		open, _, frame, win, anchor = nativeTestFormatBar(e)
		if !open {
			t.Fatal("bar not shown on first line")
		}
		if frame.X < win.X || frame.X+frame.W > win.X+win.W || frame.Y < win.Y || frame.Y+frame.H > win.Y+win.H {
			t.Errorf("first-line bar outside window: bar=%+v window=%+v", frame, win)
		}
		if frame.Y < anchor.Y+anchor.H && frame.Y+frame.H > anchor.Y {
			t.Errorf("bar covers the first line: bar=%+v anchor=%+v", frame, anchor)
		}
	})
}
