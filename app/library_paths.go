package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func openLibraryRoot(root string, create bool) (*os.Root, error) {
	if root == "" {
		return nil, fmt.Errorf("notes library is not configured")
	}
	if create {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, err
		}
	}
	return os.OpenRoot(root)
}

func libraryFolder(folder string, allowRoot bool) (string, error) {
	if folder == "" && allowRoot {
		return ".", nil
	}
	if folder == "" || filepath.IsAbs(folder) || strings.Contains(folder, "\\") {
		return "", fmt.Errorf("invalid folder name: %q", folder)
	}
	for _, part := range strings.Split(folder, "/") {
		if part == "" || part == "." || part == ".." || part == TrashName || strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("invalid folder name: %q", folder)
		}
	}
	return filepath.FromSlash(folder), nil
}

// os.Root confines operations even if a directory changes during an operation.
// Reject links as well so writes never silently target a different local note.
func checkLibraryPath(root *os.Root, rel string, allowMissing bool) error {
	current := "."
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if allowMissing && os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not managed notes paths: %s", rel)
		}
	}
	return nil
}

func libraryNotePath(dir *os.Root, root, path string) (string, error) {
	if path == "" || !mdExt(path) {
		return "", fmt.Errorf("not a Markdown note: %s", path)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil || !filepath.IsLocal(rel) || rel == "." {
		return "", fmt.Errorf("note is outside the library: %s", path)
	}
	if err := checkLibraryPath(dir, rel, false); err != nil {
		return "", err
	}
	info, err := dir.Lstat(rel)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular note file: %s", path)
	}
	return rel, nil
}

func isTrashPath(rel string) bool {
	return strings.HasPrefix(filepath.ToSlash(rel), TrashName+"/")
}

func numberedName(name string, n int) string {
	if n == 1 {
		return name
	}
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(name, ext), n, ext)
}

func moveLibraryNote(dir *os.Root, source, folder string) (string, error) {
	if filepath.Dir(source) == folder {
		return source, nil
	}
	for i := 1; ; i++ {
		dest := filepath.Join(folder, numberedName(filepath.Base(source), i))
		// Link creates the destination atomically without replacing an existing
		// file; only remove the source after its contents have another name.
		if err := dir.Link(source, dest); os.IsExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
		if err := dir.Remove(source); err != nil {
			_ = dir.Remove(dest)
			return "", err
		}
		return dest, nil
	}
}

func renameLibraryFolder(dir *os.Root, old, next string) error {
	oldParent, err := dir.Open(filepath.Dir(old))
	if err != nil {
		return err
	}
	defer oldParent.Close()
	nextParent, err := dir.Open(filepath.Dir(next))
	if err != nil {
		return err
	}
	defer nextParent.Close()
	return renameNoReplace(oldParent, filepath.Base(old), nextParent, filepath.Base(next))
}
