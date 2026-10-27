package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// FileOps bundles the file operations the menus and shortcuts call.
type FileOps struct {
	App    *App
	Window *mygo.Window
	// BeforeOpen flushes pending editor changes; false cancels the open.
	BeforeOpen func() bool
	// OnOpened, when set, is called after a path is loaded (restart
	// the external-change watcher, for example).
	OnOpened         func(path string)
	OnLibraryOpened  func(root string)
	Dispatch         func(func())
	folderLoadID     uint64
	cancelFolderLoad context.CancelFunc
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

// OpenPath opens a document, or adopts a directory as the document library.
func (f *FileOps) OpenPath(path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		f.OpenFolderPath(path)
		return
	}
	if f.App.LibraryLoading {
		f.CancelFolderLoad()
	}
	if f.BeforeOpen != nil && !f.BeforeOpen() {
		return
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		mygo.Dialog.Error("打开失败", err.Error())
		return
	}
	f.App.Loaded(path, string(data))
	f.adoptPath(path)
}

// OpenDropped opens a file dropped onto the window (called from OnFileDrop).
func (f *FileOps) OpenDropped(path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		f.OpenPath(path)
		return
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") ||
		strings.HasSuffix(lower, ".mdx") || strings.HasSuffix(lower, ".txt") {
		if f.BeforeOpen != nil || f.checkDirty() {
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
	if err := f.saveAsPath(path); err != nil {
		mygo.Dialog.Error("保存失败", err.Error())
		return false
	}
	return true
}

func (f *FileOps) saveAsPath(path string) error {
	if err := f.writePath(path, true); err != nil {
		return err
	}
	f.adoptPath(path)
	return nil
}

// adoptPath keeps document identity and library commands aligned after an
// external open or Save As; an external file must not target the old selection.
func (f *FileOps) adoptPath(path string) {
	f.App.mu.Lock()
	root := f.App.LibraryRoot
	f.App.mu.Unlock()
	folder, selected := "", ""
	if dir, err := openLibraryRoot(root, false); err == nil {
		if rel, err := libraryNotePath(dir, root, path); err == nil {
			selected = path
			folder = filepath.ToSlash(filepath.Dir(rel))
			if folder == "." {
				folder = ""
			}
			if isTrashPath(rel) {
				folder = TrashName
			}
		}
		_ = dir.Close()
	}
	f.App.mu.Lock()
	f.App.Path, f.App.Title = path, filepath.Base(path)
	f.App.SelFolder, f.App.SelNote, f.App.Query = folder, selected, ""
	f.App.pushRecent(path)
	f.App.mu.Unlock()
	f.App.ScanLibrary()
	if f.OnOpened != nil {
		f.OnOpened(path)
	}
}

func (f *FileOps) write(path string) bool {
	if err := f.writePath(path, false); err != nil {
		mygo.Dialog.Error("保存失败", err.Error())
		return false
	}
	return true
}

func (f *FileOps) writePath(path string, allowCreate bool) error {
	data := f.App.CurrentSource()
	_, statErr := os.Lstat(path)
	if allowCreate && os.IsNotExist(statErr) {
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		_, err = out.WriteString(data)
		if err == nil {
			err = out.Sync()
		}
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
			return err
		}
	} else if err := SaveExistingNote(path, data); err != nil {
		return err
	}
	f.App.mu.Lock()
	f.App.Dirty = false
	f.App.mu.Unlock()
	return nil
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

// ExportHTML renders the document to a standalone HTML page.
func (f *FileOps) ExportHTML(render func(title string) ([]byte, error)) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      f.Window,
		Title:       "导出 HTML",
		DefaultPath: strings.TrimSuffix(defaultSaveName(f.App.Path), ".md") + ".html",
		Filters:     []mygo.FileFilter{{Extensions: []string{"html"}}},
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

// ExportPandoc converts the document with pandoc and saves the result.
func (f *FileOps) ExportPandoc(label, ext, pandocArg string) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      f.Window,
		Title:       "导出 " + label,
		DefaultPath: strings.TrimSuffix(defaultSaveName(f.App.Path), ".md") + "." + ext,
		Filters:     []mygo.FileFilter{{Extensions: []string{ext}}},
	})
	if err != nil || path == "" {
		return
	}
	f.App.mu.Lock()
	src, title := f.App.CurrentSource(), f.App.Title
	f.App.mu.Unlock()
	data, err := ExportPandoc(src, pandocArg, title)
	if err != nil {
		mygo.Dialog.Error("导出失败", err.Error())
		return
	}
	if err := os.WriteFile(filepath.Clean(path), data, 0o644); err != nil {
		mygo.Dialog.Error("导出失败", err.Error())
		return
	}
}

// Import asks for a document pandoc can read and opens it as a new note.
func (f *FileOps) Import() {
	var filters []mygo.FileFilter
	for _, f := range ImportFormats {
		filters = append(filters, mygo.FileFilter{Name: f.Label, Extensions: f.Exts})
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: f.Window, Title: "导入为 Markdown", Filters: filters,
	})
	if err != nil || len(paths) == 0 {
		return
	}
	md, err := ImportPandoc(paths[0])
	if err != nil {
		mygo.Dialog.Error("导入失败", err.Error())
		return
	}
	f.App.Loaded("", md)
	f.App.Dirty = true // not on disk yet: saving asks for a path
	f.App.InvalidateUI()
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

	dispatch  func(func())
	closeOnce sync.Once
}

// NewWatcher starts watching path and delivers changes on MyGo's main thread.
// Call Close to stop. NewWatcherOnUI allows an explicit UI dispatcher.
func NewWatcher(app *App, path string) *Watcher {
	return NewWatcherOnUI(app, path, mygo.RunOnMain)
}

// NewWatcherOnUI polls on a worker and dispatches document changes to the UI.
// dispatch must serialize its callbacks with UI actions (for example win.Update).
func NewWatcherOnUI(app *App, path string, dispatch func(func())) *Watcher {
	if dispatch == nil {
		dispatch = mygo.RunOnMain
	}
	w := &Watcher{App: app, Path: path, Stop: make(chan struct{}), dispatch: dispatch}
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
			w.pollOnce()
		}
	}
}

func (w *Watcher) stopped() bool {
	select {
	case <-w.Stop:
		return true
	default:
		return false
	}
}

func (w *Watcher) pollOnce() {
	if w.stopped() {
		return
	}
	st, err := os.Stat(w.Path)
	if err != nil || !st.ModTime().After(w.Mod) {
		return
	}
	data, err := os.ReadFile(w.Path)
	if err != nil {
		return
	}
	w.Mod = st.ModTime()
	w.dispatch(func() {
		if w.stopped() || w.App.CurrentPath() != w.Path {
			return
		}
		// A save or deletion after polling makes this queued snapshot stale.
		current, err := os.Stat(w.Path)
		if err != nil || !current.ModTime().Equal(st.ModTime()) || current.Size() != st.Size() {
			return
		}
		w.App.mu.Lock()
		dirty := w.App.Dirty
		if dirty {
			w.App.Conflict = true
		}
		w.App.mu.Unlock()
		if dirty {
			w.App.InvalidateUI()
			return
		}
		w.App.Loaded(w.Path, string(data))
	})
}

// Close stops the watcher.
func (w *Watcher) Close() { w.closeOnce.Do(func() { close(w.Stop) }) }

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
