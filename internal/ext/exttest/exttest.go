// Package exttest checks extensions: Markdown round trips, documents
// generated from any schema, and commands on them.
package exttest

import (
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/doc/doctest"
	"github.com/Maidang1/corra/internal/ext"
)

// CheckMarkdown checks that each sample reads, and that writing what was
// read and reading it again gives the same document and, written once
// more, the same Markdown: the first save may normalize, later ones do not
// change anything.
func CheckMarkdown(t *testing.T, k *ext.Kit, samples ...string) {
	t.Helper()
	for _, src := range samples {
		d := k.Codec.Parse(src)
		md1 := k.Codec.Write(d)
		d2 := k.Codec.Parse(md1)
		if !doc.SameContent(d, d2) {
			t.Errorf("reading back changes the document\nsource:  %q\nwritten: %q\nfirst:  %s\nsecond: %s", src, md1, doctest.Dump(d), doctest.Dump(d2))
			continue
		}
		if md2 := k.Codec.Write(d2); md2 != md1 {
			t.Errorf("writing is not stable\nsource: %q\nfirst:  %q\nsecond: %q", src, md1, md2)
		}
	}
}

// Gen generates documents from a kit's schema.
type Gen struct {
	Kit *ext.Kit
	R   *rand.Rand
	// Words are the text pieces; they include Markdown syntax on purpose.
	Words []string
}

func NewGen(k *ext.Kit, seed uint64) *Gen {
	return &Gen{Kit: k, R: rand.New(rand.NewPCG(seed, seed+1)), Words: []string{
		"a", "bc", "中文", "😀", "x_y", "*", "1.", "#", "[x]", "`", `\`, "&amp;", "<b>", "~", "-", "+", "> q", "a|b", "!", "_",
	}}
}

func generated(name string) bool { return !strings.HasPrefix(name, "raw") }

func (g *Gen) attrs(t *doc.NodeType) doc.A {
	a := doc.A{}
	for k, spec := range t.Spec().Attrs {
		var tries []any
		switch spec.Default.(type) {
		case bool:
			tries = []any{g.R.IntN(2) == 0}
		case int:
			tries = []any{1 + g.R.IntN(6)}
		case string, nil:
			// A required attribute (no default) is taken to be a string.
			tries = []any{[]string{"", "x", "go"}[g.R.IntN(3)], "x"}
		}
		for _, v := range tries {
			if spec.Validate == nil || spec.Validate(v) == nil {
				a[k] = v
				break
			}
		}
	}
	return a
}

func (g *Gen) word() string { return g.Words[g.R.IntN(len(g.Words))] }

func (g *Gen) inline(t *doc.NodeType) doc.Inline {
	reg := g.Kit.Registry
	var marks []*doc.MarkType
	for _, m := range reg.Marks() {
		if t.AllowsMark(m) && generated(m.Name()) {
			marks = append(marks, m)
		}
	}
	var atoms []*doc.NodeType
	for _, n := range reg.Nodes() {
		if t.Allows(n) && generated(n.Name()) && n.Name() != "hard_break" {
			atoms = append(atoms, n)
		}
	}
	var segs []doc.Segment
	for i := g.R.IntN(5); i >= 0; i-- {
		if i > 0 && len(segs) > 0 {
			sep := " "
			if g.R.IntN(8) == 0 {
				// A line break between words, never at an edge.
				hb := reg.Node("hard_break")
				if hb != nil && t.Allows(hb) {
					segs = append(segs, doc.Segment{Atom: doc.NewAtom(hb, doc.A{"soft": g.R.IntN(2) == 0})})
					sep = ""
				}
			}
			segs = append(segs, doc.Segment{Text: sep})
		}
		if len(atoms) > 0 && g.R.IntN(6) == 0 {
			t := atoms[g.R.IntN(len(atoms))]
			segs = append(segs, doc.Segment{Atom: doc.NewAtom(t, g.attrs(t))})
			continue
		}
		var set doc.MarkSet
		if len(marks) > 0 && g.R.IntN(2) == 0 {
			m := marks[g.R.IntN(len(marks))]
			set = set.Add(doc.NewMark(m, g.markAttrs(m)))
		}
		segs = append(segs, doc.Segment{Text: g.word(), Marks: set})
	}
	return doc.NewInline(segs...)
}

func (g *Gen) markAttrs(m *doc.MarkType) doc.A {
	a := doc.A{}
	for k, spec := range m.Spec().Attrs {
		if _, ok := spec.Default.(string); ok {
			for _, v := range []string{[]string{"u", "http://x.y/z", "a b"}[g.R.IntN(3)], "#d44c47", "red"} {
				if spec.Validate == nil || spec.Validate(v) == nil {
					a[k] = v
					break
				}
			}
		}
	}
	return a
}

// Node generates a node of type t.
func (g *Gen) Node(t *doc.NodeType, depth int) *doc.Node {
	switch t.Content() {
	case doc.ContentInline:
		return doc.NewTextBlock(t, g.attrs(t), g.inline(t))
	case doc.ContentText:
		var lines []string
		for i := g.R.IntN(3); i >= 0; i-- {
			lines = append(lines, g.word()+" "+g.word())
		}
		return doc.NewTextBlock(t, g.attrs(t), doc.Text(strings.Join(lines, "\n")))
	case doc.ContentNone:
		return doc.NewAtom(t, g.attrs(t))
	}
	var kids []*doc.Node
	var choices []*doc.NodeType
	for _, c := range g.Kit.Registry.Nodes() {
		if t.Allows(c) && generated(c.Name()) && (depth > 0 || !c.IsContainer()) {
			choices = append(choices, c)
		}
	}
	if len(choices) == 0 {
		return doc.Empty(t)
	}
	for i := g.R.IntN(3); i >= 0; i-- {
		kids = append(kids, g.Node(choices[g.R.IntN(len(choices))], depth-1))
	}
	return doc.NewBlock(t, g.attrs(t), kids...)
}

// Doc generates a valid document.
func (g *Gen) Doc() *doc.Node {
	return doc.Normalized(g.Node(g.Kit.Registry.Root(), 3))
}

func flatText(n *doc.Node) string {
	var b strings.Builder
	n.Walk(func(d *doc.Node) bool {
		if d.Type().IsTextBlock() {
			in := d.Inline()
			for i := 0; i < in.NumSegments(); i++ {
				if seg := in.Segment(i); seg.Atom == nil {
					b.WriteString(seg.Text)
				} else if seg.Atom.Type().Name() == "hard_break" {
					// Whitespace before a line break is dropped by Markdown;
					// the break itself separates the words.
					b.WriteByte(' ')
				}
			}
			b.WriteByte(' ')
		}
		return true
	})
	return strings.Join(strings.FieldsFunc(b.String(), unicode.IsSpace), " ")
}

func countRaw(n *doc.Node) int {
	c := 0
	n.Walk(func(d *doc.Node) bool {
		if d.Type().Name() == "raw" {
			c++
		}
		if d.Type().IsTextBlock() {
			for i := 0; i < d.Inline().NumSegments(); i++ {
				for _, m := range d.Inline().Segment(i).Marks {
					if m.Type.Name() == "raw_inline" {
						c++
					}
				}
			}
		}
		return true
	})
	return c
}

// CheckRandom writes and reads back random documents. The first read may
// normalize (marks move off spaces, empty paragraphs go); after it the
// round trip must be exact, the text must survive, and nothing may read
// back as raw source.
func CheckRandom(t *testing.T, k *ext.Kit, seed uint64, n int) {
	t.Helper()
	g := NewGen(k, seed)
	for i := 0; i < n; i++ {
		d0 := g.Doc()
		md := k.Codec.Write(d0)
		d1 := k.Codec.Parse(md)
		if c := countRaw(d1); c > 0 {
			t.Fatalf("seed %d #%d: written Markdown reads back as raw source\ndoc: %s\nmd:  %q\nread: %s", seed, i, doctest.Dump(d0), md, doctest.Dump(d1))
		}
		if a, b := flatText(d0), flatText(d1); a != b {
			t.Fatalf("seed %d #%d: text changed\nwant %q\ngot  %q\ndoc: %s\nmd:  %q\nread: %s", seed, i, a, b, doctest.Dump(d0), md, doctest.Dump(d1))
		}
		md1 := k.Codec.Write(d1)
		d2 := k.Codec.Parse(md1)
		if !doc.SameContent(d1, d2) || k.Codec.Write(d2) != md1 {
			t.Fatalf("seed %d #%d: not stable after one round\nmd1: %q\nd1: %s\nd2: %s\nmd2: %q", seed, i, md1, doctest.Dump(d1), doctest.Dump(d2), k.Codec.Write(d2))
		}
	}
}

// CheckCommands runs the kit's named commands and key chains on random
// documents and selections, checking validity and the op log.
func CheckCommands(t *testing.T, k *ext.Kit, seed uint64, docs, steps int) {
	t.Helper()
	g := NewGen(k, seed)
	r := doctest.NewRand(seed)
	var names []string
	cmds := map[string]doc.Command{}
	for name, c := range k.Commands {
		names = append(names, name)
		cmds[name] = c
	}
	for key, c := range k.Chains {
		names = append(names, "key "+key)
		cmds["key "+key] = c
	}
	names = append(names, "type")
	cmds["type"] = doc.InsertText("w")
	sort.Strings(names)
	for d := 0; d < docs; d++ {
		s := doc.NewState(g.Doc())
		var trail []string
		for i := 0; i < steps; i++ {
			s.Sel = r.Selection(s)
			name := names[g.R.IntN(len(names))]
			trail = append(trail, name)
			before := doctest.DumpIDs(s.Doc)
			var after doc.State
			var c doc.Commit
			var ok bool
			func() {
				defer func() {
					if e := recover(); e != nil {
						t.Fatalf("seed %d: %s panicked: %v\nstate %s\ntrail %v", seed, name, e, doctest.DumpState(s), trail)
					}
				}()
				after, c, ok = doc.Run(s, cmds[name], doc.Meta{Kind: "structure"})
			}()
			if !ok {
				continue
			}
			if doctest.DumpIDs(s.Doc) != before {
				t.Fatalf("seed %d: %s changed the old document", seed, name)
			}
			if err := doc.Validate(after.Doc); err != nil {
				t.Fatalf("seed %d: %s: %v\nbefore %s\nafter %s\ntrail %v", seed, name, err, doctest.DumpState(s), doctest.Dump(after.Doc), trail)
			}
			doctest.CheckLog(t, c)
			s = after
		}
	}
	_ = slices.Contains[[]string]
}
