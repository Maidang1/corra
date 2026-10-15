//go:build darwin && cgo

package nativerich

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeImageClickEditsSourceInPlace(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "photo.png"))
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 60; x++ {
			img.Set(x, y, color.RGBA{R: 80, G: 140, B: 180, A: 255})
		}
	}
	if err = png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		e.SetBaseDir(dir)
		e.Load("![photo](photo.png)")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "click", "")
		if !nativeTestHasTraits(e, "source") {
			t.Error("click did not reveal editable image source")
			return
		}
		nativeTestSelect(e, 2, 5)
		nativeTestAction(e, "type", "new")
		nativeTestAction(e, "blur", "")
		got, _, err := e.Snapshot()
		if err != nil || got != "![new](photo.png)" || !nativeTestHasTraits(e, "image") {
			t.Errorf("image source edit/render: %q %v", got, err)
		}
	})
}

func TestNativeStructureScreenshot(t *testing.T) {
	path := os.Getenv("CORRA_STRUCTURE_SCREENSHOT")
	if path == "" {
		t.Skip("set CORRA_STRUCTURE_SCREENSHOT for visual inspection")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 700)
		e.SetVisible(true)
		e.Load("# Native editing\n\n- Parent\n  - Nested **bold**\n    - [x] Completed\n  - [ ] Next task\n- Sibling\n\n| Feature | Status |\n| :--- | :---: |\n| Nested lists | Ready |\n| Table commands | Ready |\n\n==Highlight== and ^superscript^\n\n```go\nfunc main() {\n\tprintln(\"native\")\n}\n```")
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
	})
}
