package editor

import "github.com/egoist/mygo/ui"

// FormatBarForTest exposes the format bar's box and labels to tests.
func (e *Editor) FormatBarForTest() (ui.Rect, []string, bool) {
	r, bs, ok := e.formatBar()
	var labels []string
	for _, b := range bs {
		labels = append(labels, b.label)
	}
	return r, labels, ok
}

// MisspelledForTest returns the misspelled ranges of top-level block i.
func (e *Editor) MisspelledForTest(i int) []ui.TextInputRange {
	return e.misspelled(e.state.Doc.Child(i))
}
