package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileOpsOpenPath(t *testing.T) {
	for _, target := range []string{"external", "library", "trash"} {
		t.Run(target+" selection", func(t *testing.T) {
			root := t.TempDir()
			folder, note := "", ""
			var path string
			switch target {
			case "external":
				path = writeNote(t, t.TempDir(), "external.md", "# Opened", 0)
			case "library":
				path = writeNote(t, root, "work/note.md", "# Opened", 0)
				folder, note = "work", path
			case "trash":
				path = writeNote(t, root, ".trash/note.md", "# Opened", 0)
				folder, note = TrashName, path
			}
			a := New()
			a.LibraryRoot = root
			a.SelFolder, a.SelNote, a.Query = TrashName, "previous.md", "no match"
			f := &FileOps{App: a}
			f.OpenPath(path)
			if a.SelFolder != folder || a.SelNote != note || a.Query != "" {
				t.Fatalf("selection after open = folder:%q note:%q query:%q, want %q %q empty query", a.SelFolder, a.SelNote, a.Query, folder, note)
			}
		})
	}
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancelled", true: "allowed"}[allow], func(t *testing.T) {
			root := t.TempDir()
			path := writeNote(t, root, "next.md", "# Next", 0)
			a := New()
			a.Loaded(filepath.Join(root, "old.md"), "# Pending")
			oldPath := a.CurrentPath()
			called, opened := 0, 0
			f := &FileOps{App: a, BeforeOpen: func() bool {
				called++
				if a.CurrentPath() != oldPath || a.CurrentSource() != "# Pending" {
					t.Fatal("document replaced before BeforeOpen")
				}
				return allow
			}, OnOpened: func(got string) {
				opened++
				if got != path {
					t.Errorf("opened %q, want %q", got, path)
				}
			}}
			f.OpenPath(path)
			if called != 1 {
				t.Fatalf("BeforeOpen calls = %d, want 1", called)
			}
			if allow {
				if a.CurrentPath() != path || a.CurrentSource() != "# Next" || opened != 1 {
					t.Fatal("approved open did not load and notify")
				}
			} else if a.CurrentPath() != oldPath || a.CurrentSource() != "# Pending" || opened != 0 {
				t.Fatal("cancelled open replaced the document or notified OnOpened")
			}
		})
	}
}

func TestFileOpsOpenDropped(t *testing.T) {
	a := New()
	a.Loaded("old.md", "# Pending")
	a.MarkSourceDirty("# Unsaved")
	called := 0
	f := &FileOps{App: a, BeforeOpen: func() bool { called++; return false }}
	f.OpenDropped(filepath.Join(t.TempDir(), "missing.md"))
	f.OpenDropped("ignored.png")
	if called != 1 || a.CurrentPath() != "old.md" || a.CurrentSource() != "# Unsaved" {
		t.Fatal("drop bypassed the cancellation hook or replaced the document")
	}
}

func TestFileOpsSaveAsPath(t *testing.T) {
	for _, dest := range []string{"library", "external", "missing parent"} {
		t.Run(dest, func(t *testing.T) {
			root := t.TempDir()
			old := writeNote(t, root, "old.md", "# Original", 0)
			a := New()
			a.LibraryRoot = root
			a.Loaded(old, "# Changed")
			a.MarkSourceDirty("# Changed")
			a.SelFolder, a.SelNote = "", old
			path := filepath.Join(root, "saved.md")
			if dest == "external" {
				path = filepath.Join(t.TempDir(), "saved.md")
			} else if dest == "missing parent" {
				path = filepath.Join(root, "missing", "saved.md")
			}
			opened := 0
			f := &FileOps{App: a, OnOpened: func(got string) {
				opened++
				if got != path || a.CurrentPath() != path {
					t.Fatal("notified before adopting saved path")
				}
			}}
			err := f.saveAsPath(path)
			if dest == "missing parent" {
				if err == nil || a.CurrentPath() != old || a.SelNote != old || !a.Dirty || opened != 0 {
					t.Fatal("failed Save As changed current document")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantNote := ""
			if dest == "library" {
				wantNote = path
				if got := len(a.Lib.Notes); got != 2 {
					t.Fatalf("library not rescanned: %d notes", got)
				}
			}
			if a.CurrentPath() != path || a.SelNote != wantNote || a.SelFolder != "" || a.Dirty || opened != 1 {
				t.Fatalf("incorrect Save As state: path=%q selection=%q dirty=%v opened=%d", a.Path, a.SelNote, a.Dirty, opened)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "# Changed" {
				t.Fatalf("saved content = %q, error %v", data, err)
			}
			original, err := os.ReadFile(old)
			if err != nil || string(original) != "# Original" {
				t.Fatal("Save As altered original file")
			}
		})
	}
}

func TestFileOpsWritePath(t *testing.T) {
	for _, scenario := range []string{"existing", "missing", "new", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "note.md")
			if scenario == "existing" {
				writeNote(t, root, "note.md", "# Old", 0)
			}
			var target string
			if scenario == "symlink" {
				target = writeNote(t, root, "target.md", "# Protected", 0)
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			a := New()
			a.Loaded(path, "# Updated")
			a.MarkSourceDirty("# Updated")
			f := &FileOps{App: a}
			err := f.writePath(path, scenario == "new")
			if scenario == "missing" || scenario == "symlink" {
				if err == nil || !a.Dirty {
					t.Fatal("unsafe write succeeded or cleared dirty state")
				}
				if scenario == "missing" {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatal("deleted file was recreated")
					}
				} else if data, err := os.ReadFile(target); err != nil || string(data) != "# Protected" {
					t.Fatal("symlink target modified")
				}
				return
			}
			if err != nil || a.Dirty {
				t.Fatalf("save failed: %v, dirty=%v", err, a.Dirty)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "# Updated" {
				t.Fatal("saved file does not match source")
			}
		})
	}
}

func TestWatcherPollOnce(t *testing.T) {
	for _, scenario := range []string{"clean", "dirty", "switched", "closed", "deleted", "newer disk", "unchanged", "missing"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			path := writeNote(t, root, "note.md", "# Before", time.Hour)
			a := New()
			a.Loaded(path, "# Before")
			st, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			var queue []func()
			w := &Watcher{App: a, Path: path, Mod: st.ModTime(), Stop: make(chan struct{}),
				dispatch: func(fn func()) { queue = append(queue, fn) }}
			t.Cleanup(w.Close)
			if scenario != "unchanged" {
				writeNote(t, root, "note.md", "# External", 0)
			}
			if scenario == "missing" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan struct{})
			go func() {
				w.pollOnce()
				close(done)
			}()
			<-done
			if a.CurrentSource() != "# Before" {
				t.Fatal("poll mutated App before UI dispatch")
			}
			if scenario == "unchanged" || scenario == "missing" {
				if len(queue) != 0 {
					t.Fatal("queued a reload without a readable changed file")
				}
				return
			}
			if len(queue) != 1 {
				t.Fatalf("queued deliveries = %d, want 1", len(queue))
			}
			switch scenario {
			case "dirty":
				a.MarkSourceDirty("# Local")
			case "switched":
				a.Loaded(filepath.Join(root, "next.md"), "# Next")
			case "closed":
				w.Close()
			case "deleted":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "newer disk":
				writeNote(t, root, "note.md", "# Saved later", -time.Hour)
				a.Loaded(path, "# Saved later")
			}
			wantPath, wantSource := a.CurrentPath(), a.CurrentSource()
			queue[0]()
			if scenario == "clean" {
				wantSource = "# External"
			}
			if a.CurrentPath() != wantPath || a.CurrentSource() != wantSource {
				t.Fatalf("delivery replaced document: path=%q source=%q, want %q %q", a.CurrentPath(), a.CurrentSource(), wantPath, wantSource)
			}
			if a.Conflict != (scenario == "dirty") {
				t.Errorf("conflict = %v in %s", a.Conflict, scenario)
			}
			if scenario == "dirty" && !a.Dirty {
				t.Fatal("delivery cleared local dirty state")
			}
			queue = nil
			w.pollOnce()
			if scenario != "newer disk" && len(queue) != 0 {
				t.Fatal("same change was dispatched twice")
			}
		})
	}
}

func TestNewWatcherOnUI(t *testing.T) {
	path := writeNote(t, t.TempDir(), "note.md", "# Note", 0)
	a := New()
	w := NewWatcherOnUI(a, path, func(func()) {})
	w.Close()
	if w.App != a || w.Path != path || w.Mod.IsZero() {
		t.Fatal("watcher constructor did not capture its initial target")
	}
}

func TestWatcherClose(t *testing.T) {
	w := &Watcher{Stop: make(chan struct{})}
	w.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("closing a stopped watcher panicked: %v", r)
		}
	}()
	w.Close()
	select {
	case <-w.Stop:
	default:
		t.Fatal("watcher stop channel remains open")
	}
}
