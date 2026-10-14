package views

import (
	"github.com/egoist/mygo/ui"
)

// toolbar renders the formatting toolbar.
func (v *View) toolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(2).Padding(6, 10).
		Background(t.Surface).Children(func() {
		tb := func(label, tip string, fn func()) {
			ui.Button(c, label).FontSize(12).Padding(4, 8).Tooltip(tip).
				OnClick(fn)
		}
		tb("B", "加粗 (Ctrl+B)", func() { v.applyWrap("**") })
		tb("I", "斜体 (Ctrl+I)", func() { v.applyWrap("*") })
		tb("S", "删除线", func() { v.applyWrap("~~") })
		tb("`", "行内代码", func() { v.applyWrap("`") })
		tb("H", "标题", func() { v.applyHeading() })
		tb("•", "无序列表", func() { v.applyList(false) })
		tb("1.", "有序列表", func() { v.applyList(true) })
		tb("❝", "引用", func() { v.applyQuote() })
		tb("🔗", "链接", func() { v.applyLink() })
		tb("⊞", "表格", func() { v.insertAtEnd(v.appendTable) })
		tb("</>", "代码块", func() { v.insertAtEnd(v.appendCode) })
		tb("☑", "任务", func() { v.insertAtEnd(v.appendTask) })
		tb("∑", "公式", func() { v.insertAtEnd(v.appendMath) })

		ui.Spacer(c)

		tb(v.modeLabel(), "切换 单栏/源码 模式 (Ctrl+E)", func() { v.toggleMode() })
		tb(v.themeLabel(), "切换 浅色/深色", func() { v.toggleTheme() })
		tb("大纲", "显示/隐藏大纲", func() {
			v.App.Sidebar = !v.App.Sidebar
			v.App.InvalidateUI()
		})
	})
}

func (v *View) modeLabel() string {
	if v.App.Mode == 0 {
		return "源码"
	}
	return "单栏"
}

func (v *View) themeLabel() string {
	switch v.App.ThemeMode {
	case 0:
		return "深色"
	case 1:
		return "深色"
	}
	return "浅色"
}

// toggleMode switches between live and source panes; leaving source mode
// re-parses the text.
func (v *View) toggleMode() {
	if v.App.Mode == 0 {
		v.App.Mode = 1
	} else {
		v.App.Mode = 0
		v.App.Refresh()
		return
	}
	v.App.InvalidateUI()
}

func (v *View) toggleTheme() {
	switch v.App.ThemeMode {
	case 0:
		v.App.ThemeMode = 2
	case 2:
		v.App.ThemeMode = 1
	default:
		v.App.ThemeMode = 2
	}
	v.App.InvalidateUI()
}

// findBar renders the search bar.
func (v *View) findBar(c *ui.Context) {
	if !v.App.FindOpen {
		return
	}
	open := true
	matches := len(v.findMatches())
	ui.FindBar(c, &open, &v.App.FindQuery, matches, &v.App.FindCurrent).
		Padding(6, 10)
	if !open {
		v.App.FindOpen = false
	}
}
