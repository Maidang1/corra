package editor

import (
	"regexp"

	"github.com/Maidang1/corra/internal/doc"
)

// PrefixRule is an input rule for block syntax typed at the start of a
// block, as "# " or "- ": when the block's text up to the caret matches re
// (anchored at both ends), the typed syntax is removed and cmd, built from
// the match, applies to the block.
func PrefixRule(re *regexp.Regexp, cmd func(m []string) doc.Command) InputRule {
	return InputRule{
		Match: re.FindStringSubmatch,
		Handle: func(s doc.State, tx *doc.Tx, m []string) bool {
			sel, ok := s.TextSel()
			if !ok || !sel.Empty() {
				return false
			}
			leaf := s.Node(sel.Head.Leaf)
			if leaf.Type().Content() != doc.ContentInline {
				return false
			}
			n := doc.UTF16Len(m[0])
			tx.ReplaceText(leaf.ID(), sel.Head.Offset-n, sel.Head.Offset, doc.Inline{})
			tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID(), Offset: sel.Head.Offset - n}))
			return cmd(m)(tx.State(), tx)
		},
	}
}

// MarkRule is an input rule for inline syntax closed at the caret, as
// "**bold**". re is anchored at the end: group 1 is the character before
// the opening delimiter, kept (it may be empty at the block's start), and
// group 2 the delimited text. The delimiters are removed, the text takes
// the mark, and typing goes on without it.
func MarkRule(re *regexp.Regexp, mark func() doc.Mark) InputRule {
	return InputRule{
		Match: re.FindStringSubmatch,
		Handle: func(s doc.State, tx *doc.Tx, m []string) bool {
			sel, ok := s.TextSel()
			if !ok || !sel.Empty() || len(m) < 3 || m[2] == "" {
				return false
			}
			leaf := s.Node(sel.Head.Leaf)
			mk := mark()
			if !leaf.Type().AllowsMark(mk.Type) {
				return false
			}
			end := sel.Head.Offset
			start := end - doc.UTF16Len(m[0]) + doc.UTF16Len(m[1])
			inner := doc.UTF16Len(m[2])
			open := (end - start - inner) / 2
			content := leaf.Inline().Slice(start+open, start+open+inner).AddMark(mk)
			tx.ReplaceText(leaf.ID(), start, end, content)
			tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID(), Offset: start + content.Len()}))
			tx.SetStoredMarks(leaf.Inline().MarksAt(end).Remove(mk.Type))
			return true
		},
	}
}
