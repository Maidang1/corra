package views

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// quickCommand is one palette action. Title and keywords are matched against
// the text after ">".
type quickCommand struct {
	title    string
	hint     string
	icon     *ui.SVG
	keywords string
	run      func()
}

// quickCommandList is the palette's action list, mirroring the menus so the
// same operations are reachable from the keyboard.
func (v *View) quickCommandList() []quickCommand {
	themeIcon := iconMoon
	if v.themeIsDark() {
		themeIcon = iconSun
	}
	return []quickCommand{
		{"新建文档", "⌘N", iconCompose, "new note create", v.NewNoteAction},
		{"新建文件夹…", "⇧⌘N", iconFolder, "new folder", v.NewFolderAction},
		{"打开文件夹…", "⌥⌘O", iconFolder, "open folder directory library", v.OpenFolderAction},
		{"目录（TOC）", "", iconTOC, "outline toc headings", v.ToggleOutlineAction},
		{"保存", "⌘S", iconDoc, "save", func() { v.FlushAutosave() }},
		{"插入链接…", "⌘K", iconPaperclip, "link insert url", v.InsertLinkAction},
		{"插入表格", "⌥⌘T", iconTable, "table insert", v.InsertTableAction},
		{"插入任务列表", "⇧⌘L", iconChecklist, "task todo list insert",
			func() { v.InsertMarkdown("\n- [ ] ") }},
		{"插入脚注", "⌥⌘F", iconDoc, "footnote insert", v.InsertFootnote},
		{"返回脚注跳转位置", "⌘⌥←", iconDoc, "footnote back return jump", v.FootnoteBackAction},
		{"查找…", "⌘F", iconSearch, "find search", v.FindAction},
		{"查找并替换…", "", iconSearch, "replace find substitute", v.FindReplaceAction},
		{"复制 Markdown", "", iconDoc, "copy markdown", v.CopyMarkdown},
		{"复制 HTML", "", iconCode, "copy html", v.CopyHTML},
		{"导出…", "", iconShare, "export pdf html word pandoc", v.OpenExport},
		{"切换文稿 / 源码", "", iconCode, "mode source rich toggle", v.ToggleModeAction},
		{"切换主题", "", themeIcon, "theme dark light appearance", v.toggleTheme},
		{"打开设置…", "⌘,", iconGear, "settings preferences", v.OpenSettings},
		{"收起 / 展开侧栏", "", iconSidebar, "sidebar toggle", v.ToggleSidebarAction},
		{"删除所选文档", "", iconTrash, "delete trash note", v.DeleteNoteAction},
	}
}

// quickCommandMode reports whether the palette is running commands instead of
// searching notes.
func (v *View) quickCommandMode() bool {
	return strings.HasPrefix(strings.TrimSpace(v.QuickQuery), ">")
}

// quickCommandMatches filters the action list by title or English keywords.
func (v *View) quickCommandMatches() []quickCommand {
	query := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v.QuickQuery), ">"))
	query = strings.ToLower(query)
	out := make([]quickCommand, 0, 8)
	for _, cmd := range v.quickCommandList() {
		if query != "" && !strings.Contains(strings.ToLower(cmd.title), query) &&
			!strings.Contains(cmd.keywords, query) {
			continue
		}
		out = append(out, cmd)
		if len(out) == 8 {
			break
		}
	}
	return out
}

// runQuickCommand closes the palette first, so a command that opens its own
// modal (settings, link) is not covered by the palette.
func (v *View) runQuickCommand(cmd quickCommand) {
	v.QuickOpen = false
	v.QuickQuery = ""
	v.App.InvalidateUI()
	if cmd.run != nil {
		cmd.run()
	}
}

// QuickOpenAction opens the quick switcher (Cmd+Shift+P): type a title or path
// and press Return to jump to a note without leaving the keyboard.
func (v *View) QuickOpenAction() {
	v.QuickQuery = ""
	v.QuickOpen = true
	v.QuickIndex = 0
	v.App.InvalidateUI()
}

// quickMatches filters the library by the quick-switch query, most recently
// edited first (the library's own order), capped to what the panel shows.
func (v *View) quickMatches() []app.Note {
	query := strings.ToLower(strings.TrimSpace(v.QuickQuery))
	out := make([]app.Note, 0, 8)
	for _, note := range v.App.VisibleNotes() {
		if query != "" && !noteMatches(note, query) {
			continue
		}
		out = append(out, note)
		if len(out) == 8 {
			break
		}
	}
	return out
}

// noteMatches looks at everything a person would type to find a note.
func noteMatches(note app.Note, query string) bool {
	haystack := strings.ToLower(note.Title + "\n" + note.Path + "\n" + note.Folder + "\n" + note.Snippet)
	return strings.Contains(haystack, query)
}

// quickRecent lists library notes opened before, newest first, excluding the
// note already open. Only notes still in the library are shown, so the row can
// reuse the same title and relative path.
func (v *View) quickRecent() []app.Note {
	byPath := make(map[string]app.Note, 8)
	for _, note := range v.App.VisibleNotes() {
		byPath[note.Path] = note
	}
	out := make([]app.Note, 0, 3)
	for _, path := range v.App.RecentPaths() {
		note, ok := byPath[path]
		if !ok || note.Path == v.App.Path {
			continue
		}
		out = append(out, note)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// quickOrdered is the row order: recent notes first when the query is empty,
// then title/path matches, without repeating a recent note.
func (v *View) quickOrdered() []app.Note {
	matches := v.quickMatches()
	if strings.TrimSpace(v.QuickQuery) != "" {
		return matches
	}
	recents := v.quickRecent()
	seen := make(map[string]bool, len(recents))
	out := make([]app.Note, 0, 8)
	for _, note := range recents {
		seen[note.Path] = true
		out = append(out, note)
	}
	for _, note := range matches {
		if seen[note.Path] {
			continue
		}
		out = append(out, note)
		if len(out) == 8 {
			break
		}
	}
	return out
}

// quickPath is the note's path relative to the library root, as the panel
// shows it.
func (v *View) quickPath(note app.Note) string {
	root := v.App.LibraryRoot
	if root != "" {
		if rel, err := filepath.Rel(root, note.Path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(note.Path)
}

// openQuickNote closes the palette and opens the note, restoring editor focus.
func (v *View) openQuickNote(note app.Note) {
	v.QuickOpen = false
	v.QuickQuery = ""
	if note.Path != v.App.Path {
		v.openNote(note.Path)
	}
	v.focusEditor = true
	v.App.InvalidateUI()
}

// quickOpenDialog is the command-palette style overlay: a search field over a
// short result list. Escape and a backdrop click close it (the shell modal
// handles both).
func (v *View) quickOpenDialog(c *ui.Context) {
	t := c.Theme()
	commandMode := v.quickCommandMode()
	placeholder := "搜索文档标题或路径…"
	if commandMode {
		placeholder = "搜索命令…（新建、插入、导出…）"
	}
	open := true
	notes := v.quickOrdered()
	cmds := v.quickCommandMatches()
	count := len(notes)
	if commandMode {
		count = len(cmds)
	}
	if v.QuickIndex >= count {
		v.QuickIndex = 0
	}
	ui.Modal(c, &open, func() {
		ui.Box(c).Width(560).Background(t.Surface).Radius(16).Padding(10).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				// Arrow keys move the highlight; the input keeps the caret keys.
				if count > 0 {
					if ui.Box(c).Absolute().Size(0, 0).OverlayShortcut(0, ui.KeyDown) {
						v.QuickIndex = (v.QuickIndex + 1) % count
						v.App.InvalidateUI()
					}
					if ui.Box(c).Absolute().Size(0, 0).OverlayShortcut(0, ui.KeyUp) {
						v.QuickIndex = (v.QuickIndex + count - 1) % count
						v.App.InvalidateUI()
					}
				}
				ui.Row(c).FillWidth().Height(44).Gap(8).Padding(0, 12).Radius(11).
					Background(fieldFill(t)).Children(func() {
					ui.Icon(c, iconSearch).Size(16, 16).TextColor(t.TextMuted)
					field := ui.TextInputBase(c, &v.QuickQuery).Label("快速打开").Placeholder(placeholder).
						FillWidth().FontSize(14).AutoFocus()
					if field.Changed() {
						v.QuickIndex = 0
					}
					if field.Submitted() {
						if commandMode {
							if v.QuickIndex < len(cmds) {
								v.runQuickCommand(cmds[v.QuickIndex])
							}
						} else if v.QuickIndex < len(notes) {
							v.openQuickNote(notes[v.QuickIndex])
						}
					}
				})
				if commandMode {
					if len(cmds) == 0 {
						ui.Column(c).FillWidth().Center().Padding(26, 18).Gap(5).Children(func() {
							ui.Text(c, "没有匹配的命令").FontSize(13).TextColor(t.TextMuted)
							ui.Text(c, "试试「新建」「插入」「导出」").FontSize(11.5).TextColor(t.TextMuted)
						})
					} else {
						ui.Column(c).FillWidth().Gap(1).Children(func() {
							for i, cmd := range cmds {
								v.quickCommandRow(c, cmd, i == v.QuickIndex)
							}
						})
					}
				} else {
					matches := notes
					if len(matches) == 0 {
						ui.Column(c).FillWidth().Center().Padding(26, 18).Gap(5).Children(func() {
							ui.Text(c, "没有匹配的文档").FontSize(13).TextColor(t.TextMuted)
							ui.Text(c, "换个标题、文件夹或路径试试").FontSize(11.5).TextColor(t.TextMuted)
						})
					} else {
						recentCount := 0
						if strings.TrimSpace(v.QuickQuery) == "" {
							recentCount = len(v.quickRecent())
						}
						ui.Column(c).FillWidth().Gap(1).Children(func() {
							if recentCount > 0 {
								v.quickSection(c, "最近打开")
							}
							for i, note := range matches {
								if recentCount > 0 && i == recentCount {
									v.quickSection(c, "全部文档")
								}
								v.quickRow(c, note, i == v.QuickIndex)
							}
						})
					}
				}
				ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(8, 4, 2, 4).Children(func() {
					if commandMode {
						ui.Text(c, "↵ 执行").FontSize(10.5).TextColor(t.TextMuted)
						ui.Text(c, "·").FontSize(10.5).TextColor(t.TextMuted).Padding(0, 5)
						ui.Text(c, "↑↓ 选择").FontSize(10.5).TextColor(t.TextMuted)
					} else {
						ui.Text(c, "↵ 打开").FontSize(10.5).TextColor(t.TextMuted)
						ui.Text(c, "·").FontSize(10.5).TextColor(t.TextMuted).Padding(0, 5)
						ui.Text(c, "↑↓ 选择").FontSize(10.5).TextColor(t.TextMuted)
						ui.Text(c, "·").FontSize(10.5).TextColor(t.TextMuted).Padding(0, 5)
						ui.Text(c, "输入 > 执行命令").FontSize(10.5).TextColor(t.TextMuted)
					}
					ui.Spacer(c)
					ui.Text(c, "esc 关闭").FontSize(10.5).TextColor(t.TextMuted)
				})
			})
		})
	})
	if !open {
		v.QuickOpen = false
		v.QuickQuery = ""
		v.focusEditor = true
		v.App.InvalidateUI()
	}
}

// quickCommandRow is one action: icon, title and its menu shortcut.
func (v *View) quickCommandRow(c *ui.Context, cmd quickCommand, primary bool) {
	t := c.Theme()
	fg, sub := t.Text, t.TextMuted
	if primary {
		fg, sub = t.AccentText, t.AccentText
	}
	row := ui.ButtonBase(c).Key("命令"+cmd.title).Label("执行命令："+cmd.title).
		FillWidth().Padding(8, 10).Radius(8)
	if primary {
		row = row.Background(t.AccentPressed)
	} else if row.Hovered() {
		row = row.Background(sidebarHover(t))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, cmd.icon).Size(16, 16).TextColor(sub)
			ui.Text(c, cmd.title).FontSize(13).TextColor(fg).Grow(1).MinWidth(0).
				SingleLine().Ellipsis("…")
			if cmd.hint != "" {
				ui.Text(c, cmd.hint).FontSize(11).TextColor(sub).NoWrap().Shrink(0)
			}
		})
	}).OnClick(func() { v.runQuickCommand(cmd) })
}

// quickSection labels a group of rows.
func (v *View) quickSection(c *ui.Context, label string) {
	t := c.Theme()
	ui.Text(c, label).FontSize(10.5).Bold().TextColor(t.TextMuted).Padding(8, 10, 2, 10)
}

// quickRow is one result: document icon, title, relative path and the time.
// The first row is highlighted because Return opens it.
func (v *View) quickRow(c *ui.Context, note app.Note, primary bool) {
	t := c.Theme()
	fg, sub := t.Text, t.TextMuted
	if primary {
		fg, sub = t.AccentText, t.AccentText
	}
	row := ui.ButtonBase(c).Key("快速打开"+note.Path).Label("打开文档："+note.Title).
		FillWidth().Padding(8, 10).Radius(8)
	if primary {
		row = row.Background(t.AccentPressed)
	} else if row.Hovered() {
		row = row.Background(sidebarHover(t))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconDoc).Size(16, 16).TextColor(sub)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, note.Title).FontSize(13).Bold().TextColor(fg).
					SingleLine().Ellipsis("…").FillWidth().MinWidth(0)
				ui.Text(c, v.quickPath(note)).FontSize(11).TextColor(sub).
					SingleLine().Ellipsis("…").FillWidth().MinWidth(0)
			})
			ui.Text(c, noteTime(note.ModTime)).FontSize(10.5).TextColor(sub).NoWrap().Shrink(0)
		})
	}).OnClick(func() { v.openQuickNote(note) })
}
