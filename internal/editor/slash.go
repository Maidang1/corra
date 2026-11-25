package editor

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// SlashItem is an entry of the "/" menu.
type SlashItem struct {
	Title, Hint string
	// Keywords match the typed query besides the title, as "h1 heading".
	Keywords string
	// Command is a named command.
	Command string
}

// Completion is a popup that typing opens: "/" for block types, "[[" for
// links to notes, ":" for emoji. Items lists the entries for what was typed
// after the trigger; Apply returns the command for the chosen one, run
// after the trigger and query are removed.
type Completion struct {
	Trigger string
	// AnyPosition opens it anywhere; otherwise only at a block's start or
	// after a space.
	AnyPosition bool
	Items       func(query string) []MenuItem
	Apply       func(item MenuItem) doc.Command
}

// MenuItem is one entry of a completion popup.
type MenuItem struct {
	Title, Hint string
	Value       any
}

// popupState is an open completion: where its trigger is, what was typed
// after it, and the highlighted entry.
type popupState struct {
	c     *Completion
	leaf  doc.NodeID
	at    int // offset of the trigger
	query string
	index int
	items []MenuItem
}

const (
	popupRowH  = 30
	popupWidth = 260
	popupRows  = 8
)

// completions returns the configured completions, the "/" menu first.
func (e *Editor) completions() []*Completion {
	if e.compl != nil {
		return e.compl
	}
	if len(e.cfg.Slash) > 0 {
		e.compl = append(e.compl, &Completion{
			Trigger: "/",
			Items: func(q string) []MenuItem {
				q = strings.ToLower(q)
				var out []MenuItem
				for _, it := range e.cfg.Slash {
					if q == "" || strings.Contains(strings.ToLower(it.Title+" "+it.Keywords), q) {
						out = append(out, MenuItem{Title: it.Title, Hint: it.Hint, Value: it.Command})
					}
				}
				return out
			},
			Apply: func(it MenuItem) doc.Command {
				cmd, ok := e.cfg.Commands[it.Value.(string)]
				if !ok {
					return func(doc.State, *doc.Tx) bool { return false }
				}
				return cmd
			},
		})
	}
	for i := range e.cfg.Completions {
		e.compl = append(e.compl, &e.cfg.Completions[i])
	}
	return e.compl
}

// updatePopup opens, follows or closes a popup after typing: it opens when
// the typed text completes a trigger, follows the text typed after it, and
// closes when the caret leaves it or a space ends the query.
func (e *Editor) updatePopup(typed string) {
	sel, ok := e.state.TextSel()
	if !ok || !sel.Empty() {
		e.popup = nil
		return
	}
	leaf := e.state.Node(sel.Head.Leaf)
	text := leaf.Inline().Slice(0, sel.Head.Offset).String()
	if e.popup == nil {
		if typed == "" || leaf.Type().Content() != doc.ContentInline {
			return
		}
		for _, c := range e.completions() {
			if !strings.HasSuffix(text, c.Trigger) {
				continue
			}
			before := strings.TrimSuffix(text, c.Trigger)
			if !c.AnyPosition && before != "" && !strings.HasSuffix(before, " ") {
				continue
			}
			at := sel.Head.Offset - doc.UTF16Len(c.Trigger)
			e.popup = &popupState{c: c, leaf: leaf.ID(), at: at, items: c.Items("")}
			if len(e.popup.items) == 0 {
				e.popup = nil
			}
			return
		}
		return
	}
	p := e.popup
	start := p.at + doc.UTF16Len(p.c.Trigger)
	if leaf.ID() != p.leaf || sel.Head.Offset < start ||
		leaf.Inline().Slice(p.at, start).String() != p.c.Trigger {
		e.popup = nil
		return
	}
	q := leaf.Inline().Slice(start, sel.Head.Offset).String()
	if strings.ContainsAny(q, " \n") {
		e.popup = nil
		return
	}
	if q != p.query || p.items == nil {
		p.query, p.index, p.items = q, 0, p.c.Items(q)
	}
	if len(p.items) == 0 {
		e.popup = nil
	}
}

// choose removes the trigger and query and applies the entry, as one
// change.
func (e *Editor) choose(it MenuItem) {
	p := e.popup
	e.popup = nil
	end := p.at + doc.UTF16Len(p.c.Trigger) + doc.UTF16Len(p.query)
	apply := p.c.Apply(it)
	e.Run(doc.Chain(func(st doc.State, tx *doc.Tx) bool {
		tx.ReplaceText(p.leaf, p.at, end, doc.Inline{})
		tx.SetSelection(doc.Caret(doc.Point{Leaf: p.leaf, Offset: p.at}))
		return true
	}, func(st doc.State, tx *doc.Tx) bool { return apply(st, tx) }), "structure")
}

// popupKey handles keys while a popup is open; it reports whether it took
// the key.
func (e *Editor) popupKey(ev ui.InputEvent) bool {
	if e.popup == nil || len(e.popup.items) == 0 {
		return false
	}
	p := e.popup
	switch ev.Key {
	case ui.KeyDown:
		p.index = (p.index + 1) % len(p.items)
	case ui.KeyUp:
		p.index = (p.index + len(p.items) - 1) % len(p.items)
	case ui.KeyEnter, ui.KeyTab:
		e.choose(p.items[min(p.index, len(p.items)-1)])
	case ui.KeyEscape:
		e.popup = nil
	default:
		return false
	}
	e.redraw()
	return true
}

// popupRect returns the popup's box in editor coordinates, under the
// trigger, and its first visible row.
func (e *Editor) popupRect() (ui.Rect, int, bool) {
	p := e.popup
	if p == nil || len(p.items) == 0 {
		return ui.Rect{}, 0, false
	}
	c, ok := e.caretRect(doc.Point{Leaf: p.leaf, Offset: p.at})
	if !ok {
		return ui.Rect{}, 0, false
	}
	first := max(0, min(p.index-popupRows+1, len(p.items)-popupRows))
	rows := min(len(p.items), popupRows)
	return ui.Rect{X: c.X, Y: c.Y + c.H + 4, W: popupWidth, H: float32(rows*popupRowH) + 8}, first, true
}

func (e *Editor) drawPopup(p *ui.Painter, origin ui.Rect) {
	r, first, ok := e.popupRect()
	if !ok {
		return
	}
	r.X += origin.X
	r.Y += origin.Y
	t := e.env.Theme
	p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 3, W: r.W, H: r.H}, ui.RGBA(0, 0, 0, 0.12), 8)
	p.Fill(r, t.Background, 8)
	p.Stroke(r, t.Border, 8, 1)
	f := e.env.Font
	f.Size *= 0.85
	hint := f
	hint.Size *= 0.85
	items := e.popup.items
	for i := first; i < len(items) && i < first+popupRows; i++ {
		row := ui.Rect{X: r.X + 4, Y: r.Y + 4 + float32(i-first)*popupRowH, W: r.W - 8, H: popupRowH}
		if i == e.popup.index {
			p.Fill(row, t.Accent.Alpha(0.15), 6)
		}
		DrawLabelIn(p, row.X+10, row, items[i].Title, f, t.Text)
		if items[i].Hint != "" {
			DrawLabelIn(p, row.X+row.W-80, row, items[i].Hint, hint, t.TextMuted)
		}
	}
}

// popupClick chooses the entry under (x, y); it reports whether the point
// was in the popup.
func (e *Editor) popupClick(x, y float32) bool {
	if e.popup == nil {
		return false
	}
	r, first, ok := e.popupRect()
	if !ok || x < r.X || x > r.X+r.W || y < r.Y || y > r.Y+r.H {
		e.popup = nil
		return false
	}
	i := first + int((y-r.Y-4)/popupRowH)
	if i >= 0 && i < len(e.popup.items) {
		e.choose(e.popup.items[i])
	}
	return true
}
