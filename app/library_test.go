package app

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func writeNote(t *testing.T, root, name, src string, age time.Duration) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Now().Add(-age)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanLibrary(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "b.md", "# B\n\nsecond", 2*time.Hour)
	writeNote(t, root, "a.md", "# A\n\nfirst", time.Hour)
	writeNote(t, root, "work/c.md", "# C\n\nthird", 3*time.Hour)
	// Not a markdown file: ignored.
	if err := os.WriteFile(filepath.Join(root, "x.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	lib := ScanLibrary(root)
	if len(lib.Notes) != 3 {
		t.Fatalf("notes = %d, want 3", len(lib.Notes))
	}
	// Most recent first.
	if got := lib.Notes[0].Title; got != "A" {
		t.Fatalf("first note = %q, want A", got)
	}
	folders := lib.Folders()
	if len(folders) != 1 || folders[0] != "work" {
		t.Fatalf("folders = %v, want [work]", folders)
	}
	if lib.Notes[0].Folder != "" || lib.Notes[2].Folder != "work" {
		t.Fatalf("folders of notes wrong: %+v", lib.Notes)
	}
	t.Run("empty folders and symlinks", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "empty", "child"), 0o755); err != nil {
			t.Fatal(err)
		}
		outside := writeNote(t, t.TempDir(), "outside.md", "# Private", 0)
		if err := os.Symlink(outside, filepath.Join(root, "link.md")); err != nil {
			t.Fatal(err)
		}
		lib := ScanLibrary(root)
		if got := lib.Folders(); !reflect.DeepEqual(got, []string{"empty", "empty/child"}) {
			t.Errorf("folders = %v, want empty folders indexed", got)
		}
		if len(lib.Notes) != 0 {
			t.Errorf("symlink note leaked into library: %+v", lib.Notes)
		}
	})
}

func TestScanLibraryIgnoresHiddenEntries(t *testing.T) {
	// An explicitly opened root may itself be hidden.
	root := filepath.Join(t.TempDir(), ".library")
	visible := writeNote(t, root, "work/visible.md", "# Visible", 0)
	trashed := writeNote(t, root, ".trash/deleted.md", "# Deleted", 0)
	for _, name := range []string{
		".hidden.md", ".git/nested/internal.md", "work/.draft.md",
		"work/.private/nested.md", "work/.trash/nested.md",
		".trash/.hidden.md", ".trash/.private/nested.md",
	} {
		writeNote(t, root, name, "# Hidden needle", 0)
	}
	if err := os.MkdirAll(filepath.Join(root, ".empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	lib := ScanLibrary(root)
	if len(lib.Notes) != 1 || lib.Notes[0].Path != visible {
		t.Fatalf("visible notes = %+v, want only %s", lib.Notes, visible)
	}
	if got := lib.Folders(); !reflect.DeepEqual(got, []string{"work"}) {
		t.Fatalf("folders = %v, want [work]", got)
	}
	if got := lib.Filter("", "needle"); len(got) != 0 {
		t.Fatalf("hidden documents leaked into search: %+v", got)
	}
	if got := lib.Filter(TrashName, ""); len(got) != 1 || got[0].Path != trashed {
		t.Fatalf("trash notes = %+v, want only %s", got, trashed)
	}
	if got := TrashCount(root); got != 1 {
		t.Fatalf("trash count = %d, want 1", got)
	}
}

func TestFilter(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "a.md", "# 苹果\n\n红色水果", time.Hour)
	writeNote(t, root, "work/b.md", "# 香蕉\n\n黄色水果", 2*time.Hour)

	lib := ScanLibrary(root)
	if got := len(lib.Filter("", "")); got != 2 {
		t.Fatalf("all = %d, want 2", got)
	}
	if got := len(lib.Filter("work", "")); got != 1 {
		t.Fatalf("work = %d, want 1", got)
	}
	if got := len(lib.Filter("", "香蕉")); got != 1 {
		t.Fatalf("search 香蕉 = %d, want 1", got)
	}
	if got := len(lib.Filter("", "红色")); got != 1 {
		t.Fatalf("search 红色 = %d, want 1", got)
	}
	if got := len(lib.Filter("", "西瓜")); got != 0 {
		t.Fatalf("search 西瓜 = %d, want 0", got)
	}
	t.Run("full source", func(t *testing.T) {
		root := t.TempDir()
		writeNote(t, root, "long.md", "# Title\n\n"+strings.Repeat("preview ", 30)+"\n\n```\nDeepNeedle\n```", 0)
		lib := ScanLibrary(root)
		if got := len(lib.Filter("", " deepneedle ")); got != 1 {
			t.Errorf("full-source matches = %d, want 1", got)
		}
	})
}

func TestAppVisibleNotes(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "active.md", "# Active", 0)
	trashed := writeNote(t, root, ".trash/deleted.md", "# Deleted\n\n"+strings.Repeat("body ", 30)+"unique-token", time.Hour)
	a := New()
	a.LibraryRoot = root
	a.SelFolder = TrashName
	a.Query = "unique-token"
	a.ScanLibrary()
	if got := a.VisibleNotes(); len(got) != 1 || got[0].Path != trashed {
		t.Fatalf("trash visible notes = %+v, want %s", got, trashed)
	}
	if len(a.Lib.Notes) != 1 || a.Lib.Notes[0].Title != "Active" {
		t.Fatalf("active library contains trash: %+v", a.Lib.Notes)
	}
}

func TestAppNewNote(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	a.SelFolder = TrashName
	a.Query = "unmatched search"
	a.Mode = ModeRich
	if err := a.NewNote(); err != nil {
		t.Fatal(err)
	}
	if a.SelFolder != "" || a.Query != "" || a.Mode != ModeEdit || len(a.VisibleNotes()) != 1 {
		t.Fatalf("new note stays hidden: folder=%q query=%q mode=%v visible=%d", a.SelFolder, a.Query, a.Mode, len(a.VisibleNotes()))
	}
}

func TestCreateNote(t *testing.T) {
	t.Run("reject unsafe folders", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		for _, folder := range []string{"../escape", ".trash", "a/../../escape", "link", outside} {
			if path, err := CreateNote(root, folder); err == nil {
				t.Errorf("unsafe folder %q created %s", folder, path)
			}
		}
	})
	t.Run("repeated creation never overwrites", func(t *testing.T) {
		root := t.TempDir()
		seen := map[string]bool{}
		for range 30 {
			path, err := CreateNote(root, "")
			if err != nil {
				t.Fatal(err)
			}
			if seen[path] {
				t.Fatalf("overwrote %s", path)
			}
			seen[path] = true
		}
	})
}

func TestDeleteNote(t *testing.T) {
	t.Run("reject outside and symlink sources", func(t *testing.T) {
		root := t.TempDir()
		outside := writeNote(t, t.TempDir(), "outside.md", "# Keep", 0)
		if err := DeleteNote(root, outside); err == nil {
			t.Error("deleted outside the library")
		}
		link := filepath.Join(root, "link.md")
		if err := os.Symlink(outside, link); err != nil {
			t.Fatal(err)
		}
		if err := DeleteNote(root, link); err == nil {
			t.Error("deleted a symlink note")
		}
	})
	t.Run("reject symlink trash", func(t *testing.T) {
		root := t.TempDir()
		path := writeNote(t, root, "keep.md", "# Keep", 0)
		if err := os.Symlink(t.TempDir(), TrashDir(root)); err != nil {
			t.Fatal(err)
		}
		if err := DeleteNote(root, path); err == nil {
			t.Fatal("wrote through symlink trash")
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatal("source was lost", err)
		}
	})
}

func TestGroupByDate(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.Local)
	mk := func(daysAgo int) Note {
		return Note{ModTime: now.AddDate(0, 0, -daysAgo)}
	}
	groups := GroupByDate([]Note{mk(0), mk(1), mk(4), mk(20), mk(35), mk(0)}, now)
	labels := []string{}
	for _, g := range groups {
		labels = append(labels, g.Label)
	}
	want := []string{"今天", "昨天", "最近 7 天", "最近 30 天", "更早"}
	if len(labels) != len(want) {
		t.Fatalf("groups = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("groups = %v, want %v", labels, want)
		}
	}
	if len(groups[0].Notes) != 2 || len(groups[4].Notes) != 1 {
		t.Fatalf("bucket counts wrong: %v", groups)
	}
}

func TestNoteMeta(t *testing.T) {
	title, snippet := NoteMeta("# 标题\n\n第一段 **加粗** 文字。\n\n第二段。")
	if title != "标题" {
		t.Fatalf("title = %q", title)
	}
	if snippet == "" || strings.Contains(snippet, "#") || strings.Contains(snippet, "**") {
		t.Fatalf("snippet = %q", snippet)
	}

	title, _ = NoteMeta("   \n\n普通段落开头\n\n# 后出现的标题")
	if title != "普通段落开头" {
		t.Fatalf("title = %q, want 普通段落开头", title)
	}

	title, _ = NoteMeta("")
	if title != "未命名文档" {
		t.Fatalf("empty title = %q", title)
	}

	_, snippet = NoteMeta("# t\n\n- [x] 已完成任务\n- 列表项")
	if strings.Contains(snippet, "[x]") || strings.Contains(snippet, "- ") {
		t.Fatalf("snippet keeps markers: %q", snippet)
	}

	_, snippet = NoteMeta("# t\n\n2026/10/5 的申請參考編號尚未完成")
	if strings.Contains(snippet, "2026") || strings.HasPrefix(snippet, "/") {
		t.Fatalf("snippet keeps date prefix: %q", snippet)
	}
}

func TestCreateAndDeleteNote(t *testing.T) {
	root := t.TempDir()
	p, err := CreateNote(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("note not created: %v", err)
	}
	namePattern := regexp.MustCompile(`^文档 \d{4}-\d{2}-\d{2} \d{2}\.\d{2}\.\d{2}\.md$`)
	if got := filepath.Base(p); !namePattern.MatchString(got) {
		t.Fatalf("new document name = %q", got)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "# 新文档\n\n"; got != want {
		t.Fatalf("new document content = %q, want %q", got, want)
	}
	lib := ScanLibrary(root)
	if len(lib.Notes) != 1 || lib.Notes[0].Title != "新文档" {
		t.Fatalf("notes = %+v", lib.Notes)
	}
	// In a folder.
	p2, err := CreateNote(root, "work")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(filepath.Dir(p2)) != filepath.ToSlash(filepath.Join(root, "work")) {
		t.Fatalf("folder note at %s", p2)
	}
	if err := DeleteNote(root, p); err != nil {
		t.Fatal(err)
	}
	if err := DeleteNote(root, p2); err != nil {
		t.Fatal(err)
	}
	if got := len(ScanLibrary(root).Notes); got != 0 {
		t.Fatalf("notes left = %d", got)
	}
	// Deleted notes land in the trash, not in oblivion.
	if got := TrashCount(root); got != 2 {
		t.Fatalf("trash = %d, want 2", got)
	}
	// And a trash note deletes permanently.
	trashed, err := CreateNote(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := DeleteNote(root, trashed); err != nil {
		t.Fatal(err)
	}
	if err := DeleteNote(root, TrashDir(root)+string(filepath.Separator)+filepath.Base(trashed)); err != nil {
		t.Fatal(err)
	}
	if got := TrashCount(root); got != 2 {
		t.Fatalf("trash after purge = %d, want 2", got)
	}
}

func TestAppNoteFlow(t *testing.T) {
	root := t.TempDir()
	a := New()
	a.LibraryRoot = root

	if err := a.NewNote(); err != nil {
		t.Fatal(err)
	}
	if a.SelNote == "" || a.CurrentPath() == "" {
		t.Fatal("new note not opened")
	}
	if got := len(a.VisibleNotes()); got != 1 {
		t.Fatalf("visible = %d, want 1", got)
	}

	// Switch folder, create there.
	a.SelFolder = "work"
	if err := a.NewNote(); err != nil {
		t.Fatal(err)
	}
	if got := len(a.VisibleNotes()); got != 1 {
		t.Fatalf("visible in work = %d, want 1", got)
	}
	a.SelFolder = ""
	if got := len(a.VisibleNotes()); got != 2 {
		t.Fatalf("visible all = %d, want 2", got)
	}

	// Delete leaves one note open.
	if err := a.DeleteSelectedNote(); err != nil {
		t.Fatal(err)
	}
	if got := len(ScanLibrary(root).Notes); got != 1 {
		t.Fatalf("notes on disk = %d, want 1", got)
	}
	if a.CurrentPath() == "" {
		t.Fatal("no note opened after delete")
	}
}

func TestNoteMetaSkipsYAMLProperties(t *testing.T) {
	title, snippet := NoteMeta("---\ntitle: 图解 Transformer\nauthor: Jay Alammar\nsource: https://example.com\n---\n\n# 图解 Transformer\n\n正文摘要。")
	if title != "图解 Transformer" || snippet != "正文摘要。" {
		t.Fatalf("metadata leaked into title or preview: %q %q", title, snippet)
	}
}

func TestLibraryFolderFilterIncludesDescendants(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "work/research/note.md", "# Nested", 0)
	if got := len(ScanLibrary(root).Filter("work", "")); got != 1 {
		t.Fatalf("parent folder hides nested documents: %d", got)
	}
}
