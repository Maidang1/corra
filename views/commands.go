package views

import (
	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
	"strings"
)

func (v *View) OpenFolderAction() {
	v.openFolderDialog("打开文件夹", v.App.LibraryRoot)
}

func (v *View) openNote(path string) {
	if path == v.App.Path {
		return
	}
	if !v.flushAutosave() {
		return
	}
	if err := v.App.OpenNote(path); err != nil {
		v.SaveError = err.Error()
		return
	}
	v.SaveError = ""
	v.edState = nil
}

func (v *View) selectFolder(folder string) {
	if !v.flushAutosave() {
		return
	}
	v.App.SelFolder = folder
	v.reconcileSelection()
	v.App.InvalidateUI()
}

func (v *View) reconcileSelection() {
	if !v.flushAutosave() {
		return
	}
	notes := v.App.VisibleNotes()
	for _, n := range notes {
		if n.Path == v.App.SelNote {
			return
		}
	}
	if len(notes) > 0 {
		v.openNote(notes[0].Path)
	} else {
		v.App.SelNote = ""
		v.App.NewDocument()
		v.edState = nil
		if v.App.NoteOpened != nil {
			v.App.NoteOpened("")
		}
	}
}

// folderMenuItems builds the folder commands of a context menu. folder is
// the menu's target: the sidebar row's folder, not the selected one, so a
// right-click acts on the row under the pointer.
func (v *View) folderMenuItems(m *ui.Menu, folder string) {
	if m.Item("打开文件夹…").Chosen() {
		v.OpenFolderAction()
	}
	if m.Item("新建文件夹…").Chosen() {
		v.openFolderDialog("新建文件夹", "")
	}
	normal := folder != "" && folder != app.TrashName
	if m.Item("重命名文件夹…").Disabled(!normal).Chosen() {
		v.openFolderDialog("重命名文件夹", folder)
	}
	if m.Item("删除空文件夹").Disabled(!normal).Chosen() {
		v.removeFolder(folder)
	}
	m.Separator()
	if m.Item("刷新文档库").Chosen() {
		if v.flushAutosave() {
			v.App.ScanLibrary()
			v.reconcileSelection()
		}
	}
}

func (v *View) noteMenu(m *ui.Menu) {
	if m.Item("恢复文档").Disabled(v.App.SelFolder != app.TrashName || v.App.SelNote == "").Chosen() {
		if err := v.App.RestoreSelectedNote(); err != nil {
			v.SaveError = err.Error()
		}
	}
	m.Submenu("移动到", func(sub *ui.Menu) {
		if sub.Item("文档库根目录").Disabled(!v.canEdit() || v.App.SelNote == "").Chosen() {
			v.moveNote("")
		}
		if v.App.Lib != nil {
			for _, folder := range v.App.Lib.Folders() {
				if sub.Item(folder).Disabled(!v.canEdit() || v.App.SelNote == "").Chosen() {
					v.moveNote(folder)
				}
			}
		}
	})
	if m.Item("删除文档…").Disabled(v.App.SelNote == "").Chosen() {
		v.deleteNote()
	}
	m.Separator()
	if m.Item("设置…").Chosen() {
		v.SettingsOpen = true
	}
}

func (v *View) moveNote(folder string) {
	if !v.canEdit() || !v.flushAutosave() {
		return
	}
	if err := v.App.MoveSelectedNote(folder); err != nil {
		v.SaveError = err.Error()
	}
	v.edState = nil
}

func (v *View) formatMenu(m *ui.Menu) {
	m.Submenu("标题级别", func(sub *ui.Menu) {
		for level, label := range []string{"一级标题", "二级标题", "三级标题", "四级标题", "五级标题", "六级标题"} {
			lv := level + 1
			if sub.Item(label).Chosen() {
				v.formatSelection(strings.Repeat("#", lv)+" ", "")
			}
		}
	})
	for _, item := range []struct{ label, prefix, suffix string }{
		{"正文", "", ""},
		{"粗体", "**", "**"}, {"斜体", "*", "*"}, {"下划线", "<u>", "</u>"},
		{"删除线", "~~", "~~"}, {"高亮", "==", "=="},
		{"上标", "^", "^"}, {"下标", "~", "~"}, {"行内代码", "`", "`"},
		{"项目符号列表", "- ", ""}, {"编号列表", "1. ", ""}, {"任务列表", "- [ ] ", ""}, {"引用", "> ", ""},
	} {
		if m.Item(item.label).Chosen() {
			v.formatSelection(item.prefix, item.suffix)
		}
	}
	m.Separator()
	if m.Item("插入链接…").Chosen() {
		v.openLinkDialog()
	}
	if m.Item("清除格式").Chosen() {
		v.formatSelection("clear", "")
	}
}

func (v *View) formatSelection(prefix, suffix string) {
	if !v.canEdit() {
		return
	}
	if !v.syncEditor() {
		return
	}
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		if command := richFormat(prefix, suffix); command != "" {
			v.rich.Format(command)
			v.rich.Focus()
			v.NativeChanged()
		}
		return
	}
	if prefix == "clear" {
		v.clearFormattingSource()
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	sel := v.edState.Selection()
	text := v.edState.Buffer.Slice(sel.Start(), sel.End())
	if suffix != "" {
		if text == "" {
			text = "文字"
		}
		v.edState.Insert(prefix + text + suffix)
		v.edState.Select(sel.Start()+len(prefix), sel.Start()+len(prefix)+len(text))
	} else {
		src := v.edState.Buffer.String()
		start := strings.LastIndex(src[:sel.Start()], "\n") + 1
		end := sel.End()
		if i := strings.IndexByte(src[end:], '\n'); i >= 0 {
			end += i
		} else {
			end = len(src)
		}
		lines := strings.Split(src[start:end], "\n")
		for i, line := range lines {
			lines[i] = prefix + strings.TrimLeft(line, "#> ")
		}
		v.edState.Select(start, end)
		v.edState.Insert(strings.Join(lines, "\n"))
	}
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}

func (v *View) FormatAction(prefix, suffix string) { v.formatSelection(prefix, suffix) }

// BlockAction runs a native structural block command (block-up, block-down,
// block-duplicate, block-delete) on the rich editor.
func (v *View) BlockAction(command string) {
	if v.rich == nil || !v.canEdit() || !v.flushAutosave() {
		return
	}
	v.rich.Format(command)
}

func (v *View) deleteDialog(c *ui.Context) {
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(360).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, "删除文档？").FontSize(16).Bold()
			message := "文档将移到最近删除，你可以在那里恢复。"
			if v.App.SelFolder == app.TrashName {
				message = "这会永久删除文档，无法恢复。"
			}
			ui.Text(c, message).FontSize(13)
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				ui.Button(c, "取消").OnClick(func() { v.DeleteOpen = false })
				ui.Button(c, "删除").TextColor(c.Theme().Danger).OnClick(func() {
					if !v.flushAutosave() {
						return
					}
					if err := v.App.DeleteSelectedNote(); err != nil {
						v.SaveError = err.Error()
						return
					}
					v.DeleteOpen = false
					v.edState = nil
					if v.App.Path == "" && v.App.NoteOpened != nil {
						v.App.NoteOpened("")
					}
				})
			})
		})
	})
	if !open {
		v.DeleteOpen = false
	}
}

func (v *View) openFolderDialog(kind, folder string) {
	v.FolderDialog, v.FolderDraft, v.FolderOriginal, v.FolderError = kind, folder, folder, ""
	v.App.InvalidateUI()
}
func (v *View) NewFolderAction() { v.openFolderDialog("新建文件夹", "") }
func (v *View) removeFolder(folder string) {
	if !v.flushAutosave() {
		return
	}
	if err := v.App.DeleteFolder(folder); err != nil {
		v.openFolderDialog("删除空文件夹", folder)
		v.FolderError = "仅可删除空文件夹：" + err.Error()
		return
	}
	v.reconcileSelection()
}
func (v *View) folderDialog(c *ui.Context) {
	if v.FolderDialog == "打开文件夹" {
		v.openLibraryDialog(c)
		return
	}
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(360).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, v.FolderDialog).FontSize(16).Bold()
			ui.TextInput(c, &v.FolderDraft).Label("文件夹名称").Placeholder("文件夹名称").AutoFocus()
			if v.FolderError != "" {
				ui.Text(c, v.FolderError).FontSize(12).TextColor(c.Theme().Danger)
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				ui.Button(c, "取消").OnClick(func() { v.FolderDialog = "" })
				ui.PrimaryButton(c, "确定").Disabled(strings.TrimSpace(v.FolderDraft) == "").OnClick(func() {
					if !v.flushAutosave() {
						v.FolderError = v.SaveError
						return
					}
					var err error
					switch v.FolderDialog {
					case "新建文件夹":
						err = v.App.CreateFolder(strings.TrimSpace(v.FolderDraft))
					case "重命名文件夹":
						err = v.App.RenameFolder(v.FolderOriginal, strings.TrimSpace(v.FolderDraft))
					default:
						err = v.App.DeleteFolder(v.FolderOriginal)
					}
					if err != nil {
						v.FolderError = err.Error()
						return
					}
					v.FolderDialog = ""
					v.edState = nil
					v.reconcileSelection()
				})
			})
		})
	})
	if !open {
		v.FolderDialog = ""
	}
}

// clearFormattingSource strips inline markers from the source-mode
// selection (or the current line when collapsed).
func (v *View) clearFormattingSource() {
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	sel := v.edState.Selection()
	start, end := sel.Start(), sel.End()
	if start == end {
		src := v.edState.Buffer.String()
		start = strings.LastIndex(src[:start], "\n") + 1
		if i := strings.IndexByte(src[end:], '\n'); i >= 0 {
			end += i
		} else {
			end = len(src)
		}
	}
	text := v.edState.Buffer.Slice(start, end)
	for _, m := range []string{"**", "__", "~~", "==", "`", "^", "~"} {
		text = strings.ReplaceAll(text, m, "")
	}
	text = strings.ReplaceAll(text, "<u>", "")
	text = strings.ReplaceAll(text, "</u>", "")
	// unwrap links: [text](url) -> text
	for {
		open := strings.IndexByte(text, '[')
		if open < 0 {
			break
		}
		mid := strings.Index(text[open:], "](")
		if mid < 0 {
			break
		}
		mid += open
		closing := strings.IndexByte(text[mid+2:], ')')
		if closing < 0 {
			break
		}
		text = text[:open] + text[open+1:mid] + text[mid+2+closing+1:]
	}
	v.edState.Select(start, end)
	v.edState.Insert(text)
	v.edState.Select(start, start+len(text))
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}
