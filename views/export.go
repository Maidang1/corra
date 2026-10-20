package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// OpenExport shows the export panel: one card per output format, so the
// writer picks from descriptions instead of guessing menu entries.
func (v *View) OpenExport() {
	if v.App.Path == "" {
		return
	}
	v.ExportOpen = true
	v.App.InvalidateUI()
}

// pandocAvailable reports whether pandoc conversions can run; tests swap the
// resolver through View.PandocPath.
func (v *View) pandocAvailable() bool {
	if v.PandocPath != nil {
		return v.PandocPath() != ""
	}
	return app.PandocPath() != ""
}

// runExport closes the panel first: the callback opens a native save dialog,
// which must not sit on top of the modal.
func (v *View) runExport(fn func()) {
	v.ExportOpen = false
	if fn != nil {
		fn()
	}
	v.App.InvalidateUI()
}

// exportDialog is the format-picker modal: built-in PDF/HTML cards plus one
// card per pandoc conversion, disabled until pandoc is installed.
func (v *View) exportDialog(c *ui.Context) {
	t := c.Theme()
	open := true
	pandoc := v.pandocAvailable()
	ui.Modal(c, &open, func() {
		ui.Box(c).Width(480).Background(t.Surface).Radius(16).Padding(22).Children(func() {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Text(c, "导出文档").FontSize(15).Bold()
				ui.Text(c, "选择一种格式，随后会询问保存位置。").
					FontSize(12).TextColor(t.TextMuted).LineHeight(1.5)
				v.exportCard(c, iconDoc, "PDF 文档 (.pdf)", "打印级版式，适合分享与存档",
					v.OnExportPDF != nil, func() { v.runExport(v.OnExportPDF) })
				v.exportCard(c, iconCode, "HTML 网页 (.html)", "自带样式的独立页面，浏览器直接打开",
					v.OnExportHTML != nil, func() { v.runExport(v.OnExportHTML) })
				for _, pf := range app.PandocFormats {
					pf := pf
					run := v.OnExportPandoc
					icon := iconDoc
					if pf.Ext == "tex" || pf.Ext == "rst" {
						icon = iconCode
					}
					v.exportCard(c, icon, pf.Label, "经 pandoc 转换",
						pandoc && run != nil, func() { v.runExport(func() { run(pf.Label, pf.Ext, pf.PandocArg) }) })
				}
				if !pandoc {
					ui.Text(c, "安装 pandoc 后可导出 Word / EPUB / LaTeX 等更多格式（brew install pandoc）。").
						FontSize(11.5).TextColor(t.TextMuted).LineHeight(1.4)
				}
				ui.Row(c).FillWidth().Children(func() {
					ui.Spacer(c)
					ui.Button(c, "取消").OnClick(func() { v.ExportOpen = false })
				})
			})
		})
	})
	if !open {
		v.ExportOpen = false
	}
}

// exportCard is one format row of the export panel: an icon, the format's
// name and blurb, and an action button on the trailing edge. The button's
// accessibility label names the format, so every card stays reachable.
func (v *View) exportCard(c *ui.Context, icon *ui.SVG, title, detail string, enabled bool, fn func()) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Padding(8, 10).Radius(10).
		Background(fieldFill(t)).Children(func() {
		ui.Icon(c, icon).Size(18, 18).TextColor(t.TextMuted)
		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().NoWrap()
			ui.Text(c, detail).FontSize(11).TextColor(t.TextMuted)
		})
		ui.Spacer(c)
		ui.Button(c, "导出").Label("导出 "+title).Disabled(!enabled).OnClick(fn)
	})
}
