package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Maidang1/corra/app"
)

// Renders the settings modal over the workspace, in both appearances, for
// visual review. Screenshots only when CORRA_SCREENSHOTS is set.
func TestSettingsShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, mode := range []app.ThemeMode{app.ThemeLight, app.ThemeDark} {
		v, tt := notesFixture(t)
		e := &fakeRichEditor{}
		v.AttachRichEditor(e)
		v.App.ThemeMode = mode
		v.SettingsOpen = true
		tt.Frame()
		name := "settings-light.png"
		if mode == app.ThemeDark {
			name = "settings-dark.png"
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
