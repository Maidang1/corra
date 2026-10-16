//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

type nativeFootnoteState struct {
	References []struct {
		Label, Number, Tooltip string
		Location, Length       int
		Attachment             bool
	}
	Definitions []struct {
		Label            string
		Location, Length int
	}
}

func readNativeFootnotes(t *testing.T, e *Editor) nativeFootnoteState {
	t.Helper()
	var state nativeFootnoteState
	if err := json.Unmarshal([]byte(nativeTestFootnotes(e)), &state); err != nil {
		t.Error(err)
	}
	return state
}

func TestNativeFootnotesRenderHoverAndRoundTrip(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		sourceRequests := 0
		e := New(window, nil, func() { sourceRequests++ })
		defer e.Close()
		source := "Text[^β] and **bold[^a]** repeated[^β].\r\n\r\n[^a]: Alpha **note**.\r\n\r\n[^β]: 中文 note\r\n\r\n    More *detail*.\r\n\r\nafter"
		e.Load(source)
		state := readNativeFootnotes(t, e)
		if len(state.References) != 3 || len(state.Definitions) != 2 {
			t.Errorf("missing native footnotes: %+v", state)
			return
		}
		for i, want := range []string{"1", "2", "1"} {
			ref := state.References[i]
			if ref.Number != want || ref.Length != 1 || !ref.Attachment {
				t.Errorf("reference %d not a numbered attachment: %+v", i, ref)
			}
		}
		if !strings.Contains(state.References[0].Tooltip, "中文 note") || !strings.Contains(state.References[0].Tooltip, "More detail") || !strings.Contains(state.References[1].Tooltip, "Alpha note") {
			t.Errorf("hover help missing definition contents: %+v", state.References)
		}
		if got, dirty, err := e.Snapshot(); got != source || dirty || err != nil {
			t.Errorf("load changed source: %q dirty=%v err=%v", got, dirty, err)
		}
		if strings.Contains(nativeTestText(e), "[^β]:") || !nativeTestHasTraits(e, "bold", "italic") {
			t.Errorf("definition did not render native inline styles: %q", nativeTestText(e))
		}
		if sourceRequests != 0 {
			t.Errorf("footnotes requested source mode")
		}
		if path := os.Getenv("CORRA_FOOTNOTE_SCREENSHOT"); path != "" {
			e.SetFrame(0, 0, 720, 500)
			if !nativeTestScreenshot(e, path) {
				t.Error("footnote screenshot failed")
			}
		}
	})
}

func TestNativeFootnoteDefinitionEditUpdatesHoverAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "Body[^note].\r\n\r\n[^note]: first **bold**\r\n\r\n    second\r\n\r\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 || len(state.References) != 1 {
			t.Errorf("missing definitions: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location, 5)
		nativeTestAction(e, "type", "新的内容")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "[^note]: 新的内容 **bold**") || !strings.Contains(got, "\r\n    second") || !strings.HasSuffix(got, "\r\n\r\nafter") {
			t.Errorf("definition edit lost structure: %q", got)
		}
		state = readNativeFootnotes(t, e)
		if len(state.References) != 1 || !strings.Contains(state.References[0].Tooltip, "新的内容 bold") {
			t.Errorf("stale hover help: %+v", state.References)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("definition undo: %q", got)
		}
		nativeTestAction(e, "redo", "")
		if state = readNativeFootnotes(t, e); len(state.References) != 1 || !strings.Contains(state.References[0].Tooltip, "新的内容") {
			t.Errorf("redo stale help: %+v", state)
		}
	})
}

func TestNativeFootnoteRevealRenameAndNavigation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^long] B[^other].\n\n[^long]: first\n\n[^other]: second"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.References) != 2 || len(state.Definitions) != 2 {
			t.Errorf("missing footnotes: %+v", state)
			return
		}
		nativeTestSelect(e, state.References[0].Location, 0)
		e.Format("footnote:definition")
		if location, length := nativeTestSelection(e); location != state.Definitions[0].Location || length != 0 {
			t.Errorf("definition navigation: %d,%d", location, length)
		}
		e.Format("footnote:reference")
		if location, _ := nativeTestSelection(e); location != state.References[0].Location {
			t.Errorf("reference navigation: %d", location)
		}
		nativeTestAction(e, "reveal", "")
		visible := nativeTestText(e)
		start := strings.Index(visible, "[^long]")
		if start < 0 || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("reference did not expand: %q", visible)
			return
		}
		nativeTestSelect(e, start+2, 4)
		nativeTestAction(e, "type", "other")
		nativeTestAction(e, "key", "cmd-return")
		state = readNativeFootnotes(t, e)
		if len(state.References) != 2 || state.References[0].Label != "other" || state.References[0].Number != "1" || state.References[1].Number != "1" {
			t.Errorf("renamed ref presentation: %+v", state)
		}
		if got, _, _ := e.Snapshot(); got != strings.Replace(original, "A[^long]", "A[^other]", 1) {
			t.Errorf("rename changed definitions: %q", got)
		}
	})
}

func TestNativeFootnoteDeletionRenumbersAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^a] B[^b].\n\n[^a]: first\n\n[^b]: second"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.References) != 2 {
			t.Errorf("missing footnotes: %+v", state)
			return
		}
		nativeTestSelect(e, state.References[0].Location, 1)
		nativeTestAction(e, "backspace", "")
		state = readNativeFootnotes(t, e)
		if len(state.References) != 1 || state.References[0].Label != "b" || state.References[0].Number != "1" {
			t.Errorf("stale numbering: %+v", state)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("reference undo lost source: %q", got)
		}
		state = readNativeFootnotes(t, e)
		if len(state.References) != 2 || state.References[1].Number != "2" {
			t.Errorf("undo stale numbering: %+v", state)
		}
	})
}

func TestNativeFootnoteInsertionIsOneUndoTransaction(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "before after"
		e.Load(original)
		nativeTestSelect(e, 7, 0)
		e.Format("footnote:insert")
		state := readNativeFootnotes(t, e)
		if len(state.References) != 1 || len(state.Definitions) != 1 {
			t.Errorf("insert missing reference or definition: %+v", state)
			return
		}
		if got, _, _ := e.Snapshot(); got != "before [^1]after\n\n[^1]: " {
			t.Errorf("atomic insertion source: %q", got)
		}
		if location, length := nativeTestSelection(e); location != state.Definitions[0].Location || length != 0 {
			t.Errorf("insertion caret not in definition: %d,%d %+v", location, length, state)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("insertion undo: %q", got)
		}
		nativeTestAction(e, "redo", "")
		state = readNativeFootnotes(t, e)
		if len(state.References) != 1 || len(state.Definitions) != 1 {
			t.Errorf("redo lost footnote: %+v", state)
		}
		e.Load("A[^1]\n\n[^2]: reserved")
		nativeTestSelect(e, 0, 0)
		e.Format("footnote:insert")
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "[^3]A[^1]") || !strings.HasSuffix(got, "[^3]: ") {
			t.Errorf("insertion reused an existing identifier: %q", got)
		}
	})
}

func TestNativeFootnoteNewlineIMEAndFormatting(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^a].\r\n\r\n[^a]: first\r\n\r\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		start := state.Definitions[0].Location
		nativeTestSelect(e, start+5, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "mark", "zhong")
		if _, _, err := e.Snapshot(); err == nil {
			t.Error("saved during marked composition")
		}
		nativeTestAction(e, "commitIME", "中")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "[^a]: first") || !strings.Contains(got, "    中") || !strings.HasSuffix(got, "\r\n\r\nafter") {
			t.Errorf("newline/IME escaped definition: %q", got)
		}
		nativeTestSelect(e, start, 5)
		e.Format("bold")
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "[^a]: **first**") {
			t.Errorf("formatting lost definition: %q", got)
		}
	})
}

func TestNativeFootnoteAdjacentReferencesAndTable(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^x][^x].\n\n| A | B |\n| - | - |\n| x[^other] | text |\n\n[^x]: first\n\n[^other]: second"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.References) != 3 {
			t.Errorf("adjacent/table refs lost: %+v", state)
			return
		}
		for i, want := range []string{"1", "1", "2"} {
			if state.References[i].Number != want || state.References[i].Length != 1 {
				t.Errorf("reference %d: %+v", i, state.References[i])
			}
		}
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("adjacent/table source changed: %q", got)
		}
		nativeTestSelect(e, state.References[2].Location, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestAction(e, "key", "cmd-return")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("table footnote reveal changed source: %q", got)
		}
	})
}

func TestNativeFootnoteContextNavigationAndNativeClick(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^a].\n\n[^a]: first"
		e.Load(original)
		e.SetFrame(0, 0, 720, 500)
		state := readNativeFootnotes(t, e)
		if len(state.References) != 1 || len(state.Definitions) != 1 {
			t.Errorf("missing footnotes: %+v", state)
			return
		}
		nativeTestSelect(e, state.References[0].Location, 0)
		nativeTestAction(e, "context", "footnote:definition")
		if location, _ := nativeTestSelection(e); location != state.Definitions[0].Location {
			t.Errorf("context did not navigate to definition: %d", location)
		}
		nativeTestAction(e, "context", "footnote:reference")
		if location, _ := nativeTestSelection(e); location != state.References[0].Location {
			t.Errorf("context did not navigate back: %d", location)
		}
		nativeTestAction(e, "click", "")
		if !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("native click did not expand footnote: %q", nativeTestText(e))
		}
		nativeTestAction(e, "key", "cmd-return")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("click changed source: %q", got)
		}
	})
}

func TestNativeFootnoteRichPasteStaysInDefinition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("one\n\ntwo")
		nativeTestSelect(e, 0, len(nativeTestText(e)))
		nativeTestAction(e, "copy", "")
		original := "A[^n].\r\n\r\n[^n]: prefix suffix\r\n\r\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location+7, 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, _ := e.Snapshot()
		doc := richtext.Parse(got)
		if len(doc.Paragraphs) != 4 || doc.Paragraphs[1].FootnoteID != "n" || doc.Paragraphs[2].FootnoteID != "n" || richtext.PlainText(doc.Paragraphs[1]) != "prefix one" || richtext.PlainText(doc.Paragraphs[2]) != "twosuffix" {
			t.Errorf("rich paste escaped definition: %q %+v", got, doc.Paragraphs)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("rich paste undo changed original: %q", got)
		}
	})
}

func TestNativeFootnoteFigureSourceRetainsContainer(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^n].\n\n[^n]: first\n\n    ![missing](missing.png)\n\nafter"
		e.Load(original)
		got, _, _ := e.Snapshot()
		if got != original {
			t.Errorf("source fallback escaped footnote: %q", got)
		}
		doc := richtext.Parse(got)
		if len(doc.Paragraphs) != 4 || doc.Paragraphs[2].FootnoteID != "n" {
			t.Errorf("figure fallback lost container: %+v", doc.Paragraphs)
		}
	})
}

func TestNativeFootnoteHTMLPasteStaysInDefinition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^n].\n\n[^n]: prefix suffix\n\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location+7, 0)
		nativeTestAction(e, "htmlPaste", "<p><strong>one</strong></p><p>two</p>")
		got, _, _ := e.Snapshot()
		doc := richtext.Parse(got)
		count := 0
		for i, p := range doc.Paragraphs {
			if i > 0 && i+1 < len(doc.Paragraphs) && p.FootnoteID != "n" {
				t.Errorf("HTML paragraph outside definition: %+v source=%q", p, got)
			}
			if p.FootnoteID == "n" {
				count++
			}
		}
		if count < 3 || !strings.Contains(got, "**one**") || !strings.HasSuffix(got, "\n\nafter") {
			t.Errorf("HTML paste lost content/container: %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("HTML paste undo: %q", got)
		}
	})
}

func TestNativeFootnoteFormulaSourceEditAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A[^n].\r\n\r\n[^n]: $$\r\n    x+1\r\n    $$\r\n\r\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location, 0)
		nativeTestAction(e, "reveal", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("formula reveal changed source: %q", got)
		}
		visible := nativeTestText(e)
		index := strings.Index(visible, "x+1")
		if index < 0 {
			t.Errorf("formula not editable: %q", visible)
			return
		}
		nativeTestSelect(e, len([]rune(visible[:index])), 1)
		nativeTestAction(e, "type", "y")
		nativeTestAction(e, "key", "cmd-return")
		got, _, _ := e.Snapshot()
		ps := richtext.Parse(got).Paragraphs
		if len(ps) != 3 || ps[1].Kind != "math" || ps[1].FootnoteID != "n" || !strings.Contains(ps[1].Raw, "y+1") || !strings.HasSuffix(got, "\r\n\r\nafter") {
			t.Errorf("formula source edit lost container or separators: %q %+v", got, ps)
		}
		nativeTestAction(e, "undo", "")
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("formula source undo changed source: %q", got)
		}
	})
}

func TestNativeFootnoteKeyboardNavigationAndTypingRules(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("A[^a].\n\n[^a]: first")
		state := readNativeFootnotes(t, e)
		if len(state.References) != 1 || len(state.Definitions) != 1 {
			t.Errorf("missing footnote: %+v", state)
			return
		}
		nativeTestSelect(e, state.References[0].Location, 0)
		nativeTestAction(e, "key", "footnote-down")
		if location, _ := nativeTestSelection(e); location != state.Definitions[0].Location {
			t.Errorf("keyboard definition navigation: %d", location)
		}
		nativeTestAction(e, "key", "footnote-up")
		if location, _ := nativeTestSelection(e); location != state.References[0].Location {
			t.Errorf("keyboard reference navigation: %d", location)
		}
		e.SetFrame(0, 0, 720, 500)
		nativeTestAction(e, "click", "")
		nativeTestAction(e, "key", "footnote-down")
		if location, _ := nativeTestSelection(e); location != state.Definitions[0].Location {
			t.Errorf("expanded reference keyboard navigation: %d", location)
		}
		e.Load("A\n\n[^a]: first")
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "type", "[^a]")
		if state = readNativeFootnotes(t, e); len(state.References) != 1 {
			t.Errorf("typed reference did not render: %+v", state)
		}
		e.Load("")
		nativeTestAction(e, "type", "[^note]: ")
		nativeTestAction(e, "type", "definition")
		if state = readNativeFootnotes(t, e); len(state.Definitions) != 1 || state.Definitions[0].Label != "note" {
			t.Errorf("typed definition did not become a container: %+v", state)
		}
		if got, _, _ := e.Snapshot(); got != "[^note]: definition" {
			t.Errorf("typed definition source: %q", got)
		}
	})
}

func TestNativeFootnoteBoundaryDeletionAndWholeRemoval(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "before\n\n[^n]: first\n\n    second\n\nafter"
		for _, action := range []struct {
			name             string
			location, length int
		}{{"backspace", 7, 0}, {"delete", 19, 0}, {"type", 8, 13}, {"mark", 8, 13}} {
			e.Load(original)
			nativeTestSelect(e, action.location, action.length)
			nativeTestAction(e, action.name, "X")
			if action.name == "mark" {
				nativeTestAction(e, "commitIME", "中")
			}
			if got, dirty, _ := e.Snapshot(); got != original || dirty {
				t.Errorf("%s swallowed definition boundary: %q", action.name, got)
			}
		}
		e.Load(original)
		nativeTestSelect(e, 7, 13)
		nativeTestAction(e, "backspace", "")
		if got, _, _ := e.Snapshot(); got != "before\n\nafter" {
			t.Errorf("whole definition removal: %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("whole definition undo: %q", got)
		}
	})
}

func TestNativeFootnoteEmptyReturnExitsDefinition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, empty := range []bool{false, true} {
			e.Load("A[^n].\n\n[^n]: first")
			if empty {
				e.Load("A[^n].\n\n[^n]: ")
			}
			state := readNativeFootnotes(t, e)
			if len(state.Definitions) != 1 {
				t.Errorf("missing definition: %+v", state)
				continue
			}
			nativeTestSelect(e, state.Definitions[0].Location+state.Definitions[0].Length, 0)
			nativeTestAction(e, "newline", "")
			if !empty {
				nativeTestAction(e, "newline", "")
			}
			nativeTestAction(e, "type", "outside")
			got, _, _ := e.Snapshot()
			doc := richtext.Parse(got)
			last := doc.Paragraphs[len(doc.Paragraphs)-1]
			if last.FootnoteID != "" || richtext.PlainText(last) != "outside" || !strings.Contains(got, "[^n]:") {
				t.Errorf("empty=%v Return stayed in definition: %q", empty, got)
			}
		}
	})
}

func TestNativeFootnoteBlockNewlineKeepsContainer(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("A[^n].\n\n[^n]: \n\nafter")
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location, 0)
		nativeTestAction(e, "type", "```go")
		nativeTestAction(e, "newline", "")
		got, _, _ := e.Snapshot()
		ps := richtext.Parse(got).Paragraphs
		if len(ps) != 3 || ps[1].Kind != "code" || ps[1].FootnoteID != "n" {
			t.Errorf("fence conversion escaped note: %q", got)
		}
		e.Load("A[^n].\n\n[^n]: # title")
		state = readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing heading definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location+5, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "body")
		got, _, _ = e.Snapshot()
		ps = richtext.Parse(got).Paragraphs
		if len(ps) != 3 || ps[2].Kind != "body" || ps[2].FootnoteID != "n" || richtext.PlainText(ps[2]) != "body" {
			t.Errorf("heading Return did not create footnote body: %q", got)
		}
		e.Load("A[^n].\n\n[^n]: ```go\n    one\n    \n    ```")
		nativeTestSelect(e, len([]rune(nativeTestText(e))), 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "after code")
		got, _, _ = e.Snapshot()
		ps = richtext.Parse(got).Paragraphs
		if len(ps) != 3 || ps[len(ps)-1].FootnoteID != "n" || richtext.PlainText(ps[len(ps)-1]) != "after code" {
			var state struct{ Document richtext.Document }
			json.Unmarshal([]byte(nativeTestFootnotes(e)), &state)
			for _, p := range state.Document.Paragraphs {
				t.Errorf("live paragraph kind=%s text=%q group=%q continuation=%v baseline=%q", p.Kind, richtext.PlainText(p), p.FootnoteGroup, p.FootnoteContinuation, p.FootnoteBaseline)
			}
			t.Errorf("code exit escaped footnote: %q native=%q", got, nativeTestText(e))
		}
	})
}

func TestNativeFootnotePasteDefinitionPreservesNestedSource(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("[^m]: one\n\n    two")
		nativeTestSelect(e, 0, len(nativeTestText(e)))
		nativeTestAction(e, "copy", "")
		original := "A[^n].\n\n[^n]: prefix suffix\n\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location+7, 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "[^m]: one") || !strings.Contains(got, "two") || !strings.HasSuffix(got, "\n\nafter") {
			t.Errorf("copied nested definition lost source: %q", got)
		}
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("nested paste undo: %q", got)
		}
	})
}

func TestNativeFootnoteFormulaWithoutBlankNeighbors(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "before\n[^n]: $$\n    x\n    $$\nafter"
		e.Load(original)
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) != 1 {
			t.Errorf("missing formula definition: %+v", state)
			return
		}
		nativeTestSelect(e, state.Definitions[0].Location, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestAction(e, "key", "cmd-return")
		if got, _, _ := e.Snapshot(); got != original {
			t.Errorf("formula fold swallowed neighbors: %q", got)
		}
		if !nativeTestHasTraits(e, "math") || strings.Contains(nativeTestText(e), "$$") {
			t.Errorf("adjacent formula did not refold: %q", nativeTestText(e))
		}
	})
}

func TestNativeFootnoteClipboardExportsMarkdown(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("A[^n].\n\n[^n]: first")
		state := readNativeFootnotes(t, e)
		if len(state.References) != 1 || len(state.Definitions) != 1 {
			t.Errorf("missing footnote: %+v", state)
			return
		}
		nativeTestSelect(e, state.References[0].Location, 1)
		nativeTestAction(e, "copy", "")
		if got := strings.TrimSpace(nativeTestClipboardText()); got != "[^n]" {
			t.Errorf("plain clipboard lost reference token: %q", got)
		}
		nativeTestSelect(e, state.Definitions[0].Location, state.Definitions[0].Length)
		nativeTestAction(e, "copy", "")
		if got := strings.TrimSpace(nativeTestClipboardText()); got != "[^n]: first" {
			t.Errorf("plain clipboard lost definition marker: %q", got)
		}
	})
}

func TestNativeFootnoteTableReplacementKeepsContainer(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("A[^n].\n\n[^n]: | h |\n    | - |\n    | b |\n\nafter")
		state := readNativeFootnotes(t, e)
		if len(state.Definitions) == 0 {
			t.Errorf("missing table definition: %+v", state)
			return
		}
		start := state.Definitions[0].Location
		nativeTestSelect(e, start, 4)
		nativeTestAction(e, "type", "replacement\n")
		got, _, _ := e.Snapshot()
		ps := richtext.Parse(got).Paragraphs
		if len(ps) != 3 || ps[1].FootnoteID != "n" || richtext.PlainText(ps[1]) != "replacement" || !strings.HasSuffix(got, "\n\nafter") {
			t.Errorf("whole table replacement escaped definition: %q", got)
		}
	})
}
