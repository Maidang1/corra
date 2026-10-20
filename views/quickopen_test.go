package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestQuickOpenPalette(t *testing.T) {
	v, tt := notesFixture(t)
	another := filepath.Join(v.App.LibraryRoot, "projects", "another.md")
	if err := os.MkdirAll(filepath.Dir(another), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(another, []byte("# Another Project\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v.App.ScanLibrary()
	tt.Frame()

	// Cmd+Shift+P opens the palette with an autofocused search field.
	v.QuickOpenAction()
	tt.Frame()
	if !tt.HasText("esc 关闭") {
		t.Fatal("quick open field missing")
	}
	// Typing filters by title.
	tt.Type("another")
	tt.Frame()
	if !tt.HasText("Another Project") {
		t.Fatal("filtered note missing")
	}
	// Return opens the first match and dismisses the palette.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v.QuickOpen {
		t.Fatal("palette stayed open after Return")
	}
	if v.App.Path != another {
		t.Fatalf("opened %q, want %q", v.App.Path, another)
	}

	// Escape dismisses without switching notes.
	v.QuickOpenAction()
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if v.QuickOpen {
		t.Fatal("escape kept the palette open")
	}
	if v.App.Path != another {
		t.Fatalf("escape switched notes to %q", v.App.Path)
	}

	// A query without matches explains itself.
	v.QuickOpenAction()
	tt.Frame()
	tt.Type("zzzzz")
	tt.Frame()
	if !tt.HasText("没有匹配的文档") {
		t.Fatal("empty state missing")
	}
}

func TestQuickOpenCommands(t *testing.T) {
	v, tt := notesFixture(t)

	// ">" switches the palette from notes to actions.
	v.QuickOpenAction()
	tt.Frame()
	tt.Type(">")
	tt.Frame()
	if !tt.HasText("新建文档") || !tt.HasText("插入表格") {
		t.Fatal("command list missing")
	}
	// English keywords find Chinese commands.
	tt.Type("theme")
	tt.Frame()
	if !tt.HasText("切换主题") {
		t.Fatal("keyword search missed the theme command")
	}
	// Return runs the highlighted command (reopen for a clean query).
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	v.QuickOpenAction()
	tt.Frame()
	tt.Type(">设置")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v.QuickOpen {
		t.Fatal("palette stayed open after running a command")
	}
	if !v.SettingsOpen {
		t.Fatal("settings command did not open the modal")
	}
}

func TestQuickOpenCommandExport(t *testing.T) {
	v, tt := notesFixture(t)
	called := false
	v.OnExportPDF = func() { called = true }
	v.QuickOpenAction()
	tt.Frame()
	tt.Type(">导出")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !called {
		t.Fatal("export command did not run")
	}
	if v.QuickOpen {
		t.Fatal("palette stayed open after export")
	}

	// A query without a command shows the empty state.
	v.QuickOpenAction()
	tt.Frame()
	tt.Type(">zzzzz")
	tt.Frame()
	if !tt.HasText("没有匹配的命令") {
		t.Fatal("command empty state missing")
	}
}

func TestQuickOpenRecentGroup(t *testing.T) {
	v, tt := notesFixture(t)
	firstPath := v.App.Path
	another := filepath.Join(v.App.LibraryRoot, "another.md")
	if err := os.WriteFile(another, []byte("# Another Project\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v.App.ScanLibrary()
	v.openNote(another)
	tt.Frame()
	if v.App.Path != another {
		t.Fatalf("setup opened %q", v.App.Path)
	}

	// An empty query groups the previous note above the whole library.
	v.QuickOpenAction()
	tt.Frame()
	if !tt.HasText("最近打开") || !tt.HasText("全部文档") {
		t.Fatal("recent section missing")
	}
	// Return opens the recent note first.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v.App.Path != firstPath {
		t.Fatalf("recent open chose %q, want %q", v.App.Path, firstPath)
	}
	// A query shows a flat, ungrouped result list.
	v.QuickOpenAction()
	tt.Frame()
	tt.Type("another")
	tt.Frame()
	if tt.HasText("最近打开") {
		t.Fatal("recent section survived a query")
	}
	if !tt.HasText("Another Project") {
		t.Fatal("filtered result missing")
	}
}

func TestQuickOpenArrowKeys(t *testing.T) {
	v, tt := notesFixture(t)
	first := v.App.Path
	another := filepath.Join(v.App.LibraryRoot, "second.md")
	if err := os.WriteFile(another, []byte("# Second Note\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v.App.ScanLibrary()
	tt.Frame()

	v.QuickOpenAction()
	tt.Frame()
	ordered := v.quickOrdered()
	if len(ordered) < 2 {
		t.Fatalf("only %d rows", len(ordered))
	}
	want := ordered[1].Path
	// Down moves the highlight to the second row.
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if v.QuickIndex != 1 {
		t.Fatalf("down left index at %d", v.QuickIndex)
	}
	// Enter opens the highlighted row, not the first one.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v.App.Path != want {
		t.Fatalf("opened %q, want the second row %q (first was %q)", v.App.Path, want, first)
	}
	// Up from the top wraps to the last row.
	v.QuickOpenAction()
	tt.Frame()
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if v.QuickIndex != len(v.quickOrdered())-1 {
		t.Fatalf("wrap index = %d", v.QuickIndex)
	}
}
