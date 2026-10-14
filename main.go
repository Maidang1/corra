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
	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/views"
)

var (
	editorApp *app.App
	fileOps   *app.FileOps
	view      *views.View
	images    *app.ImageLoader
	watcher   *app.Watcher
	configDir string
)

func main() {
	editorApp = app.New()

	mygo.App.WhenReady(func() {
		setupDirs()

		images = app.NewImageLoader(filepath.Join(configDir, "image-cache"))

		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Corra",
			Width:     1080,
			Height:    760,
			MinWidth:  640,
			MinHeight: 480,
			Content:   ui.View(buildView),
		})

		fileOps = &app.FileOps{App: editorApp, Window: win}
		view = views.New(editorApp, images)
		view.OnConflictResolve = func(reload bool) { fileOps.ResolveConflict(reload) }
		view.OnSaveHosting = func(cfg app.HostingConfig) {
			if err := app.SaveHosting(configDir, cfg); err != nil {
				mygo.Dialog.Error("保存设置失败", err.Error())
			}
			editorApp.Hosting = cfg
		}
		view.OnOpenRecent = func(path string) { fileOps.OpenPath(path) }
		view.OnExportPDF = func() { fileOps.ExportPDF(exportPDF) }
		view.OnSave = func() bool { return fileOps.Save() }
		view.OnOpen = func() { fileOps.Open() }
		view.OnInsertImage = func() { insertImageAndUpload(win) }

		editorApp.Hosting = app.LoadHosting(configDir)
		editorApp.HasSysDark = true
		editorApp.SysDark = mygo.Theme.IsDark()

		mygo.App.SetMenu(buildMenu())

		// Open a file from the command line or a file association.
		if len(os.Args) > 1 {
			fileOps.OpenPath(os.Args[1])
		}
		mygo.App.OnOpenFile(func(path string) {
			fileOps.OpenPath(path)
		})
		mygo.App.OnSecondInstance(func(args []string, _ string) {
			if len(args) > 1 {
				fileOps.OpenPath(args[1])
			}
		})

		win.OnFileDrop(func(e *mygo.FileDropEvent) {
			for _, p := range e.Paths {
				fileOps.OpenDropped(p)
			}
		})
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
	if dir, err := mygo.App.Path("userConfig"); err == nil {
		configDir = dir
	} else {
		configDir = filepath.Join(os.TempDir(), "corra-config")
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "config dir: %v\n", err)
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
		watcher = app.NewWatcher(editorApp, path)
	}
}

// exportPDF renders the current document to PDF bytes.
func exportPDF(title string) ([]byte, error) {
	doc := editorApp.CurrentDoc()
	if doc == nil || len(doc.Blocks) == 0 {
		doc = render.Parse(editorApp.CurrentSource())
	}
	return render.ExportPDF(context.Background(), doc, title)
}

// insertImageAndUpload asks for an image file, uploads it when hosting is
// configured, and inserts the markdown reference at the end.
func insertImageAndUpload(win *mygo.Window) {
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
	editorApp.AppendBlock(fmt.Sprintf("![%s](%s)", alt, url))
	watchCurrent()
}
