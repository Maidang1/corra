package doctest

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/Maidang1/corra/internal/doc"
)

// Iterations returns n, or CORRA_FUZZ_ITERS when set, so a local run can
// go longer than CI's.
func Iterations(n int) int {
	if v, err := strconv.Atoi(os.Getenv("CORRA_FUZZ_ITERS")); err == nil && v > 0 {
		return v
	}
	return n
}

// Rand generates documents, selections and commands in the test schema.
type Rand struct {
	*Builder
	R *rand.Rand
}

func NewRand(seed uint64) *Rand {
	return &Rand{Builder: NewBuilder(), R: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

var words = []string{"a", "bc", "def", "中文", "😀", " ", "x y"}

func (r *Rand) inline() []Part {
	var parts []Part
	for i := r.R.IntN(4); i >= 0; i-- {
		w := words[r.R.IntN(len(words))]
		switch r.R.IntN(6) {
		case 0:
			parts = append(parts, r.B(w))
		case 1:
			parts = append(parts, r.I(w))
		case 2:
			parts = append(parts, r.C(w))
		case 3:
			parts = append(parts, r.Link("u", w))
		case 4:
			parts = append(parts, r.Math("x"))
		default:
			parts = append(parts, w)
		}
	}
	return parts
}

func (r *Rand) block(depth int) *doc.Node {
	k := r.R.IntN(9)
	if depth <= 0 && k >= 5 {
		k = r.R.IntN(5)
	}
	switch k {
	case 0, 1:
		return r.P(r.inline()...)
	case 2:
		return r.H(1+r.R.IntN(3), r.inline()...)
	case 3:
		return r.Pre("", words[r.R.IntN(len(words))])
	case 4:
		if r.R.IntN(2) == 0 {
			return r.Img("i")
		}
		return r.Builder.HR()
	case 5, 6:
		return r.Quote(r.blocks(depth - 1)...)
	default:
		var items []*doc.Node
		for i := r.R.IntN(3); i >= 0; i-- {
			items = append(items, r.LI(r.blocks(depth-1)...))
		}
		return r.UL(items...)
	}
}

func (r *Rand) blocks(depth int) []*doc.Node {
	var out []*doc.Node
	for i := r.R.IntN(3); i >= 0; i-- {
		out = append(out, r.block(depth))
	}
	return out
}

// Doc returns a random valid document.
func (r *Rand) Doc() *doc.Node {
	d := r.Builder.Doc(r.blocks(2)...)
	if err := doc.Validate(d); err != nil {
		// Generated containers can be empty only through a bug here.
		panic(err)
	}
	return d
}

func (r *Rand) point(s doc.State) doc.Point {
	return s.PointAt(r.R.IntN(s.Doc.Size() + 1))
}

// Selection returns a random selection in s: mostly carets and ranges,
// sometimes a selected block.
func (r *Rand) Selection(s doc.State) doc.Selection {
	switch r.R.IntN(6) {
	case 0:
		var ids []doc.NodeID
		s.Doc.Walk(func(n *doc.Node) bool {
			if n.Type().IsAtom() {
				ids = append(ids, n.ID())
			}
			return true
		})
		if len(ids) > 0 {
			return doc.NodeSelection{IDs: []doc.NodeID{ids[r.R.IntN(len(ids))]}}
		}
		fallthrough
	case 1, 2:
		return doc.Caret(r.point(s))
	default:
		return doc.TextSelection{Anchor: r.point(s), Head: r.point(s)}
	}
}

// Command returns a random command and its name.
func (r *Rand) Command(s doc.State) (string, doc.Command) {
	var blocks []*doc.Node
	s.Doc.Walk(func(n *doc.Node) bool {
		if n != s.Doc {
			blocks = append(blocks, n)
		}
		return true
	})
	var containers []*doc.Node
	s.Doc.Walk(func(n *doc.Node) bool {
		if n.Type().IsContainer() {
			containers = append(containers, n)
		}
		return true
	})
	switch r.R.IntN(16) {
	case 0, 1:
		w := words[r.R.IntN(len(words))]
		return "InsertText " + w, doc.InsertText(w)
	case 2:
		return "InsertText newline", doc.InsertText("p\nq")
	case 3:
		return "DeleteSelection", doc.DeleteSelection
	case 4:
		return "SplitBlock", doc.SplitBlock
	case 5:
		return "JoinBackward", doc.JoinBackward
	case 6:
		return "JoinForward", doc.JoinForward
	case 7:
		m := []*doc.MarkType{r.Bold, r.Italic, r.Schema.Code}[r.R.IntN(3)]
		return "ToggleMark " + m.Name(), doc.ToggleMark(m)
	case 8:
		return "AddLink", doc.AddMark(doc.NewMark(r.Schema.Link, doc.A{"href": "v"}))
	case 9:
		switch r.R.IntN(3) {
		case 0:
			return "SetBlockType heading", doc.SetBlockType(r.Heading, doc.A{"level": 2})
		case 1:
			return "SetBlockType code", doc.SetBlockType(r.CodeBlock, nil)
		}
		return "SetBlockType paragraph", doc.SetBlockType(r.Paragraph, nil)
	case 10:
		if r.R.IntN(2) == 0 {
			return "Wrap list", doc.Wrap(r.BulletList, nil)
		}
		return "Wrap quote", doc.Wrap(r.Blockquote, nil)
	case 11:
		return "Lift", doc.Lift
	case 12:
		if len(blocks) == 0 || len(containers) == 0 {
			return "SelectAll", doc.SelectAll
		}
		n := blocks[r.R.IntN(len(blocks))]
		c := containers[r.R.IntN(len(containers))]
		i := r.R.IntN(c.NumChildren() + 1)
		return fmt.Sprintf("MoveBlock %s to %s@%d", n.Type().Name(), c.Type().Name(), i), doc.MoveBlock(n.ID(), c.ID(), i)
	case 13:
		return "InsertBlocks", doc.InsertBlocks(r.block(1))
	case 14:
		return "InsertFragment", doc.InsertFragment(r.blocks(1))
	default:
		return "SelectAll+Delete", doc.Chain(doc.SelectAll, doc.DeleteSelection)
	}
}

// Fuzz runs random command sequences on random documents and checks
// after each command that the document is valid, the old one unchanged,
// the op log replays both ways and undo returns the old document.
func Fuzz(t *testing.T, seed uint64, docs, steps int) {
	r := NewRand(seed)
	for d := 0; d < docs; d++ {
		s := doc.NewState(r.Doc())
		var h doc.History
		var trail []string
		for i := 0; i < steps; i++ {
			s.Sel = r.Selection(s)
			name, cmd := r.Command(s)
			trail = append(trail, name)
			before := DumpIDs(s.Doc)
			var after doc.State
			var c doc.Commit
			var ok bool
			func() {
				defer func() {
					if e := recover(); e != nil {
						t.Fatalf("seed %d doc %d: %s panicked: %v\nstate %s\ntrail %v", seed, d, name, e, DumpState(s), trail)
					}
				}()
				after, c, ok = doc.Run(s, cmd, doc.Meta{Kind: "structure"})
			}()
			if !ok {
				continue
			}
			if got := DumpIDs(s.Doc); got != before {
				t.Fatalf("seed %d: %s changed the old document\nbefore %s\nafter  %s", seed, name, before, got)
			}
			if err := doc.Validate(after.Doc); err != nil {
				t.Fatalf("seed %d: %s left an invalid document: %v\nbefore %s\nafter  %s\ntrail %v", seed, name, err, DumpState(s), DumpIDs(after.Doc), trail)
			}
			if !after.Valid(selectionHead(after)) && isText(after.Sel) {
				t.Fatalf("seed %d: %s left an invalid selection %+v in %s", seed, name, after.Sel, DumpIDs(after.Doc))
			}
			CheckLog(t, c)
			h.Push(c)
			if c.DocChanged() {
				prev, ok := h.Undo()
				if !ok || prev.Doc != s.Doc {
					t.Fatalf("seed %d: undo after %s does not return the old document", seed, name)
				}
				if next, _ := h.Redo(); next.Doc != after.Doc {
					t.Fatalf("seed %d: redo after %s does not return the new document", seed, name)
				}
			}
			s = after
		}
	}
}

func isText(s doc.Selection) bool {
	_, ok := s.(doc.TextSelection)
	return ok
}

func selectionHead(s doc.State) doc.Point {
	if sel, ok := s.Sel.(doc.TextSelection); ok {
		return sel.Head
	}
	return doc.Point{}
}
