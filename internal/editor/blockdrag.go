package editor

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// blockDrag is a top-level block being dragged by its handle.
type blockDrag struct {
	id doc.NodeID
	y  float32 // the pointer, in editor coordinates
}

const handleW = 14

// topBlockAt returns the index of the top-level block at y, or -1.
func (e *Editor) topBlockAt(y float32) int {
	if e.root == nil {
		return -1
	}
	pad := e.cfg.Padding
	for i, k := range e.root.kids {
		if y >= pad+k.y-k.before && y < pad+k.y+k.h+k.after {
			return i
		}
	}
	return -1
}

// handleRect returns the drag handle of top-level block i.
func (e *Editor) handleRect(i int) ui.Rect {
	k := e.root.kids[i]
	pad := e.cfg.Padding
	h := min(firstLineHeight(k), 26)
	return ui.Rect{X: pad - handleW - 6, Y: pad + k.y, W: handleW, H: h}
}

// dropIndex returns where a dragged block lands for the pointer at y: the
// index of the first top-level block whose middle is below it.
func (e *Editor) dropIndex(y float32) int {
	pad := e.cfg.Padding
	for i, k := range e.root.kids {
		if y < pad+k.y+k.h/2 {
			return i
		}
	}
	return len(e.root.kids)
}

// handleDown starts dragging a block when the press is on its handle.
func (e *Editor) handleDown(x, y float32) bool {
	i := e.topBlockAt(y)
	if i < 0 {
		return false
	}
	r := e.handleRect(i)
	if x < r.X-4 || x > r.X+r.W+4 || y < r.Y || y > r.Y+r.H {
		return false
	}
	e.blockDrag = &blockDrag{id: e.root.kids[i].n.ID(), y: y}
	return true
}

// handleUp drops the dragged block.
func (e *Editor) handleUp() {
	d := e.blockDrag
	e.blockDrag = nil
	if d == nil {
		return
	}
	to := e.dropIndex(d.y)
	e.Run(doc.MoveBlock(d.id, e.state.Doc.ID(), to), "structure")
}

// drawHandles paints the hovered block's handle, and where a dragged
// block would land.
func (e *Editor) drawHandles(p *ui.Painter, origin ui.Rect) {
	if e.root == nil || !e.hovering {
		return
	}
	t := e.env.Theme
	if d := e.blockDrag; d != nil {
		pad := e.cfg.Padding
		i := e.dropIndex(d.y)
		y := pad + e.root.h
		if i < len(e.root.kids) {
			k := e.root.kids[i]
			y = pad + k.y - k.before/2
		}
		p.Fill(ui.Rect{X: origin.X + pad, Y: origin.Y + y - 1, W: e.root.w, H: 2}, t.Accent, 1)
		return
	}
	i := e.topBlockAt(e.hoverY)
	if i < 0 {
		return
	}
	r := e.handleRect(i)
	r.X += origin.X
	r.Y += origin.Y
	p.Text(r.X, r.Y+r.H*0.7, "⋮⋮", e.env.Font.Size*0.8, t.TextMuted)
}
