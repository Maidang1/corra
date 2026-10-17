//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func nativeTypeCharacters(editor *Editor, value string) {
	for _, char := range value {
		nativeTestAction(editor, "type", string(char))
	}
}

func TestNativeMarkdownInputRules(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		editor := New(window, nil, nil)
		defer editor.Close()
		for _, example := range []struct{ source, kind, text string }{
			{"## hello world", "heading", "hello world"},
			{"# 中文😀", "heading", "中文😀"},
			{"- item", "bullet", "item"},
			{"1. item", "ordered", "item"},
			{"- [ ] task", "checklist", "task"},
			{"> quote", "quote", "quote"},
			{"**bold**", "body", "bold"},
			{"*italic*", "body", "italic"},
			{"~~strike~~", "body", "strike"},
			{"`code`", "body", "code"},
		} {
			editor.Load("")
			nativeTypeCharacters(editor, example.source)
			got, _, err := editor.Snapshot()
			doc := richtext.Parse(got)
			if err != nil || len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != example.kind || richtext.PlainText(doc.Paragraphs[0]) != example.text {
				t.Errorf("typed %q: got %q (%v)", example.source, got, err)
			}
			start, length := nativeTestSelection(editor)
			if start != richtext.UTF16Len(example.text) || length != 0 {
				t.Errorf("caret after %q: %d,%d", example.source, start, length)
			}
		}
	})
}

func TestNativeEditableTable(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		sources := 0
		editor := New(window, nil, func() { sources++ })
		defer editor.Close()
		original := "| A | B |\n| --- | --- |\n| one | two |"
		editor.Load(original)
		if !nativeTestHasTraits(editor, "table") || nativeTestHasTraits(editor, "opaque") {
			t.Error("table is a protected placeholder")
			return
		}
		got, changed, err := editor.Snapshot()
		if err != nil || changed || got != original {
			t.Errorf("table load: %q %v %v", got, changed, err)
		}
		nativeTestSelect(editor, 4, 3)
		nativeTestAction(editor, "type", "edited")
		got, _, err = editor.Snapshot()
		if err != nil || sources != 0 || !strings.Contains(got, "edited") || !strings.Contains(got, "two") {
			t.Errorf("cell edit: %q sources=%d err=%v", got, sources, err)
		}
		nativeTestAction(editor, "undo", "")
		got, _, _ = editor.Snapshot()
		if got != original {
			t.Errorf("table undo: %q", got)
		}
		editor.Load("")
		editor.InsertMarkdown("| A | B |\n| --- | --- |\n| | |\n")
		start, _ := nativeTestSelection(editor)
		if start != 0 {
			t.Errorf("insert table should focus first cell, got %d", start)
		}
		nativeTestSelect(editor, 4, 0)
		nativeTestAction(editor, "type", "first")
		nativeTestAction(editor, "tab", "")
		nativeTestAction(editor, "type", "second")
		got, _, _ = editor.Snapshot()
		if !strings.Contains(got, "| first | second |") {
			t.Errorf("empty cell edit / Tab: %q", got)
		}
	})
}

func TestNativeInputRuleUndoCompositionAndLiterals(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		editor := New(window, nil, nil)
		defer editor.Close()
		for _, source := range []string{"**bold**", "*italic*", "~~strike~~", "`code`", "[site](https://example.com)"} {
			editor.Load("")
			nativeTypeCharacters(editor, source)
			formatted, _, _ := editor.Snapshot()
			nativeTestAction(editor, "undo", "")
			literal, _, _ := editor.Snapshot()
			if literal != "" {
				t.Errorf("undo did not restore the state before the typing group: %q -> %q", source, literal)
			}
			nativeTestAction(editor, "redo", "")
			got, _, _ := editor.Snapshot()
			if got != formatted {
				t.Errorf("redo: %q != %q", got, formatted)
			}
		}
		for _, source := range []string{`\**literal**`, `\# title`, `hello_world_`, "**unfinished", "## pasted"} {
			editor.Load("")
			if source == "## pasted" {
				nativeTestAction(editor, "plainPaste", source)
			} else {
				nativeTypeCharacters(editor, source)
			}
			got, _, _ := editor.Snapshot()
			// A typed escape keeps its character as text and drops the slash.
			want := strings.TrimPrefix(source, `\`)
			if richtext.PlainText(richtext.Parse(got).Paragraphs[0]) != want {
				t.Errorf("literal input changed: %q -> %q", source, got)
			}
		}
		editor.Load("")
		nativeTypeCharacters(editor, "## ")
		nativeTestAction(editor, "mark", "zhong")
		if _, _, err := editor.Snapshot(); err == nil {
			t.Error("active IME snapshot must be blocked")
		}
		nativeTestAction(editor, "commit", "中文😀")
		got, _, err := editor.Snapshot()
		if err != nil || got != "## 中文😀" {
			t.Errorf("IME heading: %q %v", got, err)
		}
		nativeTestAction(editor, "newline", "")
		nativeTypeCharacters(editor, "body")
		got, _, _ = editor.Snapshot()
		if !strings.Contains(got, "\n\nbody") {
			t.Errorf("heading Return: %q", got)
		}
		editor.Load("")
		nativeTypeCharacters(editor, "**bold** plain")
		got, _, _ = editor.Snapshot()
		doc := richtext.Parse(got)
		if len(doc.Paragraphs[0].Runs) != 2 || doc.Paragraphs[0].Runs[1].Bold {
			t.Errorf("bold leaked into following text: %q", got)
		}
	})
}

func TestNativeTableNavigationClipboardAndBoundaries(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		editor := New(window, nil, nil)
		defer editor.Close()
		original := "| A | B |\n| --- | --- |\n| one | two |"
		editor.Load(original)
		nativeTestSelect(editor, 8, 0)
		nativeTestAction(editor, "tab", "")
		nativeTypeCharacters(editor, "new row")
		nativeTestAction(editor, "tab", "")
		nativeTypeCharacters(editor, "last")
		got, _, _ := editor.Snapshot()
		if !strings.Contains(got, "| new row | last |") {
			t.Errorf("Tab at last cell did not add row: %q", got)
		}
		nativeTestAction(editor, "backtab", "")
		start, _ := nativeTestSelection(editor)
		if start != 19 { // after "new row", ready to keep typing
			t.Errorf("Shift+Tab target: %d", start)
		}
		editor.Load(got)
		reloaded, changed, err := editor.Snapshot()
		if err != nil || changed || reloaded != got {
			t.Errorf("table reload: %q %v %v", reloaded, changed, err)
		}
		editor.Load(original)
		nativeTestSelect(editor, 4, 3)
		editor.Format("bold")
		got, _, _ = editor.Snapshot()
		if !strings.Contains(got, "**one**") {
			t.Errorf("table inline format: %q", got)
		}
		nativeTestAction(editor, "undo", "")
		nativeTestSelect(editor, 1, 1)
		nativeTestAction(editor, "type", "")
		got, _, _ = editor.Snapshot()
		if got != original {
			t.Errorf("cell boundary deletion corrupted table: %q", got)
		}
		nativeTestSelect(editor, 4, 3)
		nativeTestAction(editor, "paste", "hello\nworld")
		got, _, _ = editor.Snapshot()
		if !strings.Contains(got, "| hello world | two |") {
			t.Errorf("multiline cell paste: %q", got)
		}
		editor.Load("| A | B |\n| --- | --- |\n| | |")
		nativeTestSelect(editor, 0, 6)
		nativeTestAction(editor, "copy", "")
		editor.Load("")
		nativeTestAction(editor, "pasteboard", "")
		got, _, _ = editor.Snapshot()
		if !strings.Contains(got, "| | |") || !nativeTestHasTraits(editor, "table") {
			t.Errorf("empty cells lost in table copy: %q", got)
		}
		editor.Load(original)
		nativeTestSelect(editor, 0, 12)
		nativeTestAction(editor, "type", "replacement")
		got, _, _ = editor.Snapshot()
		if got != "replacement" {
			t.Errorf("whole table replacement: %q", got)
		}
		nativeTestAction(editor, "undo", "")
		got, _, _ = editor.Snapshot()
		if got != original {
			t.Errorf("whole table replacement undo: %q", got)
		}
		editor.Load("before\n\n" + original + "\n\nafter")
		nativeTestSelect(editor, 6, 1)
		nativeTestAction(editor, "type", "")
		got, _, _ = editor.Snapshot()
		if got != "before\n\n"+original+"\n\nafter" {
			t.Errorf("merge before table: %q", got)
		}
		if path := os.Getenv("CORRA_TABLE_SCREENSHOT"); path != "" {
			editor.SetFrame(0, 0, 720, 500)
			editor.SetVisible(true)
			editor.Load("# Editable table\n\n| Name | Status | Progress |\n| :--- | :---: | ---: |\n| Design | **Ready** | 100% |\n| Native editor | In review | 90% |\n| | | |\n\nClick a cell to edit. Tab moves to the next cell.")
			editor.SetStyle(1, false, true)
			if !nativeTestScreenshot(editor, path) {
				t.Error("table screenshot failed")
			}
		}
	})
}

func TestNativeTableHeaderEditDoesNotLoseAdjacentTables(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		editor := New(window, nil, nil)
		defer editor.Close()
		for _, source := range []string{
			"| A | B |\n|---|---|",
			"| A | B |\n|---|---|\n| | |\n\n| C | D |\n|---|---|\n| | |",
			"\n| A | B |\n|---|---|\n| | |\n\n\n",
		} {
			editor.Load(source)
			got, _, err := editor.Snapshot()
			if err != nil || got != source {
				t.Errorf("table edge roundtrip: %q -> %q %v", source, got, err)
			}
			data, _ := json.Marshal(richtext.Parse(got))
			if strings.Contains(string(data), `"kind":"opaque"`) {
				t.Errorf("table became opaque: %s", data)
			}
		}
	})
}
