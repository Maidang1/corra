package views

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
)

// FindAction routes Cmd+F: the native find bar in rich mode, the editor's
// own find bar in source mode.
func (v *View) FindAction() {
	if v.App.Mode == app.ModeRich && v.rich != nil {
		v.rich.ShowFindBar()
		return
	}
	if v.App.Mode == app.ModeEdit && v.edState != nil {
		v.edState.Run(editor.Find)
		v.focusEditor = true
		v.App.InvalidateUI()
	}
}

// PrintAction prints via the native print panel in rich mode; other modes
// export a PDF instead (MyGo has no print API for its own surfaces).
func (v *View) PrintAction() {
	if v.rich != nil {
		if !v.flushAutosave() {
			return
		}
		if v.App.Mode != app.ModeRich {
			v.App.Mode = app.ModeRich
			v.ensureRichEditor()
		}
		v.rich.Print()
		return
	}
	if v.OnExportPDF != nil && v.syncEditor() {
		v.OnExportPDF()
	}
}

// CopyMarkdown copies the document source to the clipboard.
func (v *View) CopyMarkdown() {
	if !v.syncEditor() {
		return
	}
	_ = mygo.Clipboard.Write(transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.Text, []byte(v.App.CurrentSource())),
	)))
}

// CopyHTML copies the rendered document as HTML + plain text.
func (v *View) CopyHTML() {
	if !v.syncEditor() || v.OnCopyHTML == nil {
		return
	}
	html, err := v.OnCopyHTML()
	if err != nil || len(html) == 0 {
		return
	}
	_ = mygo.Clipboard.Write(transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.HTML, html),
		transfer.Bytes(transfer.Text, []byte(v.App.CurrentSource())),
	)))
}

// InsertFootnote adds a footnote reference at the caret and its definition
// at the end of the document, like Typora's footnote insertion.
func (v *View) InsertFootnote() {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	// Find the next free footnote number.
	used := map[string]bool{}
	src := v.App.CurrentSource()
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "[^") {
			if i := strings.IndexByte(line, ']'); i > 2 {
				used[line[2:i]] = true
			}
		}
	}
	n := 1
	for used[strconv.Itoa(n)] {
		n++
	}
	label := strconv.Itoa(n)
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		v.rich.InsertMarkdown("[^" + label + "]")
		v.NativeChanged()
		if !v.flushAutosave() {
			return
		}
		// Append the definition at the end and reload the native buffer so
		// it reflects the change (the caret lands at the definition, as in
		// Typora's footnote insertion).
		src := strings.TrimRight(v.App.Source, "\n") + "\n\n[^" + label + "]: "
		v.App.SetSource(src, len(src))
		v.richLoaded = false
		v.ensureRichEditor()
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	v.edState.Insert("[^" + label + "]")
	v.syncEditor()
	end := v.edState.Buffer.Len()
	v.edState.Select(end, end)
	v.edState.Insert("\n\n[^" + label + "]: ")
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}

// InsertLinkAction opens the insert-link dialog (Cmd+K).
func (v *View) InsertLinkAction() { v.openLinkDialog() }

// ToggleSpellcheck flips continuous spell checking in the native editor.
func (v *View) ToggleSpellcheck() {
	v.App.Spellcheck = !v.App.Spellcheck
	if v.rich != nil {
		v.rich.SetSpellcheck(v.App.Spellcheck)
	}
	v.App.InvalidateUI()
}

// ToggleFocusMode flips focus mode (dimmed inactive paragraphs).
func (v *View) ToggleFocusMode() {
	v.App.FocusMode = !v.App.FocusMode
	if v.rich != nil {
		v.rich.SetFocusMode(v.App.FocusMode)
	}
	v.App.InvalidateUI()
}

// ToggleTypewriter flips typewriter mode (caret line stays centered).
func (v *View) ToggleTypewriter() {
	v.App.Typewriter = !v.App.Typewriter
	if v.rich != nil {
		v.rich.SetTypewriter(v.App.Typewriter)
	}
	v.App.InvalidateUI()
}
