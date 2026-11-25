package editor

import (
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// Editor edits one document. Keep it across frames and call View in the
// view.
type Editor struct {
	cfg   Config
	state doc.State
	hist  doc.History

	lay   layouter
	root  *box
	width float32
	env   Env

	scroll     ui.ScrollState
	viewH      float32
	focused    bool
	wantFocus  bool
	typewriter bool
	focusMode  bool
	goalX      float32 // column kept by Up and Down; < 0 when unset
	drag       *dragState

	// Input method composition: the document before it began, and the
	// composed range in the linear text.
	preBase  *doc.State
	preStart int
	preLen   int // the composed text's length
	preOrig  int // the length of what it replaces
	// ruleUndo is set right after an input rule applied: Backspace undoes
	// it.
	ruleUndo  bool
	popup     *popupState
	ordinals  map[doc.NodeID]int
	spellOn   bool
	spelled   map[*doc.Node][]ui.TextInputRange
	blockDrag *blockDrag
	hovering  bool
	hoverY    float32
	compl     []*Completion
	barCache  struct {
		doc     *doc.Node
		sel     doc.TextSelection
		buttons []formatButton
	}
	invalidate   func()
	invalidateFn atomic.Value // func(), for Refresh from other goroutines
	refresh      atomic.Bool

	find findState
}

type dragState struct {
	anchor doc.Point
	clicks int
}

// New returns an editor for doc with cfg.
func New(cfg Config, d *doc.Node) *Editor {
	if cfg.Views == nil {
		cfg.Views = DefaultViews()
	}
	if cfg.Padding == 0 {
		cfg.Padding = 24
	}
	e := &Editor{cfg: cfg, state: doc.NewState(d), goalX: -1}
	e.lay.views = cfg.Views
	e.lay.env = &e.env
	return e
}

// State returns the current state.
func (e *Editor) State() doc.State { return e.state }

// Doc returns the current document.
func (e *Editor) Doc() *doc.Node { return e.state.Doc }

// SetDoc replaces the document, as on loading a file, and forgets the
// history.
func (e *Editor) SetDoc(d *doc.Node) {
	e.endComposition()
	e.state = doc.NewState(d)
	e.hist.Clear()
	e.relayout()
}

// Reload replaces the document with d, read again from its source, as one
// undoable change: blocks with unchanged content keep their identity, and
// the caret stays at its offset in the text.
func (e *Editor) Reload(d *doc.Node) {
	e.endComposition()
	if doc.SameContent(e.state.Doc, d) {
		return
	}
	next := doc.Reconcile(e.state.Doc, d)
	at := 0
	if sel, ok := e.state.TextSel(); ok {
		at = e.state.GlobalOffset(sel.Head)
	}
	e.run(func(s doc.State, tx *doc.Tx) bool {
		root := s.Doc
		tx.SetOrigin(doc.OriginSource)
		if !root.Attrs().Equal(next.Attrs()) {
			tx.SetAttrs(root.ID(), next.Attrs().Map())
		}
		tx.ReplaceChildren(root.ID(), 0, root.NumChildren(), next.Children()...)
		tx.SetSelection(doc.Caret(tx.State().PointAt(at)))
		return true
	}, "source")
}

// ReplaceState moves to s as one change outside the history (an external
// edit), as reloading a changed file keeps the scroll position.
func (e *Editor) ReplaceState(s doc.State, kind string) {
	e.endComposition()
	before := e.state
	e.state = s
	c := doc.Commit{Before: before, After: s, Meta: doc.Meta{Kind: kind, NoHistory: true, Time: time.Now()}}
	if before.Doc != s.Doc {
		c.Ops = []doc.Op{{}} // marks the document as changed
	}
	e.hist.Push(c)
	e.relayout()
	if e.cfg.OnChange != nil {
		e.cfg.OnChange(c)
	}
}

// Run applies cmd as one undoable change; it reports whether cmd applied.
func (e *Editor) Run(cmd doc.Command, kind string) bool {
	e.endComposition()
	return e.run(cmd, kind)
}

func (e *Editor) run(cmd doc.Command, kind string) bool {
	after, c, ok := doc.Run(e.state, cmd, doc.Meta{Kind: kind, Time: time.Now()})
	if !ok {
		return false
	}
	e.commit(after, c)
	return true
}

func (e *Editor) commit(after doc.State, c doc.Commit) {
	e.state = after
	e.hist.Push(c)
	e.relayout()
	if c.DocChanged() {
		e.find.stale = true
		if e.cfg.OnChange != nil {
			e.cfg.OnChange(c)
		}
	}
	e.revealCaret()
	e.redraw()
}

// Command runs the named command; it reports whether it exists and applied.
func (e *Editor) Command(name string) bool {
	switch name {
	case "undo":
		return e.Undo()
	case "redo":
		return e.Redo()
	}
	cmd, ok := e.cfg.Commands[name]
	if !ok {
		return false
	}
	return e.Run(cmd, "format")
}

// Can reports whether the named command applies now, for enabling a
// button.
func (e *Editor) Can(name string) bool {
	cmd, ok := e.cfg.Commands[name]
	return ok && doc.Can(e.state, cmd)
}

func (e *Editor) CanUndo() bool { return e.hist.CanUndo() }
func (e *Editor) CanRedo() bool { return e.hist.CanRedo() }

// Undo goes back one change.
func (e *Editor) Undo() bool {
	e.endComposition()
	s, ok := e.hist.Undo()
	if !ok {
		return false
	}
	e.restore(s)
	return true
}

// Redo goes forward one undone change.
func (e *Editor) Redo() bool {
	e.endComposition()
	s, ok := e.hist.Redo()
	if !ok {
		return false
	}
	e.restore(s)
	return true
}

func (e *Editor) restore(s doc.State) {
	before := e.state
	e.state = s
	e.relayout()
	e.find.stale = true
	if e.cfg.OnChange != nil {
		e.cfg.OnChange(doc.Commit{Before: before, After: s, Ops: []doc.Op{{}}, Meta: doc.Meta{Kind: "history"}})
	}
	e.revealCaret()
	e.redraw()
}

// SetSelection moves the selection without changing the document.
func (e *Editor) SetSelection(sel doc.Selection) {
	e.endComposition()
	e.state.Sel = sel
	e.state.Stored = nil
	e.revealCaret()
	e.redraw()
}

func (e *Editor) redraw() {
	if e.invalidate != nil {
		e.invalidate()
	}
}

// relayout lays the document out again; unchanged nodes keep their boxes.
func (e *Editor) relayout() {
	if e.width <= 0 || e.env.Theme == nil {
		e.root = nil
		return
	}
	e.renumber()
	e.root = e.lay.layout(e.state.Doc, e.width-2*e.cfg.Padding)
}

// Refresh asks, from any goroutine, for the document to be laid out again
// on the next frame, as when a view's background rendering is done.
func (e *Editor) Refresh() {
	e.refresh.Store(true)
	if f, ok := e.invalidateFn.Load().(func()); ok && f != nil {
		f()
	}
}

// Relayout lays everything out again, as when pictures finished loading.
func (e *Editor) Relayout() {
	e.lay.cache = nil
	e.relayout()
	e.redraw()
}

// Height is the height of the laid-out document.
func (e *Editor) Height() float32 {
	if e.root == nil {
		return 0
	}
	return e.root.h + 2*e.cfg.Padding
}

// setEnv updates what the views draw with, laying out again from scratch
// when it changed.
func (e *Editor) setEnv(env Env, width float32) {
	if env != e.env {
		e.env = env
		e.lay.cache = nil
		e.width = width
		e.relayout()
		return
	}
	if width != e.width {
		e.width = width
		e.relayout()
	}
}

// revealCaret scrolls the caret into view.
func (e *Editor) revealCaret() {
	sel, ok := e.state.Sel.(doc.TextSelection)
	if !ok || e.viewH <= 0 {
		return
	}
	r, ok := e.caretRect(sel.Head)
	if !ok {
		return
	}
	if e.typewriter {
		// The caret's line stays in the middle of the view.
		e.scroll.Y = max(0, r.Y+r.H/2-e.viewH/2)
		return
	}
	margin := r.H
	if r.Y-margin < e.scroll.Y {
		e.scroll.Y = max(0, r.Y-margin)
	} else if r.Y+r.H+margin > e.scroll.Y+e.viewH {
		e.scroll.Y = r.Y + r.H + margin - e.viewH
	}
}

// View builds the editor: a scrolling surface that takes text input.
func (e *Editor) View(c *ui.Context, env Env) {
	e.invalidate = c.Invalidate
	if e.invalidateFn.Load() == nil {
		e.invalidateFn.Store(c.Invalidate)
	}
	if env.Theme == nil {
		env.Theme = c.Theme()
	}
	env.Refresh = e
	env.Ordinals = e
	if e.refresh.Swap(false) {
		e.lay.cache = nil
		e.relayout()
	}
	scroll := ui.Scroll(c).TrackScroll(&e.scroll).Fill()
	if b := scroll.Bounds(); b.W > 0 {
		before := e.Height()
		e.setEnv(env, b.W)
		e.viewH = b.H
		if e.Height() != before {
			c.Invalidate()
		}
	} else {
		e.env = env
	}
	scroll.Children(func() {
		// The surface fills the view, so a click below the text focuses
		// it and puts the caret at the end.
		surface := ui.Box(c).FillWidth().Height(max(e.Height(), e.viewH, 1)).
			Label("Document").Role(ui.RoleTextField).
			HandleTextInput(e).HandleInput(e.input).
			Draw(e.draw)
		if e.wantFocus {
			surface.Focus()
			e.wantFocus = false
		}
		if e.cfg.OnDropFiles != nil {
			if files := surface.DroppedFiles(); len(files) > 0 {
				if md := e.cfg.OnDropFiles(files); md != "" {
					e.PasteClip(Clip{Markdown: md})
				}
			}
		}
		e.focused = surface.Focused()
	})
}

// SetTypewriter keeps the caret's line in the middle of the view.
func (e *Editor) SetTypewriter(on bool) { e.typewriter = on; e.revealCaret(); e.redraw() }

// SetFocusMode dims every top-level block but the caret's.
func (e *Editor) SetFocusMode(on bool) { e.focusMode = on; e.redraw() }

// RequestFocus focuses the editor in the next frame.
func (e *Editor) RequestFocus() {
	e.wantFocus = true
	e.redraw()
}

// Focused reports whether the editor had the keyboard in the last frame.
func (e *Editor) Focused() bool { return e.focused }

// Ordinal implements OrdinalSource.
func (e *Editor) Ordinal(n *doc.Node) int { return e.ordinals[n.ID()] }

// renumber counts the nodes of each type in document order; a node whose
// number changed is laid out again, though it did not change itself.
func (e *Editor) renumber() {
	counts := map[*doc.NodeType]int{}
	next := make(map[doc.NodeID]int, len(e.ordinals))
	e.state.Doc.Walk(func(n *doc.Node) bool {
		counts[n.Type()]++
		next[n.ID()] = counts[n.Type()]
		if old, ok := e.ordinals[n.ID()]; ok && old != next[n.ID()] {
			delete(e.lay.cache, n)
		}
		return true
	})
	e.ordinals = next
}

// SetSpellcheck turns underlining of misspelled words on or off.
func (e *Editor) SetSpellcheck(on bool) {
	e.spellOn = on
	e.redraw()
}

// misspelled returns a prose block's misspelled ranges, checked once per
// node: an unchanged block is never checked again.
func (e *Editor) misspelled(n *doc.Node) []ui.TextInputRange {
	if !e.spellOn || e.cfg.Spell == nil || n.Type().Content() != doc.ContentInline {
		return nil
	}
	if r, ok := e.spelled[n]; ok {
		return r
	}
	if e.spelled == nil || len(e.spelled) > 4096 {
		e.spelled = map[*doc.Node][]ui.TextInputRange{}
	}
	r := e.cfg.Spell.Misspelled(n.Inline().String())
	e.spelled[n] = r
	return r
}
