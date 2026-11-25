package doc

import "math/bits"

// pmap is a persistent hash array mapped trie from node IDs to node IDs,
// used for parent links. Set and Delete copy only the path to the key, so
// every State keeps its own map while sharing almost all of it.
type pmap struct {
	root *hnode
	n    int
}

type hnode struct {
	bitmap  uint32
	entries []hentry
}

type hentry struct {
	key, val uint64
	child    *hnode // non-nil for a subtree
}

const hbits = 5

// mix is splitmix64's finalizer: a bijection, so distinct keys always
// diverge at some level and no collision buckets are needed.
func mix(k uint64) uint64 {
	k ^= k >> 30
	k *= 0xbf58476d1ce4e5b9
	k ^= k >> 27
	k *= 0x94d049bb133111eb
	k ^= k >> 31
	return k
}

func slot(h uint64, level int) uint32 { return uint32(h>>(uint(level)*hbits)) & 31 }

func (m *pmap) Len() int {
	if m == nil {
		return 0
	}
	return m.n
}

func (m *pmap) Get(key uint64) (uint64, bool) {
	if m == nil {
		return 0, false
	}
	h := mix(key)
	n := m.root
	for level := 0; n != nil; level++ {
		bit := uint32(1) << slot(h, level)
		if n.bitmap&bit == 0 {
			return 0, false
		}
		e := n.entries[bits.OnesCount32(n.bitmap&(bit-1))]
		if e.child == nil {
			if e.key == key {
				return e.val, true
			}
			return 0, false
		}
		n = e.child
	}
	return 0, false
}

// Set returns a map with key set to val.
func (m *pmap) Set(key, val uint64) *pmap {
	var root *hnode
	var n int
	if m != nil {
		root, n = m.root, m.n
	}
	newRoot, added := set(root, mix(key), key, val, 0)
	if added {
		n++
	}
	return &pmap{root: newRoot, n: n}
}

func set(n *hnode, h, key, val uint64, level int) (*hnode, bool) {
	bit := uint32(1) << slot(h, level)
	if n == nil {
		return &hnode{bitmap: bit, entries: []hentry{{key: key, val: val}}}, true
	}
	i := bits.OnesCount32(n.bitmap & (bit - 1))
	c := &hnode{bitmap: n.bitmap, entries: append([]hentry(nil), n.entries...)}
	if n.bitmap&bit == 0 {
		c.bitmap |= bit
		c.entries = append(c.entries, hentry{})
		copy(c.entries[i+1:], c.entries[i:])
		c.entries[i] = hentry{key: key, val: val}
		return c, true
	}
	e := n.entries[i]
	switch {
	case e.child != nil:
		child, added := set(e.child, h, key, val, level+1)
		c.entries[i] = hentry{child: child}
		return c, added
	case e.key == key:
		c.entries[i] = hentry{key: key, val: val}
		return c, false
	default:
		// Push the existing leaf one level down, then add the new one.
		sub, _ := set(nil, mix(e.key), e.key, e.val, level+1)
		sub, _ = set(sub, h, key, val, level+1)
		c.entries[i] = hentry{child: sub}
		return c, true
	}
}

// Delete returns a map without key.
func (m *pmap) Delete(key uint64) *pmap {
	if m == nil {
		return nil
	}
	root, removed := del(m.root, mix(key), key, 0)
	if !removed {
		return m
	}
	return &pmap{root: root, n: m.n - 1}
}

func del(n *hnode, h, key uint64, level int) (*hnode, bool) {
	if n == nil {
		return nil, false
	}
	bit := uint32(1) << slot(h, level)
	if n.bitmap&bit == 0 {
		return n, false
	}
	i := bits.OnesCount32(n.bitmap & (bit - 1))
	e := n.entries[i]
	if e.child == nil {
		if e.key != key {
			return n, false
		}
		if len(n.entries) == 1 {
			return nil, true
		}
		c := &hnode{bitmap: n.bitmap &^ bit, entries: make([]hentry, 0, len(n.entries)-1)}
		c.entries = append(c.entries, n.entries[:i]...)
		c.entries = append(c.entries, n.entries[i+1:]...)
		return c, true
	}
	child, removed := del(e.child, h, key, level+1)
	if !removed {
		return n, false
	}
	c := &hnode{bitmap: n.bitmap, entries: append([]hentry(nil), n.entries...)}
	switch {
	case child == nil:
		c.bitmap &^= bit
		c.entries = append(c.entries[:i], c.entries[i+1:]...)
		if len(c.entries) == 0 {
			return nil, true
		}
	case len(child.entries) == 1 && child.entries[0].child == nil:
		// Pull a lone leaf back up.
		c.entries[i] = child.entries[0]
	default:
		c.entries[i] = hentry{child: child}
	}
	return c, true
}
