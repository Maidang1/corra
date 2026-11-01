package app

import (
	"os"
	"path/filepath"
	"testing"
)

// copyNoReplace is the move fallback on volumes without hard links.
func TestCopyNoReplace(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "taken.md"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()

	if err := copyNoReplace(dir, "a.md", "taken.md"); !os.IsExist(err) {
		t.Fatalf("existing target: err=%v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "taken.md")); string(got) != "keep\n" {
		t.Fatalf("existing target overwritten: %q", got)
	}
	if err := copyNoReplace(dir, "a.md", "b.md"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "b.md")); string(got) != "# A\n" {
		t.Fatalf("copy: %q", got)
	}
	if err := copyNoReplace(dir, "missing.md", "c.md"); err == nil {
		t.Fatal("missing source copied")
	}
	if _, err := os.Stat(filepath.Join(root, "c.md")); !os.IsNotExist(err) {
		t.Fatalf("failed copy left a target: %v", err)
	}
}
