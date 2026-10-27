package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFolderRecursivelyAndSaveInPlace(t *testing.T) {
	root := t.TempDir()
	path := writeNote(t, root, "one/two/note.md", "# Nested", 0)
	writeNote(t, root, "root.md", "# Root", 0)
	writeNote(t, root, "ignore.json", "{}", 0)
	a := New()
	a.LibraryRoot = t.TempDir()
	a.SelFolder, a.Query = "old", "old query"
	var notified string
	f := &FileOps{App: a, BeforeOpen: func() bool { return true }, OnLibraryOpened: func(root string) { notified = root }}
	if err := openFolderForTest(t, f, root); err != nil {
		t.Fatal(err)
	}
	if a.LibraryRoot != root || notified != root || len(a.VisibleNotes()) != 2 || a.SelFolder != "" || a.Query != "" {
		t.Fatalf("folder not adopted: %+v, callback %q", a.Lib, notified)
	}
	if err := a.OpenNote(path); err != nil {
		t.Fatal(err)
	}
	a.SetSource("# Edited", 0)
	if err := f.writePath(path, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "# Edited" {
		t.Fatalf("save: %q, %v", data, err)
	}
	if a.SnapshotSettings().LibraryRoot != root {
		t.Fatal("selected library missing from settings")
	}
}

func TestOpenFolderCancellationAndInvalidPaths(t *testing.T) {
	a := New()
	old := t.TempDir()
	a.LibraryRoot = old
	a.Loaded(filepath.Join(old, "old.md"), "# Pending")
	f := &FileOps{App: a, BeforeOpen: func() bool { return false }}
	if err := openFolderForTest(t, f, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if a.LibraryRoot != old || a.CurrentSource() != "# Pending" {
		t.Fatal("cancelled switch changed state")
	}
	f.BeforeOpen = func() bool { return true }
	for _, path := range []string{filepath.Join(old, "missing"), writeNote(t, old, "file.md", "# File", 0)} {
		if err := openFolderForTest(t, f, path); err == nil {
			t.Fatalf("accepted %s", path)
		}
		if a.LibraryRoot != old || a.CurrentSource() != "# Pending" {
			t.Fatal("failed switch changed state")
		}
	}
}

func TestOpenEmptyFolderClearsOldDocument(t *testing.T) {
	a := New()
	a.Loaded("old.md", "# Old")
	a.SelNote = "old.md"
	root := t.TempDir()
	var opened string = "not called"
	f := &FileOps{App: a, OnOpened: func(path string) { opened = path }}
	if err := openFolderForTest(t, f, root); err != nil {
		t.Fatal(err)
	}
	if a.CurrentPath() != "" || a.CurrentSource() != "" || a.SelNote != "" || opened != "" {
		t.Fatal("stale document or watcher")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("opening empty folder created files")
	}
}

func TestOpenFolderThroughPathAndDrop(t *testing.T) {
	for _, drop := range []bool{false, true} {
		t.Run(map[bool]string{false: "path", true: "drop"}[drop], func(t *testing.T) {
			root := t.TempDir()
			path := writeNote(t, root, "nested/note.md", "# Nested", 0)
			a := New()
			queued := make(chan func(), 1)
			f := &FileOps{App: a, Dispatch: func(fn func()) { queued <- fn }}
			if drop {
				f.OpenDropped(root)
			} else {
				f.OpenPath(root)
			}
			nextFolderUpdate(t, queued)
			if a.LibraryRoot != root || a.CurrentPath() != path {
				t.Fatal("folder route did not load nested document")
			}
		})
	}
}

func TestOpenFolderDoesNotDiscardDirtyDocumentWithoutSaveHook(t *testing.T) {
	a := New()
	a.SetSource("unsaved", 0)
	f := &FileOps{App: a}
	if err := openFolderForTest(t, f, t.TempDir()); err == nil {
		t.Fatal("discarded dirty document")
	}
	if a.CurrentSource() != "unsaved" || !a.Dirty {
		t.Fatal("unsaved source changed")
	}
}

func TestFolderLoadScansAfterSavingCurrentDocument(t *testing.T) {
	root := t.TempDir()
	path := writeNote(t, root, "note.md", "# Before", 0)
	a := New()
	a.LibraryRoot = root
	a.Loaded(path, "# Before")
	a.SetSource("# Pending", 0)
	f := &FileOps{App: a}
	f.BeforeOpen = func() bool { return f.writePath(path, false) == nil }
	if err := openFolderForTest(t, f, root); err != nil {
		t.Fatal(err)
	}
	if a.CurrentSource() != "# Pending" {
		t.Fatal("folder load restored a stale version from before autosave")
	}
}

func openFolderForTest(t *testing.T, f *FileOps, path string) error {
	t.Helper()
	queued := make(chan func(), 1)
	f.startFolderLoad(path, func(fn func()) { queued <- fn }, readFolderLibrary)
	if f.App.LibraryLoading {
		nextFolderUpdate(t, queued)
	}
	if f.App.LibraryLoadError != "" {
		return errors.New(f.App.LibraryLoadError)
	}
	return nil
}
