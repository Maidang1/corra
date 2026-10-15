package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveExistingNote(t *testing.T) {
	t.Run("replace complete contents and preserve permissions", func(t *testing.T) {
		for _, mode := range []os.FileMode{0o600, 0o640, 0o444} {
			t.Run(mode.String(), func(t *testing.T) {
				root := t.TempDir()
				path := writeNote(t, root, "note.md", "# Old body with a long tail", 0)
				if err := os.Chmod(path, mode); err != nil {
					t.Fatal(err)
				}
				original, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := SaveExistingNote(path, "# New\n"); err != nil {
					t.Fatal(err)
				}
				assertNoteSource(t, path, "# New\n")
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != mode {
					t.Fatalf("permissions changed: info=%v err=%v", info, err)
				}
				if os.SameFile(original, info) {
					t.Fatal("save mutated the original inode instead of replacing a complete temporary file")
				}
				assertNoSaveTemps(t, root)
			})
		}
	})
	t.Run("reject missing file without resurrection", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "missing.md")
		if err := SaveExistingNote(path, "# New"); !os.IsNotExist(err) {
			t.Fatalf("missing note error = %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("missing note was created: %v", err)
		}
		assertNoSaveTemps(t, root)
	})
	t.Run("reject symlink without touching its target", func(t *testing.T) {
		root := t.TempDir()
		target := writeNote(t, root, "target.md", "# Keep", 0)
		path := filepath.Join(root, "link.md")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if err := SaveExistingNote(path, "# New"); err == nil {
			t.Fatal("save accepted a symlink")
		}
		assertNoteSource(t, target, "# Keep")
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("symlink changed: info=%v err=%v", info, err)
		}
		assertNoSaveTemps(t, root)
	})
	t.Run("reject directory", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "folder.md")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := SaveExistingNote(path, "# New"); err == nil {
			t.Fatal("save accepted directory")
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("directory changed: info=%v err=%v", info, err)
		}
		assertNoSaveTemps(t, root)
	})
	t.Run("write error preserves original and removes temporary file", func(t *testing.T) {
		root := t.TempDir()
		path := writeNote(t, root, "note.md", "# Keep", 0)
		writeErr := errors.New("source read failed after partial output")
		reader := io.MultiReader(strings.NewReader("partial"), saveReaderFunc(func([]byte) (int, error) { return 0, writeErr }))
		if err := saveExistingNote(path, reader); !errors.Is(err, writeErr) {
			t.Fatalf("write error = %v, want source error", err)
		}
		assertNoteSource(t, path, "# Keep")
		assertNoSaveTemps(t, root)
	})
	t.Run("external change cancels commit", func(t *testing.T) {
		for _, change := range []string{"removed", "replaced", "mtime", "size", "symlink"} {
			t.Run(change, func(t *testing.T) {
				root := t.TempDir()
				path := writeNote(t, root, "note.md", "# Keep", 0)
				original, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				source := strings.NewReader("# New")
				changed := false
				reader := saveReaderFunc(func(buf []byte) (int, error) {
					if !changed {
						changed = true
						switch change {
						case "removed":
							if err := os.Remove(path); err != nil {
								t.Fatal(err)
							}
						case "replaced":
							next := writeNote(t, root, "replacement.md", "# Peer", 0)
							if err := os.Chtimes(next, original.ModTime(), original.ModTime()); err != nil {
								t.Fatal(err)
							}
							if err := os.Rename(next, path); err != nil {
								t.Fatal(err)
							}
						case "mtime":
							next := original.ModTime().Add(time.Minute)
							if err := os.Chtimes(path, next, next); err != nil {
								t.Fatal(err)
							}
						case "size":
							if err := os.WriteFile(path, []byte("# External source is longer"), 0o600); err != nil {
								t.Fatal(err)
							}
							if err := os.Chtimes(path, original.ModTime(), original.ModTime()); err != nil {
								t.Fatal(err)
							}
						case "symlink":
							if err := os.Remove(path); err != nil {
								t.Fatal(err)
							}
							target := writeNote(t, root, "target.md", "# Peer", 0)
							if err := os.Symlink(target, path); err != nil {
								t.Fatal(err)
							}
						}
					}
					return source.Read(buf)
				})
				if err := saveExistingNote(path, reader); err == nil {
					t.Fatal("save overwrote an external change")
				}
				switch change {
				case "removed":
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("save resurrected note: %v", err)
					}
				case "replaced", "symlink":
					assertNoteSource(t, path, "# Peer")
				case "mtime":
					assertNoteSource(t, path, "# Keep")
				case "size":
					assertNoteSource(t, path, "# External source is longer")
				}
				assertNoSaveTemps(t, root)
			})
		}
	})
}

type saveReaderFunc func([]byte) (int, error)

func (f saveReaderFunc) Read(buf []byte) (int, error) { return f(buf) }

func assertNoteSource(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("note contents = %q, error = %v; want %q", data, err, want)
	}
}

func assertNoSaveTemps(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".corra-save-") {
			t.Errorf("temporary save file left behind: %s", entry.Name())
		}
	}
}
