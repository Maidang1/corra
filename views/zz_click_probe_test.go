package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

func TestProbeHeaderClicks(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "a.md")
	os.WriteFile(p, []byte("# A\n\nhello\n"), 0o644)
	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	a.OpenNote(p)
	v := New(a, app.NewImageLoader(""))
	tt := ui.NewTester(v.Build, 1280, 820)
	tt.SetScale(2)
	tt.SetTitleBar(ui.TitleBar{Height: 28, Left: 80})
	for _, l := range []string{"切换 浅色 / 深色", "插入任务列表", "新建备忘录 (Cmd+N)", "切换边栏"} {
		r, ok := tt.Find(l)
		t.Logf("%s => %+v %v", l, r, ok)
	}
	tm := a.ThemeMode
	if err := tt.Click("备忘录选项"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("切换 浅色 / 深色"); err != nil {
		t.Fatal(err)
	}
	if a.ThemeMode == tm {
		t.Fatal("theme did not change")
	}
	t.Logf("theme %v -> %v", tm, a.ThemeMode)
	src := a.Source
	if err := tt.Click("插入任务列表"); err != nil {
		t.Fatal(err)
	}
	if a.Source == src {
		t.Fatal("checklist was not inserted")
	}
	t.Logf("source %q -> %q", src, a.Source)
	n := len(a.Lib.Notes)
	if err := tt.Click("新建备忘录 (Cmd+N)"); err != nil {
		t.Fatal(err)
	}
	if len(a.Lib.Notes) != n+1 {
		t.Fatal("note was not created")
	}
	v.Close()
	t.Logf("notes %d -> %d", n, len(a.Lib.Notes))
}
