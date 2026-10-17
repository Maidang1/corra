package views

import (
	"github.com/egoist/mygo/ui"
)

// Hand-drawn 24×24 stroke icons, the way Apple Notes' toolbar reads.
// They paint with currentColor so the element's TextColor tints them.

const iconStroke = `xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"`

var (
	iconCompose   = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M11 5H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-6"/><path d="M16.4 3.7a2.1 2.1 0 0 1 3 3L8.5 17.6 4.5 19l1.4-4Z"/></svg>`))
	iconChecklist = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M10.5 6.5h10M10.5 12h10M10.5 17.5h10"/><path d="M3.5 6.5l1.3 1.3 2.4-2.4"/><path d="M3.5 12l1.3 1.3 2.4-2.4"/><path d="M4 17.5h3.4"/></svg>`))
	iconTable     = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><rect x="3.5" y="5" width="17" height="14" rx="1.5"/><path d="M3.5 9.7h17M3.5 14.3h17M9.8 5v14"/></svg>`))
	iconPaperclip = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M20.5 11.2l-7 7a5 5 0 0 1-7.1-7.1l7.8-7.8a3.4 3.4 0 0 1 4.8 4.8l-7.5 7.5a1.9 1.9 0 0 1-2.7-2.7l6.6-6.6"/></svg>`))
	iconShare     = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M12 15V4"/><path d="M7.5 8.2 12 3.7l4.5 4.5"/><path d="M5 12.8v6.2A1.5 1.5 0 0 0 6.5 20.5h11a1.5 1.5 0 0 0 1.5-1.5v-6.2"/></svg>`))
	iconEllipsis  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="5" cy="12" r="1.7" fill="currentColor"/><circle cx="12" cy="12" r="1.7" fill="currentColor"/><circle cx="19" cy="12" r="1.7" fill="currentColor"/></svg>`))
	iconSearch    = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><circle cx="11" cy="11" r="6.3"/><path d="M20 20l-3.9-3.9"/></svg>`))
	iconSidebar   = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M9.6 4.5v15"/></svg>`))

	iconPlusCircle = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><circle cx="12" cy="12" r="8.5"/><path d="M12 8v8M8 12h8"/></svg>`))
	iconFolder     = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M3.5 6.6A1.6 1.6 0 0 1 5.1 5h3.9l2 2.4h7.4a1.6 1.6 0 0 1 1.6 1.6v7.4a1.6 1.6 0 0 1-1.6 1.6H5.1a1.6 1.6 0 0 1-1.6-1.6Z"/></svg>`))
	iconTrash      = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M4.5 6.6h15M9.5 6V4.5A1.4 1.4 0 0 1 10.9 3h2.2a1.4 1.4 0 0 1 1.4 1.5V6"/><path d="M6.7 6.6l.8 11.4a1.5 1.5 0 0 0 1.5 1.4h6a1.5 1.5 0 0 0 1.5-1.4l.8-11.4"/><path d="M10.2 10.7v5.8M13.8 10.7v5.8"/></svg>`))
	iconMoon       = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="M20 14.2A8.2 8.2 0 0 1 9.8 4a8.2 8.2 0 1 0 10.2 10.2Z"/></svg>`))
	iconSun        = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><circle cx="12" cy="12" r="4"/><path d="M12 2.8v2M12 19.2v2M2.8 12h2M19.2 12h2M5.2 5.2l1.4 1.4M17.4 17.4l1.4 1.4M18.8 5.2l-1.4 1.4M6.6 17.4l-1.4 1.4"/></svg>`))
	iconCode       = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><path d="m8.5 7-5 5 5 5M15.5 7l5 5-5 5"/></svg>`))
	iconDoc        = ui.MustParseSVG([]byte(`<svg ` + iconStroke + `><rect x="5" y="3.5" width="14" height="17" rx="2"/><path d="M9 8.5h6M9 12h6M9 15.5h4"/></svg>`))
)

// iconButton is a header toolbar button: a quiet rounded square holding
// a 17-DIP icon, without the theme button chrome — the way Apple Notes'
// borderless toolbar icons read.
func iconButton(c *ui.Context, icon *ui.SVG, tip string, fn func()) ui.Element {
	btn := ui.ButtonBase(c).Key(tip).Label(tip).Tooltip(tip).Size(28, 28).Radius(14)
	if btn.Pressed() {
		btn.Background(c.Theme().SurfacePressed)
	} else if btn.Hovered() {
		btn.Background(c.Theme().SurfaceHover)
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(17, 17)
	})
	return btn.OnClick(fn)
}
