package editor

import "time"

const (
	maxUndo  = 200
	coalesce = time.Second
)

// Undo is an undo/redo stack of document snapshots. Edits made within
// coalesce of the previous one merge into a single undo step, so typing a
// word is one undo, not one undo per character.
type Undo struct {
	undo   []snapshot
	redo   []snapshot
	last   time.Time
	hasTop bool
}

type snapshot struct {
	src   string
	caret int
}

// PushBase records the initial state (document load). It always starts a
// new run, never coalesces.
func (u *Undo) PushBase(src string) {
	u.undo = []snapshot{{src: src}}
	u.redo = u.redo[:0]
	u.last = time.Now()
	u.hasTop = true
}

// Push records a new state. caret is the caret position that produced it.
func (u *Undo) Push(src string, caret int) {
	u.PushAt(src, caret, time.Now())
}

// PushAt is Push with an explicit time, for tests.
func (u *Undo) PushAt(src string, caret int, now time.Time) {
	if u.hasTop && now.Sub(u.last) < coalesce {
		// Same typing run: keep only the latest state of the run.
		u.undo[len(u.undo)-1] = snapshot{src, caret}
	} else {
		u.undo = append(u.undo, snapshot{src, caret})
		if len(u.undo) > maxUndo {
			u.undo = u.undo[len(u.undo)-maxUndo:]
		}
	}
	u.redo = u.redo[:0]
	u.last = now
	u.hasTop = true
}

// Top returns the current state.
func (u *Undo) Top() (string, int, bool) {
	if !u.hasTop || len(u.undo) == 0 {
		return "", 0, false
	}
	s := u.undo[len(u.undo)-1]
	return s.src, s.caret, true
}

// Undo steps back, returning the restored state and caret.
func (u *Undo) Undo(current string, caret int) (string, int, bool) {
	if len(u.undo) < 2 {
		return "", 0, false
	}
	u.redo = append(u.redo, u.undo[len(u.undo)-1])
	u.undo = u.undo[:len(u.undo)-1]
	s := u.undo[len(u.undo)-1]
	return s.src, s.caret, true
}

// Redo steps forward.
func (u *Undo) Redo(current string, caret int) (string, int, bool) {
	if len(u.redo) == 0 {
		return "", 0, false
	}
	s := u.redo[len(u.redo)-1]
	u.redo = u.redo[:len(u.redo)-1]
	u.undo = append(u.undo, s)
	return s.src, s.caret, true
}

// CanUndo reports whether a step back exists.
func (u *Undo) CanUndo() bool { return len(u.undo) >= 2 }

// CanRedo reports whether a step forward exists.
func (u *Undo) CanRedo() bool { return len(u.redo) > 0 }

// Clear empties both stacks (on document open).
func (u *Undo) Clear() {
	u.undo = u.undo[:0]
	u.redo = u.redo[:0]
	u.last = time.Time{}
	u.hasTop = false
}
