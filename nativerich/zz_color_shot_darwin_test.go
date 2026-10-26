//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes colored text and the format bar's color palette to PNGs for visual
// review, in light and dark.
// Run: CORRA_COLOR_SHOT=/tmp/color.png go test ./nativerich -run TestColorShot
func TestColorShot(t *testing.T) {
	path := os.Getenv("CORRA_COLOR_SHOT")
	if path == "" {
		t.Skip("set CORRA_COLOR_SHOT=path.png")
	}
	base := strings.TrimSuffix(path, filepath.Ext(path))
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 900, 520)
		e.SetVisible(true)
		var lines []string
		for _, c := range []struct{ name, text, bg string }{
			{"灰色", "#787774", "#f1f1ef"}, {"棕色", "#9f6b53", "#f4eeee"}, {"橙色", "#d9730d", "#fbecdd"},
			{"黄色", "#cb912f", "#fbf3db"}, {"绿色", "#448361", "#edf3ec"}, {"蓝色", "#337ea9", "#e7f3f8"},
			{"紫色", "#9065b0", "#f6f3f9"}, {"粉色", "#c14c8a", "#faf1f5"}, {"红色", "#d44c47", "#fdebec"},
		} {
			lines = append(lines, `<span style="color:`+c.text+`">`+c.name+`文字 text</span> 和 <span style="background-color:`+c.bg+`">`+c.name+`背景 background</span>`)
		}
		source := strings.Join(lines, "\n\n")
		for _, dark := range []bool{false, true} {
			suffix := ""
			if dark {
				suffix = "-dark"
			}
			e.SetStyle(1, dark, true)
			e.Load(source)
			if !nativeTestScreenshot(e, base+suffix+filepath.Ext(path)) {
				t.Errorf("editor screenshot failed")
			}
			nativeTestSelect(e, 0, 4)
			nativeTestAction(e, "formatbar-pick", "palette")
			if !nativeTestFormatBarShot(e, base+"-palette"+suffix+filepath.Ext(path)) {
				t.Errorf("palette screenshot failed")
			}
		}
	})
}
