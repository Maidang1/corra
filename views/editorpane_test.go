package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	meditor "github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
)

// TestEditorSurvivesAutosaveRebuild is the regression test for "pressing
// Enter scrolls the editor to the top": typing used to leave the rebuild
// base stale, so the autosave's invalidate rebuilt the whole editor state
// (scroll and caret reset) after every keystroke.
func TestEditorSurvivesAutosaveRebuild(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "note.md")
	if err := os.WriteFile(p, []byte("# 标题\n\n正文\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	if err := a.OpenNote(p); err != nil {
		t.Fatal(err)
	}
	a.Mode = app.ModeEdit
	v := New(a, app.NewImageLoader(""))
	ui.Render(v.Build, 900, 700, 1)

	st := v.edState
	if st == nil {
		t.Fatal("editor state not created")
	}

	// Simulate typing: the buffer gains text, the view syncs and the
	// autosave fires and invalidates the window.
	st.Buffer.Edit(meditor.Edit{Start: 0, End: 0, Text: "开头\n"})
	v.syncEditor()
	v.flushAutosave()
	ui.Render(v.Build, 900, 700, 1)

	if v.edState != st {
		t.Fatal("editor state rebuilt after typing + autosave; scroll/caret would reset")
	}
	if got := a.CurrentSource(); !strings.HasPrefix(got, "开头\n# ") {
		t.Fatalf("source not synced: %q", got)
	}
}

// TestEditorRebuiltOnNoteSwitch: changing notes must rebuild the editor
// around the new file, and keep the caret inside the new text.
func TestEditorRebuiltOnNoteSwitch(t *testing.T) {
	root := t.TempDir()
	write := func(name, src string) string {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p1 := write("one.md", "# 一\n\n第一页\n")
	p2 := write("two.md", "# 二\n\n第二页\n")

	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	if err := a.OpenNote(p1); err != nil {
		t.Fatal(err)
	}
	a.Mode = app.ModeEdit
	v := New(a, app.NewImageLoader(""))
	ui.Render(v.Build, 900, 700, 1)
	st1 := v.edState

	if err := a.OpenNote(p2); err != nil {
		t.Fatal(err)
	}
	ui.Render(v.Build, 900, 700, 1)
	if v.edState == st1 || v.edPath != p2 {
		t.Fatal("editor not rebuilt for the new note")
	}
}

// TestEditorKeepsCaretOnRebuild: an external reload (same note, new
// content) rebuilds the buffer but leaves the caret where it was,
// clamped to the new length.
func TestEditorKeepsCaretOnRebuild(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "note.md")
	if err := os.WriteFile(p, []byte("abcdef\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	if err := a.OpenNote(p); err != nil {
		t.Fatal(err)
	}
	a.Mode = app.ModeEdit
	v := New(a, app.NewImageLoader(""))
	ui.Render(v.Build, 900, 700, 1)

	st := v.edState
	st.Select(4, 4)

	// External change reloads the same note with longer text.
	if err := os.WriteFile(p, []byte("abcdefgh\nijkl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.Loaded(p, "abcdefgh\nijkl\n")
	ui.Render(v.Build, 900, 700, 1)

	if v.edState == st {
		t.Fatal("editor not rebuilt after external reload")
	}
	if got := v.edState.Cursor(); got != 4 {
		t.Fatalf("caret = %d, want 4 (kept across rebuild)", got)
	}
}
