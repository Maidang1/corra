package editor

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// formatButton is one button of the bar shown over a text selection.
type formatButton struct {
	label, command, mark string
}

var formatButtons = []formatButton{
	{"B", "bold", "bold"}, {"I", "italic", "italic"}, {"S", "strikethrough", "strike"},
	{"</>", "code", "code"}, {"==", "highlight", "highlight"}, {"🔗", "link", "link"},
}

const (
	formatButtonW = 34
	formatBarH    = 32
)

// formatBar returns the bar's buttons that apply and its box in editor
// coordinates, above the selection's start; ok is false when there is no
// text selection, or while dragging.
func (e *Editor) formatBar() (ui.Rect, []formatButton, bool) {
	sel, ok := e.state.TextSel()
	if !ok || sel.Empty() || e.drag != nil || e.popup != nil {
		return ui.Rect{}, nil, false
	}
	// Which buttons apply is worked out once per document and selection,
	// not every frame: Can runs the command.
	if e.barCache.doc == e.state.Doc && e.barCache.sel == sel {
		return e.barRect(sel, e.barCache.buttons)
	}
	var buttons []formatButton
	for _, b := range formatButtons {
		if b.command == "link" {
			if e.cfg.OnLink != nil {
				buttons = append(buttons, b)
			}
			continue
		}
		if _, ok := e.cfg.Commands[b.command]; ok && e.Can(b.command) {
			buttons = append(buttons, b)
		}
	}
	e.barCache.doc, e.barCache.sel, e.barCache.buttons = e.state.Doc, sel, buttons
	return e.barRect(sel, buttons)
}

func (e *Editor) barRect(sel doc.TextSelection, buttons []formatButton) (ui.Rect, []formatButton, bool) {
	if len(buttons) == 0 {
		return ui.Rect{}, nil, false
	}
	from, _ := e.state.Ordered(sel)
	c, ok := e.caretRect(from)
	if !ok {
		return ui.Rect{}, nil, false
	}
	w := float32(len(buttons))*formatButtonW + 8
	y := c.Y - formatBarH - 6
	if y < 0 {
		y = c.Y + c.H + 6
	}
	return ui.Rect{X: max(0, c.X-8), Y: y, W: w, H: formatBarH}, buttons, true
}

func (e *Editor) drawFormatBar(p *ui.Painter, origin ui.Rect) {
	r, buttons, ok := e.formatBar()
	if !ok || !e.focused {
		return
	}
	r.X += origin.X
	r.Y += origin.Y
	t := e.env.Theme
	p.Fill(ui.Rect{X: r.X + 1, Y: r.Y + 2, W: r.W, H: r.H}, ui.RGBA(0, 0, 0, 0.12), 8)
	p.Fill(r, t.Background, 8)
	p.Stroke(r, t.Border, 1, 8)
	sel, _ := e.state.TextSel()
	size := e.env.Font.Size * 0.8
	for i, b := range buttons {
		x := r.X + 4 + float32(i)*formatButtonW
		if mt := e.cfg.Registry.Mark(b.mark); mt != nil && e.state.RangeHasMark(sel, mt) {
			p.Fill(ui.Rect{X: x + 2, Y: r.Y + 4, W: formatButtonW - 4, H: r.H - 8}, t.Accent.Alpha(0.18), 5)
		}
		p.Text(x+(formatButtonW-float32(len([]rune(b.label)))*size*0.6)/2, r.Y+r.H*0.68, b.label, size, t.Text)
	}
}

// formatClick runs the button under (x, y); it reports whether the point
// was on the bar.
func (e *Editor) formatClick(x, y float32) bool {
	r, buttons, ok := e.formatBar()
	if !ok || x < r.X || x > r.X+r.W || y < r.Y || y > r.Y+r.H {
		return false
	}
	i := int((x - r.X - 4) / formatButtonW)
	if i < 0 || i >= len(buttons) {
		return true
	}
	if buttons[i].command == "link" {
		e.cfg.OnLink()
		return true
	}
	e.Command(buttons[i].command)
	return true
}
