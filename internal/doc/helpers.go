package doc

import (
	"slices"
	"strings"
)

// The commands here know only the kernel's structures (containers, text
// blocks, atoms); behavior that depends on a node type, as splitting a
// list item, belongs to that type's extension.

// isolating returns the nearest isolating ancestor of id, or the root.
func isolating(s State, id NodeID) NodeID {
	anc := s.Ancestors(id)
	for i := len(anc) - 1; i >= 1; i-- {
		if anc[i].typ.spec.Isolating {
			return anc[i].id
		}
	}
	return s.Doc.id
}

// parentOf returns the container holding id and id's index in it.
func parentOf(s State, id NodeID) (*Node, int) {
	anc := s.Ancestors(id)
	if len(anc) < 2 {
		return nil, -1
	}
	p := anc[len(anc)-2]
	return p, p.ChildIndex(id)
}

// marksFor keeps the marks a text block of type t allows.
func marksFor(t *NodeType, marks MarkSet) []Mark {
	var out []Mark
	for _, m := range marks {
		if t.AllowsMark(m.Type) {
			out = append(out, m)
		}
	}
	return out
}

// InsertText types text at the selection, replacing what is selected.
// Newlines split marked-text blocks; plain-text blocks keep them.
func InsertText(text string) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok {
			return false
		}
		marks := s.CurrentMarks()
		if !sel.Empty() {
			deleteSelection(tx)
		}
		lines := []string{text}
		s = tx.State()
		caret := s.Sel.(TextSelection).Head
		if s.Node(caret.Leaf).typ.spec.Content == ContentInline {
			lines = strings.Split(text, "\n")
		}
		for i, line := range lines {
			if i > 0 {
				splitBlock(tx)
			}
			caret = tx.State().Sel.(TextSelection).Head
			n := tx.State().Node(caret.Leaf)
			in := Text(line, marksFor(n.typ, marks)...)
			tx.ReplaceText(caret.Leaf, caret.Offset, caret.Offset, in)
			tx.SetSelection(Caret(Point{caret.Leaf, caret.Offset + in.Len()}))
		}
		return true
	}
}

// DeleteSelection deletes what is selected; it does not apply to a caret.
func DeleteSelection(s State, tx *Tx) bool { return deleteSelection(tx) }

func deleteSelection(tx *Tx) bool {
	s := tx.State()
	switch sel := s.Sel.(type) {
	case NodeSelection:
		for _, id := range sel.IDs {
			if p, i := parentOf(tx.State(), id); p != nil {
				tx.ReplaceChildren(p.id, i, i+1)
				removeEmptyAncestors(tx, p.id)
			}
		}
		return len(sel.IDs) > 0
	case TextSelection:
		if sel.Empty() {
			return false
		}
		from, to := s.Ordered(sel)
		if from.Leaf == to.Leaf {
			tx.ReplaceText(from.Leaf, from.Offset, to.Offset, Inline{})
			tx.SetSelection(Caret(from))
			return true
		}
		if isolating(s, from.Leaf) != isolating(s, to.Leaf) {
			// Across table cells or callouts: empty each part, keep the
			// structure.
			for _, r := range s.LeafRanges(sel) {
				tx.ReplaceText(r.Leaf, r.From, r.To, Inline{})
			}
			tx.SetSelection(Caret(from))
			return true
		}
		a, b := s.Node(from.Leaf), s.Node(to.Leaf)
		tail := b.text.Slice(to.Offset, b.text.Len())
		tx.ReplaceText(a.id, from.Offset, a.text.Len(), tail)
		removeBetween(tx, a.id, b.id)
		tx.SetSelection(Caret(from))
		return true
	}
	return false
}

// preorder lists the blocks of n in document order.
func preorder(n *Node) []*Node {
	var out []*Node
	n.Walk(func(d *Node) bool { out = append(out, d); return true })
	return out
}

// removeBetween removes block b and every block after a up to b in
// document order, except b's ancestors, which go only if left empty.
func removeBetween(tx *Tx, a, b NodeID) {
	s := tx.State()
	order := preorder(s.Doc)
	ia := slices.IndexFunc(order, func(n *Node) bool { return n.id == a })
	ib := slices.IndexFunc(order, func(n *Node) bool { return n.id == b })
	if ia < 0 || ib <= ia {
		return
	}
	ancB := map[NodeID]bool{}
	anc := s.Ancestors(b)
	for _, n := range anc[:len(anc)-1] {
		ancB[n.id] = true
	}
	whole := map[NodeID]bool{}
	var remove []NodeID
	for _, n := range order[ia+1 : ib+1] {
		p, _ := s.Parent(n.id)
		if whole[p] {
			whole[n.id] = true
			continue
		}
		if !ancB[n.id] {
			whole[n.id] = true
			remove = append(remove, n.id)
		}
	}
	for i := len(remove) - 1; i >= 0; i-- {
		if p, j := parentOf(tx.State(), remove[i]); p != nil {
			tx.ReplaceChildren(p.id, j, j+1)
		}
	}
	// b's ancestors that started after a and are now empty.
	for i := len(anc) - 2; i >= 1; i-- {
		n := tx.State().Node(anc[i].id)
		if n == nil || slices.IndexFunc(order[ia+1:ib+1], func(o *Node) bool { return o.id == n.id }) < 0 {
			continue
		}
		if len(n.kids) == 0 {
			if p, j := parentOf(tx.State(), n.id); p != nil {
				tx.ReplaceChildren(p.id, j, j+1)
			}
		}
	}
}

// removeEmptyAncestors removes id and its ancestors while they are empty
// containers, stopping below the root.
func removeEmptyAncestors(tx *Tx, id NodeID) {
	for {
		s := tx.State()
		n := s.Node(id)
		if n == nil || id == s.Doc.id || !n.typ.IsContainer() || len(n.kids) > 0 {
			return
		}
		p, i := parentOf(s, id)
		tx.ReplaceChildren(p.id, i, i+1)
		id = p.id
	}
}

// SplitBlock splits the text block at the caret, as Enter does. At the end
// of a block that is not its container's default (a heading), the new block
// is the default (a paragraph). Plain-text blocks take a newline instead.
func SplitBlock(s State, tx *Tx) bool {
	if _, ok := s.TextSel(); !ok {
		return false
	}
	return splitBlock(tx)
}

func splitBlock(tx *Tx) bool {
	if sel := tx.State().Sel.(TextSelection); !sel.Empty() {
		deleteSelection(tx)
	}
	s := tx.State()
	caret := s.Sel.(TextSelection).Head
	l := s.Node(caret.Leaf)
	if l.typ.spec.Content == ContentText {
		tx.ReplaceText(l.id, caret.Offset, caret.Offset, Text("\n"))
		tx.SetSelection(Caret(Point{l.id, caret.Offset + 1}))
		return true
	}
	p, i := parentOf(s, l.id)
	t, attrs := l.typ, l.attrs.Map()
	if caret.Offset == l.text.Len() {
		if def := s.Registry().DefaultTextBlock(p.typ); def != nil && def != t {
			t, attrs = def, nil
		}
	}
	tail := l.text.Slice(caret.Offset, l.text.Len())
	tx.ReplaceText(l.id, caret.Offset, l.text.Len(), Inline{})
	nb := NewTextBlock(t, attrs, tail)
	tx.ReplaceChildren(p.id, i+1, i+1, nb)
	tx.SetSelection(Caret(Point{nb.id, 0}))
	return true
}

// JoinBackward handles Backspace at the start of a text block: it selects
// an atom block just before it, lifts the block out of its container when
// it is the container's first, or joins it into the text block before.
func JoinBackward(s State, tx *Tx) bool {
	sel, ok := s.TextSel()
	if !ok || !sel.Empty() || sel.Head.Offset != 0 {
		return false
	}
	l := s.Node(sel.Head.Leaf)
	if l.typ.spec.Isolating {
		return false // a table cell keeps its edges
	}
	p, i := parentOf(s, l.id)
	if i > 0 && p.kids[i-1].typ.IsAtom() {
		tx.SetSelection(NodeSelection{IDs: []NodeID{p.kids[i-1].id}})
		return true
	}
	if i == 0 && p.id != s.Doc.id && !p.typ.spec.Isolating {
		return liftRange(tx, p.id, 0, 0)
	}
	prev := s.PrevLeaf(l.id)
	if prev == nil || isolating(s, prev.id) != isolating(s, l.id) {
		return false
	}
	return join(tx, prev.id, l.id)
}

// JoinForward handles Delete at the end of a text block: it joins the next
// text block into this one.
func JoinForward(s State, tx *Tx) bool {
	sel, ok := s.TextSel()
	if !ok || !sel.Empty() {
		return false
	}
	l := s.Node(sel.Head.Leaf)
	if sel.Head.Offset != l.text.Len() || l.typ.spec.Isolating {
		return false
	}
	p, i := parentOf(s, l.id)
	if i+1 < len(p.kids) && p.kids[i+1].typ.IsAtom() {
		tx.SetSelection(NodeSelection{IDs: []NodeID{p.kids[i+1].id}})
		return true
	}
	next := s.NextLeaf(l.id)
	if next == nil || isolating(s, next.id) != isolating(s, l.id) {
		return false
	}
	return join(tx, l.id, next.id)
}

// join appends text block b's content to a and removes b, and any
// container b leaves empty.
func join(tx *Tx, a, b NodeID) bool {
	s := tx.State()
	na, nb := s.Node(a), s.Node(b)
	at := na.text.Len()
	in := nb.text
	if na.typ.spec.Content == ContentText {
		in = in.PlainText()
	}
	tx.ReplaceText(a, at, at, in)
	p, i := parentOf(tx.State(), b)
	tx.ReplaceChildren(p.id, i, i+1)
	removeEmptyAncestors(tx, p.id)
	tx.SetSelection(Caret(Point{a, at}))
	return true
}

// AddMark adds m to the selection, or to the stored marks at a caret.
func AddMark(m Mark) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !s.MarkAllowed(sel, m.Type) {
			return false
		}
		if sel.Empty() {
			tx.SetStoredMarks(s.CurrentMarks().Add(m))
			return true
		}
		for _, r := range s.LeafRanges(sel) {
			n := s.Node(r.Leaf)
			if r.From == r.To || !n.typ.AllowsMark(m.Type) {
				continue
			}
			tx.ReplaceText(r.Leaf, r.From, r.To, n.text.Slice(r.From, r.To).AddMark(m))
		}
		tx.SetSelection(sel)
		return true
	}
}

// RemoveMark removes marks of type t from the selection, or from the
// stored marks at a caret.
func RemoveMark(t *MarkType) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !s.MarkAllowed(sel, t) {
			return false
		}
		if sel.Empty() {
			tx.SetStoredMarks(s.CurrentMarks().Remove(t))
			return true
		}
		for _, r := range s.LeafRanges(sel) {
			if r.From == r.To {
				continue
			}
			n := s.Node(r.Leaf)
			tx.ReplaceText(r.Leaf, r.From, r.To, n.text.Slice(r.From, r.To).RemoveMark(t))
		}
		tx.SetSelection(sel)
		return true
	}
}

// ToggleMark removes t if all of the selection has it, otherwise adds it.
func ToggleMark(t *MarkType) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok {
			return false
		}
		has := s.RangeHasMark(sel, t)
		if sel.Empty() {
			has = s.CurrentMarks().Has(t)
		}
		if has {
			return RemoveMark(t)(s, tx)
		}
		return AddMark(Mark{Type: t})(s, tx)
	}
}

// SetBlockType turns the text blocks the selection covers into type t,
// where their containers allow it.
func SetBlockType(t *NodeType, attrs A) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !t.IsTextBlock() {
			return false
		}
		want := mustAttrs(t, NewAttrs(attrs))
		applied := false
		for _, r := range s.LeafRanges(sel) {
			n := s.Node(r.Leaf)
			p, _ := parentOf(s, n.id)
			if !p.typ.Allows(t) || (n.typ == t && n.attrs.Equal(want)) {
				continue
			}
			if t.spec.Content == ContentText && n.text.hasAtomsOrMarks() {
				tx.ReplaceText(n.id, 0, n.text.Len(), n.text.PlainText())
			}
			tx.SetType(n.id, t, attrs)
			applied = true
		}
		tx.SetSelection(sel)
		return applied
	}
}

// siblingRange returns the deepest container holding both ends of the
// selection, and the indexes of the children each end is in.
func siblingRange(s State, sel TextSelection) (*Node, int, int) {
	from, to := s.Ordered(sel)
	a, b := s.Ancestors(from.Leaf), s.Ancestors(to.Leaf)
	d := 0
	for d < len(a) && d < len(b) && a[d].id == b[d].id {
		d++
	}
	if d == len(a) || d == len(b) {
		// One text block: the range is that block in its parent.
		d = len(a) - 1
	}
	parent := a[d-1]
	return parent, parent.ChildIndex(a[d].id), parent.ChildIndex(b[d].id)
}

// Wrap wraps the blocks the selection covers in a new container of type t,
// at the deepest level where the schema allows it.
func Wrap(t *NodeType, attrs A) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok || !t.IsContainer() {
			return false
		}
		p, i, j := siblingRange(s, sel)
		for {
			fits := p.typ.Allows(t)
			for _, k := range p.kids[i : j+1] {
				fits = fits && t.Allows(k.typ)
			}
			if fits {
				break
			}
			if p.id == s.Doc.id {
				return false
			}
			gp, gi := parentOf(s, p.id)
			p, i, j = gp, gi, gi
		}
		kids := slices.Clone(p.kids[i : j+1])
		tx.ReplaceChildren(p.id, i, j+1, NewBlock(t, attrs, kids...))
		tx.SetSelection(sel)
		return true
	}
}

// Lift moves the blocks the selection covers out of their container,
// splitting it around them.
func Lift(s State, tx *Tx) bool {
	sel, ok := s.TextSel()
	if !ok {
		return false
	}
	p, i, j := siblingRange(s, sel)
	if p.id == s.Doc.id {
		return false
	}
	return liftRange(tx, p.id, i, j)
}

// liftRange moves children [i, j] of container c out to c's parent: the
// children before stay in c, those after go to a copy of c after them.
func liftRange(tx *Tx, c NodeID, i, j int) bool {
	s := tx.State()
	cn := s.Node(c)
	g, ci := parentOf(s, c)
	if g == nil {
		return false
	}
	sel := s.Sel
	mid := slices.Clone(cn.kids[i : j+1])
	after := slices.Clone(cn.kids[j+1:])
	tx.ReplaceChildren(c, i, len(cn.kids))
	moved := mid
	if len(after) > 0 {
		moved = append(moved, NewBlock(cn.typ, cn.attrs.Map(), after...))
	}
	if i == 0 {
		tx.ReplaceChildren(g.id, ci, ci+1, moved...)
	} else {
		tx.ReplaceChildren(g.id, ci+1, ci+1, moved...)
	}
	tx.SetSelection(sel)
	return true
}

// MoveBlock moves block id to index in container parent.
func MoveBlock(id, parent NodeID, index int) Command {
	return func(s State, tx *Tx) bool {
		n := s.Node(id)
		p, i := parentOf(s, id)
		dest := s.Node(parent)
		if n == nil || p == nil || dest == nil || !dest.typ.IsContainer() {
			return false
		}
		for _, a := range s.Ancestors(parent) {
			if a.id == id {
				return false // into itself
			}
		}
		sel := s.Sel
		tx.ReplaceChildren(p.id, i, i+1)
		if parent == p.id && index > i {
			index--
		}
		tx.ReplaceChildren(parent, index, index, n)
		removeEmptyAncestors(tx, p.id)
		tx.SetSelection(sel)
		return true
	}
}

// InsertBlocks inserts copies of blocks after the block holding the
// caret, or in its place when it is an empty text block. The command can
// run any number of times: each inserts new copies.
func InsertBlocks(blocks ...*Node) Command {
	return func(s State, tx *Tx) bool {
		sel, ok := s.TextSel()
		if !ok || len(blocks) == 0 {
			return false
		}
		nodes := cloneAll(blocks)
		l := s.Node(sel.Head.Leaf)
		// Go up to where the blocks are allowed: from a table cell, after
		// the table.
		target := l
		p, i := parentOf(s, l.id)
		for p != nil && !p.typ.Allows(nodes[0].typ) && p.id != s.Doc.id {
			target = p
			p, i = parentOf(s, p.id)
		}
		if p == nil {
			return false
		}
		if target == l && l.text.Len() == 0 && !l.typ.spec.Isolating {
			tx.ReplaceChildren(p.id, i, i+1, nodes...)
		} else {
			tx.ReplaceChildren(p.id, i+1, i+1, nodes...)
		}
		selectEnd(tx, nodes[len(nodes)-1])
		return true
	}
}

// selectEnd puts the caret at the end of n's last text block, or selects
// n when it has none.
func selectEnd(tx *Tx, n *Node) {
	if l := lastLeaf(n); l != nil {
		tx.SetSelection(Caret(Point{l.id, l.text.Len()}))
		return
	}
	tx.SetSelection(NodeSelection{IDs: []NodeID{n.id}})
}

// InsertFragment pastes blocks at the selection. Text blocks at the edges
// of the fragment join the text before and after the caret; the fragment
// is copied with fresh IDs.
func InsertFragment(frag []*Node) Command {
	return func(s State, tx *Tx) bool {
		if _, ok := s.TextSel(); !ok || len(frag) == 0 {
			return false
		}
		frag = cloneAll(frag)
		if !s.Sel.(TextSelection).Empty() {
			deleteSelection(tx)
		}
		caret := tx.State().Sel.(TextSelection).Head
		plain := tx.State().Node(caret.Leaf).typ.spec.Content == ContentText
		if plain || (len(frag) == 1 && frag[0].typ.IsTextBlock()) {
			in := frag[0].text
			if plain {
				// Pasting into code: the fragment's text, a line per block.
				var lines []string
				for _, f := range frag {
					f.Walk(func(n *Node) bool {
						if n.typ.IsTextBlock() {
							lines = append(lines, n.text.PlainText().String())
						}
						return true
					})
				}
				in = Text(strings.Join(lines, "\n"))
			}
			tx.ReplaceText(caret.Leaf, caret.Offset, caret.Offset, in)
			tx.SetSelection(Caret(Point{caret.Leaf, caret.Offset + in.Len()}))
			return true
		}
		splitBlock(tx)
		s = tx.State()
		after := s.Sel.(TextSelection).Head.Leaf
		before := s.PrevLeaf(after).id
		p, i := parentOf(s, after)
		tx.ReplaceChildren(p.id, i, i, frag...)
		last := frag[len(frag)-1]
		if bn := tx.State().Node(before); bn.text.Len() == 0 {
			// Pasted into an empty block: the fragment takes its place.
			bp, bi := parentOf(tx.State(), before)
			tx.ReplaceChildren(bp.id, bi, bi+1)
		} else if first := frag[0]; first.typ.IsTextBlock() && len(frag) > 1 {
			join(tx, before, first.id)
		}
		if last.typ.IsTextBlock() {
			end := tx.State().Node(last.id).text.Len()
			join(tx, last.id, after)
			tx.SetSelection(Caret(Point{last.id, end}))
		} else {
			selectEnd(tx, last)
		}
		return true
	}
}

func cloneAll(ns []*Node) []*Node {
	out := make([]*Node, len(ns))
	for i, n := range ns {
		out[i] = CloneFresh(n)
	}
	return out
}

// SelectAll selects from the start of the first text block to the end of
// the last.
func SelectAll(s State, tx *Tx) bool {
	first, last := firstLeaf(s.Doc), lastLeaf(s.Doc)
	if first == nil {
		return false
	}
	tx.SetSelection(TextSelection{Anchor: Point{first.id, 0}, Head: Point{last.id, last.text.Len()}})
	return true
}
