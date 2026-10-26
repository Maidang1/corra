// Command corra is a single-pane Markdown editor with native UI, written
// in Go with the MyGo toolkit: no webview, no HTML, no JavaScript.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/nativerich"
	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/views"
)

var (
	editorApp   *app.App
	fileOps     *app.FileOps
	view        *views.View
	images      *app.ImageLoader
	watcher     *app.Watcher
	configDir   string
	libraryRoot string
	mainWindow  *mygo.Window
)

func main() {
	editorApp = app.New()

	mygo.App.WhenReady(func() {
		setupDirs()
		setupLibrary()

		images = app.NewImageLoader(filepath.Join(configDir, "image-cache"))
		view = views.New(editorApp, images)

		win := mygo.NewWindow(mygo.WindowOptions{
			Title:                "Corra",
			Width:                1280,
			Height:               820,
			MinWidth:             960,
			MinHeight:            560,
			Content:              ui.View(buildView),
			TitleBarStyle:        mygo.TitleBarHiddenInset,
			TrafficLightPosition: &mygo.Point{X: 20, Y: 26},
			TitleBarHeight:       58,
		})
		mainWindow = win
		if native := nativerich.New(win.NativeHandle(), view.NativeChanged, view.EditSource); native != nil {
			native.OnLinkCandidates = view.LinkCandidates
			native.OnOpenDocument = func(target string) bool {
				path, ok := view.ResolveLinkedDocument(target)
				if ok {
					fileOps.OpenPath(path)
				}
				return ok
			}
			view.AttachRichEditor(native)
		}

		fileOps = &app.FileOps{App: editorApp, Window: win}
		fileOps.BeforeOpen = view.FlushAutosave
		view.OnConflictResolve = func(reload bool) { fileOps.ResolveConflict(reload) }
		view.OnNoteSaved = watchCurrent // our own autosave: re-arm the watcher
		view.OnSaveHosting = func(cfg app.HostingConfig) {
			if err := app.SaveHosting(configDir, cfg); err != nil {
				mygo.Dialog.Error("保存设置失败", err.Error())
			}
			editorApp.Hosting = cfg
		}
		view.OnSaveSettings = func() {
			if err := app.SaveSettings(configDir, editorApp.SnapshotSettings()); err != nil {
				mygo.Dialog.Error("保存设置失败", err.Error())
			}
		}
		view.OnOpenRecent = func(path string) { fileOps.OpenPath(path) }
		view.OnExportPDF = func() {
			if view.SyncBuffer() {
				fileOps.ExportPDF(exportPDF)
			}
		}
		view.OnExportHTML = func() {
			if view.SyncBuffer() {
				fileOps.ExportHTML(exportHTML)
			}
		}
		view.OnExportPandoc = func(label, ext, pandocArg string) {
			if view.SyncBuffer() {
				fileOps.ExportPandoc(label, ext, pandocArg)
			}
		}
		view.OnSave = func() bool {
			return view.FlushAutosave()
		}
		view.OnOpen = func() { fileOps.Open() }
		view.OnInsertImage = func() { insertImageAndUpload(win) }
		win.OnClose(func(e *mygo.CloseEvent) {
			if !view.FlushAutosave() {
				e.PreventDefault()
			}
		})
		win.OnClosed(func() {
			if watcher != nil {
				watcher.Close()
			}
			view.Close()
		})

		editorApp.Hosting = app.LoadHosting(configDir)
		editorApp.ApplySettings(app.LoadSettings(configDir))
		editorApp.HasSysDark = true
		editorApp.SysDark = mygo.Theme.IsDark()
		mygo.Theme.OnUpdated(func() { win.Update(func() { editorApp.SysDark = mygo.Theme.IsDark(); editorApp.InvalidateUI() }) })
		editorApp.NoteOpened = func(string) { watchCurrent() }
		fileOps.OnOpened = func(string) { view.SaveError = ""; watchCurrent() }

		mygo.App.SetMenu(buildMenu())

		// A command-line path or file association opens that file;
		// otherwise the most recent library note opens.
		opened := false
		if len(os.Args) > 1 {
			fileOps.OpenPath(os.Args[1])
			opened = true
		}
		mygo.App.OnOpenFile(func(path string) {
			fileOps.OpenPath(path)
		})
		mygo.App.OnSecondInstance(func(args []string, _ string) {
			if len(args) > 1 {
				fileOps.OpenPath(args[1])
			}
		})
		if !opened {
			openMostRecent()
		}

		win.OnFileDrop(func(e *mygo.FileDropEvent) {
			for _, p := range e.Paths {
				fileOps.OpenDropped(p)
			}
		})
		win.Update(func() { editorApp.InvalidateUI() })
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// buildView is the MyGo view; it also binds window services on first build.
func buildView(c *ui.Context) {
	if view != nil {
		view.Serve(c)
	}
}

func setupDirs() {
	if dir := os.Getenv("CORRA_CONFIG_DIR"); dir != "" {
		configDir = dir
	} else if dir, err := mygo.App.Path("userConfig"); err == nil {
		configDir = dir
	} else {
		configDir = filepath.Join(os.TempDir(), "corra-config")
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "config dir: %v\n", err)
	}
}

// setupLibrary points the app at its notes folder (a folder of Markdown
// files, like Apple Notes' storage) and seeds a welcome note the first
// time the folder is empty.
func setupLibrary() {
	libraryRoot = filepath.Join(configDir, "备忘录")
	_, statErr := os.Stat(libraryRoot)
	firstLaunch := os.IsNotExist(statErr)
	if err := os.MkdirAll(libraryRoot, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "library dir: %v\n", err)
	}
	editorApp.LibraryRoot = libraryRoot
	if firstLaunch {
		path := filepath.Join(libraryRoot, "欢迎使用 Corra.md")
		if err := os.WriteFile(path, []byte(app.Welcome), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "seed note: %v\n", err)
		}
	}
	editorApp.ScanLibrary()
}

// openMostRecent opens the newest library note, if any.
func openMostRecent() {
	editorApp.ScanLibrary()
	notes := editorApp.VisibleNotes()
	if len(notes) == 0 {
		editorApp.NewDocument()
		return
	}
	if err := editorApp.OpenNote(notes[0].Path); err != nil {
		fmt.Fprintf(os.Stderr, "open note: %v\n", err)
	}
}

// watchCurrent (re)starts the external-change watcher for the open file.
func watchCurrent() {
	if watcher != nil {
		watcher.Close()
		watcher = nil
	}
	path := editorApp.CurrentPath()
	if path != "" {
		watcher = app.NewWatcherOnUI(editorApp, path, mainWindow.Update)
	}
}

// exportPDF renders the current document to PDF bytes. The source is
// parsed fresh so edits made after the last preview are never lost.
func exportPDF(title string) ([]byte, error) {
	src := editorApp.CurrentSource()
	baseDir := ""
	if p := editorApp.CurrentPath(); p != "" {
		baseDir = filepath.Dir(p)
	}
	loader := func(url string) ([]byte, error) {
		return images.LoadImageBytes(url, baseDir)
	}
	doc := render.ParseOptions(src, editorApp.SmartPunct)
	doc.MathOptions = render.MathOptions{Numbering: editorApp.MathNumbering}
	return render.ExportPDFOpts(context.Background(), doc, title, loader)
}

// exportHTML renders the current document to a standalone HTML page.
func exportHTML(title string) ([]byte, error) {
	return render.ExportHTMLOpts(context.Background(), editorApp.CurrentSource(), title, editorApp.SmartPunct, render.MathOptions{Numbering: editorApp.MathNumbering})
}

// insertImageAndUpload asks for an image file, uploads it when hosting is
// configured, and inserts the markdown reference at the end.
func insertImageAndUpload(win *mygo.Window) {
	if !view.FlushAutosave() || editorApp.CurrentPath() == "" {
		return
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: win, Title: "插入图片",
		Filters: []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "svg"}}},
	})
	if err != nil || len(paths) == 0 {
		return
	}
	p := paths[0]
	data, err := os.ReadFile(filepath.Clean(p))
	if err != nil {
		mygo.Dialog.Error("读取图片失败", err.Error())
		return
	}
	cfg := editorApp.Hosting
	var url string
	if cfg.Enabled && cfg.Token != "" {
		res, err := app.UploadImageToGitHub(cfg, filepath.Base(p), data)
		if err != nil {
			mygo.Dialog.Error("上传失败", err.Error())
			return
		}
		url = res.URL
	} else {
		// No hosting: reference relative to the document.
		if editorApp.CurrentPath() != "" {
			if rel, err := filepath.Rel(filepath.Dir(editorApp.CurrentPath()), p); err == nil {
				url = filepath.ToSlash(rel)
			}
		}
		if url == "" {
			url = p
		}
	}
	alt := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	view.InsertMarkdown(fmt.Sprintf("![%s](<%s>)", strings.ReplaceAll(alt, "]", "\\]"), strings.ReplaceAll(url, ">", "%3E")))
}
