package editor

import (
	"strings"
	"testing"
	"time"
)

func TestToggleWrap(t *testing.T) {
	src := "hello world"
	e := ToggleWrap(src, "**", 0, 5)
	got := Apply(src, e)
	if got != "**hello** world" {
		t.Errorf("wrap = %q", got)
	}
	e2 := ToggleWrap(got, "**", 0, 9)
	if got2 := Apply(got, e2); got2 != "hello world" {
		t.Errorf("unwrap = %q", got2)
	}
}

func TestToggleHeading(t *testing.T) {
	src := "plain text"
	e := ToggleHeading(src, 0, 4)
	if got := Apply(src, e); got != "# plain text" {
		t.Errorf("h1 = %q", got)
	}
	e = ToggleHeading("# plain text", 0, 4)
	if got := Apply("# plain text", e); got != "## plain text" {
		t.Errorf("h2 = %q", got)
	}
	e = ToggleHeading("###### deep", 0, 4)
	if got := Apply("###### deep", e); got != "deep" {
		t.Errorf("cycle = %q", got)
	}
}

func TestToggleListMarker(t *testing.T) {
	src := "one\ntwo"
	e := ToggleListMarker(src, false, 0, len([]rune(src)))
	if got := Apply(src, e); got != "- one\n- two" {
		t.Errorf("list = %q", got)
	}
	e = ToggleListMarker("- one\n- two", false, 0, len([]rune("- one\n- two")))
	if got := Apply("- one\n- two", e); got != "one\ntwo" {
		t.Errorf("unlist = %q", got)
	}
}

func TestToggleQuote(t *testing.T) {
	src := "line"
	e := ToggleQuote(src, 0, 4)
	if got := Apply(src, e); got != "> line" {
		t.Errorf("quote = %q", got)
	}
}

func TestInsertHelpers(t *testing.T) {
	if got := Apply("x", InsertTable("x", 1)); !strings.Contains(got, "| 列1 |") {
		t.Errorf("table = %q", got)
	}
	if got := Apply("x", InsertCodeBlock("x", "go", 1)); !strings.Contains(got, "```go") {
		t.Errorf("code = %q", got)
	}
	if got := Apply("x", InsertLink("x", "https://a", 0, 1)); got != "[x](https://a)" {
		t.Errorf("link = %q", got)
	}
}

func TestUndo(t *testing.T) {
	var u Undo
	u.PushBase("v0")
	now := time.Now()
	u.PushAt("v1", 1, now)
	u.PushAt("v12", 2, now.Add(100*time.Millisecond)) // coalesced
	u.PushAt("v123", 3, now.Add(2*time.Second))
	if !u.CanUndo() {
		t.Fatal("should undo")
	}
	s, caret, ok := u.Undo("v123", 3)
	if !ok || s != "v12" || caret != 2 {
		t.Errorf("undo = %q %d %v", s, caret, ok)
	}
	s, _, ok = u.Redo("v12", 2)
	if !ok || s != "v123" {
		t.Errorf("redo = %q %v", s, ok)
	}
	u.Undo("v123", 3)
	u.PushAt("new", 0, now.Add(3*time.Second))
	if u.CanRedo() {
		t.Error("redo should be cleared")
	}
}

func TestFindReplace(t *testing.T) {
	src := "Hello hello HELLO"
	m := Find(src, "hello")
	if len(m) != 3 {
		t.Fatalf("matches = %d", len(m))
	}
	if got := Replace(src, m[1], "hi"); got != "Hello hi HELLO" {
		t.Errorf("replace = %q", got)
	}
	if got := ReplaceAll(src, "hello", "x"); got != "x x x" {
		t.Errorf("replaceAll = %q", got)
	}
}
