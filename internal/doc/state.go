package doc

import "fmt"

// State is an editor state: a document, a selection and the marks the
// next typed text takes. It is a value; transactions produce new states.
type State struct {
	Doc *Node
	Sel Selection
	// Stored are the marks text typed at a collapsed selection takes, as
	// after Cmd+B; nil when they come from the text around the caret.
	Stored *MarkSet

	parents *pmap
}

// NewState returns a state for doc with the caret at the start of its
// first text block.
func NewState(doc *Node) State {
	s := State{Doc: doc, parents: buildParents(doc)}
	if l := firstLeaf(doc); l != nil {
		s.Sel = Caret(Point{Leaf: l.id})
	}
	return s
}

func (s State) tree() tree { return tree{root: s.Doc, parents: s.parents} }

// Registry returns the schema of the document.
func (s State) Registry() *Registry { return s.Doc.typ.reg }

// Node returns the block with id, or nil.
func (s State) Node(id NodeID) *Node { return s.tree().node(id) }

// Parent returns the ID of the container holding id.
func (s State) Parent(id NodeID) (NodeID, bool) { return s.tree().parent(id) }

// Ancestors returns the nodes from the root down to id, inclusive.
func (s State) Ancestors(id NodeID) []*Node {
	nodes, _, _ := s.tree().chain(id)
	return nodes
}

// Index returns id's index in its parent.
func (s State) Index(id NodeID) int {
	_, idx, ok := s.tree().chain(id)
	if !ok {
		return -1
	}
	return idx[len(idx)-1]
}

// GlobalOffset converts p to an offset in the linear text: every text
// block's content followed by one separator, in document order.
func (s State) GlobalOffset(p Point) int {
	g, _ := s.tree().globalOffset(p)
	return g
}

// PointAt converts a linear offset to a point.
func (s State) PointAt(g int) Point {
	p, _ := s.tree().pointAt(g)
	return p
}

// Valid reports whether p is in a text block of the document, within its
// content.
func (s State) Valid(p Point) bool {
	n := s.Node(p.Leaf)
	return n != nil && n.typ.IsTextBlock() && p.Offset >= 0 && p.Offset <= n.text.Len()
}

// Compare orders two points in document order: -1, 0 or 1.
func (s State) Compare(a, b Point) int {
	ga, gb := s.GlobalOffset(a), s.GlobalOffset(b)
	switch {
	case ga < gb:
		return -1
	case ga > gb:
		return 1
	}
	return 0
}

// Ordered returns a selection's ends in document order.
func (s State) Ordered(sel TextSelection) (from, to Point) {
	if s.Compare(sel.Anchor, sel.Head) <= 0 {
		return sel.Anchor, sel.Head
	}
	return sel.Head, sel.Anchor
}

// LeafRanges splits a text selection into the parts of the text blocks
// it covers, in document order.
func (s State) LeafRanges(sel TextSelection) []LeafRange {
	from, to := s.Ordered(sel)
	return s.tree().leafRanges(s.GlobalOffset(from), s.GlobalOffset(to))
}

// NextLeaf and PrevLeaf return the text block after or before id in
// document order, or nil.
func (s State) NextLeaf(id NodeID) *Node {
	n := s.Node(id)
	if n == nil {
		return nil
	}
	g := s.GlobalOffset(Point{Leaf: id, Offset: n.text.Len()}) + 1
	if g >= s.Doc.size {
		return nil
	}
	p := s.PointAt(g)
	return s.Node(p.Leaf)
}

func (s State) PrevLeaf(id NodeID) *Node {
	g := s.GlobalOffset(Point{Leaf: id})
	if g == 0 {
		return nil
	}
	p := s.PointAt(g - 1)
	return s.Node(p.Leaf)
}

// FirstLeaf and LastLeaf return the first and last text block inside id.
func (s State) FirstLeaf(id NodeID) *Node {
	if n := s.Node(id); n != nil {
		return firstLeaf(n)
	}
	return nil
}

func (s State) LastLeaf(id NodeID) *Node {
	if n := s.Node(id); n != nil {
		return lastLeaf(n)
	}
	return nil
}

// TextSel returns the selection as a text selection, if it is one whose
// points are both in the document.
func (s State) TextSel() (TextSelection, bool) {
	t, ok := s.Sel.(TextSelection)
	return t, ok && s.Valid(t.Anchor) && s.Valid(t.Head)
}

// CurrentMarks returns the marks typed text takes now: the stored marks,
// or those at the caret.
func (s State) CurrentMarks() MarkSet {
	if s.Stored != nil {
		return *s.Stored
	}
	sel, ok := s.TextSel()
	if !ok {
		return nil
	}
	n := s.Node(sel.Head.Leaf)
	if n == nil {
		return nil
	}
	return n.text.MarksAt(sel.Head.Offset)
}

// RangeHasMark reports whether every character the selection covers
// carries a mark of type t.
func (s State) RangeHasMark(sel TextSelection, t *MarkType) bool {
	any := false
	for _, r := range s.LeafRanges(sel) {
		if r.From == r.To {
			continue
		}
		any = true
		if !s.Node(r.Leaf).text.HasMark(t, r.From, r.To) {
			return false
		}
	}
	return any
}

// MarkAllowed reports whether some block the selection covers takes t.
func (s State) MarkAllowed(sel TextSelection, t *MarkType) bool {
	for _, r := range s.LeafRanges(sel) {
		if s.Node(r.Leaf).typ.AllowsMark(t) {
			return true
		}
	}
	return false
}

func (s State) String() string { return fmt.Sprintf("State{%d nodes}", s.parents.Len()+1) }
