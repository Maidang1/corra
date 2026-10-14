package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// Column widths, shared by the header sections and the columns below so
// the header lines up with the panes.
const (
	sidebarW = 200.0
	listW    = 300.0
)

// header renders the window's single header row, the way Apple Notes
// does it: a sidebar section, the list's title ("全部备忘录 / N 个备忘
// 录" with a delete affordance), and the note column's action icons with
// the search field on the far right.
func (v *View) header(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).FillWidth().Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Children(func() {
			v.headerSidebar(c)
			v.headerList(c)
			v.headerEditor(c)
		})
	})
	ui.Divider(c).FillWidth()
}

// headerSidebar is the gray stub above the sidebar; its button would
// toggle the sidebar (kept visible always for now).
func (v *View) headerSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(sidebarW).FillHeight().Background(sidebarBackground(t)).
		Padding(8, 10).Children(func() {
		iconButton(c, iconSidebar, "切换边栏", func() {
			// The sidebar is always visible in this version.
		})
	})
}

// headerList shows the selected folder's name and note count, with the
// delete button on the right — "All iCloud / 2 notes / …" in Apple Notes.
func (v *View) headerList(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(listW).FillHeight().Padding(7, 12).Children(func() {
		ui.Row(c).FillWidth().Center().Children(func() {
			ui.Column(c).Gap(0).Children(func() {
				ui.Text(c, folderLabel(v.App.SelFolder)).FontSize(15).Bold()
				n := len(v.App.VisibleNotes())
				word := "个备忘录"
				if v.App.SelFolder == app.TrashName {
					word = "个已删除"
				}
				ui.Text(c, itoa(n)+word).FontSize(11.5).TextColor(t.TextMuted)
			})
			ui.Spacer(c)
			iconButton(c, iconTrash, "删除所选备忘录 (Backspace)", v.deleteNote)
		})
	})
}

// headerEditor holds the note actions — compose, edit/preview, task,
// table, attachment, share, theme — and the search field on the right.
func (v *View) headerEditor(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Grow(1).FillHeight().Padding(6, 12).Children(func() {
		ui.Row(c).FillWidth().Center().Gap(2).Children(func() {
			iconButton(c, iconCompose, "新建备忘录 (Cmd+N)", v.newNote)

			// Aa switches between the rendered note and the Markdown
			// source editor; it highlights while editing.
			aa := ui.ButtonBase(c).Tooltip("切换 编辑 / 预览 (Cmd+E)").
				Padding(4, 8).Radius(6).FontSize(13).Bold()
			if v.App.Mode == app.ModeEdit {
				aa = aa.Background(t.SurfacePressed)
			}
			aa.OnClick(func() { v.toggleMode() })

			iconButton(c, iconChecklist, "插入任务列表", func() {
				v.insertSnippet("- [ ] ")
			})
			iconButton(c, iconTable, "插入表格", func() {
				v.insertSnippet("\n| 列 1 | 列 2 | 列 3 |\n| --- | --- | --- |\n|  |  |  |\n")
			})
			iconButton(c, iconPaperclip, "插入图片（可上传图床）", func() {
				if v.OnInsertImage != nil {
					v.OnInsertImage()
				}
			})
			iconButton(c, iconShare, "导出 PDF", func() {
				if v.OnExportPDF != nil {
					v.OnExportPDF()
				}
			})

			iconButton(c, v.themeIcon(), "切换 浅色 / 深色", v.toggleTheme)

			ui.Spacer(c)

			v.searchBox(c)
		})
	})
}

// searchBox is the rounded "Q 搜索" field of the header's right side.
func (v *View) searchBox(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(190).Padding(1, 10).Radius(8).
		Border(1, t.Border).Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Gap(6).Center().Children(func() {
			ui.Icon(c, iconSearch).Size(13, 13)
			ui.TextInput(c, &v.App.Query).Placeholder("搜索").
				FontSize(12)
		})
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// themeIcon returns the sun or moon for the theme toggle.
func (v *View) themeIcon() *ui.SVG {
	if v.themeIsDark() {
		return iconSun
	}
	return iconMoon
}

// toggleMode switches between the rendered note and the Markdown editor;
// entering preview flushes edits and re-parses the source first.
func (v *View) toggleMode() {
	if v.App.Mode == app.ModeEdit {
		v.flushAutosave()
		v.App.Mode = app.ModePreview
		v.App.Refresh()
	} else {
		v.App.Mode = app.ModeEdit
	}
	v.App.InvalidateUI()
}

// insertSnippet inserts markdown at the caret in edit mode, or appends a
// block in preview mode.
func (v *View) insertSnippet(s string) {
	if v.App.Mode == app.ModeEdit && v.edState != nil {
		v.edState.Insert(s)
		v.App.InvalidateUI()
		return
	}
	v.App.AppendBlock(s)
}

func (v *View) toggleTheme() {
	switch v.App.ThemeMode {
	case app.ThemeSystem:
		if v.themeIsDark() {
			v.App.ThemeMode = app.ThemeLight
		} else {
			v.App.ThemeMode = app.ThemeDark
		}
	case app.ThemeLight:
		v.App.ThemeMode = app.ThemeDark
	default:
		v.App.ThemeMode = app.ThemeLight
	}
	v.App.InvalidateUI()
}

// themeIsDark reports the effective appearance right now.
func (v *View) themeIsDark() bool {
	if v.App.ThemeMode == app.ThemeDark {
		return true
	}
	if v.App.ThemeMode == app.ThemeLight {
		return false
	}
	return v.App.HasSysDark && v.App.SysDark
}

// ToggleThemeAction cycles the theme (exported for menus).
func (v *View) ToggleThemeAction() { v.toggleTheme() }

// ToggleModeAction switches edit/preview (exported for menus).
func (v *View) ToggleModeAction() { v.toggleMode() }
