package editor

import (
	"strings"
	"unicode"

	"github.com/Maidang1/corra/internal/doc"
)

type findState struct {
	query   string
	matches []doc.LeafRange
	current int
	stale   bool
}

// Find searches the text blocks for query, ignoring case, and returns how
// many matches there are. Matches are highlighted until Find("").
func (e *Editor) Find(query string) int {
	e.find = findState{query: query, current: -1}
	e.search()
	e.redraw()
	return len(e.find.matches)
}

func (e *Editor) search() {
	e.find.matches = nil
	e.find.stale = false
	q := []rune(strings.ToLower(e.find.query))
	if len(q) == 0 {
		return
	}
	e.state.Doc.Walk(func(n *doc.Node) bool {
		if !n.Type().IsTextBlock() {
			return true
		}
		var rs []rune
		var offs []int
		off := 0
		for _, r := range n.Inline().String() {
			rs = append(rs, unicode.ToLower(r))
			offs = append(offs, off)
			off++
			if r >= 0x10000 {
				off++
			}
		}
		offs = append(offs, off)
		for i := 0; i+len(q) <= len(rs); {
			if string(rs[i:i+len(q)]) == string(q) {
				e.find.matches = append(e.find.matches, doc.LeafRange{Leaf: n.ID(), From: offs[i], To: offs[i+len(q)]})
				i += len(q)
				continue
			}
			i++
		}
		return true
	})
}

// Matches returns the current matches, searching again after an edit.
func (e *Editor) Matches() []doc.LeafRange {
	if e.find.stale {
		e.search()
	}
	return e.find.matches
}

// FindNext selects the next match after the caret, or the previous one
// when back is set, wrapping around; it reports whether there was one.
func (e *Editor) FindNext(back bool) bool {
	ms := e.Matches()
	if len(ms) == 0 {
		return false
	}
	at := 0
	if sel, ok := e.state.TextSel(); ok {
		from, to := e.state.Ordered(sel)
		at = e.global(to)
		if back {
			at = e.global(from)
		}
	}
	idx := -1
	if back {
		for i := len(ms) - 1; i >= 0; i-- {
			if e.global(doc.Point{Leaf: ms[i].Leaf, Offset: ms[i].To}) <= at {
				idx = i
				break
			}
		}
		if idx < 0 {
			idx = len(ms) - 1
		}
	} else {
		for i, m := range ms {
			if e.global(doc.Point{Leaf: m.Leaf, Offset: m.From}) >= at {
				idx = i
				break
			}
		}
		if idx < 0 {
			idx = 0
		}
	}
	m := ms[idx]
	e.find.current = idx
	e.SetSelection(doc.TextSelection{Anchor: doc.Point{Leaf: m.Leaf, Offset: m.From}, Head: doc.Point{Leaf: m.Leaf, Offset: m.To}})
	return true
}

// Replace replaces the selected match with text and finds the next one.
func (e *Editor) Replace(text string) bool {
	sel, ok := e.state.TextSel()
	if !ok || sel.Empty() {
		return e.FindNext(false)
	}
	e.Run(doc.InsertText(text), "input")
	e.find.stale = true
	return e.FindNext(false)
}

// ReplaceAll replaces every match as one change and returns how many.
func (e *Editor) ReplaceAll(text string) int {
	ms := e.Matches()
	if len(ms) == 0 {
		return 0
	}
	e.Run(func(s doc.State, tx *doc.Tx) bool {
		for i := len(ms) - 1; i >= 0; i-- {
			m := ms[i]
			marks := s.Node(m.Leaf).Inline().Slice(m.From, m.To)
			var set []doc.Mark
			if marks.NumSegments() > 0 {
				set = marks.Segment(0).Marks
			}
			tx.ReplaceText(m.Leaf, m.From, m.To, doc.Text(text, set...))
		}
		return true
	}, "input")
	e.find.stale = true
	return len(ms)
}

// matchRanges returns the matches by text block, for drawing.
func (e *Editor) matchRanges() map[doc.NodeID][]doc.LeafRange {
	ms := e.Matches()
	if len(ms) == 0 {
		return nil
	}
	out := map[doc.NodeID][]doc.LeafRange{}
	for _, m := range ms {
		out[m.Leaf] = append(out[m.Leaf], m)
	}
	return out
}
