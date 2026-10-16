package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Maidang1/corra/app"
)

// Renders the mode switcher's two active states (文稿 / 源码) for visual
// review. Screenshots only when CORRA_SCREENSHOTS is set.
func TestModeSwitcherShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, mode := range []app.Mode{app.ModeRich, app.ModeEdit} {
		v, tt := notesFixture(t)
		e := &fakeRichEditor{}
		v.AttachRichEditor(e)
		tt.Frame()
		v.App.Mode = mode
		tt.Frame()
		name := "switcher-rich.png"
		if mode == app.ModeEdit {
			name = "switcher-edit.png"
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}
