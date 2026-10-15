package views

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Maidang1/corra/app"
)

type fakeRichEditor struct {
	source                   string
	changed, visible, closed bool
	loads, focuses           int
	command                  string
	err                      error
	x, y, width, height      float64
}

func (e *fakeRichEditor) Load(s string) { e.source = s; e.changed = false; e.loads++ }
func (e *fakeRichEditor) Snapshot() (string, bool, error) {
	changed := e.changed
	e.changed = false
	return e.source, changed, e.err
}
func (e *fakeRichEditor) Format(s string)                            { e.command = s }
func (e *fakeRichEditor) InsertMarkdown(s string)                    { e.source += s; e.changed = true }
func (e *fakeRichEditor) SetFrame(x, y, w, h float64)                { e.x, e.y, e.width, e.height = x, y, w, h }
func (e *fakeRichEditor) SetVisible(v bool)                          { e.visible = v }
func (e *fakeRichEditor) SetStyle(zoom float64, dark, editable bool) {}
func (e *fakeRichEditor) SetSpellcheck(bool)                         {}
func (e *fakeRichEditor) SetFocusMode(bool)                          {}
func (e *fakeRichEditor) SetTypewriter(bool)                         {}
func (e *fakeRichEditor) ShowFindBar()                               {}
func (e *fakeRichEditor) RevealHeading(int)                          {}
func (e *fakeRichEditor) Print()                                     {}

func (e *fakeRichEditor) SetBaseDir(dir string)   {}
func (e *fakeRichEditor) SetMathNumbering(string) {}
func (e *fakeRichEditor) Focus()                  { e.focuses++ }
func (e *fakeRichEditor) Blur()                   {}
func (e *fakeRichEditor) Close()                  { e.closed = true }

func TestRichDefaultLayoutAndNoReloadOnSave(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	if v.App.Mode != app.ModeRich || !e.visible || e.width <= 0 || e.height <= 0 || e.x < listW {
		t.Fatalf("native editor not in laid-out pane: mode=%v editor=%+v", v.App.Mode, e)
	}
	loads := e.loads
	e.source = "# Native\n\n中文 😀\n"
	e.changed = true
	if !v.FlushAutosave() {
		t.Fatal(v.SaveError)
	}
	tt.Frame()
	got, _ := os.ReadFile(v.App.Path)
	if string(got) != e.source || e.loads != loads {
		t.Fatalf("save lost native text or reset selection: %q loads=%d", got, e.loads)
	}
}

func TestRichSourceSwitchSynchronizesBothDirections(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	e.source, e.changed = "# Rich\n", true
	v.toggleMode()
	tt.Frame()
	if v.App.Mode != app.ModeEdit || e.visible || v.edState.Buffer.String() != "# Rich\n" {
		t.Fatal("source mode missing latest native edits")
	}
	v.edState.Select(0, 0)
	v.edState.Insert("source ")
	v.toggleMode()
	tt.Frame()
	if v.App.Mode != app.ModeRich || !strings.HasPrefix(e.source, "source ") || !e.visible {
		t.Fatal("rich mode missing source edits")
	}
}

func TestRichActionsAndModalVisibility(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	v.formatSelection("**", "**")
	if e.command != "bold" || v.App.Mode != app.ModeRich {
		t.Fatal("formatting did not reach native selection")
	}
	v.insertSnippet("- [ ] ")
	if e.command != "checklist" {
		t.Fatal("checklist toolbar inserts raw markdown")
	}
	v.HostOpen = true
	tt.Frame()
	if e.visible {
		t.Fatal("native view obscures settings modal")
	}
	v.HostOpen = false
	tt.Frame()
	if !e.visible {
		t.Fatal("native view not restored after modal")
	}
	v.Close()
	if !e.closed {
		t.Fatal("native view leaked on close")
	}
}

func TestRichSnapshotFailureBlocksNavigationAndSave(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	e.err = errors.New("composition pending")
	v.NativeChanged()
	if !v.App.Dirty {
		t.Fatal("watcher can replace uncommitted native input")
	}
	if v.FlushAutosave() || v.SaveError == "" {
		t.Fatal("failed snapshot reported saved")
	}
	v.toggleMode()
	if v.App.Mode != app.ModeRich {
		t.Fatal("source mode replaced pending native input")
	}
	e.err = nil
	v.NativeChanged()
	if v.SaveError != "" || !v.FlushAutosave() {
		t.Fatal("committing or cancelling composition leaves autosave stuck")
	}
}

func TestRichNewNoteAndTrash(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	v.toggleMode()
	v.newNote()
	tt.Frame()
	if v.App.Mode != app.ModeRich {
		t.Fatal("new note not WYSIWYG")
	}
	v.App.SelFolder = app.TrashName
	tt.Frame()
	before := e.command
	v.formatSelection("**", "**")
	if e.command != before {
		t.Fatal("native command modified trashed note")
	}
}

func TestRichReadingPreviewRetainsRenderedDocument(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	e.source, e.changed = "# Rich preview\n\n| A | B |\n| - | - |\n| 1 | 2 |\n", true
	v.PreviewAction()
	tt.Frame()
	if e.visible || v.App.Mode != app.ModePreview || !tt.HasText("Rich preview") {
		t.Fatal("rich document cannot open full rendered reading preview")
	}
	v.RichAction()
	tt.Frame()
	if !e.visible || v.App.Mode != app.ModeRich {
		t.Fatal("cannot return to WYSIWYG from reading preview")
	}
	v.PreviewAction()
	v.formatSelection("**", "**")
	if v.App.Mode != app.ModeRich || e.command != "bold" {
		t.Fatal("formatting in reading view unexpectedly opens raw source")
	}
}

func TestRichConflictCannotSavePendingComposition(t *testing.T) {
	v, tt := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	e.err = errors.New("composition pending")
	v.App.Conflict = true
	called := false
	v.OnConflictResolve = func(bool) { called = true }
	v.saveAndReloadConflict(false)
	if called || !v.App.Conflict {
		t.Fatal("conflict overwrites disk with stale uncommitted text")
	}
}
