package views

import (
	"path"
	"sort"
	"strings"

	"github.com/Maidang1/corra/app"
	"github.com/egoist/mygo/ui"
)

type folderTreeNode struct {
	path, name string
	children   []*folderTreeNode
	notes      []app.Note
	count      int
}

func (v *View) libraryTree() *folderTreeNode {
	lib := v.App.Lib
	if v.treeRoot != nil && v.treeLibrary == lib && v.treeQuery == v.App.Query {
		return v.treeRoot
	}
	root := &folderTreeNode{}
	v.treeLibrary, v.treeQuery, v.treeRoot = lib, v.App.Query, root
	if lib == nil {
		return root
	}
	if v.folderExpanded == nil || v.treeLibraryRoot != lib.Root {
		v.treeLibraryRoot = lib.Root
		v.folderExpanded = make(map[string]bool)
	}
	nodes := map[string]*folderTreeNode{"": root}
	var folder func(string) *folderTreeNode
	folder = func(relative string) *folderTreeNode {
		if node := nodes[relative]; node != nil {
			return node
		}
		parent := path.Dir(relative)
		if parent == "." {
			parent = ""
		}
		node := &folderTreeNode{path: relative, name: path.Base(relative)}
		nodes[relative] = node
		ancestor := folder(parent)
		ancestor.children = append(ancestor.children, node)
		return node
	}
	// Derive the tree from visible documents so empty directories (including
	// asset-only folders) disappear while ancestors of documents remain.
	for _, note := range lib.Filter("", v.App.Query) {
		node := folder(note.Folder)
		node.notes = append(node.notes, note)
	}
	var finish func(*folderTreeNode) int
	finish = func(node *folderTreeNode) int {
		sort.Slice(node.children, func(i, j int) bool {
			return strings.ToLower(node.children[i].name) < strings.ToLower(node.children[j].name)
		})
		sort.Slice(node.notes, func(i, j int) bool {
			a, b := strings.ToLower(node.notes[i].Title), strings.ToLower(node.notes[j].Title)
			if a == b {
				return node.notes[i].Path < node.notes[j].Path
			}
			return a < b
		})
		node.count = len(node.notes)
		for _, child := range node.children {
			node.count += finish(child)
		}
		return node.count
	}
	finish(root)
	return root
}

func (v *View) fileTree(c *ui.Context) {
	root := v.libraryTree()
	v.treeVisibleNotes = nil
	list := ui.Scroll(c).Fill()
	list.Children(func() {
		ui.Column(c).FillWidth().Gap(1).Padding(0, 8, 8, 8).Children(func() {
			if root.count == 0 && len(root.children) == 0 {
				hint := "暂无文档"
				if v.App.Query != "" {
					hint = "没有找到匹配的文档"
				}
				ui.Text(c, hint).FontSize(13).TextColor(c.Theme().TextMuted).Padding(18, 10)
			}
			v.treeChildren(c, root, 0)
		})
	})
	if list.Shortcut(0, ui.KeyDown) {
		v.moveNoteSelection(1, true)
	}
	if list.Shortcut(0, ui.KeyUp) {
		v.moveNoteSelection(-1, true)
	}
}

func (v *View) treeChildren(c *ui.Context, node *folderTreeNode, depth int) {
	for _, child := range node.children {
		expanded, known := v.folderExpanded[child.path]
		if !known {
			expanded = depth == 0 || strings.HasPrefix(v.App.SelFolder, child.path+"/")
		}
		if v.App.Query != "" {
			expanded = true
		}
		v.treeFolderRow(c, child, depth, expanded)
		if expanded {
			v.treeChildren(c, child, depth+1)
		}
	}
	for _, note := range node.notes {
		v.treeVisibleNotes = append(v.treeVisibleNotes, note)
		v.treeNoteRow(c, note, depth)
	}
}

// Each tree row owns a full-width background. Only its contents are
// indented, so folders and documents share the same selection edges.
func (v *View) treeFolderRow(c *ui.Context, node *folderTreeNode, depth int, expanded bool) {
	t := c.Theme()
	selected := v.App.SelFolder == node.path && v.App.SelNote == ""
	row := ui.Row(c).Key("tree-folder:"+node.path).FillWidth().MinWidth(0).Height(30).
		AlignItems(ui.Center).Padding(0, 0, 0, float32(4+depth*16)).Radius(7)
	if selected {
		row.Background(t.SurfacePressed)
	} else if row.Hovered() {
		row.Background(sidebarHover(t))
	}
	row.ContextMenu(func(m *ui.Menu) { v.folderMenuItems(m, node.path) }).Children(func() {
		if len(node.children) > 0 || len(node.notes) > 0 {
			icon, label := iconChevronRight, "展开文件夹："+node.path
			if expanded {
				icon, label = iconChevronDown, "收起文件夹："+node.path
			}
			ui.ButtonBase(c).Key("disclosure:"+node.path).Label(label).Size(18, 30).Radius(5).Children(func() {
				ui.Icon(c, icon).Size(12, 12).TextColor(t.TextMuted)
			}).OnClick(func() { v.folderExpanded[node.path] = !expanded; v.App.InvalidateUI() })
		} else {
			ui.Box(c).Size(18, 30)
		}
		ui.ButtonBase(c).Key(node.path).Label("文件夹："+node.path).Tooltip(node.path).
			Grow(1).MinWidth(0).Height(30).Padding(0, 8, 0, 4).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(7).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconFolder).Size(15, 15).TextColor(t.TextMuted)
				ui.Text(c, node.name).FillWidth().MinWidth(0).FontSize(13).SingleLine().Ellipsis("…")
			})
		}).OnClick(func() { v.selectFolder(node.path) })
	})
}

func (v *View) openTreeNote(note app.Note) {
	v.openNote(note.Path)
	if v.App.Path == note.Path {
		v.App.SelFolder = note.Folder
		v.App.InvalidateUI()
	}
}

func (v *View) treeNoteRow(c *ui.Context, note app.Note, depth int) {
	t := c.Theme()
	selected := v.App.SelNote == note.Path
	fg := t.Text
	row := ui.ButtonBase(c).Key(note.Path).Label("文档："+note.Title).Tooltip(note.Title).FillWidth().MinWidth(0).Height(30).Padding(0, 8, 0, float32(26+depth*16)).Radius(7)
	if selected {
		row.Background(t.AccentPressed)
		fg = t.AccentText
	} else if row.Hovered() {
		row.Background(sidebarHover(t))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconDoc).Size(15, 15).TextColor(fg)
			ui.Text(c, note.Title).FillWidth().MinWidth(0).FontSize(13).TextColor(fg).SingleLine().Ellipsis("…")
		})
	}).OnClick(func() { v.openTreeNote(note) }).ContextMenu(func(m *ui.Menu) {
		v.openTreeNote(note)
		if v.App.SelNote != note.Path {
			m.Item("请先保存当前文档").Disabled(true)
			return
		}
		v.noteMenu(m)
	})
	if v.focusNotePath == note.Path {
		row.Focus()
		v.focusNotePath = ""
	}
}

func (v *View) sidebarTabs(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(3).Padding(0, 8, 4, 8).Children(func() {
		for _, label := range []string{"文档", "大纲"} {
			outline := label == "大纲"
			tab := ui.ButtonBase(c).Key("sidebar-tab:"+label).Label(label).Grow(1).Padding(5, 8).Radius(7).Disabled(outline && v.App.Path == "")
			if v.App.Outline == outline {
				tab.Background(t.SurfacePressed)
			} else if tab.Hovered() {
				tab.Background(sidebarHover(t))
			}
			fg := t.Text
			if outline && v.App.Path == "" {
				fg = t.TextMuted
			}
			tab.Children(func() { ui.Text(c, label).FontSize(12).TextColor(fg).Center() }).OnClick(func() {
				if !outline && v.App.SelFolder == app.TrashName {
					v.selectFolder("")
				}
				v.App.Outline = outline
				v.App.InvalidateUI()
			})
		}
	})
}
