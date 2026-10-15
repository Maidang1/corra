package app

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SaveExistingNote atomically replaces an existing regular note with a complete,
// synced temporary file. It never creates an initially missing destination.
// External-change checks are best effort: another process can still change or
// delete the destination between the final Lstat and Rename. Portable rename
// has no compare-and-swap operation to close that last race.
func SaveExistingNote(path, source string) error {
	return saveExistingNote(path, strings.NewReader(source))
}

func saveExistingNote(path string, source io.Reader) error {
	dir, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	name := filepath.Base(path)
	original, err := dir.Lstat(name)
	if err != nil {
		return err
	}
	if !original.Mode().IsRegular() {
		return fmt.Errorf("cannot save a non-regular note: %s", path)
	}
	var temp *os.File
	var tempName string
	for {
		tempName = ".corra-save-" + rand.Text()
		temp, err = dir.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if !os.IsExist(err) {
			break
		}
	}
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = dir.Remove(tempName)
		}
	}()
	if _, err := io.Copy(temp, source); err != nil {
		return err
	}
	mode := original.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	current, err := dir.Lstat(name)
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(original, current) ||
		!original.ModTime().Equal(current.ModTime()) || original.Size() != current.Size() {
		return fmt.Errorf("note changed during save: %s", path)
	}
	if err := dir.Rename(tempName, name); err != nil {
		return err
	}
	committed = true
	return nil
}
