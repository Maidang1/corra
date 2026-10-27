package views

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (v *View) submitLibraryPath() {
	path := strings.TrimSpace(v.FolderDraft)
	if path == "" || v.OnOpenFolderPath == nil {
		return
	}
	v.FolderDialog = ""
	v.OnOpenFolderPath(path)
}

func (v *View) openLibraryDialog(c *ui.Context) {
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(520).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, "打开文件夹").FontSize(16).Bold()
			ui.Text(c, "输入已有文件夹的路径，递归加载其中的 Markdown 文档。").FontSize(13).TextColor(c.Theme().TextMuted)
			input := ui.TextInput(c, &v.FolderDraft).Label("文件夹路径").Placeholder("/Users/… 或 ~/Documents").AutoFocus()
			if input.Submitted() {
				v.submitLibraryPath()
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				ui.Button(c, "取消").OnClick(func() { v.FolderDialog = "" })
				ui.PrimaryButton(c, "打开").Disabled(strings.TrimSpace(v.FolderDraft) == "").OnClick(v.submitLibraryPath)
			})
		})
	})
	if !open {
		v.FolderDialog = ""
	}
}

func (v *View) libraryLoadingDialog(c *ui.Context) {
	open := true
	cancel := func() {
		if v.OnCancelFolderLoad != nil {
			v.OnCancelFolderLoad()
		}
	}
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(460).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			if v.App.LibraryLoading {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Label("正在加载").Size(18, 18)
					ui.Text(c, "正在加载文档库…").FontSize(16).Bold()
				})
				ui.Text(c, "正在扫描子文件夹并解析 Markdown 文件。").FontSize(13).TextColor(c.Theme().TextMuted)
			} else {
				ui.Text(c, "打开文件夹失败").FontSize(16).Bold()
				ui.Text(c, v.App.LibraryLoadError).FontSize(13).TextColor(c.Theme().Danger)
			}
			ui.Text(c, v.App.LibraryLoadPath).FillWidth().FontSize(12).TextColor(c.Theme().TextMuted)
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				if v.App.LibraryLoading {
					ui.Button(c, "取消加载").OnClick(cancel)
				} else {
					ui.Button(c, "关闭").OnClick(cancel)
					ui.PrimaryButton(c, "修改路径").OnClick(func() { path := v.App.LibraryLoadPath; cancel(); v.openFolderDialog("打开文件夹", path) })
				}
			})
		})
	})
	if !open {
		cancel()
	}
}
