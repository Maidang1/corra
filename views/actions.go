package views

import (
	"strings"

	"github.com/Maidang1/corra/editor"
	"github.com/Maidang1/corra/render"
)

// The toolbar actions operate on the whole document at the end of the
// source (v1: no selection tracking across blocks), which keeps them
// predictable in single-pane mode.

func (v *View) applyWrap(marker string) {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.ToggleWrap(src, marker, caret, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) applyHeading() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.ToggleHeading(src, caret, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) applyList(ordered bool) {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.ToggleListMarker(src, ordered, caret, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) applyQuote() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.ToggleQuote(src, caret, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) applyLink() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.InsertLink(src, "", caret, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) appendTable() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.InsertTable(src, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) appendCode() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.InsertCodeBlock(src, "", caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) appendTask() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.InsertTask(src, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

func (v *View) appendMath() {
	src := v.App.Source
	caret := len([]rune(src))
	e := editor.InsertMath(src, caret)
	v.App.SetSource(editor.Apply(src, e), e.Caret)
}

// WrapAction applies a wrap marker (exported for menus).
func (v *View) WrapAction(marker string) { v.applyWrap(marker) }

// HeadingAction toggles a heading (exported for menus).
func (v *View) HeadingAction() { v.applyHeading() }

// ListAction toggles a list (exported for menus).
func (v *View) ListAction(ordered bool) { v.applyList(ordered) }

// QuoteAction toggles a quote (exported for menus).
func (v *View) QuoteAction() { v.applyQuote() }

// ToggleThemeAction cycles the theme (exported for menus).
func (v *View) ToggleThemeAction() { v.toggleTheme() }

// insertAtEnd appends a block built by fn at the end of the document.
func (v *View) insertAtEnd(fn func()) { fn() }

// findMatches returns matches of the find query in the document source.
func (v *View) findMatches() []editor.Match {
	return editor.Find(v.App.Source, v.App.FindQuery)
}

// jumpToHeading scrolls the document to the n-th heading block.
func (v *View) jumpToHeading(n int) {
	count := -1
	for i := range v.App.Doc.Blocks {
		if v.App.Doc.Blocks[i].Kind == render.KindHeading {
			count++
			if count == n {
				v.jumpTo = i
				v.App.InvalidateUI()
				return
			}
		}
	}
}

// findNext moves the find cursor to the next match.
func (v *View) findNext() {
	m := v.findMatches()
	if len(m) == 0 {
		return
	}
	v.App.FindCurrent = (v.App.FindCurrent + 1) % len(m)
	v.App.InvalidateUI()
}

// findPrev moves the find cursor to the previous match.
func (v *View) findPrev() {
	m := v.findMatches()
	if len(m) == 0 {
		return
	}
	v.App.FindCurrent = (v.App.FindCurrent - 1 + len(m)) % len(m)
	v.App.InvalidateUI()
}

var _ = strings.TrimSpace
