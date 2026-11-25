package doc

import (
	"fmt"
	"log"
)

// OnNormalizeLimit is called when normalization stops because its rules
// did not settle. Tests set it to panic; the default logs.
var OnNormalizeLimit = func(err error) { log.Print(err) }

// Fixer is what a NodeSpec's Normalize changes the document through. Its
// changes are recorded with the transaction's other changes.
type Fixer struct{ tx *Tx }

func (f *Fixer) State() State { return f.tx.State() }
func (f *Fixer) ReplaceText(leaf NodeID, from, to int, in Inline) {
	f.tx.ReplaceText(leaf, from, to, in)
}
func (f *Fixer) SetAttrs(id NodeID, attrs A)             { f.tx.SetAttrs(id, attrs) }
func (f *Fixer) SetAttr(id NodeID, k string, v any)      { f.tx.SetAttr(id, k, v) }
func (f *Fixer) SetType(id NodeID, t *NodeType, attrs A) { f.tx.SetType(id, t, attrs) }
func (f *Fixer) ReplaceChildren(parent NodeID, from, to int, nodes ...*Node) {
	f.tx.ReplaceChildren(parent, from, to, nodes...)
}

// Normalized returns d repaired to satisfy the schema, as a parser's
// output may not: every node is checked, not only touched ones.
func Normalized(d *Node) *Node {
	tx := NewState(d).Begin()
	var all []NodeID
	d.Walk(func(n *Node) bool { all = append(all, n.id); return true })
	tx.normalizeFrom(all)
	after, _ := tx.Commit()
	return after.Doc
}

// normalize repairs the nodes the transaction touched until every one
// satisfies the schema and its type's rules.
func (tx *Tx) normalize() { tx.normalizeFrom(nil) }

func (tx *Tx) normalizeFrom(extra []NodeID) {
	prev := tx.origin
	tx.origin = OriginNormalize
	defer func() { tx.origin = prev }()

	var queue []NodeID
	queued := map[NodeID]bool{}
	push := func(id NodeID) {
		if !queued[id] {
			queued[id] = true
			queue = append(queue, id)
		}
	}
	cursor := 0
	collect := func() int {
		n := 0
		for ; cursor < len(tx.ops); cursor++ {
			o := tx.ops[cursor]
			// The target and its ancestors: a rule may span children, as
			// a table's keeps its rows equally long.
			if nodes, _, ok := tx.t.chain(o.Target); ok {
				for _, a := range nodes {
					push(a.id)
				}
			} else {
				push(o.Target)
			}
			n++
			if o.Kind == OpSetType {
				if p, ok := tx.t.parent(o.Target); ok {
					push(p)
				}
			}
			if o.Kind == OpReplaceChildren {
				for _, k := range o.NewKids {
					k.Walk(func(d *Node) bool { push(d.id); n++; return true })
				}
			}
		}
		return n
	}
	for _, id := range extra {
		push(id)
	}
	budget := max(100, 10*(collect()+len(extra)))
	fix := &Fixer{tx}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		delete(queued, id)
		if !tx.t.contains(id) {
			continue
		}
		if budget--; budget < 0 {
			OnNormalizeLimit(fmt.Errorf("doc: normalization did not settle; stopped at node %d", id))
			return
		}
		before := len(tx.ops)
		tx.normalizeNode(tx.t.node(id), fix)
		if len(tx.ops) != before {
			push(id)
			collect()
		}
	}
}

// normalizeNode makes at most one repair; the caller requeues the node
// until it needs none.
func (tx *Tx) normalizeNode(n *Node, fix *Fixer) {
	if f := n.typ.spec.Normalize; f != nil {
		before := len(tx.ops)
		f(n, fix)
		if len(tx.ops) != before {
			return
		}
	}
	reg := n.typ.reg
	switch n.typ.spec.Content {
	case ContentBlocks:
		if len(n.kids) == 0 {
			if t := reg.DefaultChild(n.typ); t != nil {
				tx.ReplaceChildren(n.id, 0, 0, emptyNode(t))
			}
			return
		}
		for i, k := range n.kids {
			if n.typ.Allows(k.typ) {
				continue
			}
			switch {
			case reg.Wrapper(n.typ, k.typ) != nil:
				tx.ReplaceChildren(n.id, i, i+1, NewBlock(reg.Wrapper(n.typ, k.typ), nil, k))
			case k.typ.IsContainer():
				tx.ReplaceChildren(n.id, i, i+1, k.kids...)
			case k.typ.IsTextBlock() && reg.DefaultTextBlock(n.typ) != nil:
				t := reg.DefaultTextBlock(n.typ)
				if t.spec.Content == ContentText && k.text.hasAtomsOrMarks() {
					tx.ReplaceText(k.id, 0, k.text.Len(), k.text.PlainText())
				} else {
					tx.SetType(k.id, t, nil)
				}
			default:
				tx.ReplaceChildren(n.id, i, i+1)
			}
			return
		}
	case ContentInline:
		if clean, changed := allowedInline(n.typ, n.text); changed {
			tx.ReplaceText(n.id, 0, n.text.Len(), clean)
		}
	}
}

// allowedInline drops the marks and atoms t does not allow.
func allowedInline(t *NodeType, in Inline) (Inline, bool) {
	changed := false
	var out Inline
	for _, s := range in.segs {
		if s.Atom != nil && !t.Allows(s.Atom.typ) {
			changed = true
			continue
		}
		var marks MarkSet
		for _, m := range s.Marks {
			if t.AllowsMark(m.Type) {
				marks = append(marks, m)
			} else {
				changed = true
			}
		}
		s.Marks = marks
		out.segs, out.n = appendSegment(out.segs, out.n, s)
	}
	return out, changed
}

// emptyNode returns an empty node of type t: an empty text block, or a
// container holding its default child.
func emptyNode(t *NodeType) *Node {
	switch t.spec.Content {
	case ContentInline, ContentText:
		return NewTextBlock(t, nil, Inline{})
	case ContentBlocks:
		if c := t.reg.DefaultChild(t); c != nil && c != t {
			return NewBlock(t, nil, emptyNode(c))
		}
		return NewBlock(t, nil)
	}
	return NewAtom(t, nil)
}

// Empty returns an empty node of type t, as a new paragraph or list item.
func Empty(t *NodeType) *Node { return emptyNode(t) }

// Validate reports the first way doc breaks the schema: a child or mark a
// type does not allow, an empty container that could hold something, or
// an ID used twice.
func Validate(doc *Node) error {
	seen := map[NodeID]bool{}
	var check func(n *Node) error
	check = func(n *Node) error {
		if seen[n.id] {
			return fmt.Errorf("doc: ID %d used twice", n.id)
		}
		seen[n.id] = true
		switch n.typ.spec.Content {
		case ContentBlocks:
			if len(n.kids) == 0 && n.typ.reg.DefaultChild(n.typ) != nil {
				return fmt.Errorf("doc: empty %s (%d)", n.typ.Name(), n.id)
			}
			for _, k := range n.kids {
				if !n.typ.Allows(k.typ) {
					return fmt.Errorf("doc: %s (%d) holds %s (%d)", n.typ.Name(), n.id, k.typ.Name(), k.id)
				}
				if err := check(k); err != nil {
					return err
				}
			}
		case ContentInline:
			if _, changed := allowedInline(n.typ, n.text); changed {
				return fmt.Errorf("doc: %s (%d) holds marks or atoms it does not allow", n.typ.Name(), n.id)
			}
			for _, s := range n.text.segs {
				if s.Atom != nil {
					if err := check(s.Atom); err != nil {
						return err
					}
				}
			}
		case ContentText:
			if n.text.hasAtomsOrMarks() {
				return fmt.Errorf("doc: plain %s (%d) holds marks or atoms", n.typ.Name(), n.id)
			}
		}
		return nil
	}
	return check(doc)
}
