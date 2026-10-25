//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writes page mentions and page cards (an existing and a missing note) to
// PNGs for visual review, in light and dark.
// Run: CORRA_PAGE_SHOT=/tmp/page.png go test ./nativerich -run TestPageShot
func TestPageShot(t *testing.T) {
	path := os.Getenv("CORRA_PAGE_SHOT")
	if path == "" {
		t.Skip("set CORRA_PAGE_SHOT=path.png")
	}
	base := strings.TrimSuffix(path, filepath.Ext(path))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte("# 第四季度计划\n\n目标：发布原生富文本编辑器，补齐颜色、提及与页面卡片。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 900, 420)
		e.SetVisible(true)
		e.SetBaseDir(dir)
		source := "会议记录见 [@第四季度计划](plan.md)，由 [@周报](weekly.md) 汇总。\n\n![第四季度计划](plan.md)\n\n![旧文档](gone.md)\n\n后续段落。"
		for _, dark := range []bool{false, true} {
			suffix := ""
			if dark {
				suffix = "-dark"
			}
			e.SetStyle(1, dark, true)
			e.Load(source)
			if !nativeTestScreenshot(e, base+suffix+filepath.Ext(path)) {
				t.Errorf("screenshot failed")
			}
		}
	})
}
