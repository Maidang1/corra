package views

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

func notesFixture(t *testing.T) (*View, *ui.Tester) {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "note.md")
	if err := os.WriteFile(p, []byte("# Note\n\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	if err := a.OpenNote(p); err != nil {
		t.Fatal(err)
	}
	v := New(a, app.NewImageLoader(""))
	t.Cleanup(func() {
		if v.autosaveTimer != nil {
			v.autosaveTimer.Stop()
		}
	})
	tt := ui.NewTester(v.Build, 1280, 820)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 100})
	return v, tt
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

// waitTransition keeps frames going for d, so a wall-clock transition really
// finishes before the test pokes the controls it is moving.
func waitTransition(tt *ui.Tester, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(5 * time.Millisecond)
	}
	tt.Frame()
}

// waitFor keeps running frames until cond holds. The panel transitions run on
// the wall clock, so a frame alone cannot promise they finished under load.
func waitFor(t *testing.T, tt *ui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		tt.Frame()
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestSidebarButtonChangesVisibility(t *testing.T) {
	v, tt := notesFixture(t)
	before := v.App.Sidebar
	click(t, tt, "收起边栏")
	if v.App.Sidebar == before {
		t.Fatal("sidebar button is a no-op")
	}
}

// TestCollapsedSidebar verifies the macOS collapse pattern: the toggle
// hides folders and search (editor gets the full width), the header keeps
// its controls within bounds, and the header's leading button expands the
// sidebar again.
func TestCollapsedSidebar(t *testing.T) {
	v, tt := notesFixture(t)
	tt.SetSize(960, 560)
	click(t, tt, "收起边栏")
	if v.App.Sidebar {
		t.Fatal("sidebar did not collapse")
	}
	waitFor(t, tt, "collapse", func() bool { return !tt.HasText("全部文档") && !tt.HasText("搜索文档") })
	for _, label := range []string{"新建文档 (Cmd+N)", "文本格式", "插入表格", "共享与导出"} {
		r, ok := tt.Find(label)
		if !ok || r.X+r.W > 961 || r.W < 25 {
			t.Fatalf("collapsed toolbar %s out of bounds %+v", label, r)
		}
	}
	if dir := os.Getenv("CORRA_SCREENSHOTS"); dir != "" {
		f, err := os.Create(filepath.Join(dir, "collapsed.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// Let the collapse transition truly finish before clicking: its hit rect
	// is still moving while the exit animation runs.
	waitTransition(tt, 300*time.Millisecond)
	// The collapsed header hosts its own expand control, so the label tells
	// the two states apart even while the sidebar is still animating out.
	click(t, tt, "展开边栏")
	waitFor(t, tt, "expand", func() bool { return v.App.Sidebar && tt.HasText("全部文档") })
}

func TestFormatControlHasVisibleText(t *testing.T) {
	_, tt := notesFixture(t)
	if !tt.HasText("Aa") {
		t.Fatal("format control has no visible Aa")
	}
}

func TestFlushSynchronizesPendingBuffer(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.Mode = app.ModeEdit
	tt.Frame()
	v.edState.Select(0, 0)
	v.edState.Insert("pending ")
	v.flushAutosave()
	got, err := os.ReadFile(v.App.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "pending ") {
		t.Fatalf("unsynced buffer lost on flush: %q", got)
	}
}

func TestFlushDoesNotTouchCleanNote(t *testing.T) {
	v, _ := notesFixture(t)
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local)
	if err := os.Chtimes(v.App.Path, old, old); err != nil {
		t.Fatal(err)
	}
	v.flushAutosave()
	st, err := os.Stat(v.App.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ModTime().Equal(old) {
		t.Fatal("opening/exporting a clean note changes its date and ordering")
	}
}

func TestTrashCannotBeEditedByToolbar(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.SelFolder = app.TrashName
	before := v.App.Source
	tt.Frame()
	click(t, tt, "插入任务列表")
	if v.App.Source != before {
		t.Fatal("toolbar edits a deleted note")
	}
}

func TestToolbarMenusAndPersistence(t *testing.T) {
	v, tt := notesFixture(t)
	click(t, tt, "文本格式")
	if err := tt.ChooseMenuItem("粗体"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.App.Source, "**文字**") {
		t.Fatalf("format menu didn't edit: %q", v.App.Source)
	}
	click(t, tt, "插入表格")
	if !strings.Contains(v.App.Source, "| 列 1 |") {
		t.Fatal("table missing")
	}
	click(t, tt, "共享与导出")
	if err := tt.ChooseMenuItem("复制 Markdown"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != v.App.Source {
		t.Fatal("clipboard not current source")
	}
	if !v.FlushAutosave() {
		t.Fatal(v.SaveError)
	}
	got, _ := os.ReadFile(v.App.Path)
	if string(got) != v.App.Source {
		t.Fatal("disk not current source")
	}
}

func TestEmptyFolderClearsEditorAndTrashRestores(t *testing.T) {
	v, tt := notesFixture(t)
	path := v.App.Path
	if err := v.App.CreateFolder("Empty"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, "文件夹：Empty")
	if v.App.Path != "" || !tt.HasText("未选择文档") {
		t.Fatal("empty folder shows unrelated note")
	}
	click(t, tt, "文件夹：全部文档")
	if err := v.App.DeleteSelectedNote(); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, "文件夹：最近删除")
	if len(v.App.VisibleNotes()) != 1 || v.App.Path == "" {
		t.Fatal("trash note not loaded")
	}
	click(t, tt, "文档选项")
	if err := tt.ChooseMenuItem("恢复文档"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if v.App.SelFolder == app.TrashName {
		t.Fatal("restore didn't return to active notes")
	}
}

func TestSaveFailurePreventsNavigationAndResurrection(t *testing.T) {
	v, tt := notesFixture(t)
	if err := v.App.CreateFolder("Empty"); err != nil {
		t.Fatal(err)
	}
	path := v.App.Path
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	v.App.AppendBlock("unsaved")
	tt.Frame()
	click(t, tt, "文件夹：Empty")
	if v.App.Path != path || v.App.SelFolder != "" || !v.App.Dirty {
		t.Fatal("failed save navigated away")
	}
	if v.SaveError == "" {
		t.Fatal("save failure is silent")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("save resurrected deleted note")
	}
}

func TestNotesShellSizes(t *testing.T) {
	for _, width := range []int{960, 1280} {
		for _, dark := range []bool{false, true} {
			v, tt := notesFixture(t)
			v.App.SetSource("可视化能力加强\n\n- 评测流程支持可视化查看，可按需展示实验和分析阶段。\n- 增加实验数据行详情，支持查看输入、模型输出与评分。\n", 0)
			if !v.FlushAutosave() {
				t.Fatal(v.SaveError)
			}
			if dark {
				v.App.ThemeMode = app.ThemeDark
			}
			tt.SetSize(width, 820)
			tt.Frame() // let the panel animations settle on the new size
			compact := width-268 < 700
			bar := []string{"新建文档 (Cmd+N)", "共享与导出", "搜索文档"}
			if !compact {
				bar = append(bar, "文本格式", "插入表格")
			}
			for _, label := range bar {
				r, ok := tt.Find(label)
				if !ok || r.X < 0 || r.X+r.W > float32(width)+1 || r.W < 25 {
					t.Fatalf("%s outside %d: %+v", label, width, r)
				}
			}
			if compact {
				// The format group folds into 文档选项 when narrow.
				click(t, tt, "文档选项")
				menu := strings.Join(tt.Menu(), "|")
				for _, label := range []string{"文本格式", "插入表格"} {
					if !strings.Contains(menu, label) {
						t.Fatalf("%s missing from the compact menu at %d: %s", label, width, menu)
					}
				}
				tt.CloseMenu()
			}
			if dir := os.Getenv("CORRA_SCREENSHOTS"); dir != "" {
				name := "light"
				if dark {
					name = "dark"
				}
				f, err := os.Create(filepath.Join(dir, name+itoa(width)+".png"))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, tt.Image()); err != nil {
					t.Fatal(err)
				}
				f.Close()
			}
		}
	}
}

func TestTypingAutosavesOnUIFrame(t *testing.T) {
	v, tt := notesFixture(t)
	click(t, tt, "新建文档 (Cmd+N)")
	click(t, tt, "文档正文")
	tt.Type("keyboard edit")
	if !strings.Contains(v.App.Source, "keyboard edit") {
		t.Fatalf("typing did not reach source: %q", v.App.Source)
	}
	v.autosaveDue = time.Now().Add(-time.Second)
	tt.Frame()
	got, err := os.ReadFile(v.App.Path)
	if err != nil || string(got) != v.App.Source || v.App.Dirty {
		t.Fatalf("UI-frame autosave failed: %s / %v", got, err)
	}
}

func TestFolderDialogAndFullTextSearch(t *testing.T) {
	v, tt := notesFixture(t)
	click(t, tt, "新建文件夹")
	click(t, tt, "文件夹名称")
	tt.Type("Projects")
	click(t, tt, "确定")
	if _, err := os.Stat(filepath.Join(v.App.LibraryRoot, "Projects")); err != nil {
		t.Fatal(err)
	}
	v.App.AppendBlock(strings.Repeat("long body ", 30) + "needleXYZ")
	if !v.FlushAutosave() {
		t.Fatal(v.SaveError)
	}
	tt.Frame()
	click(t, tt, "搜索文档")
	tt.Type("needleXYZ")
	if len(v.App.VisibleNotes()) != 1 {
		t.Fatal("full body search returned nothing")
	}
}

func TestDeleteConfirmationAndRestoreClicks(t *testing.T) {
	v, tt := notesFixture(t)
	path := v.App.Path
	click(t, tt, "文档选项")
	if err := tt.ChooseMenuItem("删除文档…"); err != nil {
		t.Fatal(err)
	}
	click(t, tt, "取消")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("cancel deleted note")
	}
	click(t, tt, "文档选项")
	if err := tt.ChooseMenuItem("删除文档…"); err != nil {
		t.Fatal(err)
	}
	click(t, tt, "删除")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("delete did not move note")
	}
	click(t, tt, "文件夹：最近删除")
	click(t, tt, "文档选项")
	if err := tt.ChooseMenuItem("恢复文档"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestAttachmentAndExportCallbacks(t *testing.T) {
	v, tt := notesFixture(t)
	v.OnInsertImage = func() { v.InsertMarkdown("![image](image.png)") }
	exported := ""
	v.OnExportPDF = func() { exported = v.App.Source }
	tt.Frame()
	click(t, tt, "插入图片（可上传图床）")
	if !strings.Contains(v.App.Source, "![image]") {
		t.Fatal("attachment callback not invoked")
	}
	click(t, tt, "共享与导出")
	if err := tt.ChooseMenuItem("导出…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !v.ExportOpen {
		t.Fatal("share menu did not open the export panel")
	}
	click(t, tt, "导出 PDF 文档 (.pdf)")
	if exported != v.App.Source {
		t.Fatal("export did not receive current source")
	}
}

// TestCompactToolbar: a narrow window folds the formatting group into the
// trailing menu instead of letting the toolbar overflow the window.
func TestCompactToolbar(t *testing.T) {
	v, tt := notesFixture(t)
	v.AttachRichEditor(&fakeRichEditor{})
	tt.SetSize(860, 600)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("文本格式"); ok {
		t.Fatal("formatting group still in a narrow toolbar")
	}
	for _, label := range []string{"新建文档 (Cmd+N)", "共享与导出", "文档选项"} {
		r, ok := tt.Find(label)
		if !ok || r.X+r.W > 861 {
			t.Fatalf("narrow toolbar %s out of bounds %+v", label, r)
		}
	}
	tt.SetSize(1280, 800)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("文本格式"); !ok {
		t.Fatal("formatting group missing from a wide toolbar")
	}
}
