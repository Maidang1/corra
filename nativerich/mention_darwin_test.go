//go:build darwin && cgo

package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeMentionCompletion(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		e.OnLinkCandidates = linkCandidates

		// "@" opens the note picker in mention mode; typing filters it.
		e.Load("")
		nativeTypeCharacters(e, "see @be")
		open, filter, _, items := nativeTestLink(e)
		mention, card := nativeTestLinkMode(e)
		if !open || !mention || card || filter != "be" || len(items) != 1 {
			t.Errorf("@be: open=%v mention=%v card=%v filter=%q items=%v", open, mention, card, filter, items)
			return
		}
		nativeTestAction(e, "key", "plain-return")
		if got, _, _ := e.Snapshot(); got != "see [@Beta](sub/beta.md) " {
			t.Errorf("inserted %q", got)
			return
		}
		// Typing carries on after the chip, outside the link.
		nativeTypeCharacters(e, "ok")
		if got, _, _ := e.Snapshot(); got != "see [@Beta](sub/beta.md) ok" {
			t.Errorf("after typing: %q", got)
			return
		}
		// Backspace on the chip's last character removes the whole mention.
		e.Load("see [@Beta](sub/beta.md) ok")
		nativeTestSelect(e, len("see @Beta"), 0)
		nativeTestAction(e, "backspace", "")
		if got, _, _ := e.Snapshot(); got != "see  ok" {
			t.Errorf("backspace left %q", got)
		}
	})
}

func TestNativeMentionTriggers(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		e.OnLinkCandidates = linkCandidates
		for _, typed := range []string{"mail a@b", "@zzz", "@be ", "`@"} {
			e.Load("")
			nativeTypeCharacters(e, typed)
			if open, _, _, _ := nativeTestLink(e); open {
				t.Errorf("%q opened the note picker", typed)
			}
		}
		// A pasted-in email stays an ordinary link, not a mention chip.
		e.Load("[@Beta](https://example.com)")
		if got, _, _ := e.Snapshot(); got != "[@Beta](https://example.com)" {
			t.Errorf("web link changed: %q", got)
		}
	})
}

func TestNativeMentionOpensNote(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		var opened []string
		e.OnOpenDocument = func(target string) bool { opened = append(opened, target); return true }
		e.Load("see [@Beta](sub/beta.md) and [web](https://example.com)")
		nativeTestSelect(e, len("see @B"), 0)
		nativeTestAction(e, "click", "")
		if len(opened) != 1 || opened[0] != "sub/beta.md" {
			t.Errorf("click on the mention opened %v", opened)
		}
	})
}

func TestNativePageCard(t *testing.T) {
	nativeTestMain(func() {
		window, e := newSlashEditor(t)
		if e == nil {
			return
		}
		defer nativeTestCloseWindow(window)
		defer e.Close()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "sub", "beta.md"), []byte("# Beta 计划\n\n第一段正文。\n"), 0o644); err != nil {
			t.Error(err)
			return
		}
		e.SetBaseDir(dir)
		e.OnLinkCandidates = linkCandidates

		// The slash menu's 页面卡片 opens the picker in card mode.
		e.Load("intro\n\n")
		nativeTestSelect(e, len("intro\n"), 0)
		nativeTypeCharacters(e, "/kapian")
		nativeTestAction(e, "slash-pick", "0")
		if mention, card := nativeTestLinkMode(e); !mention || !card {
			t.Errorf("card picker: mention=%v card=%v", mention, card)
			return
		}
		nativeTypeCharacters(e, "be")
		nativeTestAction(e, "key", "plain-return")
		got, _, _ := e.Snapshot()
		if !strings.HasSuffix(got, "![Beta](sub/beta.md)") {
			t.Errorf("card saved as %q", got)
			return
		}
		cards := nativeTestPageCards(e)
		if len(cards) != 1 || !cards[0].Figure || cards[0].Title != "Beta 计划" || cards[0].Excerpt != "第一段正文。" {
			t.Errorf("cards = %+v", cards)
		}

		// A card for a missing note still loads, marked missing, and saves as is.
		e.Load("![Gone](gone.md)")
		if cards := nativeTestPageCards(e); len(cards) != 1 || !cards[0].Missing || !cards[0].Figure {
			t.Errorf("missing card = %+v", cards)
		}
		if got, _, _ := e.Snapshot(); got != "![Gone](gone.md)" {
			t.Errorf("missing card saved as %q", got)
		}

		// Clicking a card opens its note.
		var opened []string
		e.OnOpenDocument = func(target string) bool { opened = append(opened, target); return true }
		e.Load("![Beta](sub/beta.md)")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "click", "")
		if len(opened) != 1 || opened[0] != "sub/beta.md" {
			t.Errorf("click on the card opened %v", opened)
		}
	})
}
