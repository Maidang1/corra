package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAppFolderWorkflows(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	if err := a.CreateFolder("work"); err != nil {
		t.Fatal(err)
	}
	if got := a.Lib.Folders(); !reflect.DeepEqual(got, []string{"work"}) {
		t.Fatalf("empty folder missing: %v", got)
	}
	a.SelFolder = "work"
	if err := a.NewNote(); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteFolder("work"); err == nil {
		t.Fatal("deleted nonempty folder")
	}
	old := a.SelNote
	if err := a.RenameFolder("work", "renamed"); err != nil {
		t.Fatal(err)
	}
	if a.SelFolder != "renamed" || a.SelNote != filepath.Join(a.LibraryRoot, "renamed", filepath.Base(old)) || a.CurrentPath() != a.SelNote {
		t.Fatalf("rename selection stale: folder=%q note=%q path=%q", a.SelFolder, a.SelNote, a.CurrentPath())
	}
	if err := a.MoveSelectedNote(""); err != nil {
		t.Fatal(err)
	}
	if a.SelFolder != "" || filepath.Dir(a.SelNote) != a.LibraryRoot || a.CurrentPath() != a.SelNote {
		t.Fatal("move selection not updated")
	}
	if err := a.DeleteFolder("renamed"); err != nil {
		t.Fatal(err)
	}
	if len(a.Lib.Folders()) != 0 {
		t.Fatalf("deleted folder still indexed: %v", a.Lib.Folders())
	}
	t.Run("nested rename retains unsaved state and watcher", func(t *testing.T) {
		a := New()
		a.LibraryRoot = t.TempDir()
		path := writeNote(t, a.LibraryRoot, "parent/child/note.md", "# Original", 0)
		if err := a.OpenNote(path); err != nil {
			t.Fatal(err)
		}
		a.SelFolder = "parent/child"
		a.MarkSourceDirty("# Pending")
		opened := ""
		a.NoteOpened = func(path string) { opened = path }
		if err := a.RenameFolder("parent", "renamed"); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(a.LibraryRoot, "renamed", "child", "note.md")
		if a.SelFolder != "renamed/child" || a.CurrentPath() != want || a.SelNote != want || opened != want || a.Source != "# Pending" || !a.Dirty {
			t.Fatalf("nested rename lost state: folder=%q path=%q selected=%q watched=%q source=%q dirty=%t", a.SelFolder, a.CurrentPath(), a.SelNote, opened, a.Source, a.Dirty)
		}
	})
	t.Run("delete selected empty folder clears editor", func(t *testing.T) {
		a := New()
		a.LibraryRoot = t.TempDir()
		if err := a.CreateFolder("empty"); err != nil {
			t.Fatal(err)
		}
		a.SelFolder = "empty"
		a.Query = "query"
		if err := a.DeleteFolder("empty"); err != nil {
			t.Fatal(err)
		}
		if a.SelFolder != "" || a.SelNote != "" || a.Source != "" || a.Query != "" {
			t.Fatal("deleting selected folder left stale editor state")
		}
	})
}

func TestAppFolderValidation(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(a.LibraryRoot, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"", ".", "..", "../escape", "a/../b", ".trash", "nested/.trash", "linked/child", outside} {
		if err := a.CreateFolder(folder); err == nil {
			t.Errorf("created unsafe folder %q", folder)
		}
	}
	for _, folder := range []string{"", ".", "..", "../escape", ".trash", "linked"} {
		if err := a.DeleteFolder(folder); err == nil {
			t.Errorf("deleted unsafe folder %q", folder)
		}
	}
	if err := a.CreateFolder("first"); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateFolder("second"); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateFolder("first"); err == nil {
		t.Error("duplicate folder create should fail")
	}
	if err := a.RenameFolder("first", "second"); err == nil {
		t.Error("rename overwrote existing empty folder")
	}
	if err := a.RenameFolder("first", "../escape"); err == nil {
		t.Error("rename escaped root")
	}
	writeNote(t, a.LibraryRoot, "first/.keep", "not a note", 0)
	if err := a.DeleteFolder("first"); err == nil {
		t.Error("deleted folder containing a non-note file")
	}
}

func TestAppRestoreSelectedNote(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	writeNote(t, a.LibraryRoot, "same.md", "# Existing", 0)
	trashed := writeNote(t, a.LibraryRoot, ".trash/same.md", "# Restored", 0)
	a.SelFolder = TrashName
	a.Query = "unmatched"
	if err := a.OpenNote(trashed); err != nil {
		t.Fatal(err)
	}
	if err := a.RestoreSelectedNote(); err != nil {
		t.Fatal(err)
	}
	if a.SelFolder != "" || a.Query != "" || a.SelNote != filepath.Join(a.LibraryRoot, "same (2).md") || a.Source != "# Restored" {
		t.Fatalf("restore state wrong: folder=%q query=%q note=%q source=%q", a.SelFolder, a.Query, a.SelNote, a.Source)
	}
	if data, err := os.ReadFile(filepath.Join(a.LibraryRoot, "same.md")); err != nil || string(data) != "# Existing" {
		t.Fatalf("restore overwrote existing note: %q, %v", data, err)
	}
	if TrashCount(a.LibraryRoot) != 0 || len(a.VisibleNotes()) != 2 {
		t.Fatal("restore did not update index")
	}
	if err := a.RestoreSelectedNote(); err == nil {
		t.Fatal("restored non-trash note")
	}
	t.Run("symlink destination is a collision", func(t *testing.T) {
		a := New()
		a.LibraryRoot = t.TempDir()
		outside := writeNote(t, t.TempDir(), "outside.md", "# Outside", 0)
		if err := os.Symlink(outside, filepath.Join(a.LibraryRoot, "note.md")); err != nil {
			t.Fatal(err)
		}
		path := writeNote(t, a.LibraryRoot, ".trash/note.md", "# Restore", 0)
		if err := a.OpenNote(path); err != nil {
			t.Fatal(err)
		}
		if err := a.RestoreSelectedNote(); err != nil {
			t.Fatal(err)
		}
		if a.SelNote != filepath.Join(a.LibraryRoot, "note (2).md") {
			t.Fatalf("restored over symlink: %s", a.SelNote)
		}
		if data, err := os.ReadFile(outside); err != nil || string(data) != "# Outside" {
			t.Fatalf("outside note changed: %q, %v", data, err)
		}
	})
}

func TestAppMoveSelectedNote(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	path := writeNote(t, a.LibraryRoot, "same.md", "# Move me", 0)
	writeNote(t, a.LibraryRoot, "work/same.md", "# Existing", 0)
	if err := a.OpenNote(path); err != nil {
		t.Fatal(err)
	}
	a.MarkSourceDirty("# Unsaved edit")
	if err := a.MoveSelectedNote("work"); err != nil {
		t.Fatal(err)
	}
	if a.SelNote != filepath.Join(a.LibraryRoot, "work", "same (2).md") || a.Source != "# Unsaved edit" || !a.Dirty {
		t.Fatal("move lost pending editor state or collision suffix")
	}
	if data, err := os.ReadFile(filepath.Join(a.LibraryRoot, "work", "same.md")); err != nil || string(data) != "# Existing" {
		t.Fatalf("move overwrote destination: %q, %v", data, err)
	}
	for _, folder := range []string{"missing", ".trash", "../escape"} {
		if err := a.MoveSelectedNote(folder); err == nil {
			t.Errorf("moved to invalid folder %q", folder)
		}
	}
}

