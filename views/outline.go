package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
)

// outlinePanel is Typora's outline panel, shaped like a macOS 26 inspector:
// a floating inset panel that mirrors the sidebar, listing the document's
// headings; clicking one jumps to it in the active mode.
func (v *View) outlinePanel(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Key("outline").Width(236).FillHeight().Padding(0, 8, 8, 0).
		Transition(panelTransition).Children(func() {
		ui.Column(c).Fill().Gap(0).Radius(16).Clip().Background(sidebarBackground(t)).
			Border(1, cardBorder(t)).Shadow(0, 6, 18, 0, cardShadow(t)).Children(func() {
			ui.Text(c, "大纲").FontSize(12).Bold().TextColor(t.TextMuted).Padding(12, 14, 6, 14)
			v.outlineEntries(c)
		})
	})
}

func (v *View) outlineEntries(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().ScrollbarInsets(6, 4, 12, 6).Children(func() {
		ui.Column(c).FillWidth().Gap(1).Padding(4, 6).Children(func() {
			headingIdx := -1
			for i := range v.App.Doc.Blocks {
				b := &v.App.Doc.Blocks[i]
				if b.Kind != render.KindHeading {
					continue
				}
				headingIdx++
				idx, hIdx := i, headingIdx
				row := ui.ButtonBase(c).Key(idx).Label("大纲："+b.HeadingText()).FillWidth().Padding(5, 8, 5, float32(8+(b.Level-1)*12)).Radius(8)
				if row.Pressed() {
					row = row.Background(sidebarPressed(t))
				} else if row.Hovered() {
					row = row.Background(sidebarHover(t))
				}
				row.Children(func() {
					el := ui.Text(c, b.HeadingText()).FontSize(12.5).SingleLine().Ellipsis("…").FillWidth().MinWidth(0)
					if b.Level <= 2 {
						el.FontWeight(600)
					} else {
						el.TextColor(t.TextMuted)
					}
				}).OnClick(func() { v.jumpToHeading(idx, hIdx) })
			}
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
