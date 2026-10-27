package views

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

func TestSidebarFolderTreeExpandsDocumentsAndSearch(t *testing.T) {
	v, tt := notesFixture(t)
	path := filepath.Join(v.App.LibraryRoot, "work", "research", "note.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Nested\n\ndeepneedle"), 0o644); err != nil {
		t.Fatal(err)
	}
	v.App.ScanLibrary()
	tt.Frame()
	parent, parentOK := tt.Find("文件夹：work")
	child, childOK := tt.Find("文件夹：work/research")
	if !parentOK || !childOK || child.X <= parent.X || tt.HasText("Nested") {
		t.Fatal("folder paths not presented as a collapsed hierarchy")
	}
	click(t, tt, "展开文件夹：work/research")
	tt.Frame()
	click(t, tt, "文档：Nested")
	if v.App.Path != path || v.App.SelFolder != "work/research" {
		t.Fatal("tree document did not open in its folder")
	}
	click(t, tt, "收起文件夹：work")
	tt.Frame()
	if tt.HasText("Nested") || v.App.Path != path {
		t.Fatal("collapsing tree closed document or left children visible")
	}
	click(t, tt, "搜索文档")
	tt.Type("deepneedle")
	tt.Frame()
	if !tt.HasText("Nested") || !tt.HasText("research") {
		t.Fatal("search did not reveal matching document ancestors")
	}
}

func TestSidebarTOCRevealsNativeHeading(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.Loaded(v.App.Path, "# Top\n\n## Section\n\n### Detail\n\nbody")
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	click(t, tt, "显示目录（TOC）")
	tt.Frame()
	if !v.App.Sidebar || !v.App.Outline || !tt.HasText("Section") {
		t.Fatal("TOC is missing from sidebar")
	}
	saveNavigationScreenshot(t, tt, "toc-sidebar.png")
	click(t, tt, "大纲：Section")
	if e.revealedHeading != 1 {
		t.Fatalf("wrong native heading: %d", e.revealedHeading)
	}
	v.App.MarkSourceDirty("# Updated\n\n## New heading")
	tt.Frame()
	if !tt.HasText("New heading") || tt.HasText("Section") {
		t.Fatal("TOC did not refresh after editing")
	}
	click(t, tt, "文档")
	tt.Frame()
	if v.App.Outline || !tt.HasText("搜索文档") {
		t.Fatal("document tree tab did not restore navigation")
	}
}

func TestSidebarTOCSourceOffsetsIncludeFrontMatter(t *testing.T) {
	v, tt := notesFixture(t)
	source := "---\r\ntitle: Meta\r\n---\r\n\r\n# Top\r\n\r\n## Section\r\n\r\nbody"
	v.App.Loaded(v.App.Path, source)
	tt.Frame()
	v.ToggleOutlineAction()
	tt.Frame()
	click(t, tt, "大纲：Section")
	want := strings.Index(strings.ReplaceAll(source, "\r\n", "\n"), "Section")
	if got := v.edState.Cursor(); got != want {
		t.Fatalf("TOC offset = %d, want %d", got, want)
	}
}

func TestSidebarTreeKeepsTrashAccessible(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.Outline = true
	tt.Frame()
	click(t, tt, "文档")
	tt.Frame()
	click(t, tt, "文件夹：最近删除")
	if v.App.SelFolder != app.TrashName {
		t.Fatal("trash navigation lost")
	}
	click(t, tt, "文档")
	tt.Frame()
	if v.App.SelFolder == app.TrashName || !tt.HasText("Note") {
		t.Fatal("document tab did not return from trash to the document tree")
	}
}

func saveNavigationScreenshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if dir := os.Getenv("CORRA_SCREENSHOTS"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, tt.Image())
		_ = file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
