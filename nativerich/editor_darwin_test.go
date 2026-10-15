//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func init() { runtime.LockOSThread() }

func TestMain(m *testing.M) {
	go func() { nativeTestExit(m.Run()) }()
	os.Exit(nativeTestLoop())
}

func TestNativeEditor(t *testing.T) {
	nativeTestMain(func() {
		if New(0, nil, nil) != nil {
			t.Error("zero window must be unsupported")
		}
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		changes, sources := 0, 0
		e := New(window, func() { changes++ }, func() { sources++ })
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		original := "# Title\n\nA **bold** 中文😀 paragraph.\n\n- [x] task\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\n![image](photo.png)\n"
		e.Load(original)
		got, dirty, err := e.Snapshot()
		if err != nil || dirty || got != original {
			t.Errorf("load roundtrip: %q dirty=%v err=%v", got, dirty, err)
		}
		if changes != 0 {
			t.Error("Load notified change")
		}
		if !nativeTestHasTraits(e, "bold", "heading", "source") {
			t.Error("missing actual rich attributes")
		}
		nativeTestSelect(e, 0, 5)
		e.SetStyle(1.2, true, true)
		if a, b := nativeTestSelection(e); a != 0 || b != 5 {
			t.Error("style changed selection")
		}
		if changes != 0 {
			t.Error("style notified change")
		}
		e.Format("italic")
		got, dirty, err = e.Snapshot()
		if err != nil || !dirty || !strings.Contains(got, "# _Title_") {
			t.Errorf("format: %q %v %v", got, dirty, err)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("format undo: %q", got)
		}
		if !nativeTestClickCheckbox(e) {
			t.Error("no checkbox hit target")
		}
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "- [ ] task") {
			t.Errorf("checkbox click: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("checkbox undo: %q", got)
		}
		// Source lines (the image below) edit in place, no mode jump. In the
		// text storage paragraphs join with one newline and the table is four
		// cell lines, so the image line starts at 54.
		nativeTestSelect(e, 44, 0) // inside "![image](photo.png)"
		nativeTestAction(e, "type", "X")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "![Ximage](photo.png)") || sources != 0 {
			t.Errorf("source line edit: %q source=%d", got, sources)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("source line undo: %q", got)
		}
		if path := os.Getenv("CORRA_NATIVE_SCREENSHOT"); path != "" {
			nativeTestSelect(e, 0, 0)
			e.SetStyle(1, true, true)
			if !nativeTestScreenshot(e, path) {
				t.Errorf("screenshot failed: %s", path)
			}
			lightPath := strings.TrimSuffix(path, filepath.Ext(path)) + "-light.png"
			e.SetStyle(1, false, true)
			if !nativeTestScreenshot(e, lightPath) {
				t.Errorf("screenshot failed: %s", lightPath)
			}
		}
	})
}

func TestNativeTypingListsAndComposition(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		changes := 0
		e := New(window, func() { changes++ }, nil)
		defer e.Close()
		e.Load("- [x] first")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "second")
		got, _, err := e.Snapshot()
		if err != nil || !strings.Contains(got, "- [ ] second") {
			t.Errorf("checklist return: %q %v", got, err)
		}
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "body")
		got, _, _ = e.Snapshot()
		if strings.Contains(got, "- [ ] body") || !strings.Contains(got, "body") {
			t.Errorf("empty checklist exits: %q", got)
		}
		e.Load("alpha")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "paste", "\nbeta\ngamma")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "beta") || !strings.Contains(got, "gamma") {
			t.Errorf("paste lines: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != "alpha" {
			t.Errorf("paste undo: %q", got)
		}
		e.Load("hello")
		nativeTestSelect(e, 5, 0)
		before := changes
		nativeTestAction(e, "mark", "中")
		if _, dirty, err := e.Snapshot(); err == nil || !dirty {
			t.Errorf("marked text must stay dirty with error: %v %v", dirty, err)
		}
		if changes <= before {
			t.Error("composition must notify unsaved status")
		}
		nativeTestAction(e, "commit", "中文😀")
		got, dirty, err := e.Snapshot()
		if err != nil || !dirty || !strings.Contains(got, "hello中文😀") {
			t.Errorf("IME commit: %q %v %v", got, dirty, err)
		}
		e.Load("")
		nativeTestAction(e, "type", "blank is editable")
		got, _, _ = e.Snapshot()
		if got != "blank is editable" {
			t.Errorf("blank edit: %q", got)
		}
		e.InsertMarkdown("\n\n![image](photo.png)\n")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "![image](photo.png)") || !nativeTestHasTraits(e, "source") {
			t.Errorf("image insertion: %q", got)
		}
	})
}

func TestNativeBoundaryAndCallbackSafety(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		var e *Editor
		var notified string
		e = New(window, func() { notified, _, _ = e.Snapshot() }, nil)
		defer e.Close()
		e.Load("- [x] first")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "newline", "")
		if !strings.Contains(notified, "- [ ] ") {
			t.Errorf("callback observed intermediate newline: %q", notified)
		}
		nativeTestAction(e, "type", "second")
		if !strings.Contains(notified, "- [ ] second") {
			t.Errorf("list typing changed metadata: %q", notified)
		}
		e.Load("body\n\n# Heading")
		nativeTestAction(e, "type", "new ")
		if strings.HasPrefix(notified, "#") || !strings.HasPrefix(notified, "new body") {
			t.Errorf("load caret inherited last paragraph: %q", notified)
		}
		e.Load("![image](photo.png)")
		nativeTestSelect(e, 0, 19)
		nativeTestAction(e, "type", "replacement")
		if notified != "replacement" {
			t.Errorf("whole source replacement lost text: %q", notified)
		}
		nativeTestAction(e, "undo", "")
		if notified != "![image](photo.png)" {
			t.Errorf("source replacement undo: %q", notified)
		}
		e.Load("alpha")
		nativeTestSelect(e, 5, 0)
		e.InsertMarkdown("![new](new.png)")
		if !strings.Contains(notified, "alpha") || !strings.Contains(notified, "![new](new.png)") {
			t.Errorf("insert callback lost protected content: %q", notified)
		}
		nativeTestAction(e, "type", "tail")
		if !strings.Contains(notified, "![new](new.png)") || !strings.Contains(notified, "tail") {
			t.Errorf("trailing paragraph lost content: %q", notified)
		}
	})
}

func TestNativeClipboardAndProtectedDeletion(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		sources := 0
		e := New(window, nil, func() { sources++ })
		defer e.Close()
		e.Load("**styled** text")
		nativeTestSelect(e, 0, 6)
		nativeTestAction(e, "copy", "")
		nativeTestSelect(e, 11, 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, _ := e.Snapshot()
		if !strings.HasSuffix(got, "**styled**") {
			t.Errorf("rich copy/paste lost bold: %q", got)
		}
		e.Load("![image](photo.png)")
		nativeTestSelect(e, 0, 19)
		nativeTestAction(e, "copy", "")
		e.Load("")
		nativeTestAction(e, "pasteboard", "")
		got, _, _ = e.Snapshot()
		if got != "![image](photo.png)" {
			t.Errorf("source clipboard lost source: %q", got)
		}
		e.Load("before after")
		nativeTestSelect(e, 7, 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "![image](photo.png)") || !strings.Contains(got, "after") {
			t.Errorf("paste protected block mid-line: %q", got)
		}
		e.Load("![image](photo.png)")
		nativeTestSelect(e, 0, 19)
		nativeTestAction(e, "type", "")
		got, _, _ = e.Snapshot()
		if got != "" {
			t.Errorf("whole source deletion: %q", got)
		}
		e.Load("body")
		nativeTestAction(e, "unsupportedPaste", "")
		got, _, _ = e.Snapshot()
		if got != "body" || sources == 0 {
			t.Errorf("unsupported clipboard silently flattened: %q source=%d", got, sources)
		}
		before := sources
		nativeTestAction(e, "underlinePaste", "")
		got, _, _ = e.Snapshot()
		if got != "body" || sources == before {
			t.Errorf("unsupported underline paste silently flattened: %q", got)
		}
	})
}

func TestNativeCodeBlockTyping(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("")
		// ```go + Enter opens an editable code block, fences hidden.
		nativeTestAction(e, "type", "```go")
		nativeTestAction(e, "newline", "")
		got, _, _ := e.Snapshot()
		if got != "```go\n```" {
			t.Errorf("fence enter: %q", got)
		}
		nativeTestAction(e, "type", "x := 1")
		got, _, _ = e.Snapshot()
		if got != "```go\nx := 1\n```" {
			t.Errorf("code typing: %q", got)
		}
		// Enter continues the block; Enter on the empty last line leaves it.
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "after")
		got, _, _ = e.Snapshot()
		if got != "```go\nx := 1\n```\n\nafter" {
			t.Errorf("code exit: %q", got)
		}
	})
}

func TestNativeDraftGroupRerenders(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		// Typing $$ lines as plain text, then moving the caret away, renders
		// the math block.
		e.Load("before")
		nativeTestSelect(e, 6, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "$$")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "x^2")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "$$")
		nativeTestSelect(e, 0, 0)
		if !nativeTestHasTraits(e, "math") {
			got, _, _ := e.Snapshot()
			t.Errorf("leaving the group did not re-render the math block: %q", got)
		}
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "$$\nx^2\n$$") || !strings.Contains(got, "before") {
			t.Errorf("re-render changed the source: %q", got)
		}
		// Typing a fenced code block the same way renders it when the caret
		// leaves (the closing fence line was typed, caret sits in it).
		e.Load("")
		nativeTestAction(e, "type", "```")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "x := 1")
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "```")
		nativeTestSelect(e, 0, 0) // typing "```" opened a code block on Enter already; force caret out
		nativeTestSelect(e, 0, 0)
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "```\nx := 1\n```") {
			t.Errorf("typed code block: %q", got)
		}
		// A group that does not parse as a renderable block stays plain text.
		e.Load("plain text")
		nativeTestSelect(e, 0, 0)
		nativeTestSelect(e, 5, 0)
		got, _, _ = e.Snapshot()
		if got != "plain text" {
			t.Errorf("plain text rewritten: %q", got)
		}
	})
}

func TestNativeEditingEdges(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		// Deleting the blank line before a source line merges the paragraphs;
		// the source text itself must survive the round trip.
		e.Load("alpha\n\n![image](photo.png)")
		nativeTestSelect(e, 5, 1)
		nativeTestAction(e, "type", "")
		got, _, _ := e.Snapshot()
		again := richtext.Parse(got)
		if !strings.Contains(richtext.PlainText(again.Paragraphs[len(again.Paragraphs)-1]), "![image](photo.png)") {
			t.Errorf("paragraph merge destroyed source text: %q", got)
		}
		e.Load("![image](photo.png)\n\ntail")
		nativeTestSelect(e, 19, 1) // the source line's own newline
		nativeTestAction(e, "type", "")
		got, _, _ = e.Snapshot()
		if got != "![image](photo.png)tail\n\n" {
			t.Errorf("merge after source line: %q", got)
		}
		e.Load("![image](photo.png)")
		nativeTestSelect(e, 19, 0)
		nativeTestAction(e, "type", "tail")
		got, dirty, _ := e.Snapshot()
		if !dirty || got != "![image](photo.png)tail" {
			t.Errorf("source-end typing: %q dirty=%v", got, dirty)
		}
		e.Load("abc")
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "X")
		got, _, _ = e.Snapshot()
		if got != "a\n\nXbc" {
			t.Errorf("middle split: %q", got)
		}
		e.Load("body")
		nativeTestSelect(e, 4, 0)
		nativeTestAction(e, "newline", "")
		before, _, _ := e.Snapshot()
		nativeTestSelect(e, 0, 0)
		got, _, _ = e.Snapshot()
		if got != before {
			t.Errorf("selection changed trailing blank semantics: before=%q after=%q", before, got)
		}
		e.Load("")
		e.Format("heading1")
		nativeTestAction(e, "undo", "")
		nativeTestAction(e, "type", "regular")
		got, _, _ = e.Snapshot()
		if got != "regular" {
			t.Errorf("empty paragraph formatting undo: %q", got)
		}
		e.Load("")
		e.Format("heading1")
		nativeTestAction(e, "type", "Title")
		got, _, _ = e.Snapshot()
		if got != "# Title" {
			t.Errorf("empty paragraph formatting: %q", got)
		}
		e.Load("a😀b")
		nativeTestSelect(e, 1, 2)
		e.Format("bold")
		got, _, _ = e.Snapshot()
		doc := richtext.Parse(got)
		if len(doc.Paragraphs) != 1 || richtext.PlainText(doc.Paragraphs[0]) != "a😀b" || len(doc.Paragraphs[0].Runs) < 2 || !doc.Paragraphs[0].Runs[1].Bold {
			t.Errorf("UTF16 emoji formatting: %q", got)
		}
		e.Load("body")
		e.SetStyle(1, false, false)
		e.Format("heading1")
		e.InsertMarkdown("invisible")
		got, _, _ = e.Snapshot()
		if got != "body" {
			t.Errorf("readonly changed: %q", got)
		}
	})
}

func TestNativeLayoutAndFocus(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		if !nativeTestLayout(e) {
			t.Error("native frame, layering, focus, or visibility contract failed")
		}
	})
}

func TestNativeClipboardParagraphSeparators(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("a\n\nb")
		nativeTestSelect(e, 1, 1)
		nativeTestAction(e, "copy", "")
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "pasteboard", "")
		got, _, _ := e.Snapshot()
		if got != "a\n\nb\n\n" {
			t.Errorf("newline-only copy lost separator: %q", got)
		}
		e.Load("a\n\nb")
		nativeTestSelect(e, 0, 2)
		nativeTestAction(e, "copy", "")
		e.Load("")
		nativeTestAction(e, "pasteboard", "")
		got, _, _ = e.Snapshot()
		if got != "a\n\n" {
			t.Errorf("trailing copied separator lost: %q", got)
		}
		e.Load("a\n\nb")
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "newline", "")
		nativeTestSelect(e, 1, 2)
		nativeTestAction(e, "copy", "")
		e.Load("")
		nativeTestAction(e, "pasteboard", "")
		a, b := nativeTestSelection(e)
		if a != 2 || b != 0 {
			t.Errorf("two copied newlines not preserved: selection=%d,%d", a, b)
		}
	})
}

func TestNativeInvalidUTF8SourcePreservation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "before\n\n\xff\n\nafter"
		e.Load(original)
		got, changed, err := e.Snapshot()
		if err != nil || changed || got != original {
			t.Errorf("invalid UTF8 load changed bytes: got=%q changed=%v err=%v", got, changed, err)
		}
		nativeTestSelectOpaque(e, true)
		start, length := nativeTestSelection(e)
		nativeTestSelect(e, start+length+1, 5)
		nativeTestAction(e, "type", "edited")
		got, changed, err = e.Snapshot()
		if err != nil || !changed || got != "before\n\n\xff\n\nedited" {
			t.Errorf("editing adjacent text corrupted invalid UTF8: got=%q changed=%v err=%v", got, changed, err)
		}
	})
}
