//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"
)

// linkCandidates is the fake library the completion panel queries.
func linkCandidates(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	type candidate struct {
		Title string `json:"title"`
		Path  string `json:"path"`
	}
	all := []candidate{{"Alpha Note", "alpha.md"}, {"Beta", "sub/beta.md"}}
	out := make([]candidate, 0, len(all))
	for _, c := range all {
		if q == "" || strings.Contains(strings.ToLower(c.Title), q) {
			out = append(out, c)
		}
	}
	data, _ := json.Marshal(out)
	return string(data)
}

func TestNativeLinkCompletion(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		e.OnLinkCandidates = linkCandidates

		// "[[" opens the note suggestions with every visible note.
		e.Load("")
		nativeTypeCharacters(e, "[[")
		open, filter, selected, items := nativeTestLink(e)
		if !open || filter != "" || selected != 0 || len(items) != 2 {
			t.Errorf("[[: open=%v filter=%q selected=%d items=%v", open, filter, selected, items)
			return
		}
		// Typing filters through the callback.
		nativeTypeCharacters(e, "be")
		open, filter, _, items = nativeTestLink(e)
		if !open || filter != "be" || len(items) != 1 || items[0] != "Beta" {
			t.Errorf("filter: open=%v filter=%q items=%v", open, filter, items)
			return
		}
		// Return inserts a Markdown link to the note's relative path.
		nativeTestAction(e, "key", "plain-return")
		if open, _, _, _ := nativeTestLink(e); open {
			t.Errorf("menu stayed open after Return")
			return
		}
		if got, _, _ := e.Snapshot(); got != "[Beta](sub/beta.md)" {
			t.Errorf("inserted %q", got)
			return
		}
		// Undo removes the link again.
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); strings.Contains(got, "]( ") || strings.Contains(got, "[Beta]") {
			t.Errorf("undo kept the link: %q", got)
			return
		}

		// Escape dismisses without inserting.
		e.Load("")
		nativeTypeCharacters(e, "[[")
		nativeTestAction(e, "key", "plain-escape")
		if open, _, _, _ := nativeTestLink(e); open {
			t.Errorf("escape kept the menu open")
			return
		}
		if got, _, _ := e.Snapshot(); got != "[[]]" && got != "[[" {
			t.Errorf("escape left %q", got)
			return
		}

		// Titles with spaces can be searched: the query allows spaces.
		e.Load("")
		nativeTypeCharacters(e, "[[alpha note")
		open, filter, _, items = nativeTestLink(e)
		if !open || filter != "alpha note" || len(items) != 1 || items[0] != "Alpha Note" {
			t.Errorf("spaced query: open=%v filter=%q items=%v", open, filter, items)
			return
		}
		nativeTestAction(e, "key", "plain-return")
		if got, _, _ := e.Snapshot(); got != "[Alpha Note](alpha.md)" {
			t.Errorf("spaced insert = %q", got)
			return
		}

		// Code blocks never complete links.
		e.Load("```\ncode\n```")
		nativeTypeCharacters(e, "[[")
		if open, _, _, _ := nativeTestLink(e); open {
			t.Errorf("link menu opened inside a code block")
			return
		}

		// Clicking a row inserts it too.
		e.Load("")
		nativeTypeCharacters(e, "[[")
		nativeTestAction(e, "link-pick", "1")
		if got, _, _ := e.Snapshot(); got != "[Beta](sub/beta.md)" {
			t.Errorf("click inserted %q", got)
		}
	})
}

func TestNativeHeadingCompletion(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		e.OnLinkCandidates = linkCandidates

		// "[[#query" searches the open note's headings.
		e.Load("# Intro\n\n## Setup Notes\n\nbody")
		nativeTestSelect(e, len(utf16.Encode([]rune(nativeTestText(e)))), 0)
		nativeTypeCharacters(e, "[[#set")
		open, filter, _, items := nativeTestLink(e)
		if !open || filter != "#set" || len(items) != 1 || items[0] != "Setup Notes" {
			t.Errorf("heading filter: open=%v filter=%q items=%v", open, filter, items)
			return
		}
		nativeTestAction(e, "key", "plain-return")
		if got, _, _ := e.Snapshot(); !strings.HasSuffix(got, "[Setup Notes](#setup-notes)") {
			t.Errorf("heading link = %q", got)
			return
		}

		// A bare "[[#" lists every heading.
		e.Load("# Intro\n\n## Setup Notes")
		nativeTestSelect(e, len(utf16.Encode([]rune(nativeTestText(e)))), 0)
		nativeTypeCharacters(e, "[[#")
		open, filter, _, items = nativeTestLink(e)
		if !open || filter != "#" || len(items) != 2 {
			t.Errorf("all headings: open=%v filter=%q items=%v", open, filter, items)
			return
		}

		// CJK headings keep their characters in the anchor.
		e.Load("## 中文标题\n\nbody")
		nativeTestSelect(e, len(utf16.Encode([]rune(nativeTestText(e)))), 0)
		nativeTypeCharacters(e, "[[#")
		nativeTestAction(e, "key", "plain-return")
		if got, _, _ := e.Snapshot(); !strings.Contains(got, "[中文标题](#中文标题)") {
			t.Errorf("CJK heading link = %q", got)
		}
	})
}
