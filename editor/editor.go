// Package editor holds the document editing core: undo, formatting
// actions and search. It operates on plain strings and rune offsets so it
// stays fully unit-testable.
package editor

// Edit is one replace operation on the document source.
type Edit struct {
	Start, End int // rune offsets of the replaced range
	Text       string
	// Caret is the rune offset the caret should land on after the edit.
	Caret int
}

// Apply applies an edit to src.
func Apply(src string, e Edit) string {
	rs := []rune(src)
	if e.Start < 0 {
		e.Start = 0
	}
	if e.End > len(rs) {
		e.End = len(rs)
	}
	out := make([]rune, 0, len(rs)-(e.End-e.Start)+len([]rune(e.Text)))
	out = append(out, rs[:e.Start]...)
	out = append(out, []rune(e.Text)...)
	out = append(out, rs[e.End:]...)
	return string(out)
}
