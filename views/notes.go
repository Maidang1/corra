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
	return ui.Hex("#f2f2f5")
}

// foldersSidebar renders the leftmost column: the smart "all notes" and
// "recently deleted" rows and one row per folder, with note counts, over
// the sidebar's translucent gray.
func (v *View) foldersSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(sidebarW).FillHeight().Background(sidebarBackground(t)).
		Padding(8, 8).Children(func() {
		ui.Column(c).Fill().Gap(1).Children(func() {
			if v.App.Lib == nil {
				return
			}
			v.folderRow(c, "全部备忘录", iconFolder, "", len(v.App.Lib.Notes))
			for _, f := range v.App.Lib.Folders() {
				count := 0
				for _, n := range v.App.Lib.Notes {
					if n.Folder == f {
						count++
					}
				}
				v.folderRow(c, f, iconFolder, f, count)
			}
			ui.Box(c).Height(10)
			v.folderRow(c, "最近删除", iconTrash, app.TrashName,
				app.TrashCount(v.App.LibraryRoot))
		})
	})
}

// folderRow renders one sidebar row: icon, label and a right-aligned
// count. The selected row is a gray pill with accent-colored text, the
// way Apple Notes marks the current folder.
func (v *View) folderRow(c *ui.Context, label string, icon *ui.SVG, folder string, count int) {
	t := c.Theme()
	selected := v.App.SelFolder == folder
	row := ui.Box(c).FillWidth().Padding(5, 8).Radius(6)
	if selected {
		row = row.Background(t.SurfacePressed)
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(7).Center().Children(func() {
			ui.Icon(c, icon).Size(16, 16)
			txt := ui.Text(c, label).FontSize(13)
			if selected {
				txt.TextColor(t.Accent)
			}
			ui.Spacer(c)
			ui.Text(c, fmt.Sprint(count)).FontSize(12).TextColor(t.TextMuted)
		})
	}).OnClick(func() {
		v.App.SelFolder = folder
		// Keep the selection if the note is still visible, else select
		// the most recent note of the folder.
		visible := false
		for _, n := range v.App.VisibleNotes() {
			if n.Path == v.App.SelNote {
				visible = true
				break
			}
		}
		if !visible {
			if notes := v.App.VisibleNotes(); len(notes) > 0 {
				_ = v.App.OpenNote(notes[0].Path)
			}
		}
		v.App.InvalidateUI()
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
				ui.Column(c).FillWidth().Gap(2).Padding(2, 8).Children(func() {
					for _, g := range v.App.NoteGroups() {
						ui.Text(c, g.Label).FontSize(15).Bold().Padding(12, 8, 6, 8)
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
	card := ui.Box(c).FillWidth().Padding(8, 12).Radius(10)
	if selected {
		card = card.Background(t.Accent)
	}
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(1).Children(func() {
			ui.Text(c, n.Title).FontSize(13.5).TextColor(fg).Bold()
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, noteTime(n.ModTime)).FontSize(12).TextColor(sub).
					NoWrap()
				ui.Text(c, ellipsize(n.Snippet, 16)).FontSize(12).
					TextColor(sub).NoWrap().Clip()
			})
			folder := n.Folder
			if folder == app.TrashName {
				folder = "最近删除"
			}
			if folder != "" {
				ui.Row(c).FillWidth().Gap(4).Children(func() {
					ui.Icon(c, iconFolder).Size(11, 11)
					ui.Text(c, folder).FontSize(11).TextColor(sub)
				})
			}
		})
	}).OnClick(func() {
		if v.App.Dirty {
			v.flushAutosave()
		}
		if err := v.App.OpenNote(n.Path); err != nil {
			mygo.Dialog.Error("打开失败", err.Error())
		}
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
	v.flushAutosave()
	if err := v.App.NewNote(); err != nil {
		mygo.Dialog.Error("新建备忘录失败", err.Error())
	}
}

// NewNoteAction creates a note (exported for menus).
func (v *View) NewNoteAction() { v.newNote() }

// DeleteNoteAction deletes the selected note (exported for menus).
func (v *View) DeleteNoteAction() { v.deleteNote() }

// FlushAutosave writes pending edits to disk at once (exported for menus
// and for operations that read the document, like save and export).
func (v *View) FlushAutosave() { v.flushAutosave() }

// deleteNote asks, then deletes the selected note: notes go to the
// trash, notes already in the trash are removed permanently.
func (v *View) deleteNote() {
	if v.App.SelNote == "" {
		return
	}
	msg := "确定要删除这个备忘录吗？可在「最近删除」中找回。"
	if v.App.SelFolder == app.TrashName {
		msg = "该备忘录已在「最近删除」中，删除后将无法恢复。"
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageWarning,
		Title:   "删除备忘录",
		Message: msg,
		Buttons: []string{"删除", "取消"},
	})
	if err != nil || res.Button != 0 {
		return
	}
	if err := v.App.DeleteSelectedNote(); err != nil {
		mygo.Dialog.Error("删除失败", err.Error())
	}
}
