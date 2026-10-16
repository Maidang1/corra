package views

import (
	"strconv"
	"strings"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

const (
	sidebarW = 280.0
	// topRowH is the height of both cards' top rows: the toolbar line the
	// window's traffic lights land on (they sit at y=26, center y=32, the
	// rows' center at 8+24).
	topRowH = 48.0
)

// header is the workspace card's top row: the note's toolbar. Its empty
// stretches drag the window. When the sidebar is collapsed the row also
// hosts the expand control, clear of the traffic lights.
func (v *View) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Height(topRowH).Gap(10).AlignItems(ui.Center).DragWindow().
		Padding(0, max(12, c.TitleBar().Right), 0, 16).Children(func() {
		if !v.App.Sidebar {
			// Collapsed: the leading slot hosts the expand control,
			// offset past the traffic lights (the card starts at x=8).
			ui.Row(c).FillHeight().AlignItems(ui.Center).Padding(0, 4, 0, max(0, c.TitleBar().Left-8-16)).Children(func() {
				iconButton(c, iconSidebar, "切换边栏", v.toggleSidebar)
			})
		}
		if v.App.Sidebar {
			// The folder title counts as navigation: it collapses with
			// the sidebar.
			ui.Column(c).Gap(0).Children(func() {
				ui.Text(c, folderLabel(v.App.SelFolder)).FontSize(13).Bold().NoWrap().Clip()
				ui.Text(c, itoa(len(v.App.VisibleNotes()))+" 个备忘录").FontSize(11).TextColor(t.TextMuted)
			})
			toolbarCapsule(c).Children(func() {
				iconButton(c, iconEllipsis, "文件夹选项", nil).Menu(v.folderMenu)
			})
		}
		ui.Spacer(c)
		v.modeSwitcher(c)
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
	})
}

// toolbarCapsule groups toolbar buttons into one floating pill: a lifted
// white (or graphite) lozenge with a hairline rim and a faint shadow, the
// way macOS clusters toolbar controls.
func toolbarCapsule(c *ui.Context) ui.Element {
	t := c.Theme()
	bg, rim := ui.Hex("#fbfbfb"), ui.Hex("#00000014")
	if t.Dark {
		bg, rim = ui.Hex("#303034"), ui.Hex("#ffffff20")
	}
	return ui.Row(c).Gap(2).Padding(2, 4).Radius(20).Background(bg).Border(1, rim).
		Shadow(0, 2, 6, 0, ui.Hex("#0000000d"))
}

// modeSwitcher is the header's segmented control for the two editing
// surfaces: 文稿 (WYSIWYG) and 源码 (Markdown source). The current surface
// rides on a raised segment; clicking the other one switches (Cmd+E).
func (v *View) modeSwitcher(c *ui.Context) {
	enabled := v.canEdit() && v.rich != nil
	toolbarCapsule(c).Children(func() {
		v.modeSegment(c, "文稿", iconDoc, v.App.Mode == app.ModeRich, enabled, func() {
			if v.App.Mode != app.ModeRich {
				v.toggleMode()
			}
		})
		v.modeSegment(c, "源码", iconCode, v.App.Mode == app.ModeEdit, enabled, func() {
			if v.App.Mode != app.ModeEdit {
				v.EditSource()
			}
		})
	})
}

// modeSegment is one segment of the switcher: a small icon-plus-label
// lozenge, lifted with a hairline rim when it is the current mode.
func (v *View) modeSegment(c *ui.Context, label string, icon *ui.SVG, active, enabled bool, fn func()) {
	t := c.Theme()
	fg := t.TextMuted
	if active {
		fg = t.Text
	}
	seg := ui.ButtonBase(c).Key(label).Label("切换到"+label).Tooltip(label).Padding(6, 10).Radius(16).
		Disabled(!enabled)
	if active {
		bg, rim := ui.Hex("#ffffff"), ui.Hex("#00000010")
		if t.Dark {
			bg, rim = ui.Hex("#4a4a4f"), ui.Hex("#ffffff24")
		}
		seg = seg.Background(bg).Border(1, rim).Shadow(0, 1, 3, 0, ui.Hex("#00000014"))
	}
	seg.Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(15, 15).TextColor(fg)
			ui.Text(c, label).FontSize(12.5).TextColor(fg)
		})
	}).OnClick(fn)
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

// toggleMode switches between the two editing surfaces: the native
// WYSIWYG editor and the Markdown source editor (Cmd+E or the header's
// mode button). Without the native editor there is only source.
func (v *View) toggleMode() {
	if !v.canEdit() || v.rich == nil {
		return
	}
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
