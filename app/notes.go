package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ScanLibrary rescans LibraryRoot into Lib. A missing root is not an
// error; the library is simply empty.
func (a *App) ScanLibrary() {
	a.mu.Lock()
	root := a.LibraryRoot
	a.mu.Unlock()
	lib := ScanLibrary(root)
	a.mu.Lock()
	a.Lib = lib
	a.mu.Unlock()
	a.InvalidateUI()
}

// VisibleNotes returns the notes of the selected folder matching the
// search query, most recent first.
func (a *App) VisibleNotes() []Note {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Lib == nil {
		return nil
	}
	return a.Lib.Filter(a.SelFolder, a.Query)
}

// NoteGroups returns the visible notes bucketed into date sections.
func (a *App) NoteGroups() []NoteGroup {
	return GroupByDate(a.VisibleNotes(), time.Now())
}

// OpenNote loads a note file into the editor column and selects it.
func (a *App) OpenNote(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	a.Loaded(path, string(data))
	a.mu.Lock()
	a.SelNote = path
	a.mu.Unlock()
	a.InvalidateUI()
	if a.NoteOpened != nil {
		a.NoteOpened(path)
	}
	return nil
}

// NewNote creates a note in the selected folder and opens it. Deleted
// notes' folder creates in the library root instead.
func (a *App) NewNote() error {
	a.mu.Lock()
	root, folder := a.LibraryRoot, a.SelFolder
	a.mu.Unlock()
	if root == "" {
		return fmt.Errorf("未设置备忘录文件夹")
	}
	if folder == TrashName {
		folder = ""
	}
	path, err := CreateNote(root, folder)
	if err != nil {
		return err
	}
	a.ScanLibrary()
	a.mu.Lock()
	a.SelFolder = folder
	a.Query = ""
	a.Mode = ModeEdit
	a.mu.Unlock()
	return a.OpenNote(path)
}

// DeleteSelectedNote moves the selected note to the trash (or deletes it
// permanently when it is already there), rescans the library and opens
// the most recent remaining note of the folder (or clears the editor
// when none is left).
func (a *App) DeleteSelectedNote() error {
	a.mu.Lock()
	root, path := a.LibraryRoot, a.SelNote
	a.mu.Unlock()
	if path == "" {
		return nil
	}
	if err := DeleteNote(root, path); err != nil {
		return err
	}
	a.ScanLibrary()
	a.mu.Lock()
	a.SelNote = ""
	a.mu.Unlock()
	if notes := a.VisibleNotes(); len(notes) > 0 {
		return a.OpenNote(notes[0].Path)
	}
	a.NewDocument()
	return nil
}
