package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// Renders the whole window with a realistic library, light and dark, sidebar
// open and collapsed. Run: CORRA_SCREENSHOTS=/tmp/x go test ./views -run TestLayoutShot
func TestLayoutShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	root := t.TempDir()
	notes := map[string]string{
		"hello.md":   "# hello world\n\nthis is a time\n\nbug hello\n",
		"welcome.md": "# 欢迎使用 Corra\n\n一个用 Go 写的原生 Markdown 备忘录，采用 Apple Notes 风格。\n",
		"工作/周报.md":   "# 本周周报\n\n完成了编辑器性能优化，打字延迟降低一个数量级。\n",
		"工作/会议纪要.md": "# 会议纪要\n\n讨论 Typora 对齐清单。\n",
		"生活/购物清单.md": "# 购物清单\n\n- [ ] 牛奶\n- [ ] 面包\n",
		"empty.md":   "# 空白备忘录\n",
	}
	i := 0
	for name, body := range notes {
		p := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
		ts := time.Now().Add(-time.Duration(i) * 30 * time.Hour)
		os.Chtimes(p, ts, ts)
		i++
	}
	for _, dark := range []bool{false, true} {
		for _, sidebar := range []bool{true, false} {
			a := app.New()
			a.LibraryRoot = root
			a.ScanLibrary()
			a.OpenNote(filepath.Join(root, "hello.md"))
			if dark {
				a.ThemeMode = app.ThemeDark
			}
			a.Sidebar = sidebar
			v := New(a, app.NewImageLoader(""))
			v.AttachRichEditor(&fakeRichEditor{})
			w := 1280
			if os.Getenv("CORRA_LAYOUT_NARROW") != "" {
				w = 860
			}
			tt := ui.NewTester(v.Build, w, 820)
			tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 100})
			tt.Frame()
			tt.Frame()
			if os.Getenv("CORRA_LAYOUT_EXTRA") != "" {
				a.Outline = true
				v.FolderDialog = "新建文件夹"
			}
			if os.Getenv("CORRA_LAYOUT_SOURCE") != "" {
				a.Mode = app.ModeEdit
			}
			tt.Frame()
			tt.Frame()
			name := "layout"
			if dark {
				name += "-dark"
			}
			if !sidebar {
				name += "-collapsed"
			}
			if os.Getenv("CORRA_LAYOUT_EXTRA") != "" {
				name += "-extra"
			}
			if os.Getenv("CORRA_LAYOUT_NARROW") != "" {
				name += "-narrow"
			}
			if os.Getenv("CORRA_LAYOUT_SOURCE") != "" {
				name += "-source"
			}
			f, _ := os.Create(filepath.Join(dir, name+".png"))
			png.Encode(f, tt.Image())
			f.Close()
		}
	}
}
