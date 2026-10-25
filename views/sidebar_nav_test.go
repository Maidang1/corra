package views

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// sidebarNavFixture builds a library of three notes with a deterministic
// order (Alpha newest, then Beta, then Gamma) and opens Alpha.
func sidebarNavFixture(t *testing.T) (*View, *ui.Tester, []string) {
	t.Helper()
	v, tt := notesFixture(t)
	paths := []string{}
	for i, title := range []string{"Alpha", "Beta", "Gamma"} {
		path := filepath.Join(v.App.LibraryRoot, title+".md")
		if err := os.WriteFile(path, []byte("# "+title+"\n\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stamp := time.Now().Add(-time.Duration(i+1) * time.Hour)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	// notesFixture's own note.md sorts last, after the three named notes.
	note := filepath.Join(v.App.LibraryRoot, "note.md")
	stamp := time.Now().Add(-4 * time.Hour)
	if err := os.Chtimes(note, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	v.App.ScanLibrary()
	v.openNote(paths[0])
	tt.Frame()
	return v, tt, paths
}

// Arrow keys walk the sidebar's note list while a row has the keyboard
// focus, wrapping at the ends and moving the focus along, the way Apple
// Notes' list box does.
func TestSidebarNoteArrowNavigation(t *testing.T) {
	v, tt, paths := sidebarNavFixture(t)

	// Clicking a row opens the note and focuses the row.
	click(t, tt, "文档：Beta")
	tt.Frame()
	if v.App.SelNote != paths[1] {
		t.Fatalf("click did not open Beta: %q", v.App.SelNote)
	}
	// Down moves to the next note; the fixture's own note.md sorts last.
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if v.App.SelNote != paths[2] {
		t.Fatalf("arrow down did not open Gamma: %q", v.App.SelNote)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if filepath.Base(v.App.SelNote) != "note.md" {
		t.Fatalf("arrow down did not open the last note: %q", v.App.SelNote)
	}
	// Down from the last wraps to the first; up from the first wraps back.
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if v.App.SelNote != paths[0] {
		t.Fatalf("arrow down did not wrap to Alpha: %q", v.App.SelNote)
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if filepath.Base(v.App.SelNote) != "note.md" {
		t.Fatalf("arrow up did not wrap to the last note: %q", v.App.SelNote)
	}
	v.Close()
}

// While the search field has the focus, arrows move the found selection
// without leaving the field, and Return hands the focus to the editor.
func TestSidebarSearchArrowNavigation(t *testing.T) {
	v, tt, paths := sidebarNavFixture(t)

	click(t, tt, "搜索文档")
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if v.App.SelNote != paths[1] {
		t.Fatalf("search arrow did not move to Beta: %q", v.App.SelNote)
	}
	if v.focusNotePath != "" {
		t.Fatal("search navigation stole the field's focus")
	}
	// The caret is still in the field: typing keeps filtering.
	tt.Type("Beta")
	tt.Frame()
	if v.App.Query != "Beta" {
		t.Fatalf("search field lost the keys: %q", v.App.Query)
	}
	// Return picks the found note for writing.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v.focusEditor {
		t.Fatal("return did not hand the focus to the editor")
	}
	if v.App.SelNote != paths[1] {
		t.Fatalf("return changed the selection: %q", v.App.SelNote)
	}
	v.Close()
}
