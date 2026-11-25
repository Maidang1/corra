package editor

import (
	"runtime"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// The input methods see the linear text: every text block's content and
// one separator after it, in document order, counted in UTF-16 units.

func (e *Editor) linear(a, b int) string {
	s := e.state
	size := s.Doc.Size()
	a, b = max(0, min(a, size)), max(0, min(b, size))
	if a >= b {
		return ""
	}
	var out strings.Builder
	leaf := s.Node(s.PointAt(a).Leaf)
	for leaf != nil {
		start := s.GlobalOffset(doc.Point{Leaf: leaf.ID()})
		if start >= b {
			break
		}
		n := leaf.TextLen()
		from, to := max(a, start)-start, min(b, start+n)-start
		if from < to {
			out.WriteString(leaf.Inline().Slice(from, to).String())
		}
		if sep := start + n; sep >= a && sep < b {
			out.WriteByte('\n')
		}
		leaf = s.NextLeaf(leaf.ID())
	}
	return out.String()
}

func (e *Editor) point(g int) doc.Point  { return e.state.PointAt(g) }
func (e *Editor) global(p doc.Point) int { return e.state.GlobalOffset(p) }

// TextForRange implements ui.TextInputClient.
func (e *Editor) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	size := e.state.Doc.Size()
	r.Start, r.End = max(0, min(r.Start, size)), max(0, min(r.End, size))
	if r.End < r.Start {
		r.End = r.Start
	}
	return e.linear(r.Start, r.End), r
}

// Selection implements ui.TextInputClient.
func (e *Editor) Selection() ui.TextInputSelection {
	sel, ok := e.state.TextSel()
	if !ok {
		return ui.TextInputSelection{}
	}
	a, h := e.global(sel.Anchor), e.global(sel.Head)
	return ui.TextInputSelection{Range: ui.TextInputRange{Start: min(a, h), End: max(a, h)}, Reversed: h < a}
}

// MarkedRange implements ui.TextInputClient.
func (e *Editor) MarkedRange() (ui.TextInputRange, bool) {
	if e.preBase == nil {
		return ui.TextInputRange{}, false
	}
	return ui.TextInputRange{Start: e.preStart, End: e.preStart + e.preLen}, true
}

func (e *Editor) selectionRange() ui.TextInputRange { return e.Selection().Range }

// ReplaceText implements ui.TextInputClient: typing, and the end of a
// composition.
func (e *Editor) ReplaceText(r *ui.TextInputRange, text string) {
	if e.preBase != nil {
		e.commitComposition(text)
		return
	}
	want := e.selectionRange()
	if r != nil {
		want = *r
	}
	e.state.Sel = doc.TextSelection{Anchor: e.point(want.Start), Head: e.point(want.End)}
	e.typeText(text)
}

// SetMarkedText implements ui.TextInputClient. The composition is applied
// to the document from before it began, outside the history, each time it
// changes.
func (e *Editor) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	if e.preBase == nil {
		want := e.selectionRange()
		if r != nil {
			want = *r
		}
		base := e.state
		e.preBase = &base
		e.preStart, e.preOrig = want.Start, want.End-want.Start
	}
	base := *e.preBase
	base.Sel = doc.TextSelection{Anchor: base.PointAt(e.preStart), Head: base.PointAt(e.preStart + e.preOrig)}
	after, _, ok := doc.Run(base, doc.InsertText(text), doc.Meta{})
	if !ok {
		return
	}
	e.state = after
	e.preLen = doc.UTF16Len(text)
	e.state.Sel = doc.TextSelection{Anchor: e.point(e.preStart + selected.Start), Head: e.point(e.preStart + selected.End)}
	e.relayout()
	e.redraw()
}

// UnmarkText implements ui.TextInputClient: the composed text stays.
func (e *Editor) UnmarkText() {
	if e.preBase == nil {
		return
	}
	e.commitComposition(e.linear(e.preStart, e.preStart+e.preLen))
}

// endComposition keeps a composition in progress before another change.
func (e *Editor) endComposition() { e.UnmarkText() }

func (e *Editor) commitComposition(text string) {
	base := *e.preBase
	base.Sel = doc.TextSelection{Anchor: base.PointAt(e.preStart), Head: base.PointAt(e.preStart + e.preOrig)}
	e.preBase, e.preLen, e.preOrig = nil, 0, 0
	e.state = base
	if text == "" && base.Sel.(doc.TextSelection).Empty() {
		e.relayout()
		e.redraw()
		return
	}
	e.typeText(text)
}

// BoundsForRange implements ui.TextInputClient.
func (e *Editor) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	c, ok := e.caretRect(e.point(r.Start))
	return c, ui.TextInputRange{Start: r.Start, End: r.Start}, ok
}

// IndexForPoint implements ui.TextInputClient.
func (e *Editor) IndexForPoint(p ui.Point) (int, bool) {
	pt, atom, ok := e.hit(p.X, p.Y)
	if !ok || atom != nil {
		return 0, false
	}
	return e.global(pt), true
}

// typeText inserts typed text, then applies the first input rule the text
// before the caret matches, as a change of its own: Backspace right after
// undoes the rule and leaves the typed text.
func (e *Editor) typeText(text string) {
	e.ruleUndo = false
	if !e.run(doc.InsertText(text), "input") {
		return
	}
	e.updatePopup(text)
	sel, ok := e.state.TextSel()
	if !ok || len(e.cfg.InputRules) == 0 || text == "" {
		return
	}
	leaf := e.state.Node(sel.Head.Leaf)
	before := leaf.Inline().Slice(0, sel.Head.Offset).String()
	for _, rule := range e.cfg.InputRules {
		if m := rule.Match(before); m != nil {
			if e.run(func(s doc.State, tx *doc.Tx) bool { return rule.Handle(s, tx, m) }, "rule") {
				e.ruleUndo = true
				return
			}
		}
	}
}

func isMac() bool { return runtime.GOOS == "darwin" }

// wordMod is the modifier that moves by words: Option on macOS, Control
// elsewhere.
func wordMod() ui.Modifiers {
	if isMac() {
		return ui.Alt
	}
	return ui.Ctrl
}

// input handles keys, commands and the pointer.
func (e *Editor) input(ev ui.InputEvent) bool {
	if e.root == nil {
		return false
	}
	switch ev.Kind {
	case ui.InputKeyDown:
		return e.key(ev)
	case ui.InputCommand:
		return e.command(ev.Text)
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		if e.handleDown(ev.X, ev.Y) {
			return true
		}
		e.pointerDown(ev)
		return true
	case ui.InputPointerMove:
		e.hovering, e.hoverY = true, ev.Y
		if e.blockDrag != nil {
			e.blockDrag.y = ev.Y
			e.redraw()
			return true
		}
		if e.drag != nil {
			e.pointerDrag(ev)
			return true
		}
		e.redraw()
	case ui.InputPointerUp:
		e.drag = nil
		e.handleUp()
		return true
	}
	return false
}

func (e *Editor) command(name string) bool {
	switch name {
	case "copy":
		e.Copy()
	case "cut":
		e.Cut()
	case "paste", "pasteAndMatchStyle":
		e.Paste()
	case "selectAll":
		e.Run(doc.SelectAll, "select")
	case "undo":
		e.Undo()
	case "redo":
		e.Redo()
	case "delete":
		e.Run(doc.DeleteSelection, "delete")
	default:
		return e.Command(name)
	}
	return true
}

func (e *Editor) key(ev ui.InputEvent) bool {
	if e.popupKey(ev) {
		return true
	}
	defer func() {
		if e.popup != nil {
			e.updatePopup("")
		}
	}()
	mods := ev.Mods
	shift := mods&ui.Shift != 0
	plain := mods &^ ui.Shift
	for _, k := range e.cfg.Keys {
		if k.Key == ev.Key && k.Mods == mods {
			e.ruleUndo = false
			return e.Command(k.Command)
		}
	}
	if plain == ui.Cmd {
		switch ev.Key {
		case ui.KeyZ:
			if shift {
				e.Redo()
			} else {
				e.Undo()
			}
			return true
		case ui.KeyY:
			if !isMac() {
				e.Redo()
				return true
			}
		case ui.KeyA:
			return e.command("selectAll")
		case ui.KeyC:
			return e.command("copy")
		case ui.KeyX:
			return e.command("cut")
		case ui.KeyV:
			return e.command("paste")
		}
	}
	switch ev.Key {
	case ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown, ui.KeyHome, ui.KeyEnd, ui.KeyPageUp, ui.KeyPageDown:
		e.ruleUndo = false
		e.navigate(ev.Key, plain, shift)
		return true
	case ui.KeyEnter:
		e.ruleUndo = false
		if shift {
			e.insertBreak()
			return true
		}
		e.Run(e.chain("Enter", doc.SplitBlock), "structure")
		return true
	case ui.KeyBackspace, ui.KeyDelete:
		if e.ruleUndo && ev.Key == ui.KeyBackspace {
			e.ruleUndo = false
			e.Undo()
			return true
		}
		e.deleteKey(ev.Key == ui.KeyBackspace, plain == wordMod())
		return true
	case ui.KeyTab:
		e.ruleUndo = false
		name := "Tab"
		if shift {
			name = "Shift-Tab"
		}
		if !e.Run(e.chain(name, nil), "structure") && !shift {
			if sel, ok := e.state.TextSel(); ok && e.state.Node(sel.Head.Leaf).Type().Content() == doc.ContentText {
				e.typeText("\t")
			}
		}
		return true
	case ui.KeyEscape:
		if sel, ok := e.state.TextSel(); ok && !sel.Empty() {
			e.SetSelection(doc.Caret(sel.Head))
			return true
		}
	}
	return false
}

// chain returns the extensions' handlers for key name, then fallback.
func (e *Editor) chain(name string, fallback doc.Command) doc.Command {
	var cmds []doc.Command
	if c, ok := e.cfg.Chains[name]; ok {
		cmds = append(cmds, c)
	}
	if fallback != nil {
		cmds = append(cmds, fallback)
	}
	return doc.FirstOf(cmds...)
}

// insertBreak inserts a line break inside the block, as Shift+Enter.
func (e *Editor) insertBreak() {
	hb := e.cfg.Registry.Node("hard_break")
	e.Run(func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.TextSel()
		if !ok {
			return false
		}
		leaf := s.Node(sel.Head.Leaf)
		if hb == nil || !leaf.Type().Allows(hb) {
			return doc.InsertText("\n")(s, tx) && leaf.Type().Content() == doc.ContentText
		}
		if !sel.Empty() {
			doc.DeleteSelection(s, tx)
		}
		p := tx.State().Sel.(doc.TextSelection).Head
		tx.ReplaceText(p.Leaf, p.Offset, p.Offset, doc.AtomInline(doc.NewAtom(hb, nil)))
		tx.SetSelection(doc.Caret(doc.Point{Leaf: p.Leaf, Offset: p.Offset + 1}))
		return true
	}, "input")
}

// deleteKey deletes the selection, or the grapheme or word before or
// after the caret; at a block's edge the key's chain joins blocks.
func (e *Editor) deleteKey(backward, word bool) {
	sel, ok := e.state.TextSel()
	if !ok {
		e.Run(doc.DeleteSelection, "delete")
		return
	}
	if !sel.Empty() {
		e.Run(doc.DeleteSelection, "delete")
		return
	}
	p := sel.Head
	leaf := e.state.Node(p.Leaf)
	if (backward && p.Offset == 0) || (!backward && p.Offset == leaf.TextLen()) {
		name := "Delete"
		if backward {
			name = "Backspace"
		}
		e.Run(e.chain(name, nil), "delete")
		return
	}
	to := e.step(p, backward, word)
	from := p
	if backward {
		from, to = to, p
	}
	e.Run(func(s doc.State, tx *doc.Tx) bool {
		tx.ReplaceText(from.Leaf, from.Offset, to.Offset, doc.Inline{})
		tx.SetSelection(doc.Caret(from))
		return true
	}, "delete")
}

// step moves p one grapheme or word within its block.
func (e *Editor) step(p doc.Point, backward, word bool) doc.Point {
	pl, ok := e.leafPlace(p.Leaf)
	if !ok {
		return p
	}
	if word {
		p.Offset = wordBoundary(pl.b.n.Inline().String(), p.Offset, backward)
		return p
	}
	if backward {
		p.Offset = pl.b.text.PreviousBoundary(p.Offset)
	} else {
		p.Offset = pl.b.text.NextBoundary(p.Offset)
	}
	return p
}

// wordBoundary returns the UTF-16 offset of the next word edge from off.
func wordBoundary(text string, off int, backward bool) int {
	type ch struct {
		r   rune
		pos int
	}
	var cs []ch
	pos := 0
	for _, r := range text {
		cs = append(cs, ch{r, pos})
		pos++
		if r >= 0x10000 {
			pos++ // a surrogate pair
		}
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	i := 0
	for i < len(cs) && cs[i].pos < off {
		i++
	}
	if backward {
		j := i - 1
		for j >= 0 && !isWord(cs[j].r) {
			j--
		}
		for j >= 0 && isWord(cs[j].r) {
			j--
		}
		if j+1 < len(cs) {
			return cs[j+1].pos
		}
		return pos
	}
	j := i
	for j < len(cs) && !isWord(cs[j].r) {
		j++
	}
	for j < len(cs) && isWord(cs[j].r) {
		j++
	}
	if j < len(cs) {
		return cs[j].pos
	}
	return pos
}

func (e *Editor) moveTo(p doc.Point, extend bool) {
	sel, ok := e.state.TextSel()
	if extend && ok {
		e.SetSelection(doc.TextSelection{Anchor: sel.Anchor, Head: p})
	} else {
		e.SetSelection(doc.Caret(p))
	}
}

// navigate moves the caret, or extends the selection with Shift.
func (e *Editor) navigate(key ui.Key, mods ui.Modifiers, extend bool) {
	s := e.state
	sel, ok := s.TextSel()
	if !ok {
		// From a selected block, the arrows go to the text around it.
		if first := firstLeafOf(s); first != nil {
			e.SetSelection(doc.Caret(doc.Point{Leaf: first.ID()}))
		}
		return
	}
	head := sel.Head
	keepGoal := false
	var to doc.Point
	switch key {
	case ui.KeyLeft, ui.KeyRight:
		back := key == ui.KeyLeft
		switch {
		case isMac() && mods == ui.Super:
			to = e.lineEdge(head, !back)
		case !extend && !sel.Empty() && mods == 0:
			from, end := s.Ordered(sel)
			to = end
			if back {
				to = from
			}
		default:
			to = e.horizontal(head, back, mods == wordMod())
		}
	case ui.KeyUp, ui.KeyDown:
		if isMac() && mods == ui.Super {
			to = e.docEdge(key == ui.KeyDown)
		} else {
			to = e.vertical(head, key == ui.KeyDown, 1)
			keepGoal = true
		}
	case ui.KeyHome, ui.KeyEnd:
		if mods == ui.Ctrl {
			to = e.docEdge(key == ui.KeyEnd)
		} else {
			to = e.lineEdge(head, key == ui.KeyEnd)
		}
	case ui.KeyPageUp, ui.KeyPageDown:
		lines := max(1, int(e.viewH/lineHeight(e.env.Font))-1)
		to = e.vertical(head, key == ui.KeyPageDown, lines)
		keepGoal = true
	}
	goal := e.goalX
	e.moveTo(to, extend)
	if keepGoal {
		e.goalX = goal
	} else {
		e.goalX = -1
	}
	if keepGoal && e.goalX < 0 {
		if c, ok := e.caretRect(head); ok {
			e.goalX = c.X
		}
	}
}

func firstLeafOf(s doc.State) *doc.Node { return s.FirstLeaf(s.Doc.ID()) }

// horizontal moves one grapheme or word, crossing into the next or
// previous text block at an edge.
func (e *Editor) horizontal(p doc.Point, back, word bool) doc.Point {
	s := e.state
	leaf := s.Node(p.Leaf)
	if back && p.Offset == 0 {
		if prev := s.PrevLeaf(p.Leaf); prev != nil {
			return doc.Point{Leaf: prev.ID(), Offset: prev.TextLen()}
		}
		return p
	}
	if !back && p.Offset == leaf.TextLen() {
		if next := s.NextLeaf(p.Leaf); next != nil {
			return doc.Point{Leaf: next.ID()}
		}
		return p
	}
	return e.step(p, back, word)
}

// vertical moves lines lines up or down, keeping the goal column.
func (e *Editor) vertical(p doc.Point, down bool, lines int) doc.Point {
	c, ok := e.caretRect(p)
	if !ok {
		return p
	}
	x := e.goalX
	if x < 0 {
		x = c.X
		e.goalX = x
	}
	dy := c.H * float32(lines)
	y := c.Y + c.H/2 - dy
	if down {
		y = c.Y + c.H/2 + dy
	}
	if y < e.cfg.Padding {
		return e.docEdge(false)
	}
	if y > e.Height()-e.cfg.Padding {
		return e.docEdge(true)
	}
	to, atom, ok := e.hit(x, y)
	if !ok || atom != nil {
		return p
	}
	if to == p {
		// The hit landed on the same line (a tall block): go on past it.
		if down {
			if next := e.state.NextLeaf(p.Leaf); next != nil && p.Offset == e.state.Node(p.Leaf).TextLen() {
				return doc.Point{Leaf: next.ID()}
			}
		}
	}
	return to
}

// lineEdge returns the start or end of p's visual line.
func (e *Editor) lineEdge(p doc.Point, end bool) doc.Point {
	pl, ok := e.leafPlace(p.Leaf)
	if !ok {
		return p
	}
	for _, l := range pl.b.text.Lines() {
		if p.Offset >= l.Start && p.Offset <= l.End {
			if end {
				p.Offset = l.End
				// A wrapped line's end is the next one's start; stay on this
				// line before its trailing space.
				if l.End < pl.b.n.TextLen() && l.End > l.Start {
					p.Offset = l.End - 1
				}
			} else {
				p.Offset = l.Start
			}
			return p
		}
	}
	if end {
		p.Offset = pl.b.n.TextLen()
	} else {
		p.Offset = 0
	}
	return p
}

func (e *Editor) docEdge(end bool) doc.Point {
	s := e.state
	if end {
		l := s.LastLeaf(s.Doc.ID())
		return doc.Point{Leaf: l.ID(), Offset: l.TextLen()}
	}
	return doc.Point{Leaf: s.FirstLeaf(s.Doc.ID()).ID()}
}

func (e *Editor) pointerDown(ev ui.InputEvent) {
	e.endComposition()
	e.ruleUndo = false
	if e.popupClick(ev.X, ev.Y) || e.formatClick(ev.X, ev.Y) {
		return
	}
	if e.toggleTaskAt(ev.X, ev.Y) || e.openAt(ev) {
		return
	}
	p, atom, ok := e.hit(ev.X, ev.Y)
	if !ok {
		return
	}
	if atom != nil {
		e.SetSelection(doc.NodeSelection{IDs: []doc.NodeID{atom.ID()}})
		return
	}
	e.goalX = -1
	sel, hasText := e.state.TextSel()
	switch {
	case ev.Clicks == 2:
		e.selectWord(p)
	case ev.Clicks >= 3:
		leaf := e.state.Node(p.Leaf)
		e.SetSelection(doc.TextSelection{Anchor: doc.Point{Leaf: p.Leaf}, Head: doc.Point{Leaf: p.Leaf, Offset: leaf.TextLen()}})
	case ev.Mods&ui.Shift != 0 && hasText:
		e.SetSelection(doc.TextSelection{Anchor: sel.Anchor, Head: p})
	default:
		e.SetSelection(doc.Caret(p))
	}
	anchor := p
	if s, ok := e.state.TextSel(); ok {
		anchor = s.Anchor
	}
	e.drag = &dragState{anchor: anchor, clicks: ev.Clicks}
}

func (e *Editor) pointerDrag(ev ui.InputEvent) {
	p, atom, ok := e.hit(ev.X, ev.Y)
	if !ok || atom != nil {
		return
	}
	e.SetSelection(doc.TextSelection{Anchor: e.drag.anchor, Head: p})
}

func (e *Editor) selectWord(p doc.Point) {
	text := e.state.Node(p.Leaf).Inline().String()
	start := wordBoundary(text, p.Offset+1, true)
	end := wordBoundary(text, start, false)
	if start > p.Offset || end <= start {
		start, end = p.Offset, p.Offset
	}
	e.SetSelection(doc.TextSelection{Anchor: doc.Point{Leaf: p.Leaf, Offset: start}, Head: doc.Point{Leaf: p.Leaf, Offset: end}})
}

// toggleTaskAt ticks a task when its checkbox is clicked.
func (e *Editor) toggleTaskAt(x, y float32) bool {
	var found *doc.Node
	var walk func(b *box, bx, by float32)
	walk = func(b *box, bx, by float32) {
		if found != nil || !b.n.Type().IsContainer() {
			return
		}
		for i, k := range b.kids {
			kx, ky := bx+k.x, by+k.y
			if b.cont.Check != nil {
				if task, _ := b.cont.Check(i, k.n); task {
					r := e.checkRect(kx, ky, k)
					if x >= r.X-3 && x <= r.X+r.W+3 && y >= r.Y-3 && y <= r.Y+r.H+3 {
						found = k.n
						return
					}
				}
			}
			if y >= ky && y <= ky+k.h {
				walk(k, kx, ky)
			}
		}
	}
	walk(e.root, e.cfg.Padding, e.cfg.Padding)
	if found == nil {
		return false
	}
	id := found.ID()
	return e.Run(func(s doc.State, tx *doc.Tx) bool {
		n := s.Node(id)
		tx.SetAttr(id, "checked", !n.Attrs().Bool("checked"))
		return true
	}, "format")
}

// openAt opens what a click is on: a page card, or with Cmd a link. It
// reports whether it opened something.
func (e *Editor) openAt(ev ui.InputEvent) bool {
	if e.cfg.OnOpenLink == nil {
		return false
	}
	if atom, _ := e.objectAt(ev.X, ev.Y); atom != nil {
		if src := atom.Attrs().String("src"); IsPageTarget(src) {
			return e.cfg.OnOpenLink(src)
		}
	}
	if ev.Mods&ui.Cmd == 0 {
		return false
	}
	p, _, ok := e.hit(ev.X, ev.Y)
	if !ok {
		return false
	}
	in := e.state.Node(p.Leaf).Inline()
	for _, off := range []int{p.Offset, p.Offset - 1} {
		if part := in.Slice(off, off+1); part.NumSegments() == 1 {
			for _, m := range part.Segment(0).Marks {
				if m.Type.Name() == "link" {
					return e.cfg.OnOpenLink(m.Attrs.String("href"))
				}
			}
		}
	}
	return false
}

// objectAt returns the inline atom drawn under (x, y), and its text block.
func (e *Editor) objectAt(x, y float32) (*doc.Node, doc.NodeID) {
	p, atom, ok := e.hit(x, y)
	if !ok || atom != nil {
		return nil, 0
	}
	pl, ok := e.leafPlace(p.Leaf)
	if !ok {
		return nil, 0
	}
	b := pl.b
	for _, o := range b.text.Objects() {
		r := o.Rect
		ox, oy := pl.x+b.textX()+r.X, pl.y+b.look.PadY+r.Y
		if x >= ox && x <= ox+r.W && y >= oy && y <= oy+r.H {
			part := b.n.Inline().Slice(o.Index, o.Index+1)
			if part.NumSegments() == 1 && part.Segment(0).Atom != nil {
				return part.Segment(0).Atom, b.n.ID()
			}
		}
	}
	return nil, 0
}
