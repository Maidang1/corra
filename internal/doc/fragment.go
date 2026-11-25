package doc

import "strings"

// Fragment copies what a text selection covers as standalone blocks with
// fresh IDs, for the clipboard: the blocks between its ends at the level
// where they are siblings, with the text before the start and after the
// end cut off. Within one text block it is that block with the selected
// part of its text.
func (s State) Fragment(sel TextSelection) []*Node {
	from, to := s.Ordered(sel)
	if from == to {
		return nil
	}
	if from.Leaf == to.Leaf {
		l := s.Node(from.Leaf)
		return []*Node{NewTextBlock(l.typ, l.attrs.Map(), cloneInlineFresh(l.text.Slice(from.Offset, to.Offset)))}
	}
	p, i, j := siblingRange(s, sel)
	a, b := s.Ancestors(from.Leaf), s.Ancestors(to.Leaf)
	depth := len(s.Ancestors(p.id))
	var out []*Node
	for k := i; k <= j; k++ {
		var left, right []*Node
		if k == i {
			left = a[depth:]
		}
		if k == j {
			right = b[depth:]
		}
		out = append(out, clonePart(p.kids[k], left, from.Offset, right, to.Offset))
	}
	return out
}

// clonePart copies n, cut on the left along the path left (from n down to
// a text block, cut before offset lo) and on the right along right (cut
// after hi). A nil path does not cut that side.
func clonePart(n *Node, left []*Node, lo int, right []*Node, hi int) *Node {
	switch n.typ.spec.Content {
	case ContentInline, ContentText:
		a, z := 0, n.text.Len()
		if left != nil {
			a = lo
		}
		if right != nil {
			z = hi
		}
		return NewTextBlock(n.typ, n.attrs.Map(), cloneInlineFresh(n.text.Slice(a, z)))
	case ContentBlocks:
		start, end := 0, len(n.kids)-1
		if left != nil && len(left) > 1 {
			start = n.ChildIndex(left[1].id)
		}
		if right != nil && len(right) > 1 {
			end = n.ChildIndex(right[1].id)
		}
		var kids []*Node
		for k := start; k <= end; k++ {
			var l, r []*Node
			if left != nil && len(left) > 1 && k == start {
				l = left[1:]
			}
			if right != nil && len(right) > 1 && k == end {
				r = right[1:]
			}
			kids = append(kids, clonePart(n.kids[k], l, lo, r, hi))
		}
		return NewBlock(n.typ, n.attrs.Map(), kids...)
	}
	return CloneFresh(n)
}

// PlainText returns the text of blocks, a line per text block, inline
// atoms as U+FFFC.
func PlainText(blocks []*Node) string {
	var lines []string
	for _, b := range blocks {
		b.Walk(func(n *Node) bool {
			if n.typ.IsTextBlock() {
				lines = append(lines, n.text.String())
			}
			return true
		})
	}
	return strings.Join(lines, "\n")
}
