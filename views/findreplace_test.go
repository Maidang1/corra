package views

import (
	"testing"

	"github.com/Maidang1/corra/app"
)

// FindReplaceAction routes to the native find bar's replace mode in rich
// mode, and to the source editor's find-replace bar in source mode.
func TestFindReplaceActionRouting(t *testing.T) {
	// Rich mode: the native bar takes over.
	v, _ := notesFixture(t)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	v.FindReplaceAction()
	if !e.replaceBar {
		t.Fatal("rich mode did not open the replace bar")
	}
	v.FindAction()
	if !e.findBar {
		t.Fatal("rich mode did not open the find bar")
	}
	v.Close()

	// Source mode: the editor's own bar opens with the replace row showing.
	v2, tt := notesFixture(t)
	v2.App.Mode = app.ModeEdit
	tt.Frame()
	v2.FindReplaceAction()
	tt.Frame()
	if !tt.HasText("Replace All") {
		t.Fatalf("source mode replace bar missing: %v", tt.Texts())
	}
	v2.Close()
}
