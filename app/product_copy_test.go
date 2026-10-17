package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProductErrorCopy(t *testing.T) {
	t.Run("library is not configured", func(t *testing.T) {
		_, err := openLibraryRoot("", false)
		if err == nil || err.Error() != "document library is not configured" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("not a Markdown document", func(t *testing.T) {
		_, err := libraryNotePath(nil, "", "plain.bin")
		if err == nil || err.Error() != "not a Markdown document: plain.bin" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("new document needs a library", func(t *testing.T) {
		a := New()
		err := a.NewNote()
		if err == nil || err.Error() != "未设置文档库目录" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("non-regular document", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "folder")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		err := SaveExistingNote(path, "content")
		want := "cannot save a non-regular document: " + path
		if err == nil || err.Error() != want {
			t.Fatalf("error = %v, want %q", err, want)
		}
	})
}
