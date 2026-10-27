package views

import (
	"strconv"
	"strings"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

const (
	sidebarW = 268.0
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
	// The row starts at the window's top edge; the inset keeps its center
	// on the traffic lights' line, level with the sidebar panel's toolbar.
	bar := ui.Row(c).FillWidth().Height(topRowH+8).Gap(8).AlignItems(ui.Center).DragWindow().
		Padding(8, max(12, c.TitleBar().Right), 0, 16)
	// Like a macOS toolbar, a narrow window folds the formatting group into
	// the trailing menu, then drops the folder title. The width derives from
	// the window's, so toggling a panel takes effect in the same frame.
	width := v.windowW
	if v.App.Sidebar {
		width -= sidebarW
	}
	compact := width > 0 && width < 700
	tight := width > 0 && width < 480
	bar.Children(func() {
		if !v.App.Sidebar {
			// Collapsed: the leading slot hosts the expand control,
			// offset past the traffic lights (the toolbar starts at x=0).
			ui.Row(c).FillHeight().AlignItems(ui.Center).Padding(0, 4, 0, max(0, c.TitleBar().Left-16)).Children(func() {
				iconButton(c, iconSidebar, "展开边栏", v.toggleSidebar)
			})
		}
		if v.App.Sidebar {
			// The folder title counts as navigation: it collapses with
			// the sidebar.
			if !tight {
				ui.Column(c).Gap(0).Children(func() {
					ui.Text(c, folderLabel(v.App.SelFolder)).FontSize(13).Bold().NoWrap().Clip()
					ui.Text(c, itoa(len(v.App.VisibleNotes()))+" 个文档").FontSize(11).TextColor(t.TextMuted)
				}).ContextMenu(func(m *ui.Menu) {
					v.folderMenuItems(m, v.App.SelFolder)
				})
			}
		}
		ui.Spacer(c)
		iconButton(c, iconTOC, "显示目录（TOC）", v.ToggleOutlineAction)
		v.modeSwitcher(c)
		if !compact {
			v.formatCapsule(c)
		}
		toolbarCapsule(c).Children(func() {
			iconButton(c, iconCompose, "新建文档 (Cmd+N)", v.newNote)
			iconButton(c, iconShare, "共享与导出", nil).Disabled(v.App.Path == "").Menu(func(m *ui.Menu) {
				if m.Item("复制 Markdown").Chosen() {
					if v.syncEditor() {
						c.Services().WriteClipboard(v.App.Source)
					}
				}
				if m.Item("导出…").Chosen() {
					v.OpenExport()
				}
			})
			iconButton(c, iconEllipsis, "文档选项", nil).Menu(func(m *ui.Menu) {
				if compact {
					// The folded formatting group lives here when narrow.
					m.Submenu("文本格式", v.formatMenu)
					if m.Item("插入任务列表").Disabled(!v.canEdit()).Chosen() {
						v.insertSnippet("- [ ] ")
					}
					if m.Item("插入表格").Disabled(!v.canEdit()).Chosen() {
						v.insertSnippet(tableSnippet)
					}
					if m.Item("插入图片…").Disabled(!v.canEdit() || v.OnInsertImage == nil).Chosen() {
						if v.flushAutosave() && v.OnInsertImage != nil {
							v.OnInsertImage()
						}
					}
					m.Separator()
				}
				v.noteMenu(m)
			})
			iconButton(c, iconGear, "设置 (Cmd+,)", v.OpenSettings)
		})
	})
}

const tableSnippet = "\n| 列 1 | 列 2 | 列 3 |\n| --- | --- | --- |\n|  |  |  |\n"

// formatCapsule is the toolbar's formatting group: text styles, checklist,
// table and attachment.
func (v *View) formatCapsule(c *ui.Context) {
	t := c.Theme()
	toolbarCapsule(c).Children(func() {
		ui.ButtonBase(c).Key("format").Label("文本格式").Tooltip("文本格式").Size(28, 28).Radius(14).
			Disabled(!v.canEdit()).Children(func() { ui.Text(c, "Aa").FontSize(15).TextColor(t.TextMuted) }).Menu(v.formatMenu)
		iconButton(c, iconChecklist, "插入任务列表", func() { v.insertSnippet("- [ ] ") }).Disabled(!v.canEdit())
		iconButton(c, iconTable, "插入表格", func() { v.insertSnippet(tableSnippet) }).Disabled(!v.canEdit())
		iconButton(c, iconPaperclip, "插入图片（可上传图床）", func() {
			if v.flushAutosave() && v.OnInsertImage != nil {
				v.OnInsertImage()
			}
		}).Disabled(!v.canEdit() || v.OnInsertImage == nil)
	})
}

// toolbarCapsule groups toolbar buttons into one floating pill: a quiet,
// flat lozenge with a hairline rim and no shadow, the way macOS 26 clusters
// toolbar controls without lifting them off the bar.
func toolbarCapsule(c *ui.Context) ui.Element {
	t := c.Theme()
	bg, rim := ui.Hex("#f7f7f8"), ui.Hex("#0000000d")
	if t.Dark {
		bg, rim = ui.Hex("#303034"), ui.Hex("#ffffff17")
	}
	return ui.Row(c).Gap(2).Padding(2, 3).Radius(16).Background(bg).Border(1, rim)
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
// lozenge, raised with a hairline rim when it is the current mode.
func (v *View) modeSegment(c *ui.Context, label string, icon *ui.SVG, active, enabled bool, fn func()) {
	t := c.Theme()
	fg := t.TextMuted
	if active {
		fg = t.Text
	}
	seg := ui.ButtonBase(c).Key(label).Label("切换到"+label).Tooltip(label).Padding(4, 9).Radius(12).
		Disabled(!enabled)
	if active {
		bg, rim := ui.Hex("#ffffff"), ui.Hex("#00000010")
		if t.Dark {
			bg, rim = ui.Hex("#4a4a4f"), ui.Hex("#ffffff24")
		}
		seg = seg.Background(bg).Border(1, rim).Shadow(0, 1, 2, 0, ui.Hex("#00000012"))
	}
	seg.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(13, 13).TextColor(fg)
			ui.Text(c, label).FontSize(12).TextColor(fg)
		})
	}).OnClick(fn)
}

func itoa(n int) string { return strconv.Itoa(n) }

func (v *View) toggleSidebar()       { v.App.Sidebar = !v.App.Sidebar; v.App.InvalidateUI() }
func (v *View) ToggleSidebarAction() { v.toggleSidebar() }

func (v *View) ToggleOutlineAction() {
	v.App.Outline = !v.App.Sidebar || !v.App.Outline
	v.App.Sidebar = true
	v.App.InvalidateUI()
}

func (v *View) SearchAction() {
	v.App.Outline = false
	v.App.Sidebar = true
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
		// Empty blocks open natively with the caret inside them; Markdown
		// insertion cannot hold a caret in a block that has no text yet.
		switch strings.TrimSpace(s) {
		case "- [ ]":
			v.rich.Format("checklist")
		case "```\n\n```":
			v.rich.Format("insert:```")
		case "$$\n\n$$":
			v.rich.Format("insert:$$")
		case "---":
			v.rich.Format("insert:---")
		default:
			// The surrounding newlines are for source mode; the native
			// editor places blocks on their own lines itself.
			v.rich.InsertMarkdown(strings.TrimSpace(s))
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

// InsertTableAction inserts a 2×2 table (Typora: ⌥⌘T).
func (v *View) InsertTableAction() { v.insertSnippet(tableSnippet) }

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
