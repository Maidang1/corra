package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
)

// outlinePanel is Typora's outline panel: the document's headings in a
// fourth column; clicking one jumps to it in the active mode.
func (v *View) outlinePanel(c *ui.Context) {
	t := c.Theme()
	// Transparent over the card's surface: a panel background would paint
	// the card's right corners square. The scroll's thumb is inset from
	// the bottom rim for the same reason.
	ui.Column(c).Width(220).FillHeight().Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12, 6, 12).Children(func() {
			ui.Text(c, "大纲").FontSize(12).Bold()
		})
		ui.Divider(c).FillWidth()
		ui.Scroll(c).Fill().ScrollbarInsets(6, 4, 26, 6).Children(func() {
			ui.Column(c).FillWidth().Gap(1).Padding(4, 6).Children(func() {
				headingIdx := -1
				for i := range v.App.Doc.Blocks {
					b := &v.App.Doc.Blocks[i]
					if b.Kind != render.KindHeading {
						continue
					}
					headingIdx++
					idx, hIdx := i, headingIdx
					el := ui.Text(c, b.HeadingText()).
						FontSize(12.5).Padding(3, 4, 3, float32(8+(b.Level-1)*12)).
						FillWidth().Cursor(ui.CursorPointer)
					if b.Level <= 2 {
						el = el.FontWeight(600)
					} else {
						el = el.TextColor(t.TextMuted)
					}
					el.OnClick(func() { v.jumpToHeading(idx, hIdx) })
				}
			})
		})
	})
}

// jumpToHeading reveals the blockIdx-th block (the headingIdx-th heading) in
// whichever editor surface is active.
func (v *View) jumpToHeading(blockIdx, headingIdx int) {
	switch {
	case v.App.Mode == app.ModeEdit && v.edState != nil:
		b := &v.App.Doc.Blocks[blockIdx]
		start := b.Start
		if start > v.edState.Buffer.Len() {
			start = v.edState.Buffer.Len()
		}
		v.edState.Select(start, start)
		v.focusEditor = true
	case v.rich != nil:
		v.rich.RevealHeading(headingIdx)
	}
	v.App.InvalidateUI()
}
