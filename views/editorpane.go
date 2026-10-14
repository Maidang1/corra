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
// below it, either the full Markdown editor (edit mode) or the rendered
// document (preview mode).
func (v *View) editorPane(c *ui.Context) {
	t := c.Theme()
	if v.App.Mode == app.ModePreview {
		v.previewPane(c)
		return
	}
	ui.Column(c).Fill().Gap(0).Background(t.Surface).Children(func() {
		v.dateHeader(c)
		v.ensureEditor()
		if v.edState == nil {
			ui.Box(c).Fill()
			return
		}
		// Keep the editing surface the column's white as the theme
		// changes; colors left zero follow the window theme.
		v.edState.Theme = &editor.Theme{Background: t.Surface}
		editor.View(c, v.edState).Fill()
	})
	v.syncEditor()
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
	ui.Box(c).FillWidth().Padding(10, 0).Center().Children(func() {
		if label != "" {
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
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
		Buffer:      buf,
		SoftWrap:    true,
		LineHeight:  1.65,
		LineNumbers: editor.LineNumbersOff,
		Placeholder: "开始输入 Markdown…",
		Label:       "备忘录正文",
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
func (v *View) syncEditor() {
	if v.edState == nil {
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
	v.scheduleAutosave()
}

// scheduleAutosave debounces a write of the current note to disk.
func (v *View) scheduleAutosave() {
	if v.autosaveTimer != nil {
		v.autosaveTimer.Stop()
	}
	v.autosaveTimer = time.AfterFunc(autosaveDelay, v.autosave)
}

// flushAutosave writes any pending change immediately (note switches,
// new/delete) so the list and disk never show stale content.
func (v *View) flushAutosave() {
	if v.autosaveTimer != nil {
		v.autosaveTimer.Stop()
		v.autosaveTimer = nil
	}
	v.autosave()
}

// autosave writes the current note back to its file, updates the library
// (title, snippet, order) and tells main to re-arm the file watcher.
func (v *View) autosave() {
	path := v.App.CurrentPath()
	if path == "" {
		return
	}
	src := v.App.CurrentSource()
	if err := os.WriteFile(filepath.Clean(path), []byte(src), 0o644); err != nil {
		return
	}
	if v.edState != nil {
		v.edState.Buffer.MarkSaved()
	}
	v.App.MarkSaved()
	v.App.ScanLibrary()
	if v.OnNoteSaved != nil {
		v.OnNoteSaved()
	}
}

// previewPane renders the parsed document read-only, on the column's
// white surface, the way Apple Notes shows a note.
func (v *View) previewPane(c *ui.Context) {
	ui.Column(c).Fill().Gap(0).Background(c.Theme().Surface).Children(func() {
		v.dateHeader(c)
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Center().Children(func() {
				ui.Column(c).MaxWidth(760).Gap(12).Padding(20, 40, 48, 40).
					Children(func() {
						for i := range v.App.Doc.Blocks {
							v.block(c, i)
						}
						v.footnotes(c)
					})
			})
		})
	})
}
