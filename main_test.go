package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/richtext"
)

func TestSetupLibraryRestoresSelectedFolder(t *testing.T) {
	oldApp, oldConfig, oldRoot := editorApp, configDir, libraryRoot
	t.Cleanup(func() { editorApp, configDir, libraryRoot = oldApp, oldConfig, oldRoot })
	configDir = t.TempDir()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "nested", "existing.md")
	if err := os.WriteFile(path, []byte("# Existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.SaveSettings(configDir, app.Settings{LibraryRoot: root}); err != nil {
		t.Fatal(err)
	}
	editorApp = app.New()
	setupLibrary()
	loadInitialLibraryForTest(t)
	if libraryRoot != root || editorApp.CurrentPath() != path {
		t.Fatalf("did not restore folder: %s %s", libraryRoot, editorApp.CurrentPath())
	}
	if _, err := os.Stat(filepath.Join(root, "欢迎使用 Corra.md")); !os.IsNotExist(err) {
		t.Fatal("seeded external folder")
	}
}

func TestEmptyExistingLibraryDoesNotReseedWelcome(t *testing.T) {
	oldApp, oldConfig, oldRoot := editorApp, configDir, libraryRoot
	t.Cleanup(func() { editorApp, configDir, libraryRoot = oldApp, oldConfig, oldRoot })
	configDir = t.TempDir()
	editorApp = app.New()
	if err := os.Mkdir(filepath.Join(configDir, "备忘录"), 0o755); err != nil {
		t.Fatal(err)
	}
	setupLibrary()
	loadInitialLibraryForTest(t)
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

func TestTitleBarStyleForPlatform(t *testing.T) {
	tests := []struct {
		goos string
		want mygo.TitleBarStyle
	}{
		{goos: "darwin", want: mygo.TitleBarHiddenInset},
		{goos: "linux", want: mygo.TitleBarDefault},
		{goos: "windows", want: mygo.TitleBarHiddenInset},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			if got := titleBarStyleForPlatform(tt.goos); got != tt.want {
				t.Fatalf("titleBarStyleForPlatform(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
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

func loadInitialLibraryForTest(t *testing.T) {
	t.Helper()
	queued := make(chan func(), 1)
	files := &app.FileOps{App: editorApp, Dispatch: func(fn func()) { queued <- fn }}
	files.OpenFolderPath(libraryRoot)
	select {
	case fn := <-queued:
		fn()
	case <-time.After(3 * time.Second):
		t.Fatal("initial library did not finish loading")
	}
	if editorApp.LibraryLoadError != "" {
		t.Fatal(editorApp.LibraryLoadError)
	}
}
