//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Renders the hover affordances (code copy button and margin handle) to PNGs.
// Run: CORRA_HOVER_SHOT=/tmp/hover.png go test ./nativerich -run TestHoverShot
func TestHoverShot(t *testing.T) {
	path := os.Getenv("CORRA_HOVER_SHOT")
	if path == "" {
		t.Skip("set CORRA_HOVER_SHOT=path.png")
	}
	handlePath := strings.TrimSuffix(path, filepath.Ext(path)) + "-handle" + filepath.Ext(path)
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 760, 520)
		e.SetVisible(true)
		e.Load("```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n\n正文段落，用于观察左边距的块手柄。")
		if !nativeTestHoverCode(e, false) {
			t.Error("copy button not hovered")
		}
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		if !nativeTestHoverBlock(e, 1, false) {
			t.Error("block handle not hovered")
		}
		if !nativeTestScreenshot(e, handlePath) {
			t.Errorf("handle screenshot failed: %s", handlePath)
		}
		dragPath := strings.TrimSuffix(path, filepath.Ext(path)) + "-drag" + filepath.Ext(path)
		if !nativeTestDragPreview(e, 0, 3, true) {
			t.Error("drag preview not prepared")
		}
		if !nativeTestScreenshot(e, dragPath) {
			t.Errorf("drag screenshot failed: %s", dragPath)
		}
		imagePath := strings.TrimSuffix(path, filepath.Ext(path)) + "-image" + filepath.Ext(path)
		img := writeTestPNG(t, 520, 260, "hover")
		defer os.Remove(img)
		e.Load("![示例图](" + img + ")\n\n正文段落，用于观察图片的尺寸条。")
		if !nativeTestHoverImage(e, -1, false) {
			t.Error("image size pill not hovered")
		}
		if !nativeTestScreenshot(e, imagePath) {
			t.Errorf("image screenshot failed: %s", imagePath)
		}
		footnotePath := strings.TrimSuffix(path, filepath.Ext(path)) + "-footnote" + filepath.Ext(path)
		e.Load("正文引用[^1]，悬停可预览定义。\n\n[^1]: 脚注定义：悬停引用即可看到这段说明，不需要等待系统提示。")
		if !nativeTestHoverFootnote(e) {
			t.Error("footnote reference not hovered")
		}
		if !nativeTestFootnoteShot(e, footnotePath) {
			t.Errorf("footnote card screenshot failed: %s", footnotePath)
		}
	})
}
