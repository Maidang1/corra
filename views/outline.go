package views

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
)

func (v *View) currentOutlineDocument() *render.Document {
	if v.outlineDoc == nil || v.outlineSource != v.App.Source {
		v.outlineSource = v.App.Source
		v.outlineDoc = render.ParseOptions(v.App.Source, v.App.SmartPunct)
	}
	return v.outlineDoc
}

func (v *View) outlineEntries(c *ui.Context) {
	t := c.Theme()
	doc := v.currentOutlineDocument()
	ui.Scroll(c).Fill().ScrollbarInsets(6, 4, 12, 6).Children(func() {
		ui.Column(c).FillWidth().Gap(1).Padding(4, 6).Children(func() {
			headingIdx := -1
			for i := range doc.Blocks {
				b := &doc.Blocks[i]
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
			if headingIdx < 0 {
				ui.Text(c, "本文暂无标题").FontSize(13).TextColor(t.TextMuted).Padding(12, 8)
			}
		})
	})
}

// jumpToHeading reveals the blockIdx-th block (the headingIdx-th heading) in
// whichever editor surface is active.
func (v *View) jumpToHeading(blockIdx, headingIdx int) {
	doc := v.currentOutlineDocument()
	if blockIdx < 0 || blockIdx >= len(doc.Blocks) {
		return
	}
	v.App.OutlineSel = headingIdx
	switch {
	case v.App.Mode == app.ModeEdit && v.edState != nil:
		b := &doc.Blocks[blockIdx]
		start := len(strings.ReplaceAll(v.App.Source[:b.Start], "\r\n", "\n"))
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
