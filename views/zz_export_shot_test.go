package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Maidang1/corra/app"
)

// Renders the export panel over the workspace, in both appearances, for
// visual review. Screenshots only when CORRA_SCREENSHOTS is set.
func TestExportShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, mode := range []app.ThemeMode{app.ThemeLight, app.ThemeDark} {
		v, tt := notesFixture(t)
		e := &fakeRichEditor{}
		v.AttachRichEditor(e)
		v.App.ThemeMode = mode
		v.PandocPath = func() string { return "" } // disabled cards show the hint
		v.ExportOpen = true
		tt.Frame()
		name := "export-light.png"
		if mode == app.ThemeDark {
			name = "export-dark.png"
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
		v.Close()
	}
}
