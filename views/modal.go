package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// conflictDialog asks how to resolve an external change.
func (v *View) conflictDialog(c *ui.Context) {
	t := c.Theme()
	open := true
	ui.Modal(c, &open, func() {
		ui.Box(c).Width(440).Background(t.Surface).Radius(16).Padding(22).Children(func() {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Text(c, "文件已被外部修改").FontSize(15).Bold()
				ui.Text(c, "磁盘上的文件发生了变化，而这里也有未保存的编辑。").
					FontSize(13).TextColor(t.TextMuted).LineHeight(1.5)
				// macOS puts dialog actions on the trailing edge.
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					ui.Button(c, "保留我的版本").OnClick(func() {
						v.saveAndReloadConflict(false)
					})
					ui.Button(c, "重新加载").OnClick(func() {
						v.saveAndReloadConflict(true)
					})
				})
			})
		})
	})
	if !open {
		// Keep the conflict pending; dismissing is not consent to overwrite.
		v.App.Conflict = true
	}
}

// saveAndReloadConflict resolves the conflict (kept here to avoid importing
// FileOps into the view; the saver is bound at startup).
func (v *View) saveAndReloadConflict(reload bool) {
	if fn := v.OnConflictResolve; fn != nil {
		if !reload && !v.syncEditor() {
			return
		}
		fn(reload)
		v.SaveError = ""
		if reload {
			v.edState = nil
		}
	}
}

// hostingDialog edits the GitHub image hosting configuration.
func (v *View) hostingDialog(c *ui.Context) {
	t := c.Theme()
	open := true
	ui.Modal(c, &open, func() {
		ui.Box(c).Width(520).Background(t.Surface).Radius(16).Padding(22).Children(func() {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Text(c, "GitHub 图床设置").FontSize(15).Bold()
				ui.Text(c, "粘贴图片时上传到指定仓库，并插入 CDN 链接。Token 仅存本机（0600 权限）。").
					FontSize(12).TextColor(t.TextMuted).LineHeight(1.5)
				enabled := v.HostDraft.Enabled
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Switch(c, &enabled).OnChange(func() {
						v.HostDraft.Enabled = enabled
					})
					ui.Text(c, "启用图床上传").FontSize(13)
				})
				hostField := func(label string, value *string, ph string) {
					ui.Column(c).FillWidth().Gap(2).Children(func() {
						ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted)
						ui.TextInput(c, value).Placeholder(ph)
					})
				}
				hostField("仓库 Owner", &v.HostDraft.Owner, "如 Maidang1")
				hostField("仓库名", &v.HostDraft.Repo, "如 images")
				hostField("分支", &v.HostDraft.Branch, "main")
				hostField("存储路径前缀", &v.HostDraft.Prefix, "如 images/2026")
				hostField("CDN 地址模板", &v.HostDraft.CDN, app.DefaultCDN)
				hostField("GitHub Token", &v.HostDraft.Token, "ghp_…（仅存本机）")
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Spacer(c)
					ui.Button(c, "取消").OnClick(func() { v.HostOpen = false })
					ui.PrimaryButton(c, "保存").OnClick(func() {
						if fn := v.OnSaveHosting; fn != nil {
							fn(v.HostDraft)
						}
						v.HostOpen = false
					})
				})
			})
		})
	})
	if !open {
		v.HostOpen = false
	}
}

// onConflictResolve and onSaveHosting are bound from main.go.
// (declared in view.go)
