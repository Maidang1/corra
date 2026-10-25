//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Renders a hovered table with its add-row/add-column buttons.
// Run: CORRA_TABLE_SHOT=/tmp/table.png go test ./nativerich -run TestTableShot
func TestTableShot(t *testing.T) {
	path := os.Getenv("CORRA_TABLE_SHOT")
	if path == "" {
		t.Skip("set CORRA_TABLE_SHOT=path.png")
	}
	dark := strings.TrimSuffix(path, filepath.Ext(path)) + "-dark" + filepath.Ext(path)
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
		e.Load("| 项目 | 状态 | 负责人 |\n| --- | --- | --- |\n| 编辑器 | 进行中 | Ming |\n| 导出 | 完成 | Lin |")
		if !nativeTestHoverTable(e) {
			t.Error("table not hovered")
		}
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		e.SetStyle(1, true, true)
		e.Load("| 项目 | 状态 |\n| --- | --- |\n| 编辑器 | 进行中 |")
		nativeTestHoverTable(e)
		if !nativeTestScreenshot(e, dark) {
			t.Errorf("dark screenshot failed: %s", dark)
		}
	})
}
