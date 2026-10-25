package views

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/internal/mdext"
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

// FindReplaceAction opens the replace UI: the native find bar's replace mode
// in rich mode, the source editor's find-replace bar otherwise.
func (v *View) FindReplaceAction() {
	if v.App.Mode == app.ModeRich && v.rich != nil {
		v.rich.ShowFindReplaceBar()
		return
	}
	if v.App.Mode == app.ModeEdit && v.edState != nil {
		v.edState.Run(editor.FindReplace)
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

// FootnoteBackAction returns the caret to where the last footnote jump
// started; running it again toggles back. Rich mode only.
func (v *View) FootnoteBackAction() {
	if v.rich == nil || v.App.Mode != app.ModeRich {
		return
	}
	v.rich.Format("footnote:back")
}

// InsertFootnote adds a footnote reference at the caret and its definition
// at the end of the document, like Typora's footnote insertion.
func (v *View) InsertFootnote() {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		v.rich.Format("footnote:insert")
		v.NativeChanged()
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

// InsertImageSize resizes the figure under the caret. The native editor
// rewrites the size suffix on the destination and redraws the figure, so the
// document keeps asking for the size the reader picked; an empty spec asks for
// the natural size again. Source mode has no figure to size, so it does nothing.
func (v *View) InsertImageSize(spec string) {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		v.rich.Format("image-size:" + spec)
		v.rich.Focus()
		v.NativeChanged()
	}
}

// InsertLinkAction opens the insert-link dialog (Cmd+K).
func (v *View) InsertLinkAction() { v.openLinkDialog() }

// InsertAlert turns the caret's paragraph into a GitHub-style callout. In rich
// mode the native editor writes the marker itself, so the paragraph keeps its
// identity and its undo history; an empty label removes the marker again.
func (v *View) InsertAlert(label string) {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		v.rich.Format("quote-alert:" + label)
		v.rich.Focus()
		v.NativeChanged()
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	src := v.edState.Buffer.String()
	sel := v.edState.Selection()
	start := strings.LastIndex(src[:sel.Start()], "\n") + 1
	end := sel.End()
	if i := strings.IndexByte(src[end:], '\n'); i >= 0 {
		end += i
	} else {
		end = len(src)
	}
	// The quote markers go, the callout marker is rewritten, so asking for a
	// second kind replaces the first instead of stacking "[!TIP] [!NOTE]".
	body := strings.TrimLeft(src[start:end], "> \t")
	if existing, rest := mdext.AlertLabelPrefix(body); existing != "" {
		body = rest
		if label == "" {
			v.edState.Select(start, end)
			v.edState.Insert("> " + body)
			v.syncEditor()
			v.focusEditor = true
			v.App.InvalidateUI()
			return
		}
	}
	if label == "" {
		return
	}
	v.edState.Select(start, end)
	v.edState.Insert("> [!" + label + "] " + body)
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}

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

func (v *View) RefreshMath() {
	if v.rich != nil {
		v.rich.Format("math-refresh")
	}
	v.App.InvalidateUI()
}
