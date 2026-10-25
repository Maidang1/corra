package views

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// Renders the quick-open palette for visual review. Screenshots only when
// CORRA_SCREENSHOTS is set.
func TestQuickOpenShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, mode := range []app.ThemeMode{app.ThemeLight, app.ThemeDark} {
		v, tt := notesFixture(t)
		e := &fakeRichEditor{}
		v.AttachRichEditor(e)
		v.App.ThemeMode = mode
		for _, title := range []string{"Product Notes", "Reading List", "Weekly Review"} {
			path := filepath.Join(v.App.LibraryRoot, title+".md")
			if err := os.WriteFile(path, []byte("# "+title+"\n\nbody\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		v.App.ScanLibrary()
		v.openNote(filepath.Join(v.App.LibraryRoot, "Reading List.md"))
		v.openNote(filepath.Join(v.App.LibraryRoot, "Weekly Review.md"))
		tt.Frame()
		v.QuickOpenAction()
		tt.Frame()
		name := "quickopen-light.png"
		if mode == app.ThemeDark {
			name = "quickopen-dark.png"
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
		tt.Type("read")
		tt.Frame()
		filteredName := strings.Replace(name, "quickopen-", "quickopen-filtered-", 1)
		ff, err := os.Create(filepath.Join(dir, filteredName))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(ff, tt.Image()); err != nil {
			t.Fatal(err)
		}
		ff.Close()
		// Command mode shares the palette: ">" lists actions, arrows move.
		v.QuickOpenAction()
		tt.Frame()
		tt.Type(">")
		tt.Frame()
		tt.Key(0, ui.KeyDown)
		tt.Frame()
		cmdName := strings.Replace(name, "quickopen-", "quickopen-commands-", 1)
		cf, err := os.Create(filepath.Join(dir, cmdName))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(cf, tt.Image()); err != nil {
			t.Fatal(err)
		}
		cf.Close()
		v.Close()
	}
}
