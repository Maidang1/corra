package app

import (
	"os"
	"path/filepath"
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
	if title != "新备忘录" {
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
	lib := ScanLibrary(root)
	if len(lib.Notes) != 1 || lib.Notes[0].Title != "新备忘录" {
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
