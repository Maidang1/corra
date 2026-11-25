package editor

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// box is a laid-out node. Positions are relative to the parent's box, so a
// box is reused whole while its node (immutable) and width stay the same:
// an edit lays out only the path from the root to what changed.
type box struct {
	n      *doc.Node
	x, y   float32
	w, h   float32
	before float32
	after  float32
	kids   []*box

	// Text blocks.
	text   *ui.TextLayout
	look   TextLook
	objs   map[int]InlineLook // inline atoms by offset
	belowH float32
	below  func(p *ui.Painter, r ui.Rect, env *Env)

	// Containers.
	cont    ContainerLook
	markers []*ui.TextLayout

	// Block atoms.
	atom AtomLook
}

func (b *box) isText() bool { return b.n.Type().IsTextBlock() }

// textX is where a text block's text starts, from its left edge.
func (b *box) textX() float32 { return b.look.PadX + b.look.Gutter }

// layouter lays out nodes, keeping boxes by node pointer.
type layouter struct {
	env   *Env
	views map[string]ViewSpec
	cache map[*doc.Node]*box
	next  map[*doc.Node]*box
}

func (l *layouter) view(name string) ViewSpec { return l.views[name] }

// layout returns the box of root at width, reusing every box whose node
// and width are unchanged, and drops boxes of nodes no longer shown.
func (l *layouter) layout(root *doc.Node, width float32) *box {
	l.next = make(map[*doc.Node]*box, len(l.cache))
	b := l.node(root, width)
	l.cache, l.next = l.next, nil
	return b
}

func (l *layouter) keep(b *box) {
	l.next[b.n] = b
	for _, k := range b.kids {
		l.keep(k)
	}
}

func (l *layouter) node(n *doc.Node, width float32) *box {
	if b, ok := l.cache[n]; ok && b.w == width {
		l.keep(b)
		return b
	}
	b := &box{n: n, w: width}
	t := n.Type()
	switch {
	case t.IsTextBlock():
		l.text(b, width)
	case t.IsContainer():
		l.container(b, width)
	default:
		l.blockAtom(b, width)
	}
	l.next[n] = b
	return b
}

func (l *layouter) text(b *box, width float32) {
	look := bodyLook(l.env)
	if v := l.view(b.n.Type().Name()); v.Text != nil {
		look = v.Text(b.n, l.env)
	}
	b.look = look
	b.before, b.after = look.Before, look.After
	spans, objs := l.spans(b.n.Inline(), look)
	b.objs = objs
	b.text = ui.ShapeRichTextWith(spans, look.Font, ui.TextShape{Width: max(1, width-2*look.PadX-look.Gutter)})
	_, h := b.text.Size()
	if h == 0 || b.n.TextLen() == 0 {
		h = max(h, lineHeight(look.Font))
	}
	b.h = h + 2*look.PadY
	if look.Below != nil {
		b.belowH, b.below = look.Below(width)
		b.h += b.belowH
	}
}

func lineHeight(f ui.Font) float32 {
	lh := f.LineHeight
	if lh == 0 {
		lh = 1.25
	}
	return f.Size * lh
}

// spans turns content into styled spans; inline atoms become their text
// or an object box, one UTF-16 unit either way, so offsets match the
// model's.
func (l *layouter) spans(in doc.Inline, look TextLook) ([]ui.Span, map[int]InlineLook) {
	var spans []ui.Span
	var objs map[int]InlineLook
	off := 0
	for i := 0; i < in.NumSegments(); i++ {
		seg := in.Segment(i)
		s := ui.Span{Color: look.Color}
		for _, m := range seg.Marks {
			if v := l.view(m.Type.Name()); v.Mark != nil {
				v.Mark(m, l.env, &s)
			}
		}
		if seg.Atom != nil {
			il := InlineLook{Text: string(doc.ObjectChar)}
			if v := l.view(seg.Atom.Type().Name()); v.Inline != nil {
				il = v.Inline(seg.Atom, l.env)
			}
			if il.Text != "" {
				s.Text = il.Text
			} else {
				s.Text = string(doc.ObjectChar)
				s.ObjectWidth, s.ObjectHeight, s.ObjectDescent = max(il.Width, 1), il.Height, il.Descent
				if objs == nil {
					objs = map[int]InlineLook{}
				}
				objs[off] = il
			}
			off++
		} else {
			s.Text = seg.Text
			off += doc.UTF16Len(seg.Text)
		}
		spans = append(spans, s)
	}
	return spans, objs
}

func (l *layouter) container(b *box, width float32) {
	look := ContainerLook{}
	if v := l.view(b.n.Type().Name()); v.Container != nil {
		look = v.Container(b.n, l.env)
	}
	b.cont = look
	b.before, b.after = look.Before, look.After
	inner := max(1, width-look.Indent)
	if look.Horizontal && b.n.NumChildren() > 0 {
		cw := inner / float32(b.n.NumChildren())
		for i := 0; i < b.n.NumChildren(); i++ {
			k := l.node(b.n.Child(i), cw)
			k.x, k.y = look.Indent+float32(i)*cw, 0
			b.h = max(b.h, k.h)
			b.kids = append(b.kids, k)
		}
		return
	}
	y := float32(0)
	var prevAfter float32
	for i := 0; i < b.n.NumChildren(); i++ {
		k := l.node(b.n.Child(i), inner)
		if i > 0 {
			y += max(prevAfter, k.before)
		}
		k.x, k.y = look.Indent, y
		y += k.h
		prevAfter = k.after
		b.kids = append(b.kids, k)
		if look.Marker != nil {
			var marker *ui.TextLayout
			if m := look.Marker(i, b.n.Child(i)); m != "" {
				f := l.env.Font
				marker = ui.ShapeText(m, f, 0)
			}
			b.markers = append(b.markers, marker)
		}
	}
	if len(b.kids) > 0 {
		b.after = max(b.after, prevAfter)
	}
	b.h = y
}

func (l *layouter) blockAtom(b *box, width float32) {
	look := AtomLook{
		Height: func(float32) float32 { return lineHeight(l.env.Font) },
		Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
			p.Fill(r, env.Theme.Surface, 4)
		},
	}
	if v := l.view(b.n.Type().Name()); v.Atom != nil {
		look = v.Atom(b.n, l.env)
	}
	b.atom = look
	b.h = look.Height(width)
	b.after = l.env.Font.Size * 0.6
}

// place is a box with its position in the editor.
type place struct {
	b    *box
	x, y float32
}

// placesTo returns the boxes from the root down to node id, with their
// positions, following the document's ancestors.
func (e *Editor) placesTo(id doc.NodeID) []place {
	anc := e.state.Ancestors(id)
	if len(anc) == 0 || e.root == nil {
		return nil
	}
	pad := e.cfg.Padding
	out := []place{{e.root, pad, pad}}
	cur := out[0]
	for i := 1; i < len(anc); i++ {
		j := anc[i-1].ChildIndex(anc[i].ID())
		if j < 0 || j >= len(cur.b.kids) {
			return nil
		}
		k := cur.b.kids[j]
		cur = place{k, cur.x + k.x, cur.y + k.y}
		out = append(out, cur)
	}
	return out
}

// leafPlace returns the box of text block id and its position.
func (e *Editor) leafPlace(id doc.NodeID) (place, bool) {
	ps := e.placesTo(id)
	if len(ps) == 0 || !ps[len(ps)-1].b.isText() {
		return place{}, false
	}
	return ps[len(ps)-1], true
}

// caretRect returns the caret at p in editor coordinates.
func (e *Editor) caretRect(p doc.Point) (ui.Rect, bool) {
	pl, ok := e.leafPlace(p.Leaf)
	if !ok {
		return ui.Rect{}, false
	}
	b := pl.b
	r := b.text.Caret(p.Offset)
	if r.H == 0 {
		r.H = lineHeight(b.look.Font)
	}
	r.X += pl.x + b.textX()
	r.Y += pl.y + b.look.PadY
	return r, true
}

// hit returns the place in the document under (x, y), in editor
// coordinates: a point in a text block, or a block atom.
func (e *Editor) hit(x, y float32) (doc.Point, *doc.Node, bool) {
	if e.root == nil {
		return doc.Point{}, nil, false
	}
	pad := e.cfg.Padding
	b, bx, by := e.root, pad, pad
	for !b.isText() {
		if len(b.kids) == 0 {
			if b.n.Type().IsAtom() {
				return doc.Point{}, b.n, true
			}
			return doc.Point{}, nil, false
		}
		// The last child starting above y (left of x, side by side), or
		// the first.
		k := b.kids[0]
		for _, c := range b.kids {
			if b.cont.Horizontal {
				if bx+c.x <= x {
					k = c
				}
			} else if by+c.y <= y {
				k = c
			}
		}
		// Below a child but above the next: the nearer one.
		if !b.cont.Horizontal && y > by+k.y+k.h {
			idx := 0
			for i, c := range b.kids {
				if c == k {
					idx = i
				}
			}
			if idx+1 < len(b.kids) {
				next := b.kids[idx+1]
				if by+next.y-y < y-(by+k.y+k.h) {
					k = next
				}
			}
		}
		b, bx, by = k, bx+k.x, by+k.y
	}
	tx := x - bx - b.textX()
	ty := min(max(y-by-b.look.PadY, 0), max(b.h-b.belowH-2*b.look.PadY-1, 0))
	return doc.Point{Leaf: b.n.ID(), Offset: b.text.IndexAt(ui.Point{X: tx, Y: ty})}, nil, true
}
