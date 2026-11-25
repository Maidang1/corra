package doc

import (
	"fmt"
	"slices"
	"time"
)

// Meta describes a transaction for history and listeners.
type Meta struct {
	// Kind is "input", "delete", "format", "structure", "paste",
	// "source" or "external"; history merges only "input" and "delete".
	Kind string
	// NoHistory keeps the change out of undo, as for external edits.
	NoHistory bool
	Time      time.Time
}

// Commit is a finished transaction: the states before and after, and
// every change made, normalization included.
type Commit struct {
	Before, After State
	Ops           []Op
	Meta          Meta
}

// DocChanged reports whether the document changed.
func (c Commit) DocChanged() bool { return len(c.Ops) > 0 }

// Tx is a transaction: a set of changes applied to a state together and
// committed as one step. Only its four primitives change the document;
// everything else builds on them.
//
// Nodes the transaction copied belong to it and change in place, so a
// change to many blocks copies each ancestor once. Nodes recorded in its
// log are frozen and copied again before a later change, so the log keeps
// the values it recorded.
type Tx struct {
	Meta Meta

	start     State
	t         tree
	sel       Selection
	selSet    bool
	stored    *MarkSet
	storedSet bool
	ops       []Op
	owned     []*Node
	origin    OpOrigin
	fallback  int
	removed   bool
	done      bool
}

// Begin starts a transaction on s.
func (s State) Begin() *Tx {
	return &Tx{start: s, t: s.tree(), sel: s.Sel, stored: s.Stored, Meta: Meta{Time: time.Now()}}
}

// State returns the transaction's current state. Its nodes may still be
// changed in place by the transaction: read them, and look them up again
// after the next change rather than holding on to them.
func (tx *Tx) State() State {
	return State{Doc: tx.t.root, Sel: tx.sel, Stored: tx.stored, parents: tx.t.parents}
}

// Start returns the state the transaction began from.
func (tx *Tx) Start() State { return tx.start }

// Ops returns the changes recorded so far.
func (tx *Tx) Ops() []Op { return slices.Clone(tx.ops) }

// SetOrigin labels the changes recorded from now on.
func (tx *Tx) SetOrigin(o OpOrigin) { tx.origin = o }

func (tx *Tx) check() {
	if tx.done {
		panic("doc: transaction already committed")
	}
}

func (tx *Tx) canChange(n *Node) bool { return n.owner == tx && !n.frozen }

// mutable returns node id, copied into the transaction with all its
// ancestors if need be, and the chain from the root down to it.
func (tx *Tx) mutable(id NodeID) (*Node, []*Node) {
	nodes, idx, ok := tx.t.chain(id)
	if !ok {
		panic(fmt.Sprintf("doc: node %d is not in the document", id))
	}
	if !tx.canChange(nodes[0]) {
		nodes[0] = nodes[0].copyFor(tx)
		tx.t.root = nodes[0]
	}
	for i := 1; i < len(nodes); i++ {
		if !tx.canChange(nodes[i]) {
			c := nodes[i].copyFor(tx)
			nodes[i-1].kids[idx[i]] = c
			nodes[i] = c
		}
	}
	return nodes[len(nodes)-1], nodes
}

func resizeChain(chain []*Node) {
	for i := len(chain) - 1; i >= 0; i-- {
		chain[i].resize()
	}
}

func (tx *Tx) freeze(n *Node) {
	if n.owner != tx || n.frozen {
		return
	}
	n.frozen = true
	for _, k := range n.kids {
		tx.freeze(k)
	}
}

func (tx *Tx) record(o Op) {
	o.Origin = tx.origin
	tx.ops = append(tx.ops, o)
}

func (tx *Tx) mustNode(id NodeID) *Node {
	n := tx.t.node(id)
	if n == nil {
		panic(fmt.Sprintf("doc: node %d is not in the document", id))
	}
	return n
}

// ReplaceText replaces [from, to) of text block leaf's content with ins.
// Offsets are clamped to the content. A plain-text block takes only the
// plain text of ins.
func (tx *Tx) ReplaceText(leaf NodeID, from, to int, ins Inline) {
	tx.check()
	n := tx.mustNode(leaf)
	if !n.typ.IsTextBlock() {
		panic(fmt.Sprintf("doc: ReplaceText on %s", n.typ.Name()))
	}
	from = n.text.Snap(from, false)
	to = max(from, n.text.Snap(to, true))
	if n.typ.spec.Content == ContentText && ins.hasAtomsOrMarks() {
		ins = ins.PlainText()
	}
	old := n.text.Slice(from, to)
	if old.Equal(ins) {
		return
	}
	tx.record(Op{Kind: OpReplaceText, Target: leaf, From: from, To: to, OldText: old, NewText: ins})
	m, chain := tx.mutable(leaf)
	m.text = m.text.Replace(from, to, ins)
	resizeChain(chain)
	tx.mapText(leaf, from, to, ins.Len())
}

// mapText moves the selection's points in leaf past a text replacement.
func (tx *Tx) mapText(leaf NodeID, from, to, inserted int) {
	sel, ok := tx.sel.(TextSelection)
	if !ok {
		return
	}
	mapPoint := func(p Point) Point {
		if p.Leaf != leaf {
			return p
		}
		switch {
		case p.Offset < from || (p.Offset == from && from != to):
		case p.Offset >= to:
			p.Offset += inserted - (to - from)
		default:
			p.Offset = from + inserted
		}
		return p
	}
	tx.sel = TextSelection{Anchor: mapPoint(sel.Anchor), Head: mapPoint(sel.Head)}
}

// SetAttrs replaces node id's attributes; missing ones take their
// defaults.
func (tx *Tx) SetAttrs(id NodeID, attrs A) {
	tx.check()
	n := tx.mustNode(id)
	a := mustAttrs(n.typ, NewAttrs(attrs))
	if a.Equal(n.attrs) {
		return
	}
	tx.record(Op{Kind: OpSetAttrs, Target: id, OldAttrs: n.attrs, NewAttrs: a, OldType: n.typ, NewType: n.typ})
	m, _ := tx.mutable(id)
	m.attrs = a
}

// SetAttr sets one attribute of node id, keeping the others.
func (tx *Tx) SetAttr(id NodeID, k string, v any) {
	tx.SetAttrs(id, tx.mustNode(id).attrs.With(k, v).Map())
}

// SetType changes node id to type t with attrs, keeping its content. The
// types must hold the same kind of content, except that marked text may
// become plain text once it has no marks or atoms, and plain text may
// always become marked text.
func (tx *Tx) SetType(id NodeID, t *NodeType, attrs A) {
	tx.check()
	n := tx.mustNode(id)
	if id == tx.t.root.id {
		panic("doc: SetType on the root")
	}
	from, to := n.typ.spec.Content, t.spec.Content
	switch {
	case from == to:
		if from == ContentNone && n.typ.spec.Inline != t.spec.Inline {
			panic(fmt.Sprintf("doc: SetType from %s to %s", n.typ.Name(), t.Name()))
		}
	case from == ContentText && to == ContentInline:
	case from == ContentInline && to == ContentText:
		if n.text.hasAtomsOrMarks() {
			panic(fmt.Sprintf("doc: SetType to plain %s of marked text", t.Name()))
		}
	default:
		panic(fmt.Sprintf("doc: SetType from %s (%s) to %s (%s)", n.typ.Name(), from, t.Name(), to))
	}
	a := mustAttrs(t, NewAttrs(attrs))
	if t == n.typ && a.Equal(n.attrs) {
		return
	}
	tx.record(Op{Kind: OpSetType, Target: id, OldType: n.typ, NewType: t, OldAttrs: n.attrs, NewAttrs: a})
	m, chain := tx.mutable(id)
	m.typ, m.attrs = t, a
	resizeChain(chain)
}

// ReplaceChildren replaces children [from, to) of container parent with
// nodes. A node may be one removed earlier in this transaction (a move),
// but no node may be in the document twice: copy content that stays with
// CloneFresh.
func (tx *Tx) ReplaceChildren(parent NodeID, from, to int, nodes ...*Node) {
	tx.check()
	p := tx.mustNode(parent)
	if !p.typ.IsContainer() {
		panic(fmt.Sprintf("doc: ReplaceChildren on %s", p.typ.Name()))
	}
	from = max(0, min(from, len(p.kids)))
	to = max(from, min(to, len(p.kids)))
	if from == to && len(nodes) == 0 {
		return
	}
	for _, n := range nodes {
		if n == nil || n.typ.spec.Inline {
			panic("doc: ReplaceChildren with a nil or inline node")
		}
	}
	old := slices.Clone(p.kids[from:to])
	news := slices.Clone(nodes)
	if len(old) > 0 && !tx.removed {
		if g, ok := tx.t.start(parent); ok {
			tx.fallback, tx.removed = g+p.sizes[from], true
		}
	}
	for _, o := range old {
		tx.unlink(o)
	}
	for _, n := range news {
		tx.link(n, parent)
	}
	for _, n := range old {
		tx.freeze(n)
	}
	for _, n := range news {
		tx.freeze(n)
	}
	tx.record(Op{Kind: OpReplaceChildren, Target: parent, From: from, To: to, OldKids: old, NewKids: news})
	m, chain := tx.mutable(parent)
	m.kids = slices.Concat(m.kids[:from:from], news, m.kids[to:])
	resizeChain(chain)
}

func (tx *Tx) link(n *Node, parent NodeID) {
	if tx.t.contains(n.id) {
		panic(fmt.Sprintf("doc: node %d (%s) is already in the document", n.id, n.typ.Name()))
	}
	tx.t.parents = tx.t.parents.Set(uint64(n.id), uint64(parent))
	for _, k := range n.kids {
		tx.link(k, n.id)
	}
}

func (tx *Tx) unlink(n *Node) {
	tx.t.parents = tx.t.parents.Delete(uint64(n.id))
	for _, k := range n.kids {
		tx.unlink(k)
	}
}

// SetSelection sets the selection the transaction ends with.
func (tx *Tx) SetSelection(sel Selection) {
	tx.check()
	tx.sel, tx.selSet = sel, true
}

// SetStoredMarks sets the marks the next typed text takes.
func (tx *Tx) SetStoredMarks(m MarkSet) {
	tx.check()
	tx.stored, tx.storedSet = &m, true
}

// Commit normalizes the document, ends the transaction and returns the
// new state with the record of what changed. The transaction cannot be
// used afterwards.
func (tx *Tx) Commit() (State, Commit) {
	tx.check()
	tx.normalize()
	sel := tx.validSelection()
	stored := tx.stored
	if !tx.storedSet && (len(tx.ops) > 0 || !sameSelection(sel, tx.start.Sel)) {
		stored = nil
	}
	after := State{Doc: tx.t.root, Sel: sel, Stored: stored, parents: tx.t.parents}
	for _, n := range tx.owned {
		n.owner, n.frozen = nil, false
	}
	tx.owned = nil
	tx.done = true
	return after, Commit{Before: tx.start, After: after, Ops: tx.ops, Meta: tx.Meta}
}

func sameSelection(a, b Selection) bool {
	switch a := a.(type) {
	case TextSelection:
		b, ok := b.(TextSelection)
		return ok && a == b
	case NodeSelection:
		b, ok := b.(NodeSelection)
		return ok && slices.Equal(a.IDs, b.IDs)
	case CellSelection:
		b, ok := b.(CellSelection)
		return ok && a == b
	}
	return a == nil && b == nil
}

// validSelection returns the selection with points clamped to their
// blocks, or a caret near the last removal if its blocks are gone.
func (tx *Tx) validSelection() Selection {
	st := tx.State()
	clamp := func(p Point) (Point, bool) {
		n := st.Node(p.Leaf)
		if n == nil || !n.typ.IsTextBlock() {
			return p, false
		}
		p.Offset = max(0, min(p.Offset, n.text.Len()))
		return p, true
	}
	switch sel := tx.sel.(type) {
	case TextSelection:
		a, okA := clamp(sel.Anchor)
		h, okH := clamp(sel.Head)
		if okA && okH {
			return TextSelection{a, h}
		}
		if okH {
			return Caret(h)
		}
		if okA {
			return Caret(a)
		}
	case NodeSelection:
		var ids []NodeID
		for _, id := range sel.IDs {
			if st.Node(id) != nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			return NodeSelection{IDs: ids}
		}
	case CellSelection:
		if n := st.Node(sel.Table); n != nil {
			return sel
		}
	}
	g := 0
	if tx.removed {
		g = tx.fallback
	}
	if p, ok := tx.t.pointAt(g); ok {
		return Caret(p)
	}
	return nil
}
