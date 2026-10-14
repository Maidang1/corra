package main

import (
	"github.com/egoist/mygo"
)

// buildMenu builds the application menu.
func buildMenu() *mygo.Menu {
	item := func(label string, accel string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: accel, Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			if fn != nil {
				fn()
			}
		}}
	}
	role := func(r mygo.MenuRole) *mygo.MenuItem {
		return &mygo.MenuItem{Role: r}
	}

	fileMenu := &mygo.MenuItem{
		Label: "文件",
		Submenu: []*mygo.MenuItem{
			item("新建备忘录", "CmdOrCtrl+N", func() { view.NewNoteAction() }),
			item("打开…", "CmdOrCtrl+O", func() { fileOps.Open() }),
			mygo.Separator(),
			item("保存", "CmdOrCtrl+S", func() {
				view.FlushAutosave()
				fileOps.Save()
			}),
			item("另存为…", "CmdOrCtrl+Shift+S", func() {
				view.FlushAutosave()
				fileOps.SaveAs()
			}),
			mygo.Separator(),
			item("导出 PDF…", "", func() {
				view.FlushAutosave()
				fileOps.ExportPDF(exportPDF)
			}),
			item("插入图片（可上传图床）…", "", func() { view.OnInsertImage() }),
			mygo.Separator(),
			item("图床设置…", "", func() {
				view.HostDraft = editorApp.Hosting
				view.HostOpen = true
				editorApp.InvalidateUI()
			}),
			mygo.Separator(),
			role(mygo.RoleClose),
			role(mygo.RoleQuit),
		},
	}

	editMenu := &mygo.MenuItem{
		Label: "编辑",
		Submenu: []*mygo.MenuItem{
			// Roles dispatch to the focused editor: the Markdown editor
			// column handles undo, cut/copy/paste and select-all itself.
			role(mygo.RoleUndo),
			role(mygo.RoleRedo),
			mygo.Separator(),
			role(mygo.RoleCut),
			role(mygo.RoleCopy),
			role(mygo.RolePaste),
			role(mygo.RoleSelectAll),
			mygo.Separator(),
			item("删除所选备忘录", "", func() { view.DeleteNoteAction() }),
		},
	}

	viewMenu := &mygo.MenuItem{
		Label: "视图",
		Submenu: []*mygo.MenuItem{
			item("预览 / 编辑", "CmdOrCtrl+E", func() { view.ToggleModeAction() }),
			mygo.Separator(),
			item("放大", "CmdOrCtrl+=", func() { zoomBy(0.1) }),
			item("缩小", "CmdOrCtrl+-", func() { zoomBy(-0.1) }),
			item("实际大小", "CmdOrCtrl+0", func() { view.App.Zoom = 1; editorApp.InvalidateUI() }),
			mygo.Separator(),
			item("切换深浅色", "", func() { view.ToggleThemeAction() }),
			role(mygo.RoleToggleFullScreen),
		},
	}

	helpMenu := &mygo.MenuItem{
		Label: "帮助",
		Submenu: []*mygo.MenuItem{
			role(mygo.RoleAbout),
		},
	}

	return mygo.NewMenu([]*mygo.MenuItem{fileMenu, editMenu, viewMenu, helpMenu})
}

// zoomBy adjusts the document zoom.
func zoomBy(d float32) {
	z := view.App.Zoom + d
	if z < 0.7 {
		z = 0.7
	}
	if z > 1.8 {
		z = 1.8
	}
	view.App.Zoom = z
	editorApp.InvalidateUI()
}
