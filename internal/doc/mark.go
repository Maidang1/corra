package doc

import "strings"

// MarkSpec declares a mark: formatting over a range of inline content.
type MarkSpec struct {
	Name string
	// Rank orders nesting when Markdown is written: lower ranks open
	// first, outside higher ones (link outside bold outside code).
	Rank int
	// Inclusive marks continue when typing at their end (bold does,
	// link and code do not).
	Inclusive bool
	// Excludes names the marks this one cannot coexist with; "_" means
	// all of them, as code excludes every other mark.
	Excludes []string
	Attrs    map[string]AttrSpec
}

// MarkType is a registered mark.
type MarkType struct {
	spec    MarkSpec
	rank    int
	excl    map[*MarkType]bool
	exclAll bool
}

func (t *MarkType) Name() string    { return t.spec.Name }
func (t *MarkType) Spec() MarkSpec  { return t.spec }
func (t *MarkType) Inclusive() bool { return t.spec.Inclusive }
func (t *MarkType) Rank() int       { return t.rank }

// Excludes reports whether t cannot coexist with o. A mark always
// excludes another of its own type: the newer replaces the older.
func (t *MarkType) Excludes(o *MarkType) bool {
	return t == o || t.exclAll || t.excl[o]
}

// Mark is a mark type with attributes.
type Mark struct {
	Type  *MarkType
	Attrs Attrs
}

// NewMark resolves attrs against t's spec; it panics on invalid attrs.
func NewMark(t *MarkType, attrs A) Mark {
	a, err := resolveAttrs(t.spec.Attrs, NewAttrs(attrs), "mark "+t.spec.Name)
	if err != nil {
		panic(err)
	}
	return Mark{Type: t, Attrs: a}
}

func (m Mark) Equal(o Mark) bool { return m.Type == o.Type && m.Attrs.Equal(o.Attrs) }

func (m Mark) String() string {
	if m.Type == nil {
		return "<nil>"
	}
	if m.Attrs.Len() == 0 {
		return m.Type.Name()
	}
	var b strings.Builder
	b.WriteString(m.Type.Name())
	b.WriteByte('{')
	for i, k := range m.Attrs.Keys() {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(formatValue(m.Attrs.Get(k)))
	}
	b.WriteByte('}')
	return b.String()
}

// MarkSet is a set of marks sorted by rank. It is immutable: methods
// return new sets and never modify the receiver.
type MarkSet []Mark

// Add returns s with m. A mark of the same type is replaced; marks m
// excludes are dropped; if a mark in s excludes m, s is returned as is.
func (s MarkSet) Add(m Mark) MarkSet {
	out := make(MarkSet, 0, len(s)+1)
	placed := false
	for _, o := range s {
		if m.Equal(o) {
			return s
		}
		if m.Type.Excludes(o.Type) {
			continue
		}
		if o.Type.Excludes(m.Type) {
			return s
		}
		if !placed && o.Type.rank > m.Type.rank {
			out = append(out, m)
			placed = true
		}
		out = append(out, o)
	}
	if !placed {
		out = append(out, m)
	}
	return out
}

// Remove returns s without marks of type t.
func (s MarkSet) Remove(t *MarkType) MarkSet {
	if !s.Has(t) {
		return s
	}
	out := make(MarkSet, 0, len(s)-1)
	for _, o := range s {
		if o.Type != t {
			out = append(out, o)
		}
	}
	return out
}

func (s MarkSet) Has(t *MarkType) bool {
	_, ok := s.Get(t)
	return ok
}

func (s MarkSet) Get(t *MarkType) (Mark, bool) {
	for _, o := range s {
		if o.Type == t {
			return o, true
		}
	}
	return Mark{}, false
}

// Toggle removes t if present, otherwise adds it without attributes.
func (s MarkSet) Toggle(t *MarkType) MarkSet {
	if s.Has(t) {
		return s.Remove(t)
	}
	return s.Add(Mark{Type: t})
}

func (s MarkSet) Equal(o MarkSet) bool {
	if len(s) != len(o) {
		return false
	}
	for i := range s {
		if !s[i].Equal(o[i]) {
			return false
		}
	}
	return true
}
