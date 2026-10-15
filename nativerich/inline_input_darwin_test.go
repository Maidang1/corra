//go:build darwin && cgo

package nativerich

import (
	"testing"

	"github.com/Maidang1/corra/richtext"
)

func TestNativeExtendedInlineInput(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, tc := range []struct{ source, text, trait string }{
			{"==marked==", "marked", "highlight"},
			{"^super^", "super", "sup"},
			{"~sub~", "sub", "sub"},
			{"<u>under</u>", "under", "underline"},
			{"<mark>marked</mark>", "marked", "highlight"},
			{"<sup>super</sup>", "super", "sup"},
			{"<sub>sub</sub>", "sub", "sub"},
		} {
			e.Load("")
			nativeTypeCharacters(e, tc.source)
			if !nativeTestHasTraits(e, tc.trait) {
				t.Errorf("%q did not format while typing", tc.source)
			}
			nativeTypeCharacters(e, " plain")
			got, _, err := e.Snapshot()
			p := richtext.Parse(got).Paragraphs[0]
			if err != nil || richtext.PlainText(p) != tc.text+" plain" || len(p.Runs) < 2 {
				t.Errorf("%q content: %q %v", tc.source, got, err)
				continue
			}
			last := p.Runs[len(p.Runs)-1]
			if last.Highlight || last.Underline || last.Sup || last.Sub {
				t.Errorf("%q format leaked into plain text: %+v", tc.source, last)
			}
			nativeTestAction(e, "undo", "")
			undone, _, _ := e.Snapshot()
			if undone != "" {
				t.Errorf("%q typing undo: %q", tc.source, undone)
			}
			nativeTestAction(e, "redo", "")
			redone, _, _ := e.Snapshot()
			if redone != got {
				t.Errorf("%q typing redo: %q", tc.source, redone)
			}
		}
	})
}

func TestNativeLinkTitlesAndResolvedReferences(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, original := range []string{
			"[site](https://example.com \"A title\")",
			"[site][ref]\n\n[ref]: https://example.com \"A title\"\n",
		} {
			e.Load(original)
			got, changed, err := e.Snapshot()
			if err != nil || changed || got != original {
				t.Errorf("link load lost raw source: %q %v", got, err)
			}
			nativeTestSelect(e, 1, 2)
			e.Format("bold")
			got, _, err = e.Snapshot()
			p := richtext.Parse(got).Paragraphs[0]
			if err != nil || richtext.PlainText(p) != "site" {
				t.Errorf("link edit not rich: %q %v", got, err)
				continue
			}
			for _, run := range p.Runs {
				if run.Link != "https://example.com" || run.LinkTitle != "A title" {
					t.Errorf("link metadata lost: %+v", run)
				}
			}
			nativeTestAction(e, "undo", "")
			got, _, _ = e.Snapshot()
			if got != original {
				t.Errorf("link undo: %q", got)
			}
		}
		e.Load("")
		nativeTypeCharacters(e, "[site](https://example.com \"A title\")")
		got, _, _ := e.Snapshot()
		p := richtext.Parse(got).Paragraphs[0]
		if richtext.PlainText(p) != "site" || p.Runs[0].LinkTitle != "A title" {
			t.Errorf("typed titled link stayed literal: %q", got)
		}
	})
}

func TestNativeOrderedTaskInputRule(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		nativeTypeCharacters(e, "3. [ ] task")
		got, _, _ := e.Snapshot()
		if got != "3. [ ] task" || !nativeTestHasTraits(e, "checklist") {
			t.Errorf("ordered task input: %q", got)
		}
	})
}

func TestNativeCodeSelectionIndentation(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "```go\none\n  two\n```"
		e.Load(original)
		nativeTestSelect(e, 0, 9)
		nativeTestAction(e, "tab", "")
		got, _, _ := e.Snapshot()
		if got != "```go\n\tone\n\t  two\n```" {
			t.Errorf("code indent destroyed selection: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("code indent undo: %q", got)
		}
		nativeTestAction(e, "redo", "")
		nativeTestAction(e, "backtab", "")
		got, _, _ = e.Snapshot()
		if got != original {
			t.Errorf("code outdent: %q", got)
		}
		e.Load("```go\n    alpha\n\tbeta\n```")
		nativeTestSelect(e, 0, 15)
		nativeTestAction(e, "key", "cmd-left")
		got, _, _ = e.Snapshot()
		if got != "```go\nalpha\nbeta\n```" {
			t.Errorf("code mixed whitespace outdent: %q", got)
		}
	})
}
