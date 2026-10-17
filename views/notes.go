package views

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// folderLabel maps a folder key to its sidebar display name.
func folderLabel(folder string) string {
	if folder == "" {
		return "全部文档"
	}
	if folder == app.TrashName {
		return "最近删除"
	}
	return folder
}

// sidebarBackground is the neutral gray of a macOS sidebar panel, a step
// off the page so the floating panel reads against it.
func sidebarBackground(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#2c2d32")
	}
	return ui.Hex("#f4f4f6")
}

// fieldFill is the translucent fill of a borderless macOS search field.
func fieldFill(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#ffffff12")
	}
	return ui.Hex("#0000000b")
}

// foldersSidebar renders the left floating card below the toolbar: the
// smart "all notes" and "recently deleted" rows plus one row per folder at
// the top, the search field, and the selected folder's notes (grouped by
// date, filtered by the search query) in the scroll below, over the
// sidebar's translucent gray.
func (v *View) foldersSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(sidebarW).FillHeight().Padding(8, 4, 8, 8).Children(func() {
		ui.Column(c).Fill().Gap(0).Radius(20).Clip().
			Background(sidebarBackground(t)).
			Border(1, cardBorder(t)).Shadow(0, 8, 24, 0, cardShadow(t)).Children(func() {
			// The card's top row is the sidebar's toolbar: the traffic
			// lights land on it, the toggle right of them, on the same
			// center line as the workspace card's header.
			ui.Row(c).FillWidth().Height(topRowH).AlignItems(ui.Center).DragWindow().
				Padding(0, 8, 0, max(12, c.TitleBar().Left-8)).Children(func() {
				iconButton(c, iconSidebar, "切换边栏", v.toggleSidebar)
			})
			ui.Column(c).FillWidth().Gap(1).Padding(2, 4, 8, 6).Children(func() {
				ui.Text(c, "本地文档库").FontSize(12).Bold().TextColor(t.TextMuted).Padding(0, 7, 5, 7)
				if v.App.Lib != nil {
					v.folderRow(c, "全部文档", iconFolder, "", len(v.App.Lib.Notes))
					for _, f := range v.App.Lib.Folders() {
						v.folderRow(c, f, iconFolder, f, len(v.App.Lib.Filter(f, "")))
					}
					v.folderRow(c, "最近删除", iconTrash, app.TrashName, len(v.App.Lib.Filter(app.TrashName, "")))
				}
			})
			ui.Divider(c).FillWidth().Padding(10, 8, 10, 8)
			v.searchBox(c)
			v.sidebarNotes(c)
			// The new-folder row floats inside the card's bottom corners;
			// its hover wash stays a rounded rect clear of the rim.
			ui.Box(c).FillWidth().Padding(0, 8, 8, 8).Children(func() {
				btn := ui.ButtonBase(c).Label("新建文件夹").FillWidth().Padding(9, 10).Radius(10)
				if btn.Pressed() {
					btn.Background(sidebarPressed(t))
				} else if btn.Hovered() {
					btn.Background(sidebarHover(t))
				}
				// Leading-aligned "⊕ 新建文件夹", like Notes' sidebar footer.
				btn.Children(func() {
					ui.Row(c).FillWidth().Gap(7).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconPlusCircle).Size(16, 16).TextColor(t.TextMuted)
						ui.Text(c, "新建文件夹").FontSize(13).TextColor(t.TextMuted)
					})
				}).OnClick(func() { v.openFolderDialog("新建文件夹", "") })
			})
		})
	})
}

// sidebarHover and sidebarPressed darken rows over the sidebar's light
// gradient, the way Finder's sidebar washes rows on hover; the theme's own
// hover gray is lighter than the gradient and would read as a highlight.
func sidebarHover(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#ffffff0a")
	}
	return ui.Hex("#00000006")
}

func sidebarPressed(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#ffffff12")
	}
	return ui.Hex("#0000000a")
}

// folderRow renders one sidebar row: icon, label and a right-aligned
// count. The selected row is a gray pill with accent-colored text, the
// way Apple Notes marks the current folder.
func (v *View) folderRow(c *ui.Context, label string, icon *ui.SVG, folder string, count int) {
	t := c.Theme()
	selected := v.App.SelFolder == folder
	row := ui.ButtonBase(c).Key(folder).Label("文件夹："+label).FillWidth().Padding(7, 7).Radius(7)
	if selected {
		row = row.Background(t.SurfacePressed)
	} else if row.Pressed() {
		row = row.Background(sidebarPressed(t))
	} else if row.Hovered() {
		row = row.Background(sidebarHover(t))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(7).Center().Children(func() {
			ic := ui.Icon(c, icon).Size(17, 17)
			if selected {
				ic.TextColor(t.Accent)
			}
			txt := ui.Text(c, label).FontSize(14)
			if selected {
				txt.TextColor(t.Accent)
			}
			ui.Spacer(c)
			ui.Text(c, fmt.Sprint(count)).FontSize(12).TextColor(t.TextMuted)
		})
	}).OnClick(func() { v.selectFolder(folder) }).ContextMenu(func(m *ui.Menu) {
		if m.Item("新建文件夹…").Chosen() {
			v.openFolderDialog("新建文件夹", "")
		}
		if folder != "" && folder != app.TrashName {
			if m.Item("重命名文件夹…").Chosen() {
				v.openFolderDialog("重命名文件夹", folder)
			}
			if m.Item("删除空文件夹").Chosen() {
				v.removeFolder(folder)
			}
		}
	})
}

// sidebarNotes renders the notes of the selected folder (filtered by the
// header's search query) in the sidebar's scroll area, grouped into date
// sections.
func (v *View) searchBox(c *ui.Context) {
	t := c.Theme()
	// The filter field lives inside the sidebar, directly above the notes
	// list it filters — the way Apple Notes scopes search to its list.
	ui.Box(c).FillWidth().Padding(4, 8, 6, 8).Children(func() {
		ui.Row(c).Key("search").FillWidth().Height(28).Gap(6).Padding(0, 10).Radius(14).
			Background(fieldFill(t)).Children(func() {
			ui.Icon(c, iconSearch).Size(14, 14).TextColor(t.TextMuted)
			field := ui.TextInputBase(c, &v.App.Query).Label("搜索文档").Placeholder("搜索").FillWidth().FontSize(13).
				OnChange(func() { v.reconcileSelection() })
			if v.focusSearch {
				field.Focus()
				v.focusSearch = false
			}
		})
	})
}

func (v *View) sidebarNotes(c *ui.Context) {
	t := c.Theme()
	notes := v.App.VisibleNotes()
	if len(notes) == 0 {
		// Fill like the list would, so the footer stays at the bottom.
		ui.Column(c).Fill().Center().Padding(18, 14).Children(func() {
			hint := "暂无文档"
			if v.App.SelFolder == app.TrashName {
				hint = "最近删除为空"
			} else if v.App.Query != "" {
				hint = "没有找到匹配的文档"
			}
			ui.Text(c, hint).FontSize(13).TextColor(t.TextMuted)
		})
		return
	}
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(1).Padding(0, 4, 0, 8).Children(func() {
			for _, g := range v.App.NoteGroups() {
				ui.Text(c, g.Label).FontSize(11).Bold().TextColor(t.TextMuted).Padding(12, 10, 4, 10)
				for _, n := range g.Notes {
					v.noteRow(c, n)
				}
			}
		})
	})
}

// noteRow renders one note of the sidebar list the way Apple Notes does:
// the bold title, then the time followed by a one-line preview. The
// selected note is a filled accent-colored rounded pill with white text.
func (v *View) noteRow(c *ui.Context, n app.Note) {
	t := c.Theme()
	selected := n.Path == v.App.SelNote
	fg := t.Text
	sub := t.TextMuted
	if selected {
		fg, sub = t.AccentText, t.AccentText
	}
	row := ui.ButtonBase(c).Key(n.Path).Label("文档："+n.Title).FillWidth().Padding(6, 10).Radius(7)
	if selected {
		row = row.Background(t.Accent)
	} else if row.Pressed() {
		row = row.Background(sidebarPressed(t))
	} else if row.Hovered() {
		row = row.Background(sidebarHover(t))
	}
	row.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, n.Title).FontSize(13).TextColor(fg).Bold().SingleLine().Ellipsis("…").FillWidth().MinWidth(0)
			preview := n.Snippet
			if preview == "" {
				preview = "暂无正文"
			}
			ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
				ui.Text(c, noteTime(n.ModTime)).FontSize(11).TextColor(fg).NoWrap().Shrink(0)
				ui.Text(c, preview).FontSize(11).TextColor(sub).SingleLine().Ellipsis("…").Grow(1).Shrink(1).MinWidth(0)
			})
		})
	}).OnClick(func() { v.openNote(n.Path) }).ContextMenu(func(m *ui.Menu) {
		if n.Path != v.App.SelNote {
			v.openNote(n.Path)
		}
		if n.Path != v.App.SelNote {
			m.Item("请先保存当前文档").Disabled(true)
			return
		}
		v.noteMenu(m)
	})
}

// noteTime formats a note's time the way the list does: the clock time
// for today, "M月d日" within the year, a full date otherwise.
func noteTime(tm time.Time) string {
	now := time.Now()
	if tm.Year() == now.Year() && tm.YearDay() == now.YearDay() {
		return tm.Format("15:04")
	}
	if tm.Year() == now.Year() {
		return fmt.Sprintf("%d月%d日", tm.Month(), tm.Day())
	}
	return fmt.Sprintf("%d年%d月%d日", tm.Year(), tm.Month(), tm.Day())
}

// newNote creates a note in the selected folder and opens it.
func (v *View) newNote() {
	if !v.flushAutosave() {
		return
	}
	v.SaveError = ""
	if err := v.App.NewNote(); err != nil {
		mygo.Dialog.Error("新建文档失败", err.Error())
	} else {
		v.edState = nil
		if v.rich != nil {
			v.App.Mode = app.ModeRich
		}
		v.focusEditor = true
	}
}

// NewNoteAction creates a note (exported for menus).
func (v *View) NewNoteAction() { v.newNote() }

// DeleteNoteAction deletes the selected note (exported for menus).
func (v *View) DeleteNoteAction() { v.deleteNote() }

// FlushAutosave writes pending edits to disk at once (exported for menus
// and for operations that read the document, like save and export).
func (v *View) FlushAutosave() bool { return v.flushAutosave() }

// SyncBuffer snapshots pending input without requiring a writable original
// file, so Save As and export can recover a document after a save failure.
func (v *View) SyncBuffer() bool { return v.syncEditor() }

// deleteNote asks, then deletes the selected note: notes go to the
// trash, notes already in the trash are removed permanently.
func (v *View) deleteNote() {
	if v.App.SelNote == "" || !v.flushAutosave() {
		return
	}
	v.DeleteOpen = true
	v.App.InvalidateUI()
}
