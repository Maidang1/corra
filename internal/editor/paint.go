package editor

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// draw paints the visible part of the document, its selection, search
// matches and caret. r is the surface's box in the window.
func (e *Editor) draw(p *ui.Painter, r ui.Rect) {
	if e.root == nil {
		return
	}
	vis := p.Visible()
	sel := e.selectionRanges()
	matches := e.matchRanges()
	var nodeSel map[doc.NodeID]bool
	if ns, ok := e.state.Sel.(doc.NodeSelection); ok {
		nodeSel = map[doc.NodeID]bool{}
		for _, id := range ns.IDs {
			nodeSel[id] = true
		}
	}
	var walk func(b *box, x, y float32)
	walk = func(b *box, x, y float32) {
		ax, ay := r.X+x, r.Y+y
		if ay > vis.Y+vis.H || ay+b.h < vis.Y {
			return
		}
		rect := ui.Rect{X: ax, Y: ay, W: b.w, H: b.h}
		switch {
		case b.isText():
			e.drawText(p, b, ax, ay, sel[b.n.ID()], matches[b.n.ID()])
		case b.n.Type().IsContainer():
			if b.cont.Paint != nil {
				b.cont.Paint(p, rect, &e.env)
			}
			for i, k := range b.kids {
				walk(k, x+k.x, y+k.y)
				e.drawMarker(p, b, i, ax, ay+k.y, k)
			}
		default:
			b.atom.Paint(p, rect, &e.env)
		}
		if nodeSel[b.n.ID()] {
			p.Stroke(ui.Rect{X: ax - 2, Y: ay - 2, W: b.w + 4, H: b.h + 4}, e.env.Theme.Accent, 4, 2)
		}
	}
	walk(e.root, e.cfg.Padding, e.cfg.Padding)
	e.dimOthers(p, r)
	e.drawHandles(p, r)
	e.drawCaret(p, r)
	e.drawPopup(p, r)
	e.drawFormatBar(p, r)
}

func (e *Editor) drawText(p *ui.Painter, b *box, x, y float32, sel, matches []doc.LeafRange) {
	look := b.look
	if look.Background.A > 0 {
		p.Fill(ui.Rect{X: x, Y: y, W: b.w, H: b.h - b.belowH}, look.Background, look.Radius)
	}
	tx, ty := x+b.textX(), y+look.PadY
	for _, m := range matches {
		for _, rr := range b.text.SelectionRects(ui.TextInputRange{Start: m.From, End: m.To}) {
			p.Fill(ui.Rect{X: tx + rr.X, Y: ty + rr.Y, W: rr.W, H: rr.H}, ui.RGBA(255, 200, 0, 0.35), 2)
		}
	}
	for _, s := range sel {
		rects := b.text.SelectionRects(ui.TextInputRange{Start: s.From, End: s.To})
		if s.From == s.To || len(rects) == 0 {
			// A block the selection passes through but has no text: a
			// sliver, so the selection reads as continuous.
			c := b.text.Caret(s.From)
			if c.H == 0 {
				c.H = lineHeight(look.Font)
			}
			rects = []ui.Rect{{X: c.X, Y: c.Y, W: look.Font.Size * 0.4, H: c.H}}
		}
		for _, rr := range rects {
			p.Fill(ui.Rect{X: tx + rr.X, Y: ty + rr.Y, W: rr.W, H: rr.H}, e.env.Theme.Selection, 0)
		}
	}
	p.TextLayout(b.text, tx, ty, look.Color)
	for _, m := range e.misspelled(b.n) {
		for _, rr := range b.text.SelectionRects(m) {
			// A dotted red line under the word.
			for x := rr.X; x < rr.X+rr.W; x += 3 {
				p.Fill(ui.Rect{X: tx + x, Y: ty + rr.Y + rr.H - 2, W: 1.5, H: 1.5}, e.env.Theme.Danger, 0)
			}
		}
	}
	if look.Decorate != nil {
		look.Decorate(p, ui.Point{X: tx, Y: ty}, b.text.Lines(), b.n.Inline().String(), &e.env)
	}
	if b.below != nil {
		b.below(p, ui.Rect{X: x, Y: y + b.h - b.belowH, W: b.w, H: b.belowH}, &e.env)
	}
	if len(b.objs) > 0 {
		for _, o := range b.text.Objects() {
			if il, ok := b.objs[o.Index]; ok && il.Paint != nil {
				il.Paint(p, ui.Rect{X: tx + o.Rect.X, Y: ty + o.Rect.Y, W: o.Rect.W, H: o.Rect.H}, &e.env)
			}
		}
	}
	// The composition's underline.
	if e.preBase != nil && e.preLen > 0 {
		start := e.state.GlobalOffset(doc.Point{Leaf: b.n.ID()})
		a, z := max(e.preStart, start)-start, min(e.preStart+e.preLen, start+b.n.TextLen())-start
		if a < z {
			for _, rr := range b.text.SelectionRects(ui.TextInputRange{Start: a, End: z}) {
				p.Fill(ui.Rect{X: tx + rr.X, Y: ty + rr.Y + rr.H - 1.5, W: rr.W, H: 1.5}, look.Color, 0)
			}
		}
	}
}

// drawMarker draws a list item's bullet, number or checkbox left of its
// first line, on that line's baseline.
func (e *Editor) drawMarker(p *ui.Painter, b *box, i int, x, y float32, k *box) {
	size := e.env.Font.Size
	base, mid := firstLine(k)
	base, mid = y+base, y+mid
	right := x + k.x - size*0.5
	t := e.env.Theme
	if b.cont.Check != nil {
		if task, checked := b.cont.Check(i, k.n); task {
			r := e.checkRect(x+k.x, y, k)
			d := r.W
			if checked {
				p.Fill(r, t.Accent, d*0.25)
				p.StrokePolyline([][2]float32{
					{r.X + d*0.26, r.Y + d*0.52},
					{r.X + d*0.43, r.Y + d*0.69},
					{r.X + d*0.75, r.Y + d*0.33},
				}, max(1.5, d*0.12), t.AccentText)
			} else {
				p.Stroke(r, t.TextMuted.Alpha(0.6), d*0.25, 1.5)
			}
			return
		}
	}
	if i >= len(b.markers) {
		return
	}
	switch m := b.markers[i]; {
	case m.dot:
		d := size * 0.36
		p.Fill(ui.Rect{X: right - d - size*0.1, Y: mid - d/2, W: d, H: d}, t.TextMuted, d/2)
	case m.text != nil:
		w, _ := m.text.Size()
		var mb float32
		if lines := m.text.Lines(); len(lines) > 0 {
			mb = lines[0].Baseline
		}
		p.TextLayout(m.text, right-w, base-mb, t.TextMuted)
	}
}

// checkRect returns the checkbox of task item k, whose box is at (x, y).
func (e *Editor) checkRect(x, y float32, k *box) ui.Rect {
	d := e.env.Font.Size * 0.95
	_, mid := firstLine(k)
	return ui.Rect{X: x - e.env.Font.Size*0.5 - d, Y: y + mid - d/2, W: d, H: d}
}

// firstLine returns, from b's top, the baseline of the first line of text
// in b and that line's middle: about where a lowercase letter's center
// and a Han character's are.
func firstLine(b *box) (baseline, mid float32) {
	y := float32(0)
	for !b.isText() {
		if len(b.kids) == 0 {
			return y + b.h*0.6, y + b.h/2 // an atom, as a picture
		}
		y += b.kids[0].y
		b = b.kids[0]
	}
	y += b.look.PadY
	size := b.look.Font.Size
	baseline = y + lineHeight(b.look.Font)*0.75
	if lines := b.text.Lines(); len(lines) > 0 {
		baseline = y + lines[0].Baseline
	}
	return baseline, baseline - size*0.33
}

// DrawLabel draws one line of s in font f, its left at x and its baseline
// at baseline, and returns its width.
func DrawLabel(p *ui.Painter, x, baseline float32, s string, f ui.Font, c ui.Color) float32 {
	l := ui.ShapeText(s, f, 0)
	w, _ := l.Size()
	var b float32
	if lines := l.Lines(); len(lines) > 0 {
		b = lines[0].Baseline
	}
	p.TextLayout(l, x, baseline-b, c)
	return w
}

// DrawLabelIn draws one line of s in font f, its left at x, centered in
// r from top to bottom, and returns its width.
func DrawLabelIn(p *ui.Painter, x float32, r ui.Rect, s string, f ui.Font, c ui.Color) float32 {
	l := ui.ShapeText(s, f, 0)
	w, h := l.Size()
	p.TextLayout(l, x, r.Y+(r.H-h)/2, c)
	return w
}

func (e *Editor) drawCaret(p *ui.Painter, r ui.Rect) {
	sel, ok := e.state.Sel.(doc.TextSelection)
	if !ok || !sel.Empty() || !e.focused {
		return
	}
	c, ok := e.caretRect(sel.Head)
	if !ok {
		return
	}
	p.Fill(ui.Rect{X: r.X + c.X, Y: r.Y + c.Y, W: max(c.W, 1.5), H: c.H}, e.env.Theme.Accent, 0)
}

// selectionRanges returns the selected part of each text block.
func (e *Editor) selectionRanges() map[doc.NodeID][]doc.LeafRange {
	sel, ok := e.state.TextSel()
	if !ok || sel.Empty() {
		return nil
	}
	out := map[doc.NodeID][]doc.LeafRange{}
	for _, r := range e.state.LeafRanges(sel) {
		out[r.Leaf] = append(out[r.Leaf], r)
	}
	return out
}

// dimOthers veils every top-level block but the caret's, in focus mode.
func (e *Editor) dimOthers(p *ui.Painter, r ui.Rect) {
	sel, ok := e.state.TextSel()
	if !e.focusMode || !ok {
		return
	}
	anc := e.state.Ancestors(sel.Head.Leaf)
	if len(anc) < 2 {
		return
	}
	current := anc[1].ID()
	veil := e.env.Theme.Background.Alpha(0.65)
	pad := e.cfg.Padding
	for _, k := range e.root.kids {
		if k.n.ID() != current {
			p.Fill(ui.Rect{X: r.X + pad + k.x - 4, Y: r.Y + pad + k.y - 2, W: k.w + 8, H: k.h + 4}, veil, 0)
		}
	}
}
