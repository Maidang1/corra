package doc

// Command is an editing operation. It reads s, the transaction's current
// state, and changes the document through tx. It returns whether it
// applied; one that returns false must not have changed tx.
type Command func(s State, tx *Tx) bool

// Can reports whether cmd applies to s, as for enabling a button. It runs
// cmd on a scratch transaction it throws away; the state is immutable, so
// nothing leaks.
func Can(s State, cmd Command) bool {
	return cmd(s, s.Begin())
}

// Run applies cmd to s as one transaction.
func Run(s State, cmd Command, meta Meta) (State, Commit, bool) {
	tx := s.Begin()
	if meta.Kind != "" {
		tx.Meta.Kind = meta.Kind
	}
	tx.Meta.NoHistory = meta.NoHistory
	if !meta.Time.IsZero() {
		tx.Meta.Time = meta.Time
	}
	if !cmd(tx.State(), tx) {
		return s, Commit{}, false
	}
	after, c := tx.Commit()
	return after, c, true
}

// Chain runs cmds in order in one transaction, stopping at the first
// that does not apply; it applies only if all of them do.
func Chain(cmds ...Command) Command {
	return func(s State, tx *Tx) bool {
		sp := tx.save()
		for _, c := range cmds {
			if !c(tx.State(), tx) {
				tx.restore(sp)
				return false
			}
		}
		return true
	}
}

// FirstOf runs the first of cmds that applies, as Enter, Backspace and
// Tab dispatch to whatever handles the caret's context.
func FirstOf(cmds ...Command) Command {
	return func(s State, tx *Tx) bool {
		for _, c := range cmds {
			sp := tx.save()
			if c(tx.State(), tx) {
				return true
			}
			tx.restore(sp)
		}
		return false
	}
}

// savepoint is a transaction's state to roll back to when a command that
// changed it reports it did not apply.
type savepoint struct {
	root      *Node
	parents   *pmap
	sel       Selection
	selSet    bool
	stored    *MarkSet
	storedSet bool
	ops       int
	fallback  int
	removed   bool
}

// save freezes the nodes the transaction owns, so later changes copy
// them and restore can bring back the tree as it is now.
func (tx *Tx) save() savepoint {
	for _, n := range tx.owned {
		n.frozen = true
	}
	return savepoint{tx.t.root, tx.t.parents, tx.sel, tx.selSet, tx.stored, tx.storedSet, len(tx.ops), tx.fallback, tx.removed}
}

func (tx *Tx) restore(sp savepoint) {
	tx.t.root, tx.t.parents = sp.root, sp.parents
	tx.sel, tx.selSet, tx.stored, tx.storedSet = sp.sel, sp.selSet, sp.stored, sp.storedSet
	tx.ops = tx.ops[:sp.ops]
	tx.fallback, tx.removed = sp.fallback, sp.removed
}
