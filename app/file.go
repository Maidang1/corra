package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

// FileOps bundles the file operations the menus and shortcuts call.
type FileOps struct {
	App    *App
	Window *mygo.Window
	// OnOpened, when set, is called after a path is loaded (restart
	// the external-change watcher, for example).
	OnOpened func(path string)
}

var mdFilters = []mygo.FileFilter{
	{Extensions: []string{"md", "markdown", "mdx", "txt"}},
}

// Open shows the open dialog and loads the chosen file.
func (f *FileOps) Open() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: f.Window, Title: "打开 Markdown 文件", Filters: mdFilters,
	})
	if err != nil || len(paths) == 0 {
		return
	}
	f.OpenPath(paths[0])
}

// OpenPath loads a file by path.
func (f *FileOps) OpenPath(path string) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		mygo.Dialog.Error("打开失败", err.Error())
		return
	}
	f.App.Loaded(path, string(data))
	if f.OnOpened != nil {
		f.OnOpened(path)
	}
}

// OpenDropped opens a file dropped onto the window (called from OnFileDrop).
func (f *FileOps) OpenDropped(path string) {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") ||
		strings.HasSuffix(lower, ".mdx") || strings.HasSuffix(lower, ".txt") {
		if f.checkDirty() {
			f.OpenPath(path)
		}
	}
}

// Save writes the document to its path, or asks for one.
func (f *FileOps) Save() bool {
	if f.App.Path == "" {
		return f.SaveAs()
	}
	return f.write(f.App.Path)
}

// SaveAs asks for a path and writes.
func (f *FileOps) SaveAs() bool {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: f.Window, Title: "保存 Markdown 文件",
		DefaultPath: defaultSaveName(f.App.Path),
		Filters:     mdFilters,
	})
	if err != nil || path == "" {
		return false
	}
	if f.write(path) {
		f.App.mu.Lock()
		f.App.Path = path
		f.App.Title = path
		f.App.pushRecent(path)
		f.App.mu.Unlock()
		return true
	}
	return false
}

func (f *FileOps) write(path string) bool {
	f.App.mu.Lock()
	data := f.App.Source
	f.App.mu.Unlock()
	if err := os.WriteFile(filepath.Clean(path), []byte(data), 0o644); err != nil {
		mygo.Dialog.Error("保存失败", err.Error())
		return false
	}
	f.App.mu.Lock()
	f.App.Dirty = false
	f.App.mu.Unlock()
	return true
}

// ExportPDF renders the document to PDF and asks where to save it.
func (f *FileOps) ExportPDF(render func(title string) ([]byte, error)) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: f.Window, Title: "导出 PDF",
		DefaultPath: strings.TrimSuffix(defaultSaveName(f.App.Path), ".md") + ".pdf",
		Filters:     []mygo.FileFilter{{Extensions: []string{"pdf"}}},
	})
	if err != nil || path == "" {
		return
	}
	f.App.mu.Lock()
	title := f.App.Title
	f.App.mu.Unlock()
	data, err := render(title)
	if err != nil {
		mygo.Dialog.Error("导出失败", err.Error())
		return
	}
	if err := os.WriteFile(filepath.Clean(path), data, 0o644); err != nil {
		mygo.Dialog.Error("导出失败", err.Error())
		return
	}
}

// CheckDirty asks before discarding unsaved changes (exported for menus).
func (f *FileOps) CheckDirty() bool { return f.checkDirty() }

// checkDirtyExport is CheckDirty under a name menus use for New.
func (f *FileOps) checkDirtyExport() bool { return f.checkDirty() }

// checkDirty asks before discarding unsaved changes.
func (f *FileOps) checkDirty() bool {
	f.App.mu.Lock()
	dirty := f.App.Dirty
	f.App.mu.Unlock()
	if !dirty {
		return true
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:  f.Window,
		Type:    mygo.MessageWarning,
		Title:   "未保存的更改",
		Message: "当前文档有未保存的更改，继续将丢弃这些更改。",
		Buttons: []string{"保存并继续", "丢弃更改", "取消"},
	})
	if err != nil {
		return false
	}
	switch res.Button {
	case 0:
		return f.Save()
	case 1:
		return true
	}
	return false
}

func defaultSaveName(path string) string {
	if path == "" {
		return "未命名.md"
	}
	return filepath.Base(path)
}

// ---- External change watching (polling; MyGo has no public watch API) ----

// Watcher polls the open file for external changes.
type Watcher struct {
	App  *App
	Path string
	Mod  time.Time
	Stop chan struct{}
}

// NewWatcher starts watching path; call Close to stop.
func NewWatcher(app *App, path string) *Watcher {
	w := &Watcher{App: app, Path: path, Stop: make(chan struct{})}
	if st, err := os.Stat(path); err == nil {
		w.Mod = st.ModTime()
	}
	go w.loop()
	return w
}

func (w *Watcher) loop() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-w.Stop:
			return
		case <-tick.C:
			st, err := os.Stat(w.Path)
			if err != nil || !st.ModTime().After(w.Mod) {
				continue
			}
			w.Mod = st.ModTime()
			w.App.mu.Lock()
			dirty := w.App.Dirty
			w.App.Conflict = dirty
			w.App.mu.Unlock()
			if !dirty {
				// No local edits: reload silently.
				if data, err := os.ReadFile(w.Path); err == nil {
					w.App.Loaded(w.Path, string(data))
				}
			} else {
				w.App.InvalidateUI()
			}
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() { close(w.Stop) }

// ResolveConflict reloads the file, discarding local edits.
func (f *FileOps) ResolveConflict(reload bool) {
	f.App.mu.Lock()
	f.App.Conflict = false
	path := f.App.Path
	f.App.mu.Unlock()
	if !reload {
		// Keep our version: save it back over the external change.
		f.Save()
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		f.App.Loaded(path, string(data))
	} else {
		mygo.Dialog.Error("重新加载失败", fmt.Sprintf("%v", err))
	}
}
