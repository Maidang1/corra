package doc

import "sort"

// tree is a root and its parent links; State and Tx both read through it.
type tree struct {
	root    *Node
	parents *pmap
}

func buildParents(root *Node) *pmap {
	var m *pmap
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, k := range n.kids {
			m = m.Set(uint64(k.id), uint64(n.id))
			walk(k)
		}
	}
	walk(root)
	return m
}

// chain returns the nodes from the root down to id, and each one's index
// in its parent (the root's is -1).
func (t tree) chain(id NodeID) ([]*Node, []int, bool) {
	var ids []NodeID
	for cur := id; cur != t.root.id; {
		ids = append(ids, cur)
		p, ok := t.parents.Get(uint64(cur))
		if !ok {
			return nil, nil, false
		}
		cur = NodeID(p)
		if len(ids) > 10000 {
			return nil, nil, false
		}
	}
	nodes := make([]*Node, 0, len(ids)+1)
	idx := make([]int, 0, len(ids)+1)
	nodes = append(nodes, t.root)
	idx = append(idx, -1)
	cur := t.root
	for i := len(ids) - 1; i >= 0; i-- {
		j := cur.ChildIndex(ids[i])
		if j < 0 {
			return nil, nil, false
		}
		cur = cur.kids[j]
		nodes = append(nodes, cur)
		idx = append(idx, j)
	}
	return nodes, idx, true
}

func (t tree) node(id NodeID) *Node {
	if id == t.root.id {
		return t.root
	}
	nodes, _, ok := t.chain(id)
	if !ok {
		return nil
	}
	return nodes[len(nodes)-1]
}

func (t tree) parent(id NodeID) (NodeID, bool) {
	p, ok := t.parents.Get(uint64(id))
	return NodeID(p), ok
}

func (t tree) contains(id NodeID) bool {
	if id == t.root.id {
		return true
	}
	_, ok := t.parents.Get(uint64(id))
	return ok
}

// start returns the global offset where node id begins.
func (t tree) start(id NodeID) (int, bool) {
	nodes, idx, ok := t.chain(id)
	if !ok {
		return 0, false
	}
	off := 0
	for i := 1; i < len(nodes); i++ {
		off += nodes[i-1].sizes[idx[i]]
	}
	return off, true
}

// globalOffset converts a point to an offset in the linear text.
func (t tree) globalOffset(p Point) (int, bool) {
	off, ok := t.start(p.Leaf)
	if !ok {
		return 0, false
	}
	return off + p.Offset, true
}

// pointAt converts a linear offset to a point, clamped to the text.
func (t tree) pointAt(g int) (Point, bool) {
	n := t.root
	if n.size == 0 {
		return Point{}, false
	}
	g = max(0, min(g, n.size-1))
	for !n.typ.IsTextBlock() {
		// The last child starting at or before g that has any size.
		i := sort.Search(len(n.kids), func(i int) bool { return n.sizes[i] > g }) - 1
		for i >= 0 && n.kids[i].size == 0 {
			i--
		}
		if i < 0 {
			// g sits before the first sized child: take that child.
			for i = 0; i < len(n.kids) && n.kids[i].size == 0; i++ {
			}
			if i == len(n.kids) {
				return Point{}, false
			}
		}
		g -= n.sizes[i]
		n = n.kids[i]
	}
	return Point{Leaf: n.id, Offset: min(g, n.text.Len())}, true
}

// leafRanges splits the global range [a, b] into the text blocks it
// touches, in document order. A block the range only passes through is
// covered whole.
func (t tree) leafRanges(a, b int) []LeafRange {
	var out []LeafRange
	var walk func(n *Node, base int)
	walk = func(n *Node, base int) {
		if n.typ.IsTextBlock() {
			lo, hi := base, base+n.text.Len()
			if hi < a || lo > b {
				return
			}
			out = append(out, LeafRange{Leaf: n.id, From: max(a, lo) - lo, To: min(b, hi) - lo})
			return
		}
		for i, k := range n.kids {
			kb := base + n.sizes[i]
			if kb > b {
				break
			}
			if k.size == 0 || kb+k.size-1 < a {
				continue
			}
			walk(k, kb)
		}
	}
	walk(t.root, 0)
	return out
}

// firstLeaf and lastLeaf return the first and last text block in n.
func firstLeaf(n *Node) *Node {
	if n.typ.IsTextBlock() {
		return n
	}
	for _, k := range n.kids {
		if l := firstLeaf(k); l != nil {
			return l
		}
	}
	return nil
}

func lastLeaf(n *Node) *Node {
	if n.typ.IsTextBlock() {
		return n
	}
	for i := len(n.kids) - 1; i >= 0; i-- {
		if l := lastLeaf(n.kids[i]); l != nil {
			return l
		}
	}
	return nil
}
