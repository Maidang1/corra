//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Temporary typography review harness (loop task): renders a representative
// document to a PNG so spacing/size changes can be compared visually.
// Run: CORRA_TYPO_SHOT=/tmp/typo.png go test ./nativerich -run TestTypographyShot
const typoDoc = `# 欢迎使用 Corra

一个纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑。这一行故意写长一点，用来观察正文的行长、行高与段落在宽栏里的阅读节奏是否舒适。

## 文档库与编辑区

- **文档库**：管理库根目录下的 Markdown 文件夹
- **文档列表**：按修改日期分组，顶部可全文搜索
- **编辑区**：顶部显示文档修改时间；「视图」菜单可开启大纲面板与字数统计

## 编辑与预览

macOS 默认使用**原生富文本编辑**，标题、正文和格式直接可见，修改自动保存。
Cmd+E 切换 Markdown 源码，Cmd+Z 撤销；源码模式有语法高亮、多光标、自动配对与查找替换（Cmd+F）。

### 三级标题的样子

#### 四级标题的样子

普通正文紧跟在标题之后，用来检验标题的段前段后间距是否形成清晰的章节节奏。` + "\n\n```go\nfunc main() {\n\tfmt.Println(\"hello\") // 注释\n}\n```" + `

- 粗体 **重点**、斜体 *强调*、删除线 ~~划掉~~、高亮 ==荧光==
- 数学公式：行内 $E=mc^2$ 与块级公式；图表：mermaid、d2、goat
- [ ] 试试在预览里点击复选框
- [x] 已完成

> 文件菜单导出 PDF / HTML / Word；粘贴或拖入图片自动保存为附件。引用块应该有一条左侧竖线与更安静的文本颜色。

1. 有序列表第一项
2. 有序列表第二项

` + "```chart\ntype: bar\ntitle: 示例\nlabels: [一, 二, 三, 四]\nseries:\n  - name: 销量\n    data: [3, 7, 5, 9]\n```" + `

文末段落，观察与上图之间的留白。
`

func TestTypographyShot(t *testing.T) {
	path := os.Getenv("CORRA_TYPO_SHOT")
	if path == "" {
		t.Skip("set CORRA_TYPO_SHOT=path.png")
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
		e.SetFrame(0, 0, 900, 1640)
		e.SetVisible(true)
		e.Load(typoDoc)
		nativeTestSelect(e, 0, 0)
		e.SetStyle(1, false, true)
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
		darkPath := strings.TrimSuffix(path, filepath.Ext(path)) + "-dark.png"
		e.SetStyle(1, true, true)
		if !nativeTestScreenshot(e, darkPath) {
			t.Errorf("screenshot failed: %s", darkPath)
		}
	})
}
