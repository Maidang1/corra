package main

import (
	"github.com/egoist/mygo"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/views"
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
	checked := func(label string, on bool, fn func()) *mygo.MenuItem {
		it := item(label, "", fn)
		it.Checked = on
		return it
	}
	disabled := func(label string, off bool, fn func()) *mygo.MenuItem {
		it := item(label, "", fn)
		it.Disabled = off
		return it
	}
	role := func(r mygo.MenuRole) *mygo.MenuItem {
		return &mygo.MenuItem{Role: r}
	}

	pandoc := app.PandocPath() != ""
	exportSub := []*mygo.MenuItem{
		item("导出 PDF…", "", func() {
			if view.SyncBuffer() {
				fileOps.ExportPDF(exportPDF)
			}
		}),
		item("导出 HTML…", "", func() {
			if view.SyncBuffer() {
				fileOps.ExportHTML(exportHTML)
			}
		}),
	}
	for _, pf := range app.PandocFormats {
		pf := pf
		label := pf.Label
		if !pandoc {
			label += "（需要 pandoc）"
		}
		exportSub = append(exportSub, disabled(label, !pandoc, func() {
			if view.SyncBuffer() {
				fileOps.ExportPandoc(pf.Label, pf.Ext, pf.PandocArg)
			}
		}))
	}

	fileMenu := &mygo.MenuItem{
		Label: "文件",
		Submenu: []*mygo.MenuItem{
			item("新建文档", "CmdOrCtrl+N", func() { view.NewNoteAction() }),
			item("新建文件夹…", "CmdOrCtrl+Shift+N", func() { view.NewFolderAction() }),
			item("快速打开…", "CmdOrCtrl+Shift+P", func() { view.QuickOpenAction() }),
			item("打开…", "CmdOrCtrl+O", func() { fileOps.Open() }),
			item("打开文件夹…", "CmdOrCtrl+Alt+O", func() { view.OpenFolderAction() }),
			{Label: "最近打开", Submenu: recentItems(item)},
			disabled("导入（Word / EPUB / HTML…）", !pandoc, func() { fileOps.Import() }),
			mygo.Separator(),
			item("保存", "CmdOrCtrl+S", func() {
				view.FlushAutosave()
			}),
			item("另存为…", "CmdOrCtrl+Shift+S", func() {
				if view.SyncBuffer() {
					fileOps.SaveAs()
				}
			}),
			mygo.Separator(),
			{Label: "导出", Submenu: exportSub},
			item("打印…", "CmdOrCtrl+P", func() { view.PrintAction() }),
			mygo.Separator(),
			item("插入图片（可上传图床）…", "CmdOrCtrl+Shift+I", func() { view.OnInsertImage() }),
			item("插入目录 [TOC]", "", func() { view.InsertMarkdown("\n[TOC]\n") }),
			item("插入分页符", "", func() { view.InsertMarkdown("\n<div style=\"page-break-after: always\"></div>\n") }),
			mygo.Separator(),
			item("设置…", "CmdOrCtrl+,", func() { view.OpenSettings() }),
			mygo.Separator(),
			role(mygo.RoleClose),
			role(mygo.RoleQuit),
		},
	}

	editMenu := &mygo.MenuItem{
		Label: "编辑",
		Submenu: []*mygo.MenuItem{
			// Roles follow the native responder chain in rich mode and the
			// MyGo editor in source mode.
			role(mygo.RoleUndo),
			role(mygo.RoleRedo),
			mygo.Separator(),
			role(mygo.RoleCut),
			role(mygo.RoleCopy),
			role(mygo.RolePaste),
			role(mygo.RoleSelectAll),
			mygo.Separator(),
			item("文档内查找…", "CmdOrCtrl+F", func() { view.FindAction() }),
			item("查找并替换…", "", func() { view.FindReplaceAction() }),
			mygo.Separator(),
			item("复制 Markdown", "", func() { view.CopyMarkdown() }),
			item("复制 HTML", "", func() { view.CopyHTML() }),
			mygo.Separator(),
			item("插入任务列表", "CmdOrCtrl+Shift+L", func() { view.InsertMarkdown("\n- [ ] ") }),
			item("插入脚注", "CmdOrCtrl+Alt+F", func() { view.InsertFootnote() }),
			item("删除所选文档", "", func() { view.DeleteNoteAction() }),
		},
	}

	formatMenu := &mygo.MenuItem{
		Label: "格式",
		Submenu: []*mygo.MenuItem{
			item("粗体", "CmdOrCtrl+B", func() { view.FormatAction("**", "**") }),
			item("斜体", "CmdOrCtrl+I", func() { view.FormatAction("*", "*") }),
			item("下划线", "CmdOrCtrl+U", func() { view.FormatAction("<u>", "</u>") }),
			item("删除线", "CmdOrCtrl+Shift+X", func() { view.FormatAction("~~", "~~") }),
			item("高亮", "CmdOrCtrl+Shift+H", func() { view.FormatAction("==", "==") }),
			item("上标", "CmdOrCtrl+Shift+.", func() { view.FormatAction("^", "^") }),
			item("下标", "CmdOrCtrl+Shift+,", func() { view.FormatAction("~", "~") }),
			item("行内代码", "CmdOrCtrl+Shift+`", func() { view.FormatAction("`", "`") }),
			item("清除格式", "CmdOrCtrl+\\", func() { view.FormatAction("clear", "") }),
			mygo.Separator(),
			item("正文", "CmdOrCtrl+0", func() { view.FormatAction("", "") }),
			item("一级标题", "CmdOrCtrl+1", func() { view.FormatAction("# ", "") }),
			item("二级标题", "CmdOrCtrl+2", func() { view.FormatAction("## ", "") }),
			item("三级标题", "CmdOrCtrl+3", func() { view.FormatAction("### ", "") }),
			item("四级标题", "CmdOrCtrl+4", func() { view.FormatAction("#### ", "") }),
			item("五级标题", "CmdOrCtrl+5", func() { view.FormatAction("##### ", "") }),
			item("六级标题", "CmdOrCtrl+6", func() { view.FormatAction("###### ", "") }),
			mygo.Separator(),
			item("项目符号列表", "CmdOrCtrl+Alt+U", func() { view.FormatAction("- ", "") }),
			item("编号列表", "CmdOrCtrl+Alt+O", func() { view.FormatAction("1. ", "") }),
			item("任务列表", "CmdOrCtrl+Alt+L", func() { view.FormatAction("- [ ] ", "") }),
			item("引用", "CmdOrCtrl+Alt+Q", func() { view.FormatAction("> ", "") }),
			alertMenu(view),
			mygo.Separator(),
			item("插入链接…", "CmdOrCtrl+K", func() { view.InsertLinkAction() }),
			imageSizeMenu(view),
			item("插入表格", "CmdOrCtrl+Alt+T", func() { view.InsertTableAction() }),
			item("插入代码块", "CmdOrCtrl+Alt+C", func() { view.InsertMarkdown("\n```\n\n```\n") }),
			item("插入数学公式块", "CmdOrCtrl+Alt+M", func() { view.InsertMarkdown("\n$$\n\n$$\n") }),
			item("插入水平分割线", "CmdOrCtrl+Shift+_", func() { view.InsertMarkdown("\n---\n") }),
			mygo.Separator(),
			{Label: "块操作", Submenu: []*mygo.MenuItem{
				item("上移块", "", func() { view.BlockAction("block-up") }),
				item("下移块", "", func() { view.BlockAction("block-down") }),
				item("复制块", "", func() { view.BlockAction("block-duplicate") }),
				item("删除块", "", func() { view.BlockAction("block-delete") }),
			}},
		},
	}

	viewMenu := &mygo.MenuItem{
		Label: "视图",
		Submenu: []*mygo.MenuItem{
			item("切换源码 / 文稿", "CmdOrCtrl+E", func() { view.ToggleModeAction() }),
			item("显示 / 隐藏边栏", "CmdOrCtrl+Alt+S", func() { view.ToggleSidebarAction() }),
			item("目录（TOC）", "CmdOrCtrl+Ctrl+1", func() { view.ToggleOutlineAction() }),
			item("搜索文档", "CmdOrCtrl+Shift+F", func() { view.SearchAction() }),
			mygo.Separator(),
			checked("字数统计", editorApp.WordCount, func() { editorApp.WordCount = !editorApp.WordCount; editorApp.InvalidateUI() }),
			checked("拼写检查", editorApp.Spellcheck, func() { view.ToggleSpellcheck() }),
			checked("专注模式", editorApp.FocusMode, func() { view.ToggleFocusMode() }),
			checked("打字机模式", editorApp.Typewriter, func() { view.ToggleTypewriter() }),
			mygo.Separator(),
			checked("代码块显示行号", editorApp.CodeLineNumbers, func() { editorApp.CodeLineNumbers = !editorApp.CodeLineNumbers; editorApp.InvalidateUI() }),
			checked("智能标点", editorApp.SmartPunct, func() { editorApp.SmartPunct = !editorApp.SmartPunct; editorApp.Refresh() }),
			{Label: "公式编号", Submenu: []*mygo.MenuItem{
				checked("不自动编号", editorApp.MathNumbering == "" || editorApp.MathNumbering == "none", func() { editorApp.MathNumbering = "none"; editorApp.InvalidateUI() }),
				checked("按 AMS 规则编号", editorApp.MathNumbering == "ams", func() { editorApp.MathNumbering = "ams"; editorApp.InvalidateUI() }),
				checked("全部块公式编号", editorApp.MathNumbering == "all", func() { editorApp.MathNumbering = "all"; editorApp.InvalidateUI() }),
			}},
			item("重新排版公式", "CmdOrCtrl+Alt+Shift+M", func() { view.RefreshMath() }),
			mygo.Separator(),
			item("放大", "CmdOrCtrl+=", func() { zoomBy(0.1) }),
			item("缩小", "CmdOrCtrl+-", func() { zoomBy(-0.1) }),
			item("实际大小", "CmdOrCtrl+Shift+0", func() { view.App.Zoom = 1; editorApp.InvalidateUI() }),
			mygo.Separator(),
			role(mygo.RoleToggleFullScreen),
		},
	}

	helpMenu := &mygo.MenuItem{
		Label: "帮助",
		Submenu: []*mygo.MenuItem{
			role(mygo.RoleAbout),
		},
	}

	return mygo.NewMenu([]*mygo.MenuItem{role(mygo.RoleAppMenu), fileMenu, editMenu, formatMenu, viewMenu, role(mygo.RoleWindowMenu), helpMenu})
}

// recentItems lists the most recently opened files.
func recentItems(item func(string, string, func()) *mygo.MenuItem) []*mygo.MenuItem {
	recent := editorApp.RecentPaths()
	if len(recent) == 0 {
		return []*mygo.MenuItem{{Label: "（无最近文件）", Disabled: true}}
	}
	out := make([]*mygo.MenuItem, 0, len(recent))
	for _, p := range recent {
		path := p
		out = append(out, item(path, "", func() { fileOps.OpenPath(path) }))
	}
	return out
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

// alertMenu lists the GitHub callout kinds. Each one turns the caret's
// paragraph into "> [!LABEL]" and every renderer paints it as a callout.
func alertMenu(view *views.View) *mygo.MenuItem {
	kind := func(label string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			view.InsertAlert(label)
		}}
	}
	return &mygo.MenuItem{
		Label: "提示块",
		Submenu: []*mygo.MenuItem{
			kind("Note"), kind("Tip"), kind("Important"), kind("Warning"), kind("Caution"),
			mygo.Separator(),
			{Label: "移除提示块", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { view.InsertAlert("") }},
		},
	}
}

// imageSizeMenu resizes the figure under the caret. The sizes are the suffixes
// Typora writes onto the destination, so the document keeps the reader's choice
// and every renderer draws it the same way.
func imageSizeMenu(view *views.View) *mygo.MenuItem {
	size := func(label, spec string) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
			view.InsertImageSize(spec)
		}}
	}
	return &mygo.MenuItem{
		Label: "图片大小",
		Submenu: []*mygo.MenuItem{
			size("原始大小", ""),
			size("25%", "25%"), size("50%", "50%"), size("75%", "75%"), size("100%", "100%"),
			mygo.Separator(),
			size("400 像素宽", "400x"), size("600 像素宽", "600x"), size("800 像素宽", "800x"),
		},
	}
}
