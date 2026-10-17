package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CreateFolder creates a local folder without replacing an existing entry.
func (a *App) CreateFolder(folder string) error {
	rel, err := libraryFolder(folder, false)
	if err != nil {
		return err
	}
	a.mu.Lock()
	root := a.LibraryRoot
	a.mu.Unlock()
	dir, err := openLibraryRoot(root, true)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := checkLibraryPath(dir, rel, true); err != nil {
		return err
	}
	if err := dir.Mkdir(rel, 0o755); err != nil {
		return err
	}
	a.ScanLibrary()
	return nil
}

// RenameFolder renames a folder and keeps the open note and unsaved source.
// Both names are relative to the library root; an existing target is an error.
func (a *App) RenameFolder(folder, newFolder string) error {
	old, err := libraryFolder(folder, false)
	if err != nil {
		return err
	}
	next, err := libraryFolder(newFolder, false)
	if err != nil {
		return err
	}
	a.mu.Lock()
	root := a.LibraryRoot
	a.mu.Unlock()
	dir, err := openLibraryRoot(root, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := checkLibraryPath(dir, old, false); err != nil {
		return err
	}
	info, err := dir.Stat(old)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a document library folder: %s", folder)
	}
	if old == next {
		return nil
	}
	if err := checkLibraryPath(dir, next, true); err != nil {
		return err
	}
	if err := renameLibraryFolder(dir, old, next); err != nil {
		return err
	}
	a.mu.Lock()
	if a.SelFolder == folder || strings.HasPrefix(a.SelFolder, folder+"/") {
		a.SelFolder = newFolder + strings.TrimPrefix(a.SelFolder, folder)
	}
	oldPath, newPath := filepath.Join(root, old), filepath.Join(root, next)
	remap := func(path string) string {
		rel, err := filepath.Rel(oldPath, path)
		if path != "" && err == nil && filepath.IsLocal(rel) {
			return filepath.Join(newPath, rel)
		}
		return path
	}
	previous := a.Path
	a.Path, a.SelNote = remap(a.Path), remap(a.SelNote)
	for i, recent := range a.Recent {
		a.Recent[i] = remap(recent)
	}
	path, opened := a.Path, a.NoteOpened
	a.mu.Unlock()
	a.ScanLibrary()
	if previous != path && opened != nil {
		opened(path)
	}
	return nil
}

// DeleteFolder removes only an empty folder, including no hidden/non-note files.
func (a *App) DeleteFolder(folder string) error {
	rel, err := libraryFolder(folder, false)
	if err != nil {
		return err
	}
	a.mu.Lock()
	root := a.LibraryRoot
	a.mu.Unlock()
	dir, err := openLibraryRoot(root, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := checkLibraryPath(dir, rel, false); err != nil {
		return err
	}
	info, err := dir.Stat(rel)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a document library folder: %s", folder)
	}
	if err := dir.Remove(rel); err != nil {
		return err
	}
	a.mu.Lock()
	selected := a.SelFolder == folder
	if selected {
		a.SelFolder = ""
		a.SelNote = ""
		a.Query = ""
	}
	a.mu.Unlock()
	if selected {
		a.NewDocument()
	}
	a.ScanLibrary()
	return nil
}

// MoveSelectedNote moves to an existing active folder, selecting the result.
// Pending source edits remain in memory for the caller's save lifecycle.
func (a *App) MoveSelectedNote(folder string) error {
	rel, err := libraryFolder(folder, true)
	if err != nil {
		return err
	}
	return a.relocateSelectedNote(rel, folder, false)
}

// RestoreSelectedNote restores to the library root; collisions gain a suffix.
func (a *App) RestoreSelectedNote() error {
	return a.relocateSelectedNote(".", "", true)
}

func (a *App) relocateSelectedNote(dest, folder string, restore bool) error {
	a.mu.Lock()
	root, path := a.LibraryRoot, a.SelNote
	a.mu.Unlock()
	dir, err := openLibraryRoot(root, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	rel, err := libraryNotePath(dir, root, path)
	if err != nil {
		return err
	}
	if isTrashPath(rel) != restore {
		return fmt.Errorf("restore requires a deleted document; move requires an active document")
	}
	if err := checkLibraryPath(dir, dest, false); err != nil {
		return err
	}
	info, err := dir.Stat(dest)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a document library folder: %s", folder)
	}
	next, err := moveNoteWithAttachments(dir, root, rel, dest)
	if err != nil {
		return err
	}
	newPath := filepath.Join(root, next)
	a.mu.Lock()
	a.SelFolder, a.Query, a.SelNote = folder, "", newPath
	if a.Path == path {
		a.Source = rebaseAttachmentPaths(a.Source, filepath.Dir(path), filepath.Dir(newPath))
		a.reparse()
		a.Path = newPath
		a.Title = filepath.Base(newPath)
	}
	for i, recent := range a.Recent {
		if recent == path {
			a.Recent[i] = newPath
		}
	}
	opened := a.NoteOpened
	a.mu.Unlock()
	a.ScanLibrary()
	if opened != nil {
		opened(newPath)
	}
	return nil
}
