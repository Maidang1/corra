//go:build darwin && cgo

package nativerich

import "testing"

func TestNativeTextColorFromFormatBar(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		e.Load("hello world")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "palette")
		if open, _, _ := nativeTestFormatBarPalette(e); !open {
			t.Error("palette button did not unfold the palette")
			return
		}
		nativeTestAction(e, "formatbar-pick", "color:#d44c47")
		open, color, _ := nativeTestFormatBarPalette(e)
		if open || color != "#d44c47" {
			t.Errorf("after a pick: palette=%v color=%q", open, color)
		}
		nativeTestSelect(e, 6, 5)
		nativeTestAction(e, "formatbar-pick", "background:#fbf3db")
		source, _, err := e.Snapshot()
		if err != nil {
			t.Error(err)
			return
		}
		want := `<span style="color:#d44c47">hello</span> <span style="background-color:#fbf3db">world</span>`
		if source != want {
			t.Errorf("saved %q, want %q", source, want)
		}

		// Default clears the color again.
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "formatbar-pick", "color:")
		source, _, _ = e.Snapshot()
		if want := `hello <span style="background-color:#fbf3db">world</span>`; source != want {
			t.Errorf("after clearing: %q, want %q", source, want)
		}
		// Clear formatting strips colors with the other traits.
		nativeTestSelect(e, 0, 11)
		nativeTestAction(e, "formatbar-pick", "clear")
		if source, _, _ = e.Snapshot(); source != "hello world" {
			t.Errorf("after clear: %q", source)
		}
	})
}

func TestNativeTextColorLoadsAndRoundTrips(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()

		source := `a <span style="color:#337ea9">blue **bold**</span> b`
		e.Load(source)
		nativeTestSelect(e, 2, 4)
		if _, color, _ := nativeTestFormatBarPalette(e); color != "#337ea9" {
			t.Errorf("loaded color %q", color)
		}
		// Typing inside the colored phrase keeps the color.
		nativeTestSelect(e, 4, 0)
		nativeTestAction(e, "type", "X")
		got, _, err := e.Snapshot()
		if err != nil {
			t.Error(err)
			return
		}
		if want := `a <span style="color:#337ea9">blXue **bold**</span> b`; got != want {
			t.Errorf("saved %q, want %q", got, want)
		}
	})
}

func TestNativeColorAndMentionSurviveCopyPaste(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		e.Load("<span style=\"color:#d44c47\">red</span> [@Beta](beta.md)\n\nend")
		nativeTestSelect(e, 0, len("red @Beta"))
		nativeTestAction(e, "copy", "")
		nativeTestSelect(e, len("red @Beta\nend"), 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, err := e.Snapshot()
		if err != nil {
			t.Error(err)
			return
		}
		want := "<span style=\"color:#d44c47\">red</span> [@Beta](beta.md)\n\nend<span style=\"color:#d44c47\">red</span> [@Beta](beta.md)"
		if got != want {
			t.Errorf("after paste %q, want %q", got, want)
		}
	})
}
