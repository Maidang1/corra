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
		return "全部备忘录"
	}
	if folder == app.TrashName {
		return "最近删除"
	}
	return folder
}

// sidebarBackground is the translucent gray of Apple Notes' sidebar.
func sidebarBackground(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#26262b")
	}
	return ui.Hex("#f6f7f8")
}

// foldersSidebar renders the leftmost column: the smart "all notes" and
// "recently deleted" rows and one row per folder, with note counts, over
// the sidebar's translucent gray.
func (v *View) foldersSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(sidebarW).FillHeight().Padding(8, 0, 8, 8).Children(func() {
		ui.Column(c).Fill().Gap(0).Radius(24).
			LinearGradient(ui.LinearGradient{From: sidebarBackground(t), To: sidebarBottom(t), Angle: 170}).
			Border(1, t.Surface).Shadow(4, 4, 24, 0, ui.Hex("#0000000c")).Children(func() {
			ui.Row(c).FillWidth().Height(toolbarH-8).Padding(0, 8, 0, max(100, c.TitleBar().Left-8)).DragWindow().Children(func() {
				iconButton(c, iconSidebar, "切换边栏", v.toggleSidebar)
			})
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).FillWidth().Gap(1).Padding(8, 10).Children(func() {
					ui.Text(c, "在我的 Mac 上").FontSize(12).Bold().TextColor(t.TextMuted).Padding(0, 7, 5, 7)
					if v.App.Lib == nil {
						return
					}
					v.folderRow(c, "全部备忘录", iconFolder, "", len(v.App.Lib.Notes))
					for _, f := range v.App.Lib.Folders() {
						v.folderRow(c, f, iconFolder, f, len(v.App.Lib.Filter(f, "")))
					}
					v.folderRow(c, "最近删除", iconTrash, app.TrashName, len(v.App.Lib.Filter(app.TrashName, "")))
				})
			})
			ui.ButtonBase(c).Label("新建文件夹").FillWidth().Padding(12, 16).Children(func() {
				ui.Row(c).Gap(8).Children(func() { ui.Icon(c, iconFolder).Size(17, 17); ui.Text(c, "新建文件夹").FontSize(12) })
			}).OnClick(func() { v.openFolderDialog("新建文件夹", "") })
		})
	})
}

func sidebarBottom(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#282d2c")
	}
	return ui.Hex("#edf5f1")
}

// folderRow renders one sidebar row: icon, label and a right-aligned
// count. The selected row is a gray pill with accent-colored text, the
// way Apple Notes marks the current folder.
func (v *View) folderRow(c *ui.Context, label string, icon *ui.SVG, folder string, count int) {
	t := c.Theme()
	selected := v.App.SelFolder == folder
	row := ui.ButtonBase(c).Key(folder).Label("文件夹："+label).FillWidth().Padding(7, 7).Radius(9)
	if selected {
		row = row.Background(t.SurfacePressed)
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(7).Center().Children(func() {
			ic := ui.Icon(c, icon).Size(20, 20)
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

// notesList renders the middle column: the notes of the selected folder
// (filtered by the header's search query) grouped into date sections.
func (v *View) notesList(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(listW).FillHeight().Background(t.Surface).Children(func() {
		ui.Column(c).Fill().Gap(0).Children(func() {
			notes := v.App.VisibleNotes()
			if len(notes) == 0 {
				ui.Box(c).FillWidth().Padding(24, 14).Children(func() {
					hint := "无备忘录"
					if v.App.SelFolder == app.TrashName {
						hint = "最近删除为空"
					} else if v.App.Query != "" {
						hint = "无匹配的备忘录"
					}
					ui.Text(c, hint).FontSize(12.5).TextColor(t.TextMuted)
				})
				return
			}

			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).FillWidth().Gap(2).Padding(0, 10).Children(func() {
					for _, g := range v.App.NoteGroups() {
						ui.Text(c, g.Label).FontSize(16).Bold().Padding(13, 7, 10, 7)
						for _, n := range g.Notes {
							v.noteCard(c, n)
						}
					}
				})
			})
		})
	})
}

// noteCard renders one note of the list. The selected note is a filled
// accent-colored rounded card with white text, exactly like Apple Notes;
// the rest are quiet rows with title, time and snippet.
func (v *View) noteCard(c *ui.Context, n app.Note) {
	t := c.Theme()
	selected := n.Path == v.App.SelNote
	fg := t.Text
	sub := t.TextMuted
	if selected {
		fg, sub = t.AccentText, t.AccentText
	}
	card := ui.ButtonBase(c).Key(n.Path).Label("备忘录："+n.Title).FillWidth().Padding(10, 18).Radius(9)
	if selected {
		card = card.Background(t.Accent)
	}
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(1).Children(func() {
			ui.Text(c, n.Title).FontSize(14).TextColor(fg).Bold().NoWrap().Clip()
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, noteTime(n.ModTime)).FontSize(12).TextColor(sub).
					NoWrap()
				ui.Text(c, ellipsize(n.Snippet, 11)).FontSize(12).
					TextColor(sub).Grow(1).NoWrap().Clip()
			})
			folder := n.Folder
			if folder == app.TrashName {
				folder = "最近删除"
			}
			if folder == "" {
				folder = "备忘录"
			}
			if folder != "" {
				ui.Row(c).FillWidth().Gap(4).Children(func() {
					ui.Icon(c, iconFolder).Size(11, 11)
					ui.Text(c, folder).FontSize(11).TextColor(sub)
				})
			}
		})
	}).OnClick(func() { v.openNote(n.Path) }).ContextMenu(func(m *ui.Menu) {
		if n.Path != v.App.SelNote {
			v.openNote(n.Path)
		}
		if n.Path != v.App.SelNote {
			m.Item("请先保存当前备忘录").Disabled(true)
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

// ellipsize shortens s to max runes with an ellipsis.
func ellipsize(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// newNote creates a note in the selected folder and opens it.
func (v *View) newNote() {
	if !v.flushAutosave() {
		return
	}
	v.SaveError = ""
	if err := v.App.NewNote(); err != nil {
		mygo.Dialog.Error("新建备忘录失败", err.Error())
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
