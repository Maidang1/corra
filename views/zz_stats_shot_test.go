package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Maidang1/corra/app"
)

// Renders the expanded document-stats strip for visual review. Screenshots
// only when CORRA_SCREENSHOTS is set.
func TestStatsShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, mode := range []app.ThemeMode{app.ThemeLight, app.ThemeDark} {
		v, tt := notesFixture(t)
		v.App.ThemeMode = mode
		v.App.WordCount = true
		v.App.Source = "# 项目计划\n\n## 目标\n\n- [ ] 完成编辑器\n- [x] 表格控制\n\n参考 [文档](notes.md) 与 ![图](img.png)。\n\n```go\nfunc main() {}\n```\n"
		v.StatsOpen = true
		tt.Frame()
		name := "stats-light.png"
		if mode == app.ThemeDark {
			name = "stats-dark.png"
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
