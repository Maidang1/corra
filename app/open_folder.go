package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Maidang1/corra/render"
)

// OpenFolderPath starts scanning without waiting on the UI thread.
func (f *FileOps) OpenFolderPath(path string) {
	f.startFolderLoad(path, f.Dispatch, readFolderLibrary)
}

func (f *FileOps) CancelFolderLoad() {
	f.folderLoadID++
	if f.cancelFolderLoad != nil {
		f.cancelFolderLoad()
		f.cancelFolderLoad = nil
	}
	f.App.LibraryLoading = false
	f.App.LibraryLoadError = ""
	f.App.InvalidateUI()
}

func (f *FileOps) startFolderLoad(path string, dispatch func(func()), scan func(context.Context, string) (*Library, error)) {
	f.CancelFolderLoad()
	f.App.LibraryLoadPath = path
	if f.BeforeOpen != nil {
		if !f.BeforeOpen() {
			return
		}
	} else if f.App.Dirty {
		f.App.LibraryLoadError = "请先保存当前文档，再打开文件夹"
		f.App.InvalidateUI()
		return
	}
	previousPath, previousSource := f.App.CurrentPath(), f.App.CurrentSource()
	id := f.folderLoadID
	ctx, cancel := context.WithCancel(context.Background())
	f.cancelFolderLoad = cancel
	f.App.LibraryLoading = true
	f.App.InvalidateUI()
	smart := f.App.SmartPunct
	go func() {
		lib, err := scan(ctx, path)
		var doc *render.Document
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			source := ""
			if len(lib.Notes) > 0 {
				source = lib.Notes[0].Source
			}
			doc = render.ParseOptions(source, smart)
		}
		dispatch(func() {
			if id != f.folderLoadID {
				return
			}
			cancel()
			f.cancelFolderLoad = nil
			f.App.LibraryLoading = false
			if err != nil {
				f.App.LibraryLoadError = err.Error()
				f.App.InvalidateUI()
				return
			}
			if f.App.Dirty || f.App.CurrentPath() != previousPath || f.App.CurrentSource() != previousSource || f.App.SmartPunct != smart {
				f.startFolderLoad(path, dispatch, scan)
				return
			}
			f.adoptLibrary(lib, doc)
		})
	}()
}

func readFolderLibrary(ctx context.Context, path string) (*Library, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("请输入文件夹路径")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return scanLibraryContext(ctx, root)
}

func (f *FileOps) adoptLibrary(lib *Library, doc *render.Document) {
	f.App.mu.Lock()
	f.App.LibraryRoot, f.App.Lib = lib.Root, lib
	f.App.SelFolder, f.App.SelNote, f.App.Query = "", "", ""
	f.App.mu.Unlock()
	path, source := "", ""
	if len(lib.Notes) > 0 {
		path, source = lib.Notes[0].Path, lib.Notes[0].Source
	}
	f.App.loadParsedDocument(path, source, doc)
	f.App.mu.Lock()
	f.App.SelNote = path
	f.App.mu.Unlock()
	if f.OnOpened != nil {
		f.OnOpened(path)
	}
	if f.OnLibraryOpened != nil {
		f.OnLibraryOpened(lib.Root)
	}
}
