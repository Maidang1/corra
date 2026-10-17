package views

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// LinkDialog state for 插入链接 (Cmd+K).

// openLinkDialog opens the insert-link modal, seeding the selected text.
func (v *View) openLinkDialog() {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	v.LinkText = ""
	if v.App.Mode == app.ModeEdit && v.edState != nil {
		sel := v.edState.Selection()
		v.LinkText = v.edState.Buffer.Slice(sel.Start(), sel.End())
	}
	v.LinkURL = ""
	v.LinkOpen = true
	v.App.InvalidateUI()
}

// linkDialog renders the insert-link modal.
func (v *View) linkDialog(c *ui.Context) {
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(400).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, "插入链接").FontSize(16).Bold()
			ui.TextInput(c, &v.LinkText).Label("链接文本").Placeholder("显示文本（留空则用网址）")
			ui.TextInput(c, &v.LinkURL).Label("链接地址").Placeholder("https://、#标题 或 文件.md").AutoFocus()
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				ui.Button(c, "取消").OnClick(func() { v.LinkOpen = false })
				ui.PrimaryButton(c, "确定").Disabled(strings.TrimSpace(v.LinkURL) == "").OnClick(v.applyLink)
			})
		})
	})
	if !open {
		v.LinkOpen = false
	}
}

// applyLink inserts the link into the active editor surface.
func (v *View) applyLink() {
	url := strings.TrimSpace(v.LinkURL)
	text := strings.TrimSpace(v.LinkText)
	if text == "" {
		text = url
	}
	v.LinkOpen = false
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		// With no selection the native editor inserts the text as a link.
		v.rich.Format("link:" + url + "\n" + strings.TrimSpace(v.LinkText))
		v.rich.Focus()
		v.NativeChanged()
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	sel := v.edState.Selection()
	selected := v.edState.Buffer.Slice(sel.Start(), sel.End())
	if selected != "" {
		text = selected
	}
	v.edState.Insert("[" + text + "](" + url + ")")
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}
