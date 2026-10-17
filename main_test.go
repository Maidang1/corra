package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/richtext"
)

func TestEmptyExistingLibraryDoesNotReseedWelcome(t *testing.T) {
	oldApp, oldConfig, oldRoot := editorApp, configDir, libraryRoot
	t.Cleanup(func() { editorApp, configDir, libraryRoot = oldApp, oldConfig, oldRoot })
	configDir = t.TempDir()
	editorApp = app.New()
	if err := os.Mkdir(filepath.Join(configDir, "备忘录"), 0o755); err != nil {
		t.Fatal(err)
	}
	setupLibrary()
	openMostRecent()
	if editorApp.Source != "" || len(editorApp.VisibleNotes()) != 0 {
		t.Fatal("empty library displays seed or creates welcome")
	}
}

// TestBuildMenuSmoke builds the full menu bar and walks every item.
func TestBuildMenuSmoke(t *testing.T) {
	oldApp := editorApp
	t.Cleanup(func() { editorApp = oldApp })
	editorApp = app.New()
	menu := buildMenu()
	if menu == nil {
		t.Fatal("no menu")
	}
	var count int
	var walk func(items []*mygo.MenuItem)
	walk = func(items []*mygo.MenuItem) {
		for _, it := range items {
			count++
			walk(it.Submenu)
		}
	}
	walk(menu.Items())
	if count < 40 {
		t.Fatalf("menu too small: %d items", count)
	}
}

func TestWelcomeNoteParsesAndRoundTrips(t *testing.T) {
	doc := render.Parse(app.Welcome)
	if len(doc.Blocks) < 10 {
		t.Fatalf("welcome blocks: %d", len(doc.Blocks))
	}
	rt := richtext.Parse(app.Welcome)
	if got := richtext.Serialize(rt); got != app.Welcome {
		t.Fatalf("welcome round trip differs:\n%q", got)
	}
	if _, err := render.ExportHTML(app.Welcome, "welcome", false); err != nil {
		t.Fatal(err)
	}
	if _, err := render.ExportPDF(nil, doc, "welcome"); err != nil {
		t.Fatal(err)
	}
}

func TestWelcomeUsesMarkdownEditorPositioning(t *testing.T) {
	for _, want := range []string{
		"原生 Markdown 桌面编辑器",
		"## 文档库与编辑区",
		"全部文档",
		"文档修改时间",
	} {
		if !strings.Contains(app.Welcome, want) {
			t.Errorf("welcome does not contain %q", want)
		}
	}
	for _, legacy := range []string{"备忘录", "笔记", "Apple Notes", "## 两栏布局"} {
		if strings.Contains(app.Welcome, legacy) {
			t.Errorf("welcome retains legacy positioning %q", legacy)
		}
	}
}
