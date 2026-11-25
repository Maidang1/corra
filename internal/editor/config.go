// Package editor draws and edits a doc tree on MyGo's text-input and
// text-layout primitives, the same on every platform. It owns the
// selection, input methods, keys, clipboard and undo; what each node type
// looks like comes from the views extensions register.
package editor

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/mdcodec"
)

// Env is what views draw with.
type Env struct {
	Theme *ui.Theme
	// Font is the body font; Mono the family of code.
	Font ui.Font
	Mono string
	Dark bool
	// Images loads pictures for image views; nil shows placeholders.
	Images ImageSource
	// BaseDir resolves relative image paths.
	BaseDir string
	// CodeLineNumbers shows line numbers in code blocks.
	CodeLineNumbers bool
	// MathNumbering numbers display formulas: "none" or "", or "all".
	MathNumbering string
	// Ordinals, set by the editor, numbers nodes in document order.
	Ordinals OrdinalSource
	// Pages describes notes for page cards; nil shows them as pictures.
	Pages PageSource
	// Refresh, set by the editor, is what views call from any goroutine
	// when background work they started (a diagram, a picture) is done.
	Refresh Refresher
}

// PageSource tells what a page card shows of the note a link points at.
type PageSource interface {
	Page(target, baseDir string) (title, excerpt string, ok bool)
}

// OrdinalSource tells a node's place among the nodes of its type, from 1,
// in document order: an equation's number.
type OrdinalSource interface{ Ordinal(n *doc.Node) int }

// Refresher lays the document out again on the next frame.
type Refresher interface{ Refresh() }

// ImageSource loads a picture by its source, in the background: until it
// is ready, Image reports false, and the app calls Editor.Relayout when it
// is.
type ImageSource interface {
	Image(src, baseDir string) (bmp *ui.Bitmap, ready bool)
}

func (e Env) key() Env { return e }

// Tint is the quiet fill behind code, formulas and placeholders: a shade
// of the page itself, as the light theme's Surface is the page's white.
func (e Env) Tint() ui.Color {
	if e.Dark {
		return ui.RGBA(255, 255, 255, 0.07)
	}
	return ui.RGBA(0, 0, 0, 0.045)
}

// Rule is the color of quote bars and dividers: stronger than a border,
// quieter than text.
func (e Env) Rule() ui.Color {
	if e.Dark {
		return ui.RGBA(255, 255, 255, 0.22)
	}
	return ui.RGBA(0, 0, 0, 0.16)
}

// TextLook is how a text block looks.
type TextLook struct {
	Font  ui.Font
	Color ui.Color
	// Background fills the block, padded by PadX and PadY, as code.
	Background ui.Color
	Radius     float32
	PadX, PadY float32
	// Before and After are the room above and below the block.
	Before, After float32
	// Gutter moves the text in from the left by that much more than PadX,
	// leaving room for Decorate, as code's line numbers.
	Gutter float32
	// Decorate paints over the block after its text; origin is where the
	// text starts, lines its laid-out lines and text its content.
	Decorate func(p *ui.Painter, origin ui.Point, lines []ui.TextLine, text string, env *Env)
	// Below, when set, adds an area under the text, inside the block,
	// painted by its paint function: a formula's or a diagram's preview
	// under its source.
	Below func(width float32) (height float32, paint func(p *ui.Painter, r ui.Rect, env *Env))
	// Colorize, when set, splits a block of plain text into colored spans
	// whose texts join back into it, as code's syntax highlighting.
	Colorize func(text string) []ui.Span
}

// ContainerLook is how a container lays out and decorates its children.
type ContainerLook struct {
	// Indent moves the children in from the left, IndentRight from the
	// right.
	Indent, IndentRight float32
	// Before and After are the room above and below the container.
	Before, After float32
	// PadTop and PadBottom are the room inside the container above its
	// first child and below its last, as a callout's title.
	PadTop, PadBottom float32
	// Gap, when set, is the room between children in place of their own
	// Before and After, as between a tight list's items.
	Gap float32
	// Color, when set, is the text color of the prose inside, as a
	// quote's quieter ink; code keeps its own.
	Color ui.Color
	// Marker returns text drawn left of child i's first line, as a list's
	// number, in the indent; "" draws none, and Bullet a dot.
	Marker func(i int, child *doc.Node) string
	// Check, when set, says child i is a task: a checkbox is drawn in
	// place of the marker, ticked when checked.
	Check func(i int, child *doc.Node) (task, checked bool)
	// Paint decorates the container's box, as a quote's bar.
	Paint func(p *ui.Painter, r ui.Rect, env *Env)
	// Horizontal lays the children side by side in equal widths, as a
	// table row's cells, rather than one under another.
	Horizontal bool
}

// AtomLook draws a block atom: an image, a rule, a formula.
type AtomLook struct {
	Height func(width float32) float32
	Paint  func(p *ui.Painter, r ui.Rect, env *Env)
}

// InlineLook draws an inline atom. Text, one UTF-16 unit, stands in for it
// in the text layout, as "\n" for a line break; otherwise it is a box of
// Width×Height on the baseline, Descent more below, painted by Paint.
type InlineLook struct {
	Text                   string
	Width, Height, Descent float32
	Paint                  func(p *ui.Painter, r ui.Rect, env *Env)
}

// ViewSpec says how a node or mark type looks. Set the field for the
// type's kind of content.
type ViewSpec struct {
	Text      func(n *doc.Node, env *Env) TextLook
	Container func(n *doc.Node, env *Env) ContainerLook
	Atom      func(n *doc.Node, env *Env) AtomLook
	Inline    func(n *doc.Node, env *Env) InlineLook
	Mark      func(m doc.Mark, env *Env, s *ui.Span)
}

// KeyBinding runs a named command for a key.
type KeyBinding struct {
	Mods    ui.Modifiers
	Key     ui.Key
	Command string
}

// InputRule turns typed Markdown syntax into structure: when the text
// before the caret in its block, after typing, matches Match, Handle runs
// in place of the typing, as "# " making a heading.
type InputRule struct {
	// Match is matched against the block's text up to the caret.
	Match func(before string) []string
	// Handle applies the rule; the matched text is still in the block.
	Handle func(s doc.State, tx *doc.Tx, match []string) bool
}

// SpellChecker finds misspelled words: the UTF-16 ranges of text it
// rejects. The app connects the system's speller.
type SpellChecker interface {
	Misspelled(text string) []ui.TextInputRange
}

// Clipboard is where copy and paste go; the app connects the system's.
type Clipboard interface {
	Write(c Clip)
	Read() Clip
}

// Clip is clipboard content in the forms the editor reads and writes.
type Clip struct {
	Plain, Markdown, HTML string
	// Fragment is the editor's own form, kept in memory, which pastes
	// back with no loss.
	Fragment []*doc.Node
	// Image is a picture's PNG bytes, as a screenshot pasted.
	Image []byte
}

// Bullet is the Marker that draws a dot rather than text: a font's bullet
// shows tiny beside CJK text.
const Bullet = "•"

// Config wires an editor.
type Config struct {
	Registry *doc.Registry
	Codec    *mdcodec.Codec
	// Commands are run by name from keys, menus and the app.
	Commands map[string]doc.Command
	// Chains handle "Enter", "Backspace", "Delete", "Tab", "Shift-Tab".
	Chains     map[string]doc.Command
	Views      map[string]ViewSpec
	Keys       []KeyBinding
	InputRules []InputRule
	// Slash lists the "/" menu's entries.
	Slash []SlashItem
	// Completions are other popups typing opens, as "[[" for note links.
	Completions []Completion
	Clipboard   Clipboard
	// HTMLToMarkdown converts pasted HTML; nil pastes its plain text.
	HTMLToMarkdown func(html string) (string, error)
	// Spell checks prose text blocks when spell checking is on.
	Spell SpellChecker
	// OnPasteImage stores a pasted picture, as in the note's assets, and
	// returns the Markdown that shows it; nil pastes no pictures.
	OnPasteImage func(png []byte) (string, error)
	// OnDropFiles stores files dropped on the editor and returns the
	// Markdown that shows or links them.
	OnDropFiles func(paths []string) string
	// OnLink asks the app for a link to set on the selection, from the
	// format bar; nil leaves the button out.
	OnLink func()
	// OnOpenLink opens a link Cmd+clicked, or a page card clicked; it
	// reports whether it did.
	OnOpenLink func(href string) bool
	// OnChange runs after each committed change to the document.
	OnChange func(c doc.Commit)
	// Padding is the room around the text.
	Padding float32
}
