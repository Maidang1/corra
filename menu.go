package main

import (
	"github.com/egoist/mygo"

	"github.com/Maidang1/corra/app"
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
	check := func(label string, accel string, fn func(checked bool)) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Accelerator: accel, Type: mygo.MenuItemCheckbox,
			Click: func(it *mygo.MenuItem, _ *mygo.Window) { fn(it.IsChecked()) }}
	}

	fileMenu := &mygo.MenuItem{
		Label: "文件",
		Submenu: []*mygo.MenuItem{
			item("新建", "CmdOrCtrl+N", func() {
				if fileOps.CheckDirty() {
					editorApp.NewDocument()
					watchCurrent()
				}
			}),
			item("打开…", "CmdOrCtrl+O", func() { fileOps.Open() }),
			mygo.Separator(),
			item("保存", "CmdOrCtrl+S", func() { fileOps.Save() }),
			item("另存为…", "CmdOrCtrl+Shift+S", func() { fileOps.SaveAs() }),
			mygo.Separator(),
			item("导出 PDF…", "", func() { fileOps.ExportPDF(exportPDF) }),
			item("插入图片（可上传图床）…", "", func() { view.OnInsertImage() }),
			mygo.Separator(),
			item("图床设置…", "", func() {
				view.HostDraft = editorApp.Hosting
				view.HostOpen = true
				editorApp.InvalidateUI()
			}),
			mygo.Separator(),
			item("退出", "", func() { mygo.App.Quit() }),
		},
	}

	editMenu := &mygo.MenuItem{
		Label: "编辑",
		Submenu: []*mygo.MenuItem{
			item("撤销", "CmdOrCtrl+Z", func() { editorApp.Undo() }),
			item("重做", "CmdOrCtrl+Shift+Z", func() { editorApp.Redo() }),
			mygo.Separator(),
			item("查找", "CmdOrCtrl+F", func() {
				view.App.FindOpen = true
				editorApp.InvalidateUI()
			}),
			mygo.Separator(),
			item("加粗", "CmdOrCtrl+B", func() { view.WrapAction("**") }),
			item("斜体", "CmdOrCtrl+I", func() { view.WrapAction("*") }),
			item("标题", "CmdOrCtrl+1", func() { view.HeadingAction() }),
			item("无序列表", "CmdOrCtrl+Shift+8", func() { view.ListAction(false) }),
			item("有序列表", "CmdOrCtrl+Shift+7", func() { view.ListAction(true) }),
			item("引用", "CmdOrCtrl+Shift+.", func() { view.QuoteAction() }),
		},
	}

	viewMenu := &mygo.MenuItem{
		Label: "视图",
		Submenu: []*mygo.MenuItem{
			check("源码模式", "CmdOrCtrl+E", func(checked bool) {
				if checked {
					view.App.Mode = app.ModeSource
				} else {
					view.App.Mode = app.ModeLive
					view.App.Refresh()
				}
				editorApp.InvalidateUI()
			}),
			check("显示大纲", "CmdOrCtrl+B", func(checked bool) {
				view.App.Sidebar = checked
				editorApp.InvalidateUI()
			}),
			mygo.Separator(),
			item("放大", "CmdOrCtrl+=", func() { zoomBy(0.1) }),
			item("缩小", "CmdOrCtrl+-", func() { zoomBy(-0.1) }),
			item("实际大小", "CmdOrCtrl+0", func() { view.App.Zoom = 1; editorApp.InvalidateUI() }),
			mygo.Separator(),
			item("切换深浅色", "", func() { view.ToggleThemeAction() }),
		},
	}

	helpMenu := &mygo.MenuItem{
		Label: "帮助",
		Submenu: []*mygo.MenuItem{
			item("关于 Corra", "", func() {
				mygo.App.ShowAboutPanel(mygo.AboutPanelOptions{})
			}),
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
