package views

import (
	"strconv"
	"strings"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

const (
	sidebarW = 248.0
	listW    = 224.0
	toolbarH = 58.0
)

func (v *View) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Height(toolbarH).Gap(0).Background(t.Surface).
		Shadow(0, 5, 16, 0, ui.Hex("#00000008")).Children(func() {
		if !v.App.Sidebar {
			ui.Row(c).Padding(0, 8, 0, max(12, c.TitleBar().Left)).Children(func() {
				iconButton(c, iconSidebar, "切换边栏", v.toggleSidebar)
			})
		}
		ui.Row(c).Width(listW).FillHeight().Padding(6, 14, 6, 20).Gap(8).Children(func() {
			ui.Column(c).Grow(1).Gap(0).DragWindow().Children(func() {
				ui.Text(c, folderLabel(v.App.SelFolder)).FontSize(14).Bold().NoWrap().Clip()
				ui.Text(c, itoa(len(v.App.VisibleNotes()))+" 个备忘录").FontSize(12).TextColor(t.TextMuted)
			})
			toolbarCapsule(c).Children(func() {
				iconButton(c, iconEllipsis, "文件夹选项", nil).Menu(v.folderMenu)
			})
		})
		ui.Row(c).Grow(1).FillHeight().Padding(6, max(10, c.TitleBar().Right), 6, 12).Gap(12).Children(func() {
			toolbarCapsule(c).Children(func() { iconButton(c, iconCompose, "新建备忘录 (Cmd+N)", v.newNote) })
			toolbarCapsule(c).Children(func() {
				ui.ButtonBase(c).Key("format").Label("文本格式").Tooltip("文本格式").Size(36, 36).Radius(18).
					Disabled(!v.canEdit()).Children(func() { ui.Text(c, "Aa").FontSize(18).TextColor(t.TextMuted) }).Menu(v.formatMenu)
				iconButton(c, iconChecklist, "插入任务列表", func() { v.insertSnippet("- [ ] ") }).Disabled(!v.canEdit())
				iconButton(c, iconTable, "插入表格", func() { v.insertSnippet("\n| 列 1 | 列 2 | 列 3 |\n| --- | --- | --- |\n|  |  |  |\n") }).Disabled(!v.canEdit())
				iconButton(c, iconPaperclip, "插入图片（可上传图床）", func() {
					if v.flushAutosave() && v.OnInsertImage != nil {
						v.OnInsertImage()
					}
				}).Disabled(!v.canEdit() || v.OnInsertImage == nil)
			})
			toolbarCapsule(c).Children(func() {
				iconButton(c, iconShare, "共享与导出", nil).Disabled(v.App.Path == "").Menu(func(m *ui.Menu) {
					if m.Item("复制 Markdown").Chosen() {
						if v.syncEditor() {
							c.Services().WriteClipboard(v.App.Source)
						}
					}
					if m.Item("导出 PDF…").Disabled(v.OnExportPDF == nil).Chosen() {
						if v.syncEditor() {
							v.OnExportPDF()
						}
					}
				})
				iconButton(c, iconEllipsis, "备忘录选项", nil).Menu(v.noteMenu)
			})
			v.searchBox(c)
		})
	})
	ui.Divider(c).FillWidth()
}

func toolbarCapsule(c *ui.Context) ui.Element {
	t := c.Theme()
	bg, rim := ui.Hex("#fafafa"), ui.Hex("#ffffff")
	if t.Dark {
		bg, rim = ui.Hex("#303032"), ui.Hex("#48484a")
	}
	return ui.Row(c).Gap(2).Padding(1, 3).Radius(24).Background(bg).Border(1, rim).
		Shadow(0, 3, 12, 0, ui.Hex("#0000000a"))
}

func (v *View) searchBox(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Key("search").Grow(1).MinWidth(90).Height(40).Gap(7).Padding(0, 12).Radius(22).
		Background(t.SurfaceHover).Border(1, t.Border).Children(func() {
		ui.Icon(c, iconSearch).Size(17, 17).TextColor(t.TextMuted)
		field := ui.TextInputBase(c, &v.App.Query).Label("搜索备忘录").Placeholder("搜索").FillWidth().FontSize(14).
			OnChange(func() { v.reconcileSelection() })
		if v.focusSearch {
			field.Focus()
			v.focusSearch = false
		}
	})
}

func itoa(n int) string { return strconv.Itoa(n) }

func (v *View) toggleSidebar()       { v.App.Sidebar = !v.App.Sidebar; v.App.InvalidateUI() }
func (v *View) ToggleSidebarAction() { v.toggleSidebar() }
func (v *View) SearchAction() {
	if v.rich != nil {
		v.rich.Blur()
	}
	v.focusSearch = true
	v.App.InvalidateUI()
}

func (v *View) canEdit() bool { return v.App.Path != "" && v.App.SelFolder != app.TrashName }

func (v *View) toggleMode() {
	if !v.canEdit() {
		return
	}
	if v.rich != nil {
		if v.App.Mode != app.ModeEdit {
			v.EditSource()
			return
		}
		if !v.flushAutosave() {
			return
		}
		v.App.Mode = app.ModeRich
		v.focusEditor = true
		v.App.InvalidateUI()
		return
	}
	if v.App.Mode == app.ModeEdit {
		if !v.flushAutosave() {
			return
		}
		v.App.Mode = app.ModePreview
		v.App.Refresh()
	} else {
		v.App.Mode = app.ModeEdit
		v.focusEditor = true
	}
	v.App.InvalidateUI()
}

func (v *View) insertSnippet(s string) {
	if !v.canEdit() {
		return
	}
	if !v.syncEditor() {
		return
	}
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		if strings.TrimSpace(s) == "- [ ]" {
			v.rich.Format("checklist")
		} else {
			v.rich.InsertMarkdown(s)
		}
		v.rich.Focus()
		v.NativeChanged()
		return
	}
	wasPreview := v.App.Mode != app.ModeEdit
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	if wasPreview {
		end := v.edState.Buffer.Len()
		v.edState.Select(end, end)
		if end > 0 {
			s = "\n" + s
		}
	}
	v.edState.Insert(s)
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}

func (v *View) InsertMarkdown(s string) { v.insertSnippet(s) }

func (v *View) toggleTheme() {
	if v.themeIsDark() {
		v.App.ThemeMode = app.ThemeLight
	} else {
		v.App.ThemeMode = app.ThemeDark
	}
	v.App.InvalidateUI()
}
func (v *View) themeIsDark() bool {
	if v.App.ThemeMode == app.ThemeDark {
		return true
	}
	if v.App.ThemeMode == app.ThemeLight {
		return false
	}
	return v.App.HasSysDark && v.App.SysDark
}
func (v *View) ToggleThemeAction() { v.toggleTheme() }
func (v *View) ToggleModeAction()  { v.toggleMode() }
