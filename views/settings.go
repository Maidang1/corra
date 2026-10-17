package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// OpenSettings shows the settings modal (menu Cmd+, and the header gear).
func (v *View) OpenSettings() {
	v.SettingsOpen = true
	v.App.InvalidateUI()
}

// saveSettings persists the preferences changed in the settings modal.
func (v *View) saveSettings() {
	if v.OnSaveSettings != nil {
		v.OnSaveSettings()
	}
	v.App.InvalidateUI()
}

// settingsDialog is the preferences modal: appearance, editor typography
// and the writing aids, in one place instead of scattered across menus.
func (v *View) settingsDialog(c *ui.Context) {
	t := c.Theme()
	open := true
	ui.Modal(c, &open, func() {
		ui.Box(c).Width(480).Background(t.Surface).Radius(16).Padding(22).Children(func() {
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				ui.Text(c, "设置").FontSize(16).Bold()

				ui.Text(c, "外观").FontSize(12).Bold().TextColor(t.TextMuted)
				ui.RadioGroup(c, func() {
					ui.Row(c).Gap(14).Children(func() {
						ui.Radio(c, &v.App.ThemeMode, app.ThemeSystem, "跟随系统").OnChange(v.saveSettings)
						ui.Radio(c, &v.App.ThemeMode, app.ThemeLight, "浅色").OnChange(v.saveSettings)
						ui.Radio(c, &v.App.ThemeMode, app.ThemeDark, "深色").OnChange(v.saveSettings)
					})
				}).Label("外观")

				ui.Divider(c).FillWidth()
				ui.Text(c, "字体").FontSize(12).Bold().TextColor(t.TextMuted)
				fontRow(c, "正文字体", v.App.BodyFont, app.BodyFonts, func(value string) {
					v.App.BodyFont = value
					v.saveSettings()
				})
				fontRow(c, "代码字体", v.App.MonoFont, app.MonoFonts, func(value string) {
					v.App.MonoFont = value
					v.saveSettings()
				})

				ui.Divider(c).FillWidth()
				ui.Text(c, "写作").FontSize(12).Bold().TextColor(t.TextMuted)
				settingsSwitch(c, "字数统计", v.App.WordCount, func(on bool) {
					v.App.WordCount = on
					v.App.InvalidateUI()
				})
				settingsSwitch(c, "拼写检查", v.App.Spellcheck, func(on bool) {
					v.App.Spellcheck = on
					if v.rich != nil {
						v.rich.SetSpellcheck(on)
					}
					v.App.InvalidateUI()
				})
				settingsSwitch(c, "专注模式", v.App.FocusMode, func(on bool) {
					v.App.FocusMode = on
					if v.rich != nil {
						v.rich.SetFocusMode(on)
					}
					v.App.InvalidateUI()
				})
				settingsSwitch(c, "打字机模式", v.App.Typewriter, func(on bool) {
					v.App.Typewriter = on
					if v.rich != nil {
						v.rich.SetTypewriter(on)
					}
					v.App.InvalidateUI()
				})

				ui.Divider(c).FillWidth()
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Button(c, "图床设置…").OnClick(func() {
						v.SettingsOpen = false
						v.HostDraft = v.App.Hosting
						v.HostOpen = true
					})
					ui.Spacer(c)
					ui.PrimaryButton(c, "完成").OnClick(func() { v.SettingsOpen = false })
				})
			})
		})
	})
	if !open {
		v.SettingsOpen = false
	}
}

// fontRow is one font picker: a muted label and a pop-up button showing
// the current family, whose menu lists the choices with a check mark.
func fontRow(c *ui.Context, label, current string, choices []app.FontChoice, set func(string)) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(13)
		ui.Spacer(c)
		name := choices[0].Label
		for _, ch := range choices {
			if ch.Value == current {
				name = ch.Label
			}
		}
		ui.MenuButton(c, name, func(m *ui.Menu) {
			for _, ch := range choices {
				ch := ch
				if m.Item(ch.Label).Checked(ch.Value == current).Chosen() {
					set(ch.Value)
				}
			}
		}).Label(label)
	})
}

// settingsSwitch is one labeled toggle row of the settings modal.
func settingsSwitch(c *ui.Context, label string, on bool, set func(bool)) {
	value := on
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(13)
		ui.Spacer(c)
		ui.Switch(c, &value).OnChange(func() { set(value) })
	})
}
