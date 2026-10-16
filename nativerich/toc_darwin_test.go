//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"testing"
)

type nativeTOC struct {
	Entries []struct {
		Level int    `json:"level"`
		Text  string `json:"text"`
	} `json:"entries"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func readNativeTOCs(t *testing.T, e *Editor) []nativeTOC {
	t.Helper()
	var tocs []nativeTOC
	if err := json.Unmarshal([]byte(nativeTestTOC(e)), &tocs); err != nil {
		t.Error(err)
	}
	return tocs
}

// [TOC] renders the document's headings as one figure, and the marker itself is
// the only text the document keeps.
func TestNativeTOCListsHeadings(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "[TOC]\n\n# First\n\ntext\n\n## Second\n\n### Deep\n\n# Last\n"
		e.Load(source)
		tocs := readNativeTOCs(t, e)
		if len(tocs) != 1 {
			t.Errorf("toc figure missing: %+v", tocs)
			return
		}
		want := []struct {
			level int
			text  string
		}{{1, "First"}, {2, "Second"}, {3, "Deep"}, {1, "Last"}}
		if len(tocs[0].Entries) != len(want) {
			t.Fatalf("entries: %+v", tocs[0].Entries)
		}
		for i, entry := range want {
			got := tocs[0].Entries[i]
			if got.Level != entry.level || got.Text != entry.text {
				t.Errorf("entry %d = %+v, want %d/%q", i, got, entry.level, entry.text)
			}
		}
		// The figure is drawn as wide as the column and one line per entry.
		if tocs[0].Width <= 0 || tocs[0].Height <= 0 {
			t.Errorf("figure not drawn: %+v", tocs[0])
		}
		if got, dirty, err := e.Snapshot(); got != source || dirty || err != nil {
			t.Errorf("load changed source: %q dirty=%v err=%v", got, dirty, err)
		}
	})
}

// The list follows the headings: renaming one, adding one and removing one all
// redraw it, and the marker keeps its place in the document.
func TestNativeTOCFollowsHeadings(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("[TOC]\n\n# One\n\n## Two\n")
		texts := func() []string {
			tocs := readNativeTOCs(t, e)
			if len(tocs) != 1 {
				t.Fatalf("toc figure missing: %+v", tocs)
			}
			var out []string
			for _, entry := range tocs[0].Entries {
				out = append(out, entry.Text)
			}
			return out
		}
		// The caret starts on the marker; the list is drawn as loaded.
		nativeTestSelect(e, 0, 0)
		if got := texts(); len(got) != 2 || got[0] != "One" || got[1] != "Two" {
			t.Fatalf("initial list: %v", got)
		}
		// Renaming a heading through the source rewrites the entry.
		e.Load("[TOC]\n\n# Renamed\n\n## Two\n")
		if got := texts(); len(got) != 2 || got[0] != "Renamed" {
			t.Errorf("rename not followed: %v", got)
		}
		e.Load("[TOC]\n\n# Renamed\n\n## Two\n\n### Three\n")
		if got := texts(); len(got) != 3 || got[2] != "Three" {
			t.Errorf("added heading not listed: %v", got)
		}
		e.Load("[TOC]\n\n# Renamed\n\n### Three\n")
		if got := texts(); len(got) != 2 || got[1] != "Three" {
			t.Errorf("removed heading still listed: %v", got)
		}
		e.Load("[TOC]\n\nno headings left\n")
		if got := texts(); len(got) != 0 {
			t.Errorf("empty document still listed: %v", got)
		}
		if got, _, _ := e.Snapshot(); got != "[TOC]\n\nno headings left\n" {
			t.Errorf("marker lost: %q", got)
		}
	})
}

// A document without the marker has no figure at all, and one whose marker was
// typed into is source text again.
func TestNativeTOCOnlyForTheMarker(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("# One\n\n## Two\n")
		if tocs := readNativeTOCs(t, e); len(tocs) != 0 {
			t.Errorf("figure without a marker: %+v", tocs)
		}
		e.Load("[TOC]\n\n# One\n")
		if tocs := readNativeTOCs(t, e); len(tocs) != 1 {
			t.Fatalf("marker not rendered: %+v", tocs)
		}
		// Typing into the figure turns the marker back into editable source.
		nativeTestSelect(e, 0, 1)
		nativeTestAction(e, "type", "x")
		if tocs := readNativeTOCs(t, e); len(tocs) != 0 {
			t.Errorf("edited marker still a figure: %+v", tocs)
		}
		if got, _, _ := e.Snapshot(); got != "x\n\n# One\n" {
			t.Errorf("source changed: %q", got)
		}
	})
}

// The marker opens as the source it is and folds back into the list, so the
// document keeps the same bytes whichever of the two is on screen.
func TestNativeTOCRevealsTheMarker(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "[TOC]\n\n# One\n"
		e.Load(source)
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "context", "toc:reveal")
		if tocs := readNativeTOCs(t, e); len(tocs) != 0 {
			t.Errorf("revealed marker still a figure: %+v", tocs)
		}
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("reveal changed source: %q", got)
		}
		// Leaving the revealed marker re-renders the list over it.
		nativeTestAction(e, "blur", "")
		if tocs := readNativeTOCs(t, e); len(tocs) != 1 {
			t.Fatalf("marker did not fold back: %+v", tocs)
		}
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("fold changed source: %q", got)
		}
	})
}

// Picking an entry puts the caret on the heading it names.
func TestNativeTOCJump(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		// The marker is one character in the text storage, so the first heading
		// starts at 2 and the second at 6.
		e.Load("[TOC]\n\n# One\n\n## Two\n")
		if tocs := readNativeTOCs(t, e); len(tocs) != 1 || len(tocs[0].Entries) != 2 {
			t.Fatalf("toc figure: %+v", tocs)
		}
		for _, entry := range []struct{ index, want int }{{0, 2}, {1, 6}} {
			nativeTestSelect(e, 0, 0)
			nativeTestAction(e, "toc", string(rune('0'+entry.index)))
			if start, length := nativeTestSelection(e); start != entry.want || length != 0 {
				t.Errorf("entry %d moved the caret to %d+%d, want %d", entry.index, start, length, entry.want)
			}
		}
	})
}
