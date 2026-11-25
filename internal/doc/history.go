package doc

import "time"

const (
	historyLimit = 200
	mergeWindow  = time.Second
)

// History is one undo history for the document. Each entry keeps the
// states before and after a change; since states share untouched nodes,
// an entry costs about as much as the change itself.
type History struct {
	undo, redo []entry
}

type entry struct {
	before, after State
	kind          string
	leaf          NodeID
	end           int // where the last merged edit ended in leaf
	at            time.Time
}

// Push records a committed transaction. Consecutive typing or deleting
// in one text block, each edit starting where the last ended and within a
// second of it, merge into one entry.
func (h *History) Push(c Commit) {
	if !c.DocChanged() || c.Meta.NoHistory {
		if c.DocChanged() {
			// An edit outside the history invalidates it: its states no
			// longer lead to the current document.
			h.undo, h.redo = nil, nil
		}
		return
	}
	e := entry{before: c.Before, after: c.After, kind: c.Meta.Kind, at: c.Meta.Time}
	if leaf, from, to, end, ok := singleTextEdit(c); ok {
		e.leaf, e.end = leaf, end
		if n := len(h.undo); n > 0 {
			last := &h.undo[n-1]
			adjacent := from == last.end || (e.kind == "delete" && to == last.end)
			if (e.kind == "input" || e.kind == "delete") && last.kind == e.kind && last.leaf == leaf &&
				adjacent && e.at.Sub(last.at) < mergeWindow {
				last.after, last.end, last.at = c.After, end, e.at
				h.redo = nil
				return
			}
		}
	}
	h.undo = append(h.undo, e)
	if len(h.undo) > historyLimit {
		h.undo = append(h.undo[:0:0], h.undo[len(h.undo)-historyLimit:]...)
	}
	h.redo = nil
}

// singleTextEdit reports whether a commit is one text replacement: the
// replaced range, and where the new text ends.
func singleTextEdit(c Commit) (leaf NodeID, from, to, end int, ok bool) {
	if len(c.Ops) != 1 || c.Ops[0].Kind != OpReplaceText {
		return 0, 0, 0, 0, false
	}
	o := c.Ops[0]
	return o.Target, o.From, o.To, o.From + o.NewText.Len(), true
}

func (h *History) CanUndo() bool { return len(h.undo) > 0 }
func (h *History) CanRedo() bool { return len(h.redo) > 0 }

// Undo returns the state before the last entry.
func (h *History) Undo() (State, bool) {
	if len(h.undo) == 0 {
		return State{}, false
	}
	e := h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	h.redo = append(h.redo, e)
	return e.before, true
}

// Redo returns the state after the last undone entry.
func (h *History) Redo() (State, bool) {
	if len(h.redo) == 0 {
		return State{}, false
	}
	e := h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	h.undo = append(h.undo, e)
	return e.after, true
}

// Break ends merging: the next edit starts a new entry.
func (h *History) Break() {
	if n := len(h.undo); n > 0 {
		h.undo[n-1].kind = ""
	}
}

// Clear forgets all entries.
func (h *History) Clear() { h.undo, h.redo = nil, nil }
