package views

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// docShotSample exercises every block and mark the document editor draws.
const docShotSample = `# 文档编辑器样式

正文段落，包含 **粗体**、*斜体*、~~删除线~~、` + "`inline code`" + ` 与 [链接](https://example.com)，以及中英文混排 mixed text 的长句子，用来观察换行后的行距是否舒适。

## 二级标题

第二段正文。==高亮== 与 <kbd>⌘</kbd>。

### 三级标题

- 无序列表第一项
- 第二项，文字较长一些，用来观察换行后与项目符号的对齐情况是否正确
  - 嵌套项
- 第三项

1. 有序列表
2. 第二项

- [ ] 待办事项
- [x] 已完成事项

> 这是一段引用，引用的文字应该比正文更安静。
> 第二行引用。

> [!NOTE]
> 这是一个提示块。

> [!WARNING]
> 这是一个警告块。

` + "```go" + `
func main() {
	fmt.Println("hello")
}
` + "```" + `

| 名称 | 说明 |
| --- | --- |
| 表头下第一行 | 内容 |
| 第二行 | 更多内容 |

---

#### 四级标题

$$
E = mc^2
$$

脚注引用[^1]。

[^1]: 这是脚注内容。
`

// Renders the document editor with every block type, light and dark.
// Run: CORRA_SCREENSHOTS=/tmp/x go test ./views -run TestDocRichShot
func TestDocRichShot(t *testing.T) {
	dir := os.Getenv("CORRA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set CORRA_SCREENSHOTS")
	}
	for _, th := range []*ui.Theme{light, dark} {
		d := NewDocRich(nil, nil, nil)
		d.Load(docShotSample)
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(th)
			ui.Box(c).Fill().Background(th.Background).Children(func() { d.Render(c) })
		}, 900, 2000)
		tt.Frame()
		tt.Frame()
		tt.Frame()
		name := "docrich"
		if th.Dark {
			name += "-dark"
		}
		saveShot(t, filepath.Join(dir, name+".png"), tt)

		// The popups: "/" at the end of a paragraph opens the slash menu;
		// a selection shows the format bar.
		tt.ClickAt(860, 240)
		tt.Frame()
		tt.Type(" /")
		tt.Frame()
		saveShot(t, filepath.Join(dir, name+"-slash.png"), tt)
		tt.Key(0, ui.KeyEscape)
		tt.Key(ui.Shift, ui.KeyHome)
		tt.Frame()
		saveShot(t, filepath.Join(dir, name+"-format.png"), tt)
	}
}

func saveShot(t *testing.T, path string, tt *ui.Tester) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
