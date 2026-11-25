package doc

import (
	"fmt"
	"slices"
	"sync/atomic"
)

// NodeID names a node for the session. IDs are never written to Markdown.
type NodeID uint64

var lastID atomic.Uint64

func newID() NodeID { return NodeID(lastID.Add(1)) }

// Node is a document node. Committed nodes are immutable: edits make new
// nodes and share every untouched subtree with the old document.
//
// While a transaction runs, the nodes it copied belong to it and change in
// place; they become immutable when it commits.
type Node struct {
	id    NodeID
	typ   *NodeType
	attrs Attrs
	kids  []*Node // ContentBlocks
	text  Inline  // ContentInline, ContentText
	size  int     // length in the linear text, see Size
	sizes []int   // sizes[i] is the summed size of kids[:i]

	owner  *Tx  // the transaction that may change this node in place
	frozen bool // recorded in its owner's log: copy before changing
}

// NewBlock returns a container. It panics if t is not a container or an
// attribute is invalid; whether t allows the children is checked by
// normalization, not here, so a transaction can pass through invalid
// states.
func NewBlock(t *NodeType, attrs A, kids ...*Node) *Node {
	if t.spec.Content != ContentBlocks {
		panic(fmt.Sprintf("doc: NewBlock of %s, which holds %s", t.Name(), t.spec.Content))
	}
	for _, k := range kids {
		if k == nil {
			panic("doc: nil child")
		}
		if k.typ.spec.Inline {
			panic(fmt.Sprintf("doc: inline %s as a block", k.typ.Name()))
		}
	}
	n := &Node{id: newID(), typ: t, attrs: mustAttrs(t, NewAttrs(attrs)), kids: slices.Clone(kids)}
	n.resize()
	return n
}

// NewTextBlock returns a text block. Plain-text types take only the
// plain text of in.
func NewTextBlock(t *NodeType, attrs A, in Inline) *Node {
	if !t.IsTextBlock() {
		panic(fmt.Sprintf("doc: NewTextBlock of %s, which holds %s", t.Name(), t.spec.Content))
	}
	if t.spec.Content == ContentText && in.hasAtomsOrMarks() {
		in = in.PlainText()
	}
	n := &Node{id: newID(), typ: t, attrs: mustAttrs(t, NewAttrs(attrs)), text: in}
	n.resize()
	return n
}

// NewAtom returns an atom, a block or, for an inline type, one to put in
// a text block's content with AtomInline.
func NewAtom(t *NodeType, attrs A) *Node {
	if t.spec.Content != ContentNone {
		panic(fmt.Sprintf("doc: NewAtom of %s, which holds %s", t.Name(), t.spec.Content))
	}
	n := &Node{id: newID(), typ: t, attrs: mustAttrs(t, NewAttrs(attrs))}
	n.resize()
	return n
}

func mustAttrs(t *NodeType, a Attrs) Attrs {
	r, err := resolveAttrs(t.spec.Attrs, a, t.Name())
	if err != nil {
		panic(err)
	}
	return r
}

// resize recomputes size and the children's prefix sums.
func (n *Node) resize() {
	switch n.typ.spec.Content {
	case ContentBlocks:
		if cap(n.sizes) < len(n.kids) {
			n.sizes = make([]int, len(n.kids))
		}
		n.sizes = n.sizes[:len(n.kids)]
		total := 0
		for i, k := range n.kids {
			n.sizes[i] = total
			total += k.size
		}
		n.size = total
	case ContentInline, ContentText:
		n.size = n.text.Len() + 1
	default:
		if n.typ.spec.Inline {
			n.size = 1
		} else {
			n.size = 0
		}
	}
}

func (n *Node) ID() NodeID        { return n.id }
func (n *Node) Type() *NodeType   { return n.typ }
func (n *Node) Attrs() Attrs      { return n.attrs }
func (n *Node) Attr(k string) any { return n.attrs.Get(k) }

// Inline returns a text block's content.
func (n *Node) Inline() Inline { return n.text }

// TextLen is a text block's content length in UTF-16 units.
func (n *Node) TextLen() int { return n.text.Len() }

// Size is the node's length in the linear text: a text block's content
// plus one separator after it, a container's children summed, one for an
// inline atom and nothing for a block atom.
func (n *Node) Size() int { return n.size }

func (n *Node) NumChildren() int     { return len(n.kids) }
func (n *Node) Child(i int) *Node    { return n.kids[i] }
func (n *Node) Children() []*Node    { return slices.Clone(n.kids) }
func (n *Node) ChildStart(i int) int { return n.sizes[i] }

// ChildIndex returns the index of the child with id, or -1.
func (n *Node) ChildIndex(id NodeID) int {
	for i, k := range n.kids {
		if k.id == id {
			return i
		}
	}
	return -1
}

// Walk calls f for n and its descendant blocks in document order, and
// skips a node's children when f returns false. Inline atoms are not
// visited.
func (n *Node) Walk(f func(*Node) bool) {
	if !f(n) {
		return
	}
	for _, k := range n.kids {
		k.Walk(f)
	}
}

// CloneFresh copies n deeply, giving every node, inline atoms included,
// a new ID: for duplicating content that is already in a document.
func CloneFresh(n *Node) *Node {
	c := &Node{id: newID(), typ: n.typ, attrs: n.attrs}
	switch n.typ.spec.Content {
	case ContentBlocks:
		c.kids = make([]*Node, len(n.kids))
		for i, k := range n.kids {
			c.kids[i] = CloneFresh(k)
		}
	case ContentInline:
		c.text = cloneInlineFresh(n.text)
	case ContentText:
		c.text = n.text
	}
	c.resize()
	return c
}

func cloneInlineFresh(in Inline) Inline {
	if !hasAtomsOrMarks(in.segs) {
		return in
	}
	segs := make([]Segment, len(in.segs))
	for i, s := range in.segs {
		if s.Atom != nil {
			s.Atom = CloneFresh(s.Atom)
		}
		segs[i] = s
	}
	return Inline{segs: segs, n: in.n}
}

// copyFor returns a copy of n that tx owns and may change in place.
func (n *Node) copyFor(tx *Tx) *Node {
	c := &Node{id: n.id, typ: n.typ, attrs: n.attrs, text: n.text, size: n.size, owner: tx}
	if n.kids != nil {
		c.kids = slices.Clone(n.kids)
		c.sizes = slices.Clone(n.sizes)
	}
	tx.owned = append(tx.owned, c)
	return c
}

// Equal reports whether a and b are the same tree: IDs, types, attributes
// and content.
func Equal(a, b *Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.id != b.id || a.typ != b.typ || !a.attrs.Equal(b.attrs) || len(a.kids) != len(b.kids) || !a.text.Equal(b.text) {
		return false
	}
	for i := range a.kids {
		if !Equal(a.kids[i], b.kids[i]) {
			return false
		}
	}
	return true
}

// SameContent is Equal ignoring IDs, as for comparing a document with its
// Markdown read back.
func SameContent(a, b *Node) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.typ != b.typ || !a.attrs.Equal(b.attrs) || len(a.kids) != len(b.kids) || !sameInline(a.text, b.text) {
		return false
	}
	for i := range a.kids {
		if !SameContent(a.kids[i], b.kids[i]) {
			return false
		}
	}
	return true
}

func sameInline(a, b Inline) bool {
	if a.n != b.n || len(a.segs) != len(b.segs) {
		return false
	}
	for i, s := range a.segs {
		t := b.segs[i]
		if s.Text != t.Text || !s.Marks.Equal(t.Marks) || (s.Atom == nil) != (t.Atom == nil) {
			return false
		}
		if s.Atom != nil && !SameContent(s.Atom, t.Atom) {
			return false
		}
	}
	return true
}

func formatValue(v any) string {
	switch v := v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprint(v)
	}
}
