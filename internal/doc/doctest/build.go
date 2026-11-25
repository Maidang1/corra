package doctest

import (
	"fmt"
	"strings"

	"github.com/Maidang1/corra/internal/doc"
)

// Part is inline content for P and H: a string, or the result of B, I, C,
// Link or Math. In strings, "|" marks a caret, "[" a selection's anchor
// and "]" its head; they are removed from the text.
type Part = any

type piece struct {
	text  string
	marks []doc.Mark
	atom  *doc.Node
}

// Builder builds trees in the test schema and remembers selection markers.
type Builder struct {
	*Schema
	anchor, head *doc.Point
}

// NewBuilder returns a builder over a new test schema.
func NewBuilder() *Builder { return &Builder{Schema: NewSchema()} }

func (b *Builder) pieces(parts []Part, marks []doc.Mark) []piece {
	var out []piece
	for _, p := range parts {
		switch p := p.(type) {
		case string:
			out = append(out, piece{text: p, marks: marks})
		case []piece:
			for _, q := range p {
				q.marks = append(append([]doc.Mark(nil), marks...), q.marks...)
				out = append(out, q)
			}
		default:
			panic(fmt.Sprintf("doctest: inline part %T", p))
		}
	}
	return out
}

func (b *Builder) marked(m doc.Mark, parts []Part) []piece { return b.pieces(parts, []doc.Mark{m}) }

// B, I and C mark their parts bold, italic and code.
func (b *Builder) B(parts ...Part) []piece { return b.marked(doc.Mark{Type: b.Bold}, parts) }
func (b *Builder) I(parts ...Part) []piece { return b.marked(doc.Mark{Type: b.Italic}, parts) }
func (b *Builder) C(parts ...Part) []piece { return b.marked(doc.Mark{Type: b.Schema.Code}, parts) }

// Link links its parts to href.
func (b *Builder) Link(href string, parts ...Part) []piece {
	return b.marked(doc.NewMark(b.Schema.Link, doc.A{"href": href}), parts)
}

// Math is an inline formula.
func (b *Builder) Math(tex string) []piece {
	return []piece{{atom: doc.NewAtom(b.Schema.Math, doc.A{"tex": tex})}}
}

// inline builds content, recording markers against leaf (set later).
type marker struct {
	kind byte
	off  int
}

func (b *Builder) inline(parts []Part) (doc.Inline, []marker) {
	var segs []doc.Segment
	var marks []marker
	off := 0
	for _, p := range b.pieces(parts, nil) {
		if p.atom != nil {
			var set doc.MarkSet
			for _, m := range p.marks {
				set = set.Add(m)
			}
			segs = append(segs, doc.Segment{Atom: p.atom, Marks: set})
			off++
			continue
		}
		var text strings.Builder
		for _, r := range p.text {
			if r == '|' || r == '[' || r == ']' {
				marks = append(marks, marker{byte(r), off + doc.UTF16Len(text.String())})
				continue
			}
			text.WriteRune(r)
		}
		var set doc.MarkSet
		for _, m := range p.marks {
			set = set.Add(m)
		}
		segs = append(segs, doc.Segment{Text: text.String(), Marks: set})
		off += doc.UTF16Len(text.String())
	}
	return doc.NewInline(segs...), marks
}

func (b *Builder) textBlock(t *doc.NodeType, attrs doc.A, parts []Part) *doc.Node {
	in, marks := b.inline(parts)
	n := doc.NewTextBlock(t, attrs, in)
	for _, m := range marks {
		p := &doc.Point{Leaf: n.ID(), Offset: m.off}
		switch m.kind {
		case '|':
			b.anchor, b.head = p, p
		case '[':
			b.anchor = p
		case ']':
			b.head = p
		}
	}
	return n
}

func (b *Builder) Doc(kids ...*doc.Node) *doc.Node { return doc.NewBlock(b.Schema.Doc, nil, kids...) }
func (b *Builder) P(parts ...Part) *doc.Node       { return b.textBlock(b.Paragraph, nil, parts) }
func (b *Builder) H(level int, parts ...Part) *doc.Node {
	return b.textBlock(b.Heading, doc.A{"level": level}, parts)
}
func (b *Builder) Quote(kids ...*doc.Node) *doc.Node { return doc.NewBlock(b.Blockquote, nil, kids...) }
func (b *Builder) UL(items ...*doc.Node) *doc.Node   { return doc.NewBlock(b.BulletList, nil, items...) }
func (b *Builder) LI(kids ...*doc.Node) *doc.Node    { return doc.NewBlock(b.ListItem, nil, kids...) }
func (b *Builder) Pre(lang string, text string) *doc.Node {
	return b.textBlock(b.CodeBlock, doc.A{"language": lang}, []Part{text})
}
func (b *Builder) Img(src string) *doc.Node { return doc.NewAtom(b.Image, doc.A{"src": src}) }
func (b *Builder) HR() *doc.Node            { return doc.NewAtom(b.Schema.HR, nil) }

// State returns a state for d with the selection its markers give, or
// the default caret when it has none. Markers are consumed.
func (b *Builder) State(d *doc.Node) doc.State {
	s := doc.NewState(d)
	if b.anchor != nil || b.head != nil {
		a, h := b.anchor, b.head
		if a == nil {
			a = h
		}
		if h == nil {
			h = a
		}
		s.Sel = doc.TextSelection{Anchor: *a, Head: *h}
	}
	b.anchor, b.head = nil, nil
	return s
}
