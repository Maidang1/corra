// Package app wires the editor state: the document, file handling,
// settings, image hosting and the async render cache.
package app

import (
	"image"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/editor"
	"github.com/Maidang1/corra/render"
)

// ThemeMode selects the appearance.
type ThemeMode int

const (
	ThemeSystem ThemeMode = iota
	ThemeLight
	ThemeDark
)

// Mode is the pane mode. The native rich editor is the default when attached;
// platforms without it keep the Markdown source editor.
type Mode int

const (
	ModeEdit Mode = iota // full Markdown source editing
	ModeRich             // native WYSIWYG editing
)

// App is the editor's root state. The view is a function of App; mutations
// go through the methods, which keep the parsed document in sync.
type App struct {
	mu sync.Mutex

	// Document state.
	Source string
	Doc    *render.Document
	Path   string
	Dirty  bool
	Title  string

	// View state.
	Mode        Mode
	ThemeMode   ThemeMode
	Zoom        float32 // 1 = 100%
	Sidebar     bool    // navigation sidebar visible (folders + notes list)
	Outline     bool    // outline (TOC) panel visible
	OutlineSel  int     // selected outline row
	FindOpen    bool
	FindQuery   string
	FindCurrent int

	// Typora-parity display options.
	SmartPunct      bool   // typographic quotes/dashes in the source editor & export
	CodeLineNumbers bool   // line numbers in code fences
	MathNumbering   string // none (default), ams, or all
	WordCount       bool   // word-count status bar
	Spellcheck      bool   // continuous spell checking in the native editor
	FocusMode       bool   // dim all but the caret's paragraph
	Typewriter      bool   // keep the caret line vertically centered

	// Editing.
	Hist editor.Undo

	// Recent files, most recent first.
	Recent []string

	// Notes library: a folder of Markdown files, listed like Apple Notes.
	LibraryRoot string
	Lib         *Library
	SelFolder   string // selected sidebar folder, "" = all notes
	SelNote     string // path of the note open in the editor column
	Query       string // search filter for the notes list

	// External change conflict.
	Conflict      bool
	conflictDirty bool // we had unsaved changes when the file changed

	// Hosting settings (图床).
	Hosting HostingConfig

	// System appearance, sampled once at startup (mygo.Theme blocks before
	// the app runtime is up, e.g. in headless tests).
	HasSysDark bool
	SysDark    bool

	// Async rendered images (math, diagrams, charts), keyed by block content.
	images     map[string]*imageEntry
	mathImages *mathImageBatch
	invalidate func() // window invalidate, set at startup
	services   ui.Services

	// NoteOpened, when set, is called after OpenNote loads a note (main
	// uses it to re-arm the external-change watcher).
	NoteOpened func(path string)
}

type imageEntry struct {
	ready bool
	img   image.Image
	err   error
}

// New creates the app with defaults and a welcome document.
func New() *App {
	a := &App{
		Doc:     &render.Document{},
		Zoom:    1,
		Sidebar: true,
		images:  map[string]*imageEntry{},
	}
	a.Source = Welcome
	a.reparse()
	a.Hist.PushBase(a.Source)
	a.Title = "未命名"
	return a
}

// Welcome is the seeded first note of a fresh library.
const Welcome = `# 欢迎使用 Corra

Corra 是一款纯原生 Markdown 桌面编辑器，提供文档库、所见即所得与源码编辑，并支持 GFM、数学公式、图表及多格式导出。

## 文档库与编辑区

- **文档库**：「全部文档」汇集库中的 Markdown 文件，各文件夹与「最近删除」用于组织文档；文档列表按修改日期分组，搜索框可检索完整 Markdown 源码
- **编辑区**：直接写作、编辑与预览，顶部显示文档修改时间；「视图」菜单可开启大纲面板与字数统计

边栏左上角按钮或「视图」菜单（Cmd+Alt+S）可收起边栏，专注写作。

## 编辑与预览

macOS 默认使用**原生富文本编辑**，标题、正文和格式直接可见，修改自动保存。
Cmd+E 切换 Markdown 源码，Cmd+Z 撤销；源码模式有语法高亮、多光标、自动配对与查找替换（Cmd+F）。
代码块隐藏围栏后直接编辑，公式和图片就地渲染，表格可在编辑区直接编辑。

## 格式一览

- 粗体 **重点**、斜体 *强调*、删除线 ~~划掉~~、下划线 <u>下划线</u>、高亮 ==荧光==
- 上标 x^2^、下标 H~2~O、键盘 <kbd>⌘</kbd> <kbd>K</kbd>、emoji :smile: :rocket:
- 数学公式：行内 $E=mc^2$ 与块级公式；图表：mermaid、d2、goat、数据图表
- 脚注[^1]、目录标记 [TOC]、YAML 头部信息、任务列表、表格
- 链接可点击：外部 [Typora](https://typora.io)、页内 #格式一览 锚点

[^1]: 脚注显示在文末，导出 PDF / HTML 时保留。

$$
a^2 + b^2 = c^2
$$

- [ ] 试试在预览里点击复选框
- [x] 已完成

` + "```chart\ntype: bar\ntitle: 示例\nlabels: [一, 二, 三, 四]\nseries:\n  - name: 销量\n    data: [3, 7, 5, 9]\n```" + `

> 文件菜单导出 PDF / HTML / Word（Word 需要 pandoc）；粘贴或拖入图片自动保存为附件（或上传 GitHub 图床）。
`

// reparse refreshes Doc from Source. Callers hold no lock; internal only.
func (a *App) reparse() {
	a.Doc = render.ParseOptions(a.Source, a.SmartPunct)
}

// SetSource replaces the document, records undo and reparses.
func (a *App) SetSource(src string, caret int) {
	a.mu.Lock()
	a.Source = src
	a.reparse()
	a.Hist.Push(src, caret)
	a.Dirty = true
	a.mu.Unlock()
	a.InvalidateUI()
}

// MarkSourceDirty notes a source edit made in the editor pane without
// reparsing on every keystroke; the document re-parses when the pane
// toggles back to preview mode.
func (a *App) MarkSourceDirty(src string) {
	a.mu.Lock()
	a.Source = src
	a.Dirty = true
	a.mu.Unlock()
}

// MarkSaved clears the dirty flag after an autosave.
func (a *App) MarkSaved() {
	a.mu.Lock()
	a.Dirty = false
	a.mu.Unlock()
	a.InvalidateUI()
}

// Refresh reparses Source (used when leaving source mode).
func (a *App) Refresh() {
	a.mu.Lock()
	a.reparse()
	a.Hist.Push(a.Source, 0)
	a.mu.Unlock()
	a.InvalidateUI()
}

// SetSourceQuiet replaces the document without an undo step (load, undo/redo).
func (a *App) SetSourceQuiet(src string, caret int) {
	a.mu.Lock()
	a.Source = src
	a.reparse()
	a.Dirty = true
	a.mu.Unlock()
	a.InvalidateUI()
}

// Loaded resets state for a freshly opened file.
func (a *App) Loaded(path, src string) {
	a.mu.Lock()
	a.Path = path
	a.Source = src
	a.reparse()
	a.Dirty = false
	if path == "" {
		a.Title = "未命名"
	} else {
		a.Title = filepath.Base(path)
	}
	a.Hist.PushBase(src)
	a.Conflict = false
	a.pushRecent(path)
	a.mu.Unlock()
	a.InvalidateUI()
}

// NewDocument resets to an empty document.
func (a *App) NewDocument() {
	a.mu.Lock()
	a.Path = ""
	a.Source = ""
	a.reparse()
	a.Dirty = false
	a.Title = "未命名"
	a.Hist.PushBase("")
	a.mu.Unlock()
	a.InvalidateUI()
}

// AppendBlock adds a new block at the end of the document.
func (a *App) AppendBlock(source string) {
	a.mu.Lock()
	sep := ""
	if a.Source != "" && !endsWithNewline(a.Source) {
		sep = "\n\n"
	} else if a.Source != "" {
		sep = "\n"
	}
	a.Source += sep + source
	a.reparse()
	a.Hist.Push(a.Source, 0)
	a.Dirty = true
	a.mu.Unlock()
	a.InvalidateUI()
}

// CanUndo/CanRedo report stack state.
func (a *App) CanUndo() bool { return a.Hist.CanUndo() }
func (a *App) CanRedo() bool { return a.Hist.CanRedo() }

// Undo reverts the last change.
func (a *App) Undo() {
	if src, caret, ok := a.Hist.Undo(a.Source, 0); ok {
		a.SetSourceQuiet(src, caret)
	}
}

// Redo reapplies the last undone change.
func (a *App) Redo() {
	if src, caret, ok := a.Hist.Redo(a.Source, 0); ok {
		a.SetSourceQuiet(src, caret)
	}
}

// RecentPaths returns a copy of the recent-files list.
func (a *App) RecentPaths() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.Recent...)
}

func (a *App) pushRecent(path string) {
	out := []string{path}
	for _, p := range a.Recent {
		if p != path {
			out = append(out, p)
		}
	}
	if len(out) > 10 {
		out = out[:10]
	}
	a.Recent = out
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// CurrentPath returns the open file path ("" for a new document).
func (a *App) CurrentPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Path
}

// CurrentSource returns the document source.
func (a *App) CurrentSource() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Source
}

// CurrentDoc returns the parsed document.
func (a *App) CurrentDoc() *render.Document {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Doc
}

func endsWithNewline(s string) bool {
	return len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r')
}
