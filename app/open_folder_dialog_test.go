package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func nextFolderUpdate(t *testing.T, queued <-chan func()) {
	t.Helper()
	select {
	case fn := <-queued:
		fn()
	case <-time.After(3 * time.Second):
		t.Fatal("folder load did not dispatch completion")
	}
}

func TestFolderLoadRunsOffUIAndAppliesParsedResult(t *testing.T) {
	root := t.TempDir()
	path := writeNote(t, root, "nested/note.md", "# Nested", 0)
	lib := ScanLibrary(root)
	a := New()
	oldRoot := t.TempDir()
	a.LibraryRoot = oldRoot
	saved := false
	f := &FileOps{App: a, BeforeOpen: func() bool { saved = true; return true }}
	queued := make(chan func(), 1)
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	f.startFolderLoad(root, func(fn func()) { queued <- fn }, func(ctx context.Context, path string) (*Library, error) {
		close(started)
		<-release
		return lib, nil
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("scan not started")
	}
	if !a.LibraryLoading || a.LibraryLoadPath != root || a.LibraryRoot != oldRoot || !saved {
		t.Fatal("UI did not enter loading state before scan")
	}
	release <- struct{}{}
	nextFolderUpdate(t, queued)
	if a.LibraryLoading || a.LibraryLoadError != "" || a.LibraryRoot != root || a.CurrentPath() != path || !saved || len(a.Doc.Blocks) == 0 {
		t.Fatal("parsed folder result not adopted")
	}
}

func TestFolderLoadFailureKeepsPreviousLibrary(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	old := a.LibraryRoot
	f := &FileOps{App: a}
	queued := make(chan func(), 1)
	f.startFolderLoad("missing", func(fn func()) { queued <- fn }, func(context.Context, string) (*Library, error) { return nil, errors.New("unreadable folder") })
	nextFolderUpdate(t, queued)
	if a.LibraryLoading || a.LibraryLoadError != "unreadable folder" || a.LibraryRoot != old {
		t.Fatal("load failure changed library or lost error")
	}
}

func TestCancelledFolderLoadCannotReplaceNewerSelection(t *testing.T) {
	a := New()
	old := t.TempDir()
	a.LibraryRoot = old
	f := &FileOps{App: a}
	queued := make(chan func(), 2)
	release := make(chan struct{})
	defer close(release)
	f.startFolderLoad("first", func(fn func()) { queued <- fn }, func(context.Context, string) (*Library, error) { <-release; return &Library{Root: "first"}, nil })
	f.CancelFolderLoad()
	if a.LibraryLoading {
		t.Fatal("cancel left loading state")
	}
	root := t.TempDir()
	f.startFolderLoad(root, func(fn func()) { queued <- fn }, readFolderLibrary)
	nextFolderUpdate(t, queued)
	release <- struct{}{}
	nextFolderUpdate(t, queued)
	if a.LibraryRoot != root || a.LibraryLoading || a.LibraryLoadError != "" {
		t.Fatal("stale completion replaced newer library")
	}
}

func TestFolderScanHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readFolderLibrary(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("scan ignored cancellation: %v", err)
	}
}
