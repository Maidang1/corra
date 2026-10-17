package views

import (
	"errors"
	"strings"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

// dirOf returns the directory part of a slash-separated path.
func dirOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}

// RichEditor owns a native editing surface. Calls run on the UI thread; layout
// uses MyGo's final logical-point rectangle, not guessed column dimensions.
type RichEditor interface {
	Load(source string)
	Snapshot() (source string, changed bool, err error)
	Format(command string)
	InsertMarkdown(source string)
	SetFrame(x, y, width, height float64)
	SetVisible(bool)
	SetStyle(zoom float64, dark, editable bool)
	// SetFont applies the settings' typefaces: body family for prose,
	// mono family for code; empty keeps the system fonts.
	SetFont(body, mono string)
	SetBaseDir(dir string)
	SetMathNumbering(mode string)
	SetCodeLineNumbers(enabled bool)
	// Typora-parity extensions; no-ops on platforms without a rich editor.
	SetSpellcheck(enabled bool)
	SetFocusMode(enabled bool)
	SetTypewriter(enabled bool)
	ShowFindBar()
	// RevealHeading scrolls to the index-th heading paragraph (0-based).
	RevealHeading(index int)
	// Print prints the rendered document (macOS native print panel).
	Print()
	Focus()
	Blur()
	Close()
}

func (v *View) AttachRichEditor(e RichEditor) {
	v.rich = e
	if e != nil {
		v.App.Mode = app.ModeRich
	}
}

// NativeChanged is called after an AppKit edit, before a watcher can replace
// the document. Marked IME input stays in the native buffer until committed.
func (v *View) NativeChanged() {
	if !v.syncEditor() && v.canEdit() {
		// The source is not serializable during IME composition, but it is
		// already user input: the external watcher must treat it as unsaved.
		v.App.MarkSourceDirty(v.App.Source)
	}
	v.App.InvalidateUI()
}

func (v *View) ensureRichEditor() {
	if v.richLoaded && v.richPath == v.App.Path && v.richBase == v.App.Source {
		return
	}
	v.rich.SetBaseDir(dirOf(v.App.Path))
	v.rich.Load(v.App.Source)
	v.richPath, v.richBase, v.richLoaded = v.App.Path, v.App.Source, true
}

func (v *View) syncEditor() bool {
	if v.rich == nil || v.App.Mode != app.ModeRich {
		v.syncSourceEditor()
		return true
	}
	if !v.richLoaded || v.richPath != v.App.Path || v.richBase != v.App.Source || !v.canEdit() {
		return true
	}
	source, changed, err := v.rich.Snapshot()
	var composing interface{ Composing() bool }
	if errors.As(err, &composing) && composing.Composing() {
		// Mid-composition the buffer simply has nothing new to save yet;
		// the commit notifies again and syncs.
		return false
	}
	if err != nil {
		v.SaveError = err.Error()
		v.richSnapshotError = true
		return false
	}
	if v.richSnapshotError {
		v.richSnapshotError = false
		v.SaveError = ""
	}
	if changed && source != v.App.Source {
		v.richBase = source
		v.App.MarkSourceDirty(source)
		v.SaveError = ""
		v.scheduleAutosave()
	}
	return true
}

func (v *View) richVisible() bool {
	return v.rich != nil && v.App.Mode == app.ModeRich &&
		(v.App.Path != "" || v.App.Source != "") &&
		!v.App.Conflict && !v.HostOpen && !v.SettingsOpen && v.FolderDialog == "" && !v.DeleteOpen
}

func (v *View) richPane(c *ui.Context) {
	v.rich.SetMathNumbering(v.App.MathNumbering)
	v.rich.SetCodeLineNumbers(v.App.CodeLineNumbers)
	v.ensureRichEditor()
	v.rich.SetStyle(float64(v.App.Zoom), c.Theme().Dark, v.canEdit())
	v.rich.SetFont(v.App.BodyFont, v.App.MonoFont)
	// The native editor is a rectangular hole punched through the window:
	// inset it from the card's rim so its square corners never cover the
	// card's rounded ones.
	ui.Column(c).Fill().Gap(0).Children(func() {
		v.dateHeader(c)
		ui.Box(c).Fill().Padding(0, 6, 8, 6).Children(func() {
			ui.Box(c).Key("native-editor").Fill().Draw(func(_ *ui.Painter, r ui.Rect) {
				v.rich.SetFrame(float64(r.X), float64(r.Y), float64(r.W), float64(r.H))
				visible := v.richVisible()
				v.rich.SetVisible(visible)
				if visible && v.focusEditor {
					v.rich.Focus()
					v.focusEditor = false
				}
			})
		})
	})
}

// EditSource is also the native protected-block action. Flush before hiding the
// native buffer so an in-progress composition or failed save cannot be lost.
func (v *View) EditSource() {
	if !v.canEdit() || !v.flushAutosave() {
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	v.focusEditor = true
	if v.rich != nil {
		v.rich.SetVisible(false)
	}
	v.App.InvalidateUI()
}

func (v *View) RichAction() {
	if v.rich == nil || !v.canEdit() || !v.flushAutosave() {
		return
	}
	v.App.Mode = app.ModeRich
	v.focusEditor = true
	v.App.InvalidateUI()
}

func richFormat(prefix, suffix string) string {
	if prefix == "clear" {
		return "clear"
	}
	if suffix != "" {
		return map[string]string{
			"**": "bold", "*": "italic", "~~": "strike", "`": "code",
			"<u>": "underline", "==": "highlight", "^": "sup", "~": "sub",
		}[prefix]
	}
	return map[string]string{
		"# ": "heading1", "## ": "heading2", "### ": "heading3", "#### ": "heading4",
		"##### ": "heading5", "###### ": "heading6",
		"": "body", "- ": "bullet", "1. ": "ordered", "- [ ] ": "checklist", "> ": "quote",
	}[prefix]
}
