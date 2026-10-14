package views

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/render"
)

// sidebar renders the outline (headings) and recent files.
func (v *View) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Width(220).FillHeight().Background(t.Background).Padding(10, 6).
		Children(func() {
			ui.Column(c).Fill().Gap(2).Children(func() {
				ui.Text(c, "大纲").FontSize(11).TextColor(t.TextMuted).Padding(4, 6)
				headingNo := 0
				sel := v.App.OutlineSel
				for i := range v.App.Doc.Blocks {
					b := &v.App.Doc.Blocks[i]
					if b.Kind != render.KindHeading {
						continue
					}
					n := headingNo
					row := ui.Button(c, b.HeadingText()).
						FontSize(12.5).Padding(3, 6+float32(b.Level-1)*10).
						FillWidth()
					if b.Level <= 2 {
						row = row.Bold()
					}
					if sel == n {
						row = row.Background(t.SurfacePressed)
					}
					row.OnClick(func() {
						v.App.OutlineSel = n
						v.jumpToHeading(n)
					})
					headingNo++
				}
				if headingNo == 0 {
					ui.Text(c, "（无标题）").FontSize(11.5).TextColor(t.TextMuted).Padding(4, 6)
				}

				if len(v.App.Recent) > 0 {
					ui.Divider(c).FillWidth().PaddingY(6)
					ui.Text(c, "最近打开").FontSize(11).TextColor(t.TextMuted).Padding(4, 6)
					for _, p := range v.App.Recent {
						path := p
						ui.Button(c, ellipsize(filepath.Base(path), 22)).
							FontSize(12).Padding(3, 6).FillWidth().
							Tooltip(path).
							OnClick(func() {
								if v.OnOpenRecent != nil {
									v.OnOpenRecent(path)
								}
							})
					}
				}
			})
		})
}

func ellipsize(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

var _ = strings.TrimSpace
