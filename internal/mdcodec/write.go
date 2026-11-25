package mdcodec

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Maidang1/corra/internal/doc"
)

// Write writes a document as Markdown, ending with a newline.
func (c *Codec) Write(d *doc.Node) string {
	w := &Writer{c: c, Style: c.Style}
	w.Block(d)
	w.ensureNewline()
	out := strings.TrimRight(w.out.String(), "\n")
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return out + "\n"
}

// Writer writes Markdown line by line. Containers push line prefixes ("> ",
// list indentation) that every line inside them starts with; blocks are
// separated by a blank line, or a single newline in tight lists.
type Writer struct {
	c     *Codec
	Style Style

	out bytes.Buffer
	// encodeNext writes the next character as a character reference if
	// it is a letter or digit, so a closing delimiter before it counts.
	encodeNext bool
	levels     []level
	lineOpen   bool // the current line has content after its prefix
	started    bool // something was written: later blocks need separation
	pending    int  // separation owed before the next block: 1 newline, 2 blank line
	tight      bool
	prev       *doc.Node
	noBlank    bool
	breaks     []bool // line breaks waiting for content after them
	inAtom     bool   // writing an inline atom: its first write flushes waiting breaks
}

type level struct {
	first, rest string
	used        bool
}

// Codec returns the writer's codec.
func (w *Writer) Codec() *Codec { return w.c }

// Block writes n by its type's rule. A type without one writes its text
// blocks' plain text.
func (w *Writer) Block(n *doc.Node) {
	if f, ok := w.c.rules.Writes[n.Type().Name()]; ok && n.Type().IsInline() {
		f(w, n)
		return
	}
	if f, ok := w.c.rules.Writes[n.Type().Name()]; ok {
		if n.Type() != w.c.reg.Root() {
			if w.loose(n.Type().Name()) {
				w.BlankBefore()
			}
			w.separate()
		}
		f(w, n)
		w.endBlock()
		return
	}
	switch {
	case n.Type().IsContainer():
		w.Children(n, w.tight)
	case n.Type().IsTextBlock():
		w.separate()
		w.Inline(n.Inline())
		w.endBlock()
	}
}

func (w *Writer) loose(name string) bool {
	for _, l := range w.c.rules.Loose {
		if l == name {
			return true
		}
	}
	return false
}

// Children writes n's children as blocks; tight separates them with a
// single newline instead of a blank line.
func (w *Writer) Children(n *doc.Node, tight bool) {
	saved, savedPrev := w.tight, w.prev
	w.tight, w.prev = tight, nil
	for i := 0; i < n.NumChildren(); i++ {
		w.Block(n.Child(i))
		w.prev = n.Child(i)
	}
	w.tight, w.prev = saved, savedPrev
	// What came last inside n says nothing about n's own neighbors.
	w.noBlank = false
}

// Prev returns the sibling written before the current block, or nil: two
// lists of one kind in a row need different markers, or they merge.
func (w *Writer) Prev() *doc.Node { return w.prev }

// Section writes fn's content as one block among its siblings, as a list
// writes each of its items.
func (w *Writer) Section(fn func()) {
	w.separate()
	fn()
	w.endBlock()
}

// Tight runs fn with blocks separated by single newlines, or not.
func (w *Writer) Tight(tight bool, fn func()) {
	saved := w.tight
	w.tight = tight
	fn()
	w.tight = saved
}

// Escape escapes s as text in the middle of a line.
func (w *Writer) Escape(s string) string { return w.escape(s, false) }

// BlankBefore asks for a blank line before the current block even in a
// tight context, as a thematic break needs after a paragraph.
func (w *Writer) BlankBefore() {
	// After an empty list item there is no paragraph to underline, and a
	// blank line would end the item around it.
	if w.pending > 0 && !w.noBlank {
		w.pending = 2
	}
}

func (w *Writer) separate() {
	if !w.started {
		return
	}
	switch w.pending {
	case 1:
		w.ensureNewline()
	case 2:
		w.ensureNewline()
		w.out.WriteString(strings.TrimRight(w.prefix(), " "))
		w.out.WriteByte('\n')
	}
	w.pending = 0
}

func (w *Writer) endBlock() {
	if !w.started {
		return
	}
	if w.tight || w.noBlank {
		w.pending = 1
	} else {
		w.pending = 2
	}
}

// NoBlankAfter separates the next block by a single newline even where a
// blank line is due, as after an empty list item, which a blank line
// would end along with the item around it.
func (w *Writer) NoBlankAfter() { w.noBlank = true }

// prefix returns the current line's prefix, using each level's first
// prefix once.
func (w *Writer) prefix() string {
	var b strings.Builder
	for i := range w.levels {
		l := &w.levels[i]
		if !l.used {
			b.WriteString(l.first)
		} else {
			b.WriteString(l.rest)
		}
	}
	return b.String()
}

// WithPrefix writes fn's content with every line prefixed: first on the
// first line, rest after.
func (w *Writer) WithPrefix(first, rest string, fn func()) {
	w.levels = append(w.levels, level{first: first, rest: rest})
	fn()
	if !w.levels[len(w.levels)-1].used {
		// Nothing was written: the prefix alone marks the container, as
		// an empty list item or quote.
		w.write("")
	}
	w.levels = w.levels[:len(w.levels)-1]
}

func (w *Writer) ensureNewline() {
	if w.lineOpen {
		w.out.WriteByte('\n')
		w.lineOpen = false
	}
}

// write writes s, which has no newline, on the current line.
func (w *Writer) write(s string) {
	if s != "" && w.inAtom && len(w.breaks) > 0 {
		w.inAtom = false // flushing writes too
		w.flushBreaks()
		w.inAtom = true
	}
	if !w.lineOpen {
		p := w.prefix()
		if s == "" {
			p = strings.TrimRight(p, " ")
		}
		w.out.WriteString(p)
		for i := range w.levels {
			w.levels[i].used = true
		}
		w.lineOpen = true
	}
	// Next to a delimiter of the same character an escaped "*", "_" or
	// "~" still joins its run; a character reference does not.
	if s != "" && strings.IndexByte(edgeEncode, s[0]) >= 0 {
		if b := w.out.Bytes(); len(b) >= 2 && b[len(b)-2] == '\\' && b[len(b)-1] == s[0] {
			w.out.Truncate(len(b) - 2)
			fmt.Fprintf(&w.out, "&#x%X;", s[0])
		}
	}
	if len(s) >= 2 && s[0] == '\\' && strings.IndexByte(edgeEncode, s[1]) >= 0 && w.lastRune() == rune(s[1]) {
		s = fmt.Sprintf("&#x%X;", s[1]) + s[2:]
	}
	if s != "" {
		w.noBlank = false
	}
	if w.encodeNext && s != "" {
		w.encodeNext = false
		if r, size := utf8.DecodeRuneInString(s); wordChar(r) {
			s = fmt.Sprintf("&#x%X;", r) + s[size:]
		}
	}
	w.out.WriteString(s)
	w.started = true
}

// wordChar reports whether r is neither space nor punctuation, the
// characters next to which an emphasis delimiter does not count when the
// text inside starts or ends with punctuation.
func wordChar(r rune) bool {
	return r != utf8.RuneError && !unicode.IsSpace(r) && !unicode.IsPunct(r) && !unicode.IsSymbol(r)
}

func punct(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }

// lastRune returns the last character written on the current line.
func (w *Writer) lastRune() rune {
	if !w.lineOpen {
		return ' '
	}
	r, _ := utf8.DecodeLastRune(w.out.Bytes())
	return r
}

// ReplaceSuffix replaces old at the end of the current line with new, and
// reports whether it was there.
func (w *Writer) ReplaceSuffix(old, new string) bool {
	b := w.out.Bytes()
	if !w.lineOpen || !bytes.HasSuffix(b, []byte(old)) {
		return false
	}
	w.out.Truncate(len(b) - len(old))
	w.out.WriteString(new)
	return true
}

// LastRune returns the last character written on the current line, or a
// space at its start.
func (w *Writer) LastRune() rune { return w.lastRune() }

// EncodeLast rewrites the last character written as a character
// reference, as a heading's final "#" that would read as closing syntax.
func (w *Writer) EncodeLast() { w.encodeLast() }

// encodeLast rewrites the last character written as a character
// reference.
func (w *Writer) encodeLast() {
	b := w.out.Bytes()
	r, size := utf8.DecodeLastRune(b)
	w.out.Truncate(len(b) - size)
	fmt.Fprintf(&w.out, "&#x%X;", r)
}

// Raw writes s verbatim; its newlines start new prefixed lines.
func (w *Writer) Raw(s string) {
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			w.ensureNewline()
		}
		w.write(line)
	}
}

// Newline ends the current line.
func (w *Writer) Newline() {
	if !w.lineOpen {
		w.write("")
	}
	w.ensureNewline()
}

// InlineString writes content alone, on no line, and returns it: for
// syntax that puts inline content inside its own, as a table's cells.
func (w *Writer) InlineString(in doc.Inline) string {
	sub := &Writer{c: w.c, Style: w.Style}
	sub.Inline(in)
	return strings.TrimRight(sub.out.String(), "\n")
}

// Started reports whether anything has been written: a document's first
// line has meanings others do not, as "---" opening front matter.
func (w *Writer) Started() bool { return w.started }

// LinePrefix returns the prefix the next line will start with, as "- "
// on a list item's first line.
func (w *Writer) LinePrefix() string {
	if w.lineOpen {
		return ""
	}
	return w.prefix()
}

// AtLineStart reports whether the next text starts a line.
func (w *Writer) AtLineStart() bool { return !w.lineOpen }

// Text writes s escaped for Markdown text.
func (w *Writer) Text(s string) {
	// Nothing to write (as spaces dropped at a line's start) must not
	// start the line: an empty write ends it with its prefix trimmed.
	if e := w.escape(s, !w.lineOpen); e != "" {
		w.write(e)
	}
}

// Inline writes text block content with its marks.
func (w *Writer) Inline(in doc.Inline) {
	var open []doc.Mark
	var closers []string // the delimiter each open mark closes with
	closeTo := func(n int) {
		before := w.lastRune()
		delimited := false
		for len(open) > n {
			m := open[len(open)-1]
			if syn, ok := w.c.rules.Marks[m.Type.Name()]; ok && syn.Close != nil {
				w.write(closers[len(closers)-1])
				delimited = delimited || syn.Delimited
			}
			open = open[:len(open)-1]
			closers = closers[:len(closers)-1]
		}
		// "x.**a" does not close: the delimiter needs punctuation or space
		// after it when punctuation is before it.
		if delimited && punct(before) {
			w.encodeNext = true
		}
	}
	count := in.NumSegments()
	for i := 0; i < count; i++ {
		seg := in.Segment(i)
		if seg.Atom == nil && seg.Text != "" && len(w.breaks) > 0 {
			w.flushBreaks()
		}
		var target, perSeg []doc.Mark
		var verbatim *doc.Mark
		for _, m := range seg.Marks {
			if syn, ok := w.c.rules.Marks[m.Type.Name()]; ok && syn.Verbatim {
				mm := m
				verbatim = &mm
				continue
			} else if ok && syn.PerSegment {
				perSeg = append(perSeg, m)
				continue
			}
			target = append(target, m)
		}
		common := 0
		for common < len(open) && common < len(target) && open[common].Equal(target[common]) {
			common++
		}
		text := seg.Text
		lead, trail := "", ""
		// keep is how many of target's marks continue into the next
		// segment; the rest close after this one.
		keep := 0
		if i+1 < count {
			next := nonVerbatim(w, in.Segment(i+1).Marks)
			for keep < len(target) && keep < len(next) && target[keep].Equal(next[keep]) {
				keep++
			}
		}
		if seg.Atom == nil && verbatim == nil && len(perSeg) == 0 {
			// Delimiters next to spaces do not count in Markdown: move the
			// spaces outside the marks that open or close here.
			if common < len(target) {
				t := strings.TrimLeftFunc(text, unicode.IsSpace)
				lead, text = text[:len(text)-len(t)], t
			}
			if keep < len(target) {
				t := strings.TrimRightFunc(text, unicode.IsSpace)
				trail, text = text[len(t):], t
			}
		}
		closeTo(common)
		if lead != "" {
			w.Text(lead)
		}
		opening := false
		for _, m := range target[common:] {
			if syn, ok := w.c.rules.Marks[m.Type.Name()]; ok && syn.Delimited {
				opening = true
			}
		}
		// "a**[" does not open: after a letter the delimiter needs a letter
		// or digit after it. Anything but plain text starts with
		// punctuation.
		if opening && wordChar(w.lastRune()) {
			first := '!'
			if seg.Atom == nil && verbatim == nil {
				if e := w.escape(text, false); e != "" {
					first, _ = utf8.DecodeRuneInString(e)
				}
			}
			if punct(first) {
				w.encodeLast()
			}
		}
		for _, m := range target[common:] {
			closer := ""
			if syn, ok := w.c.rules.Marks[m.Type.Name()]; ok && syn.Open != nil {
				opener := syn.Open(m, w.Style)
				closer = syn.Close(m, w.Style)
				// A delimiter right after one of the same character joins
				// its run ("**" then "*"): switch to the other spelling.
				if syn.Alt != nil && opener != "" && w.lastRune() == rune(opener[0]) {
					opener, closer = syn.Alt(m, w.Style)
				}
				w.write(opener)
			}
			open = append(open, m)
			closers = append(closers, closer)
		}
		switch {
		case seg.Atom != nil:
			// Breaks before an atom are written when it writes something.
			w.inAtom = true
			w.Block(seg.Atom)
			w.inAtom = false
		case verbatim != nil:
			syn := w.c.rules.Marks[verbatim.Type.Name()]
			w.write(w.wrapPerSegment(perSeg, syn.Wrap(*verbatim, text), w.lastRune(), '!'))
		case len(perSeg) > 0:
			next := rune(0) // marks opening or an atom: unknown
			if i+1 < count {
				if ns := in.Segment(i + 1); ns.Atom == nil && sameMarkSet(nonVerbatim(w, ns.Marks), target) {
					next, _ = utf8.DecodeRuneInString(w.escape(ns.Text, false) + " ")
				}
			} else {
				next = ' '
			}
			w.write(w.wrapPerSegment(perSeg, w.escape(text, !w.lineOpen), w.lastRune(), next))
		default:
			w.inlineText(text)
		}
		if trail != "" {
			closeTo(keep)
			w.Text(trail)
		}
	}
	closeTo(0)
	w.breaks = nil
}

// wrapPerSegment wraps written text in per-segment marks, innermost
// (highest rank) first.
func (w *Writer) wrapPerSegment(marks []doc.Mark, body string, prev, next rune) string {
	for i := len(marks) - 1; i >= 0; i-- {
		body = w.c.rules.Marks[marks[i].Type.Name()].WrapNear(marks[i], body, prev, next)
	}
	return body
}

func sameMarkSet(a, b []doc.Mark) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// Break ends a line inside a paragraph: a hard break ("\" at the line's
// end) or a soft one (a plain newline). It is written only once more
// content follows, so a break at a paragraph's end, which Markdown cannot
// hold, is dropped, and one that would leave a line empty is made hard.
func (w *Writer) Break(soft bool) { w.breaks = append(w.breaks, soft) }

func (w *Writer) flushBreaks() {
	bs := w.breaks
	w.breaks = nil
	for _, soft := range bs {
		if !soft || !w.lineOpen {
			// An escaped backslash before the break's would read as
			// escaping it.
			w.ReplaceSuffix(`\\`, "&#x5C;")
			w.write(`\`)
		}
		w.ensureNewline()
	}
}

// EncodeNext writes the next character as a character reference if it is
// a letter or digit, as after "$x$", which a digit would keep from closing.
func (w *Writer) EncodeNext() { w.encodeNext = true }

func nonVerbatim(w *Writer, s doc.MarkSet) []doc.Mark {
	var out []doc.Mark
	for _, m := range s {
		if syn, ok := w.c.rules.Marks[m.Type.Name()]; ok && (syn.Verbatim || syn.PerSegment) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// inlineText writes text that may hold newlines, as hard line breaks.
func (w *Writer) inlineText(s string) {
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			w.write("\\")
			w.ensureNewline()
		}
		if line != "" {
			w.Text(line)
		}
	}
}
