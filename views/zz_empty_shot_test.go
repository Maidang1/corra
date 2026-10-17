package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// Run: CORRA_SCREENSHOTS=/tmp/x go test ./views -run TestEmptyShot
func TestEmptyShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	a := app.New()
	a.LibraryRoot = t.TempDir()
	a.ScanLibrary()
	v := New(a, app.NewImageLoader(""))
	v.AttachRichEditor(&fakeRichEditor{})
	tt := ui.NewTester(v.Build, 1280, 820)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 100})
	tt.Frame()
	tt.Frame()
	f, _ := os.Create(filepath.Join(dir, "empty.png"))
	png.Encode(f, tt.Image())
	f.Close()
}
