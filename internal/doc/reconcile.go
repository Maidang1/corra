package doc

// Reconcile returns next with every subtree that has the same content as
// one in prev, in the same order, replaced by prev's node: those keep
// their IDs (and whatever is keyed by them, as laid-out text) when a
// document is read again from its source.
func Reconcile(prev, next *Node) *Node {
	if SameContent(prev, next) {
		return prev
	}
	if prev.typ != next.typ || !prev.typ.IsContainer() {
		return next
	}
	kids := make([]*Node, len(next.kids))
	changed := false
	j := 0 // prev's children are matched in order
	for i, k := range next.kids {
		kids[i] = k
		for m := j; m < len(prev.kids); m++ {
			if SameContent(prev.kids[m], k) {
				kids[i], j = prev.kids[m], m+1
				break
			}
		}
		if kids[i] == k && k.typ.IsContainer() {
			// No equal subtree: the next container of the same type is
			// matched inside, as a quote with one paragraph edited.
			for m := j; m < len(prev.kids); m++ {
				if prev.kids[m].typ == k.typ {
					kids[i], j = Reconcile(prev.kids[m], k), m+1
					break
				}
			}
		}
		changed = changed || kids[i] != k
	}
	if !changed {
		return next
	}
	n := &Node{id: next.id, typ: next.typ, attrs: next.attrs, kids: kids}
	n.resize()
	return n
}
