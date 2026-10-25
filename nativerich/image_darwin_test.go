//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

type nativeImageFigure struct {
	URL, Spec     string
	Width, Height float64
}

func readNativeImages(t *testing.T, e *Editor) []nativeImageFigure {
	t.Helper()
	var figures []nativeImageFigure
	if err := json.Unmarshal([]byte(nativeTestImages(e)), &figures); err != nil {
		t.Error(err)
	}
	return figures
}

// writeTestPNG drops a solid PNG of w×h pixels into the job's temp directory and
// returns its path, so a figure can be loaded without a network.
func writeTestPNG(t *testing.T, w, h int, name string) string {
	t.Helper()
	path := filepath.Join(os.TempDir(), "corra-native-image-"+name+".png")
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0x30, 0x60, 0xa0, 0xff})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// The size suffix on an image destination is part of the document: the native
// figure has to be drawn at that size, and the source has to keep it.
func TestNativeImageSizeRoundTrip(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "sized")
		defer os.Remove(path)
		source := "![alt](" + path + " =200x100)"
		e.Load(source)
		figures := readNativeImages(t, e)
		if len(figures) != 1 {
			t.Errorf("image figure missing: %+v", figures)
			return
		}
		if figures[0].Spec != "200x100" || figures[0].Width != 200 || figures[0].Height != 100 {
			t.Errorf("size suffix ignored: %+v", figures[0])
		}
		if got, dirty, err := e.Snapshot(); got != source || dirty || err != nil {
			t.Errorf("load changed source: %q dirty=%v err=%v", got, dirty, err)
		}
	})
}

// A percentage scales the natural size, and an absent suffix draws it at the
// natural size clamped to the column.
func TestNativeImageSizePercentAndNatural(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "percent")
		defer os.Remove(path)
		e.Load("![alt](" + path + " =50%)")
		figures := readNativeImages(t, e)
		if len(figures) != 1 || figures[0].Width != 200 || figures[0].Height != 150 {
			t.Errorf("percentage ignored: %+v", figures)
			return
		}
		e.Load("![alt](" + path + ")")
		figures = readNativeImages(t, e)
		if len(figures) != 1 || figures[0].Width != 400 || figures[0].Height != 300 {
			t.Errorf("natural size wrong: %+v", figures)
		}
	})
}

// The resize command rewrites the suffix in the document and redraws the figure,
// so the preview and the exports see the same size the reader chose. Undo takes
// both back together.
func TestNativeImageSizeCommand(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "command")
		defer os.Remove(path)
		original := "![alt](" + path + ")"
		e.Load(original)
		nativeTestSelect(e, 0, 0)
		for _, tc := range []struct{ spec, want string }{
			{"200x100", "![alt](" + path + " =200x100)"},
			{"50%", "![alt](" + path + " =50%)"},
			{"x80", "![alt](" + path + " =x80)"},
			{"", original},
		} {
			e.Format("image-size:" + tc.spec)
			if got, _, _ := e.Snapshot(); got != tc.want {
				t.Errorf("size %q wrote %q, want %q", tc.spec, got, tc.want)
			}
		}
		figures := readNativeImages(t, e)
		if len(figures) != 1 || figures[0].Width != 400 || figures[0].Spec != "" {
			t.Errorf("removal did not restore the natural size: %+v", figures)
		}
		// A second size replaces the first instead of stacking suffixes.
		e.Format("image-size:300x")
		if got, _, _ := e.Snapshot(); got != "![alt]("+path+" =300x)" {
			t.Errorf("replacement stacked the suffix: %q", got)
		}
		if figures = readNativeImages(t, e); len(figures) != 1 || figures[0].Width != 300 || figures[0].Height != 225 {
			t.Errorf("width-only size ignored the aspect ratio: %+v", figures)
		}
		// A width the column cannot hold is scaled down to it, keeping the
		// ratio, rather than running off the edge of the text view.
		e.Format("image-size:4000x")
		if figures = readNativeImages(t, e); len(figures) != 1 || figures[0].Width >= 4000 || figures[0].Width*3 != figures[0].Height*4 {
			t.Errorf("oversized figure was not clamped: %+v", figures)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("undo lost the original source: %q", got)
		}
		if figures = readNativeImages(t, e); len(figures) != 1 || figures[0].Width != 400 {
			t.Errorf("undo did not restore the figure: %+v", figures)
		}
	})
}

// A destination that carries a title, or spaces inside <...>, keeps both when
// the size changes; the URL itself is unchanged.
func TestNativeImageSizeKeepsDestination(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "title")
		defer os.Remove(path)
		e.Load("![alt](" + path + " \"a title\" =100x100)")
		figures := readNativeImages(t, e)
		if len(figures) != 1 || figures[0].Width != 100 {
			t.Errorf("titled figure not sized: %+v", figures)
			return
		}
		if figures[0].URL != path {
			t.Errorf("URL lost the destination: %q", figures[0].URL)
		}
		e.Format("image-size:200x")
		if got, _, _ := e.Snapshot(); got != "![alt]("+path+" \"a title\" =200x)" {
			t.Errorf("title lost: %q", got)
		}
	})
}

// Replacing a figure swaps its file but keeps everything around the URL: the
// alt text, the quoted title and the size suffix, which now scales the new
// picture. Undo brings the original file back.
func TestNativeImageReplaceKeepsDestination(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		oldPath := writeTestPNG(t, 400, 300, "replace-old")
		defer os.Remove(oldPath)
		newPath := writeTestPNG(t, 200, 100, "replace-new")
		defer os.Remove(newPath)
		original := "![alt](" + oldPath + " \"a title\" =50%)"
		e.Load(original)
		if !nativeTestImageReplace(e, newPath) {
			t.Errorf("replace failed")
			return
		}
		want := "![alt](" + newPath + " \"a title\" =50%)"
		if got, _, _ := e.Snapshot(); got != want {
			t.Errorf("replace wrote %q, want %q", got, want)
		}
		figures := readNativeImages(t, e)
		if len(figures) != 1 || figures[0].URL != newPath || figures[0].Spec != "50%" || figures[0].Width != 100 || figures[0].Height != 50 {
			t.Errorf("replaced figure not redrawn from the new file: %+v", figures)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("undo lost the original file: %q", got)
		}
		if figures = readNativeImages(t, e); len(figures) != 1 || figures[0].URL != oldPath {
			t.Errorf("undo did not restore the figure: %+v", figures)
		}
	})
}

// A no-op replacement (same URL, or no figure at the caret) changes nothing
// and reports failure, so the panel flow can tell apart a cancel.
func TestNativeImageReplaceRejectsNoop(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "replace-noop")
		defer os.Remove(path)
		original := "![alt](" + path + " =200x)"
		e.Load(original)
		if nativeTestImageReplace(e, path) {
			t.Errorf("same-URL replace reported a change")
		}
		if got, dirty, _ := e.Snapshot(); got != original || dirty {
			t.Errorf("no-op replace touched the document: %q dirty=%v", got, dirty)
		}
		e.Load("just text")
		if nativeTestImageReplace(e, path) {
			t.Errorf("replace ran without a figure")
		}
		if got, _, _ := e.Snapshot(); got != "just text" {
			t.Errorf("replace changed prose: %q", got)
		}
	})
}

// A replacement file inside the document's folder is written as a relative
// path, so the note keeps working when the folder moves.
func TestNativeImageReplaceWritesRelativePath(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		oldPath := writeTestPNG(t, 400, 300, "rel-old")
		defer os.Remove(oldPath)
		newPath := writeTestPNG(t, 200, 100, "rel-new")
		defer os.Remove(newPath)
		e.SetBaseDir(filepath.Dir(newPath))
		e.Load("![alt](" + oldPath + ")")
		if !nativeTestImageReplace(e, newPath) {
			t.Errorf("replace failed")
			return
		}
		want := "![alt](" + filepath.Base(newPath) + ")"
		if got, _, _ := e.Snapshot(); got != want {
			t.Errorf("replace wrote %q, want relative %q", got, want)
		}
		if figures := readNativeImages(t, e); len(figures) != 1 || figures[0].Width != 200 {
			t.Errorf("relative figure not redrawn: %+v", figures)
		}
	})
}

// A path holding a space only loads inside angle brackets, and the brackets
// have to come back when the size is rewritten.
func TestNativeImageSizeBracketedDestination(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		path := writeTestPNG(t, 400, 300, "with space")
		defer os.Remove(path)
		e.Load("![alt](<" + path + "> =100x50)")
		figures := readNativeImages(t, e)
		if len(figures) != 1 || figures[0].Spec != "100x50" || figures[0].Width != 100 || figures[0].Height != 50 {
			t.Errorf("bracketed figure not sized: %+v", figures)
			return
		}
		if figures[0].URL != path {
			t.Errorf("URL lost the destination: %q", figures[0].URL)
		}
		e.Format("image-size:200x")
		want := "![alt](<" + path + "> =200x)"
		if got, _, _ := e.Snapshot(); got != want {
			t.Errorf("brackets lost: %q != %q", got, want)
		}
		if figures = readNativeImages(t, e); len(figures) != 1 || figures[0].Width != 200 || figures[0].Height != 150 {
			t.Errorf("resized bracketed figure: %+v", figures)
		}
	})
}
