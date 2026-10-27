package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestOpenFolderPathSubmitAndLoading(t *testing.T) {
	v, tt := notesFixture(t)
	var path string
	v.OnOpenFolderPath = func(p string) { path = p; v.App.LibraryLoading = true; v.App.LibraryLoadPath = p }
	click(t, tt, "打开文件夹…")
	tt.Frame()
	click(t, tt, "文件夹路径")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("/tmp/example-documents")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if path != "/tmp/example-documents" || !tt.HasText("正在加载文档库…") {
		t.Fatalf("path not submitted or loading state missing: %q", path)
	}
	cancelled := false
	v.OnCancelFolderLoad = func() { cancelled = true; v.App.LibraryLoading = false }
	click(t, tt, "取消加载")
	if !cancelled {
		t.Fatal("cancel did not stop folder load")
	}
}

func TestFolderLoadingAndFailureStayAboveNativeEditor(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	v.App.LibraryLoading = true
	tt.Frame()
	if e.visible || !tt.HasText("正在加载文档库…") {
		t.Fatal("native editor obscures loading state")
	}
	v.App.LibraryLoading = false
	v.App.LibraryLoadError = "目录不可读"
	tt.Frame()
	if e.visible || !tt.HasText("目录不可读") {
		t.Fatal("load failure is not visible")
	}
	v.OnCancelFolderLoad = func() { v.App.LibraryLoadError = "" }
	click(t, tt, "关闭")
	tt.Frame()
	if !e.visible {
		t.Fatal("previous document did not reappear after dismissing failure")
	}
}
