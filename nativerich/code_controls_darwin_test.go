//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

type codeControlState struct {
	Enabled      bool    `json:"enabled"`
	Gutter       float64 `json:"gutter"`
	ActualIndent float64 `json:"actualIndent"`
	Blocks       []struct {
		Location, Length int
		Open, Language   string
	} `json:"blocks"`
	Lines []struct {
		Number, Location, Length int
		X, Y, Width, Height      float64
		GlyphY, GlyphHeight      float64
	} `json:"lines"`
}

func readCodeControls(t *testing.T, e *Editor) codeControlState {
	t.Helper()
	var state codeControlState
	if err := json.Unmarshal([]byte(nativeTestCodeControls(e)), &state); err != nil {
		t.Error(err)
	}
	return state
}

func TestNativeCodeLanguagePreservesSourceAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, original := range []string{
			"before\n\n  ~~~~~go title=demo\n  var x = 1\n  ~~~~~\n\nafter",
			"```go\nvar x = 1\n```\n```python\nprint(2)\n```",
			"```go\n```",
		} {
			e.Load(original)
			state := readCodeControls(t, e)
			if len(state.Blocks) == 0 {
				t.Errorf("no native code block: %q", original)
				continue
			}
			location := state.Blocks[0].Location
			nativeTestSelect(e, location, 0)
			e.Format("code-language:rust")
			got, changed, err := e.Snapshot()
			want := strings.Replace(original, "go", "rust", 1)
			if err != nil || !changed || got != want {
				t.Errorf("language: got %q want %q changed=%v err=%v", got, want, changed, err)
			}
			if a, b := nativeTestSelection(e); a != location || b != 0 {
				t.Errorf("selection changed: %d,%d", a, b)
			}
			if strings.Contains(nativeTestText(e), "```") || strings.Contains(nativeTestText(e), "~~~~~") {
				t.Errorf("language exposed fence")
			}
			nativeTestAction(e, "undo", "")
			undone, _, _ := e.Snapshot()
			if undone != original {
				t.Errorf("undo: %q != %q", undone, original)
			}
			nativeTestAction(e, "redo", "")
			redone, _, _ := e.Snapshot()
			if redone != want {
				t.Errorf("redo: %q", redone)
			}
		}
	})
}

func TestNativeCodeLanguageGuardsAndHighlight(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("```\nvar answer = 1\n```")
		nativeTestSelect(e, 0, 0)
		e.Format("code-language:go")
		if !nativeTestHasTraits(e, "hl:keyword") {
			t.Error("selected language did not update highlighting")
		}
		nativeTestAction(e, "type", "// ")
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "```go\n// var") {
			t.Errorf("typing inherited stale language metadata: %q", got)
		}
		e.Load("```go\nvar answer = 1\n```")
		nativeTestSelect(e, 0, 0)
		for _, invalid := range []string{"bad\n```", "two words", "`go"} {
			before, _, _ := e.Snapshot()
			e.Format("code-language:" + invalid)
			after, _, _ := e.Snapshot()
			if after != before {
				t.Errorf("invalid language mutated source: %q", after)
			}
		}
		nativeTestAction(e, "mark", "zhong")
		e.Format("code-language:python")
		nativeTestAction(e, "commitIME", "中")
		got, _, _ := e.Snapshot()
		if !strings.HasPrefix(got, "```go\n中") {
			t.Errorf("composition changed language or text: %q", got)
		}
		e.Load("normal prose")
		e.Format("code-language:go")
		got, dirty, _ := e.Snapshot()
		if got != "normal prose" || dirty {
			t.Errorf("prose modified: %q", got)
		}
	})
}

func TestNativeCodeLineNumbersUseLogicalLinesAndReset(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		changes := 0
		e := New(window, func() { changes++ }, nil)
		defer e.Close()
		e.SetFrame(0, 0, 320, 500)
		e.SetVisible(true)
		original := "```go\n" + strings.Repeat("longIdentifier ", 16) + "\n\nlast\n```\n```python\nprint(1)\n```"
		e.Load(original)
		nativeTestSelect(e, 0, 0)
		e.Format("code-line-numbers:1")
		state := readCodeControls(t, e)
		if !state.Enabled || state.Gutter <= 0 || state.ActualIndent != state.Gutter {
			t.Errorf("gutter not in actual text layout: %+v", state)
		}
		if len(state.Blocks) != 2 || len(state.Lines) != 4 {
			t.Errorf("adjacent block/logical line counts: %+v", state)
			return
		}
		for i, want := range []int{1, 2, 3, 1} {
			if state.Lines[i].Number != want {
				t.Errorf("line %d number=%d want=%d", i, state.Lines[i].Number, want)
			}
		}
		if state.Lines[1].Y-state.Lines[0].Y <= state.Lines[0].Height*2 {
			t.Error("long logical line did not wrap in actual layout")
		}
		for _, line := range state.Lines {
			if line.X < 0 || line.Width <= 0 || line.Height <= 0 {
				t.Errorf("invalid gutter rect: %+v", line)
			}
			if line.GlyphHeight > 0 && math.Abs((line.Y+line.Height/2)-(line.GlyphY+line.GlyphHeight/2)) > 1 {
				t.Errorf("number not aligned with first wrapped code glyph line: %+v", line)
			}
		}
		got, dirty, _ := e.Snapshot()
		if got != original || dirty || changes != 0 {
			t.Errorf("line numbers modified document: %q dirty=%v changes=%d", got, dirty, changes)
		}
		nativeTestSelect(e, state.Blocks[1].Location, 0)
		e.Format("code-language:ruby")
		changed, _, _ := e.Snapshot()
		if changed != strings.Replace(original, "python", "ruby", 1) {
			t.Errorf("second adjacent block language targeted wrong fence: %q", changed)
		}
		nativeTestAction(e, "undo", "")
		e.SetStyle(1.5, false, true)
		zoomed := readCodeControls(t, e)
		if zoomed.Gutter <= state.Gutter || zoomed.Lines[0].Height <= state.Lines[0].Height {
			t.Error("gutter does not follow zoom")
		}
		if path := os.Getenv("CORRA_CODE_SCREENSHOT"); path != "" {
			e.SetFrame(0, 0, 720, 500)
			e.SetStyle(1, false, true)
			e.Load("# Native code editing\n\nChoose a language without exposing fences.\n\n```go\npackage main\n\nfunc main() {\n\tprintln(\"Native macOS\")\n}\n```\n```python\ndef greet(name):\n    return f\"Hello {name}\"\n```\n\nLine numbers stay outside saved and copied text.")
			if !nativeTestScreenshot(e, path) {
				t.Error("screenshot failed")
			}
			e.Load("```go\nfmt.Println(\"First block\")\n```")
			if !nativeTestScreenshot(e, strings.TrimSuffix(path, ".png")+"-first.png") {
				t.Error("first-code screenshot failed")
			}
		}
		e.Format("code-line-numbers:0")
		off := readCodeControls(t, e)
		if off.Enabled || off.Gutter != 0 || len(off.Lines) != 0 || off.ActualIndent >= state.Gutter {
			t.Errorf("toggle off left gutter: %+v", off)
		}
	})
}

func TestNativeCodeLanguageContextMenu(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		e.Load("```go\nfirst\n```")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "context", "code-language:python")
		if got, _, _ := e.Snapshot(); got != "```python\nfirst\n```" {
			t.Errorf("native context language action not connected: %q", got)
		}
	})
}

func TestNativeCodeLanguageTargetsFinalEmptyBlock(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, source := range []string{"```go\na\n```\n```python\n```", "```go\n```\n```python\n```"} {
			e.Load(source)
			nativeTestSelect(e, len(nativeTestText(e)), 0)
			e.Format("code-language:rust")
			if got, _, _ := e.Snapshot(); got != strings.Replace(source, "python", "rust", 1) {
				t.Errorf("language command selected preceding fence instead of final empty block: %q", got)
			}
		}
	})
}

func TestNativeCodeLanguageContextTargetsClickedBlock(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 500)
		e.SetVisible(true)
		source := "prose\n\n```go\nfirst\n```\n```python\nsecond\n```"
		e.Load(source)
		index := strings.Index(nativeTestText(e), "second")
		nativeTestSelect(e, 0, index+2)
		nativeTestAction(e, "context-at", strconv.Itoa(index)+"|code-language:rust")
		if got, _, _ := e.Snapshot(); got != strings.Replace(source, "python", "rust", 1) {
			t.Errorf("context language applied to selection start instead of clicked code: %q", got)
		}
		if start, length := nativeTestSelection(e); start != 0 || length != index+2 {
			t.Errorf("context language changed the native selection: %d %d", start, length)
		}
	})
}

func TestNativeCodeNewlinePreservesEdgesAndUndo(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "~~~~go\r\nfirst\r\n~~~~\r\n\r\nafter"
		for _, tc := range []struct {
			location int
			body     string
		}{{0, "\r\nfirst"}, {2, "fi\r\nrst"}, {5, "first\r\n"}} {
			e.Load(original)
			nativeTestSelect(e, tc.location, 0)
			nativeTestAction(e, "newline", "")
			want := "~~~~go\r\n" + tc.body + "\r\n~~~~\r\n\r\nafter"
			got, _, _ := e.Snapshot()
			if got != want {
				t.Errorf("Enter at %d: %q want %q", tc.location, got, want)
			}
			nativeTestAction(e, "undo", "")
			undone, _, _ := e.Snapshot()
			if undone != original {
				t.Errorf("Enter undo at %d: %q", tc.location, undone)
			}
			nativeTestAction(e, "redo", "")
			redone, _, _ := e.Snapshot()
			if redone != want {
				t.Errorf("Enter redo at %d: %q", tc.location, redone)
			}
		}
		e.Load("~~~~go\nfirst\n\n~~~~\n\nafter")
		nativeTestSelect(e, 6, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "outside")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "first\n~~~~\n\noutside") || !strings.HasSuffix(got, "\n\nafter") {
			t.Errorf("empty last line failed to leave original fence: %q", got)
		}
		e.Load("```go\na\nb\nc\n```")
		nativeTestSelect(e, 1, 3)
		nativeTestAction(e, "newline", "")
		got, _, _ = e.Snapshot()
		if got != "```go\na\nc\n```" {
			t.Errorf("selection split: %q", got)
		}
		e.Load("```go\na\n```\n```python\nb\n```")
		nativeTestSelect(e, 0, 3)
		nativeTestAction(e, "newline", "")
		got, dirty, _ := e.Snapshot()
		if got != "```go\na\n```\n```python\nb\n```" || dirty {
			t.Errorf("cross-block selection flattened code: %q", got)
		}
	})
}

func TestNativeCodeNumbersFollowEditingAndPreserveFenceEndings(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "~~~~go\r\nfirst\r\n~~~~\r\n\r\nafter"
		e.Load(original)
		e.Format("code-line-numbers:1")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "newline", "")
		nativeTestAction(e, "type", "second")
		state := readCodeControls(t, e)
		if len(state.Blocks) != 1 || len(state.Lines) != 2 {
			t.Errorf("split stale closing metadata made a second block: %+v", state)
			return
		}
		for i, line := range state.Lines {
			if line.Number != i+1 {
				t.Errorf("edited logical line number: %+v", line)
			}
		}
		e.Format("code-language:rust")
		got, _, _ := e.Snapshot()
		if got != "~~~~rust\r\nfirst\r\nsecond\r\n~~~~\r\n\r\nafter" {
			t.Errorf("editing lost fence/CRLF/boundary: %q", got)
		}
	})
}

func TestNativeCodeReplacementKeepsFenceMetadata(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, fence := range []struct{ open, close, nl string }{
			{"  ~~~~~go title=sample", "  ~~~~~~~", "\n"},
			{"``````go", "````````", "\n"},
			{"~~~~go", "~~~~~~", "\r\n"},
		} {
			for _, action := range []struct{ name, text string }{{"type", "replacement"}, {"paste", "left\r\nright"}, {"backspace", ""}, {"delete", ""}} {
				body := "one" + fence.nl + "two" + fence.nl + "three"
				original := "before" + fence.nl + fence.nl + fence.open + fence.nl + body + fence.nl + fence.close + fence.nl + fence.nl + "after"
				e.Load(original)
				state := readCodeControls(t, e)
				if len(state.Blocks) != 1 {
					t.Errorf("missing block in replacement fixture: %+v", state)
					continue
				}
				start := state.Blocks[0].Location
				nativeTestSelect(e, start, len("one\ntwo\nthree"))
				nativeTestAction(e, action.name, action.text)
				replacement := strings.ReplaceAll(action.text, "\r\n", fence.nl)
				middle := replacement
				if middle != "" {
					middle += fence.nl
				}
				want := "before" + fence.nl + fence.nl + fence.open + fence.nl + middle + fence.close + fence.nl + fence.nl + "after"
				got, _, err := e.Snapshot()
				if got != want || err != nil {
					t.Errorf("%s %q: got %q want %q err=%v", action.name, fence.open, got, want, err)
				}
				nativeTestAction(e, "undo", "")
				undone, _, _ := e.Snapshot()
				if undone != original {
					t.Errorf("%s undo lost fence: %q", action.name, undone)
				}
				nativeTestAction(e, "redo", "")
				redone, _, _ := e.Snapshot()
				if redone != want {
					t.Errorf("%s redo lost fence: %q", action.name, redone)
				}
			}
		}
	})
}

func TestNativeCodeMultilinePasteAndLineJoin(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "~~~~go\r\none\r\ntwo\r\n~~~~~~\r\n\r\nafter"
		for _, action := range []struct {
			name, text    string
			start, length int
			body          string
		}{
			{"paste", "a\r\nb", 1, 0, "oa\r\nbne\r\ntwo"},
			{"backspace", "", 4, 0, "onetwo"},
			{"delete", "", 3, 0, "onetwo"},
			{"type", "X", 4, 3, "one\r\nX"},
			{"backspace", "", 4, 3, "one\r\n"},
		} {
			e.Load(original)
			nativeTestSelect(e, action.start, action.length)
			nativeTestAction(e, action.name, action.text)
			want := "~~~~go\r\n" + action.body + "\r\n~~~~~~\r\n\r\nafter"
			got, _, _ := e.Snapshot()
			if got != want {
				t.Errorf("%s selection%d,%d: got %q want %q", action.name, action.start, action.length, got, want)
			}
			nativeTestAction(e, "undo", "")
			undone, _, _ := e.Snapshot()
			if undone != original {
				t.Errorf("%s undo: %q", action.name, undone)
			}
		}
		for _, fixture := range []string{original, "before\r\n\r\n" + strings.Replace(original, "~~~~~~\r\n\r\nafter", "~~~~~~\r\n\r\n\r\n\r\nafter", 1)} {
			e.Load(fixture)
			state := readCodeControls(t, e)
			if len(state.Blocks) != 1 {
				t.Errorf("missing IME code block: %+v", state)
				continue
			}
			nativeTestSelect(e, state.Blocks[0].Location, len("one\ntwo"))
			nativeTestAction(e, "mark", "zhong")
			nativeTestAction(e, "commitIME", "中")
			want := strings.Replace(fixture, "one\r\ntwo", "中", 1)
			if got, _, _ := e.Snapshot(); got != want {
				t.Errorf("IME replacing code body lost fences/separators: %q want %q", got, want)
			}
			nativeTestAction(e, "undo", "")
			if got, _, _ := e.Snapshot(); got != fixture {
				t.Errorf("IME undo lost original code block: %q want %q", got, fixture)
			}
		}
	})
}

func TestNativeCodeReplacementGuardsBlockBoundaries(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "~~~~go\na\n~~~~~~\n~~~~python\nb\n~~~~~~"
		for _, action := range []string{"type", "paste", "backspace", "delete", "mark"} {
			e.Load(original)
			nativeTestSelect(e, 0, 3)
			nativeTestAction(e, action, "X")
			if action == "mark" {
				if text := nativeTestText(e); text != "a\nb" {
					t.Errorf("mark replaced protected body: %q", text)
				}
				nativeTestAction(e, "commitIME", "中")
			}
			if got, dirty, _ := e.Snapshot(); got != original || dirty {
				t.Errorf("%s flattened adjacent fences: %q", action, got)
			}
		}
		original = "before\n\n~~~~go\none\ntwo\n~~~~~~\n\nafter"
		for _, action := range []struct {
			name     string
			location int
		}{{"backspace", 7}, {"delete", 14}} {
			e.Load(original)
			nativeTestSelect(e, action.location, 0)
			nativeTestAction(e, action.name, "")
			if got, dirty, _ := e.Snapshot(); got != original || dirty {
				t.Errorf("%s crossed prose/code boundary: %q", action.name, got)
			}
		}
		for _, action := range []string{"type", "paste"} {
			e.Load("~~~~go\na\n~~~~~~")
			nativeTestSelect(e, 0, 1)
			nativeTestAction(e, action, "a")
			if start, length := nativeTestSelection(e); start != 1 || length != 0 {
				t.Errorf("%s identical replacement kept selection %d,%d", action, start, length)
			}
			if got, dirty, _ := e.Snapshot(); got != "~~~~go\na\n~~~~~~" || dirty {
				t.Errorf("%s identical replacement changed source: %q", action, got)
			}
		}
	})
}
