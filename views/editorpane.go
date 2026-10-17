package views

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
)

// autosaveDelay debounces writes while typing.
const autosaveDelay = 600 * time.Millisecond

// editorPane renders the rightmost column: a centered date header and,
// below it, either the native WYSIWYG editor (rich mode) or the full
// Markdown source editor (edit mode).
func (v *View) editorPane(c *ui.Context) {
	t := c.Theme()
	if v.App.Path == "" && v.App.Source == "" {
		ui.Column(c).Fill().Center().Gap(12).Children(func() {
			ui.Text(c, "未选择文档").FontSize(20).TextColor(t.TextMuted)
			ui.Button(c, "新建文档").OnClick(v.newNote)
		})
		return
	}
	if v.App.Mode == app.ModeRich && v.rich != nil {
		ui.Column(c).Fill().Gap(0).Children(func() {
			ui.Box(c).Fill().Children(func() { v.richPane(c) })
			v.statusBar(c)
		})
		return
	}
	// The card under the pane already paints the white surface; a second
	// full-bleed background here would square off the card's corners.
	ui.Column(c).Fill().Gap(0).Children(func() {
		v.dateHeader(c)
		v.ensureEditor()
		if v.edState == nil {
			ui.Box(c).Fill()
			return
		}
		// Keep the editing surface the column's white as the theme
		// changes; colors left zero follow the window theme.
		v.edState.Theme = &editor.Theme{Background: t.Surface}
		v.edState.ReadOnly = !v.canEdit()
		v.edState.Font = ui.Font{Family: fontMono, Size: 14 * v.App.Zoom}
		// The same centered reading column as the rich editor (Typora keeps
		// one measure in both modes).
		side := float32(22)
		pane := v.windowW
		if v.App.Sidebar {
			pane -= sidebarW
		}
		if v.App.Outline {
			pane -= 236
		}
		if measure := 760 * float32(v.App.Zoom); pane-measure > 2*side {
			side = (pane - measure) / 2
		}
		ui.Box(c).Fill().Padding(2, side, 12, side).Children(func() {
			e := editor.View(c, v.edState).Fill()
			if v.focusEditor {
				e.Focus()
				v.focusEditor = false
			}
		})
		v.statusBar(c)
	})
	v.syncEditor()
}

// statusBar is the word-count strip at the editor column's bottom edge
// (Typora shows counts in its status bar; toggleable in the view menu).
func (v *View) statusBar(c *ui.Context) {
	if !v.App.WordCount {
		return
	}
	t := c.Theme()
	ui.Row(c).FillWidth().Padding(3, 22, 6, 22).Children(func() {
		ui.Spacer(c)
		ui.Text(c, countWords(v.App.CurrentSource()).String()).FontSize(11).TextColor(t.TextMuted).NoWrap()
	})
}

// dateHeader shows the note's modification time centered on top, the way
// Apple Notes dates every note above its text.
func (v *View) dateHeader(c *ui.Context) {
	t := c.Theme()
	tm := v.noteModTime()
	label := ""
	if !tm.IsZero() {
		label = fmt.Sprintf("%d年%d月%d日 %s", tm.Year(), tm.Month(), tm.Day(),
			tm.Format("15:04"))
	} else if v.App.Path != "" {
		label = "尚未保存"
	}
	ui.Column(c).FillWidth().Padding(6, 0, 4, 0).Center().Children(func() {
		if label != "" {
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).NoWrap()
		}
		if v.SaveError != "" {
			ui.Text(c, "未保存："+v.SaveError).FontSize(11).TextColor(t.Danger)
		}
		if v.App.SelFolder == app.TrashName {
			ui.Text(c, "已删除的文档 · 恢复后可编辑").FontSize(11).TextColor(t.TextMuted)
		}
	})
}

// noteModTime finds the modification time of the open note in the library.
func (v *View) noteModTime() time.Time {
	if v.App.Lib == nil {
		return time.Time{}
	}
	for _, n := range v.App.Lib.Notes {
		if n.Path == v.App.Path {
			return n.ModTime
		}
	}
	if v.App.Path != "" {
		if st, err := os.Stat(filepath.Clean(v.App.Path)); err == nil {
			return st.ModTime()
		}
	}
	return time.Time{}
}

// ensureEditor (re)builds the code editor when the note changed or the
// source moved under it (open, autosave-less external reload, modal edits).
// A rebuild keeps the caret where it was, clamped to the new text.
func (v *View) ensureEditor() {
	src := v.App.Source
	if v.edState != nil && v.edPath == v.App.Path && v.edBase == src {
		return
	}
	caret := 0
	if v.edState != nil && v.edPath == v.App.Path {
		caret = v.edState.Cursor()
	}
	buf := editor.NewBuffer(src)
	if caret > buf.Len() {
		caret = buf.Len()
	}
	buf.SetLanguage(editor.LanguageFor("note.md"))
	v.edState = &editor.State{
		Buffer:          buf,
		SoftWrap:        true,
		LineHeight:      1.45,
		NoActiveLine:    true,
		NoIndentGuides:  true,
		NoScrollPastEnd: true,
		LineNumbers:     editor.LineNumbersOff,
		Placeholder:     "开始输入 Markdown…",
		Label:           "文档正文",
	}
	if caret > 0 {
		v.edState.Select(caret, caret)
	}
	v.edPath = v.App.Path
	v.edBase = src
	v.edVer = buf.Version()
}

// syncEditor copies editor changes into App.Source and schedules the
// autosave. The document itself re-parses when preview opens, so typing
// never triggers a rebuild of the widget under the caret.
func (v *View) syncSourceEditor() {
	if v.edState == nil || v.App.Mode != app.ModeEdit || v.edPath != v.App.Path || v.edBase != v.App.Source || !v.canEdit() {
		return
	}
	ver := v.edState.Buffer.Version()
	if ver == v.edVer {
		return
	}
	v.edVer = ver
	src := v.edState.Buffer.String()
	// The buffer is now the source of truth: record it as the base so the
	// next rebuild (autosave's invalidate) does not mistake our own edit
	// for an external change and rebuild the editor — that reset the
	// scroll position and caret after every keystroke.
	v.edBase = src
	v.App.MarkSourceDirty(src)
	v.SaveError = ""
	v.scheduleAutosave()
}

// scheduleAutosave debounces a write of the current note to disk.
func (v *View) scheduleAutosave() {
	if v.autosaveTimer != nil {
		v.autosaveTimer.Stop()
	}
	v.autosaveDue = time.Now().Add(autosaveDelay)
	v.autosaveSource = v.App.Source
	// Timers only request a frame. Buffer access and disk writes stay on
	// the UI thread, so an old timer cannot save a different note.
	wake := v.invalidate
	v.autosaveTimer = time.AfterFunc(autosaveDelay, func() {
		if wake != nil {
			wake()
		}
	})
}

func (v *View) processAutosave() {
	if !v.App.Dirty {
		return
	}
	if v.autosaveTimer == nil || v.autosaveSource != v.App.Source {
		v.scheduleAutosave()
	} else if !time.Now().Before(v.autosaveDue) && v.SaveError == "" && !v.App.Conflict {
		v.autosave()
	}
}

// flushAutosave writes any pending change immediately (note switches,
// new/delete) so the list and disk never show stale content.
func (v *View) flushAutosave() bool {
	if !v.syncEditor() {
		return false
	}
	if v.autosaveTimer != nil {
		v.autosaveTimer.Stop()
		v.autosaveTimer = nil
	}
	return v.autosave()
}

// autosave writes the current note back to its file, updates the library
// (title, snippet, order) and tells main to re-arm the file watcher.
func (v *View) autosave() bool {
	if !v.App.Dirty {
		return true
	}
	if v.App.Conflict {
		v.SaveError = "请先处理外部修改冲突"
		return false
	}
	path := v.App.CurrentPath()
	if path == "" || !v.canEdit() {
		v.SaveError = "当前文档无法保存"
		return false
	}
	src := v.App.CurrentSource()
	err := app.SaveExistingNote(path, src)
	if err != nil {
		v.SaveError = err.Error()
		v.App.InvalidateUI()
		return false
	}
	v.SaveError = ""
	if v.edState != nil && v.edPath == path && v.edBase == src {
		v.edState.Buffer.MarkSaved()
	}
	v.App.MarkSaved()
	v.App.ScanLibrary()
	if v.OnNoteSaved != nil {
		v.OnNoteSaved()
	}
	return true
}
