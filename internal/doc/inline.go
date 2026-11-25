package doc

import (
	"strings"
	"unicode/utf8"
)

// ObjectChar stands for an inline atom in plain text and in the linear
// text input methods see.
const ObjectChar = '￼'

// Segment is a run of text with one set of marks, or one inline atom.
type Segment struct {
	Text  string  // empty for an atom
	Marks MarkSet //
	Atom  *Node   // an inline atom node, or nil
}

func (s Segment) len() int {
	if s.Atom != nil {
		return 1
	}
	return UTF16Len(s.Text)
}

// Inline is the content of a text block. It is immutable: methods return
// new values. Adjacent text segments with equal marks are always merged
// and empty text segments dropped, so equal content has one form.
type Inline struct {
	segs []Segment
	n    int
}

// NewInline builds content from segments, normalizing them.
func NewInline(segs ...Segment) Inline {
	var in Inline
	for _, s := range segs {
		in.segs, in.n = appendSegment(in.segs, in.n, s)
	}
	return in
}

// Text returns plain text content with marks.
func Text(s string, marks ...Mark) Inline {
	var set MarkSet
	for _, m := range marks {
		set = set.Add(m)
	}
	return NewInline(Segment{Text: s, Marks: set})
}

// AtomInline returns content holding one inline atom.
func AtomInline(atom *Node, marks ...Mark) Inline {
	var set MarkSet
	for _, m := range marks {
		set = set.Add(m)
	}
	return NewInline(Segment{Atom: atom, Marks: set})
}

func appendSegment(segs []Segment, n int, s Segment) ([]Segment, int) {
	if s.Atom == nil && s.Text == "" {
		return segs, n
	}
	if s.Atom == nil && len(segs) > 0 {
		last := &segs[len(segs)-1]
		if last.Atom == nil && last.Marks.Equal(s.Marks) {
			// segs may share a backing array with another Inline; copy
			// before changing an element.
			segs = append(segs[:len(segs)-1:len(segs)-1], Segment{Text: last.Text + s.Text, Marks: last.Marks})
			return segs, n + UTF16Len(s.Text)
		}
	}
	return append(segs[:len(segs):len(segs)], s), n + s.len()
}

// Len is the length in UTF-16 units; an atom counts as one.
func (in Inline) Len() int { return in.n }

func (in Inline) Empty() bool { return in.n == 0 }

// NumSegments and Segment iterate the normalized segments.
func (in Inline) NumSegments() int      { return len(in.segs) }
func (in Inline) Segment(i int) Segment { return in.segs[i] }
func (in Inline) Segments() []Segment   { return append([]Segment(nil), in.segs...) }
func (in Inline) segments() []Segment   { return in.segs }
func (in Inline) hasAtomsOrMarks() bool { return hasAtomsOrMarks(in.segs) }

func hasAtomsOrMarks(segs []Segment) bool {
	for _, s := range segs {
		if s.Atom != nil || len(s.Marks) > 0 {
			return true
		}
	}
	return false
}

// String returns the plain text, an atom as U+FFFC.
func (in Inline) String() string {
	var b strings.Builder
	for _, s := range in.segs {
		if s.Atom != nil {
			b.WriteRune(ObjectChar)
		} else {
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

// Slice returns the content between UTF-16 offsets from and to, clamped.
// An offset inside a surrogate pair rounds down.
func (in Inline) Slice(from, to int) Inline {
	from, to = max(0, from), min(in.n, to)
	if from >= to {
		return Inline{}
	}
	var out Inline
	pos := 0
	for _, s := range in.segs {
		l := s.len()
		a, z := max(from, pos), min(to, pos+l)
		if a < z {
			if s.Atom != nil {
				out.segs, out.n = appendSegment(out.segs, out.n, s)
			} else {
				t := s.Text[byteOffset(s.Text, a-pos):byteOffset(s.Text, z-pos)]
				out.segs, out.n = appendSegment(out.segs, out.n, Segment{Text: t, Marks: s.Marks})
			}
		}
		pos += l
		if pos >= to {
			break
		}
	}
	return out
}

// Snap moves off off the middle of a surrogate pair: down, or up when up
// is set. Offsets outside the content clamp to it.
func (in Inline) Snap(off int, up bool) int {
	off = max(0, min(off, in.n))
	pos := 0
	for _, s := range in.segs {
		l := s.len()
		if off < pos+l {
			if s.Atom != nil {
				return off
			}
			b := byteOffset(s.Text, off-pos)
			down := pos + UTF16Len(s.Text[:b])
			if down == off || !up {
				return down
			}
			_, size := utf8.DecodeRuneInString(s.Text[b:])
			return pos + UTF16Len(s.Text[:b+size])
		}
		pos += l
	}
	return off
}

// Concat returns in followed by o.
func (in Inline) Concat(o Inline) Inline {
	out := Inline{segs: in.segs[:len(in.segs):len(in.segs)], n: in.n}
	for _, s := range o.segs {
		out.segs, out.n = appendSegment(out.segs, out.n, s)
	}
	return out
}

// Replace returns in with [from, to) replaced by ins.
func (in Inline) Replace(from, to int, ins Inline) Inline {
	return in.Slice(0, from).Concat(ins).Concat(in.Slice(to, in.n))
}

// Map returns the content with every segment's marks passed through f.
func (in Inline) mapMarks(f func(MarkSet) MarkSet) Inline {
	var out Inline
	for _, s := range in.segs {
		s.Marks = f(s.Marks)
		out.segs, out.n = appendSegment(out.segs, out.n, s)
	}
	return out
}

// AddMark adds m to all of the content.
func (in Inline) AddMark(m Mark) Inline {
	return in.mapMarks(func(s MarkSet) MarkSet { return s.Add(m) })
}

// RemoveMark removes marks of type t from all of the content.
func (in Inline) RemoveMark(t *MarkType) Inline {
	return in.mapMarks(func(s MarkSet) MarkSet { return s.Remove(t) })
}

// StripMarks removes all marks and turns atoms into nothing, as content
// becoming plain text does.
func (in Inline) PlainText() Inline {
	var b strings.Builder
	for _, s := range in.segs {
		if s.Atom == nil {
			b.WriteString(s.Text)
		}
	}
	return Text(b.String())
}

// MarksAt returns the marks text typed at offset off takes: those of the
// character before it, keeping only inclusive marks at a segment's end,
// or of the character after it at offset 0.
func (in Inline) MarksAt(off int) MarkSet {
	if len(in.segs) == 0 {
		return nil
	}
	pos := 0
	for i, s := range in.segs {
		l := s.len()
		if off == 0 && i == 0 {
			return inclusiveOnly(s.Marks)
		}
		if off > pos && off < pos+l {
			return s.Marks
		}
		if off == pos+l {
			if i+1 < len(in.segs) {
				// Between two segments: the marks both share, plus the
				// inclusive marks of the one before.
				next := in.segs[i+1].Marks
				out := MarkSet(nil)
				for _, m := range s.Marks {
					if m.Type.Inclusive() || next.Has(m.Type) {
						out = out.Add(m)
					}
				}
				return out
			}
			return inclusiveOnly(s.Marks)
		}
		pos += l
	}
	return nil
}

// inclusiveOnly keeps the marks typing continues at an edge of the text,
// where no neighbor shares the others.
func inclusiveOnly(s MarkSet) MarkSet {
	var out MarkSet
	for _, m := range s {
		if m.Type.Inclusive() {
			out = out.Add(m)
		}
	}
	return out
}

// HasMark reports whether every character in [from, to) carries a mark
// of type t; an empty range reports false.
func (in Inline) HasMark(t *MarkType, from, to int) bool {
	part := in.Slice(from, to)
	if part.n == 0 {
		return false
	}
	for _, s := range part.segs {
		if !s.Marks.Has(t) {
			return false
		}
	}
	return true
}

// Equal compares content, including inline atoms (by Equal).
func (in Inline) Equal(o Inline) bool {
	if in.n != o.n || len(in.segs) != len(o.segs) {
		return false
	}
	for i, s := range in.segs {
		t := o.segs[i]
		if s.Text != t.Text || !s.Marks.Equal(t.Marks) || (s.Atom == nil) != (t.Atom == nil) {
			return false
		}
		if s.Atom != nil && !Equal(s.Atom, t.Atom) {
			return false
		}
	}
	return true
}
