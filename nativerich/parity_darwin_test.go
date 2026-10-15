//go:build darwin && cgo

package nativerich

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Maidang1/corra/internal/mdext"
	"github.com/Maidang1/corra/richtext"
)

// Typora-parity additions: new inline styles, link command, emoji input rule,
// auto-pair, smart paste, focus/typewriter modes, find bar, heading reveal.

func TestNativeParityInlineStyles(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		original := "A <u>under</u> ==high== ^sup^ ~sub~ end"
		e.Load(original)
		got, dirty, err := e.Snapshot()
		if err != nil || dirty || got != original {
			t.Errorf("inline styles roundtrip: %q dirty=%v err=%v", got, dirty, err)
		}
		if !nativeTestHasTraits(e, "underline", "highlight", "sup", "sub") {
			t.Error("missing display attributes for the new inline styles")
		}
		for _, example := range []struct{ command, want string }{
			{"underline", "a <u>word</u>"},
			{"highlight", "a ==word=="},
			{"sup", "a ^word^"},
			{"sub", "a ~word~"},
		} {
			e.Load("a word")
			nativeTestSelect(e, 2, 4)
			e.Format(example.command)
			got, _, _ = e.Snapshot()
			if got != example.want {
				t.Errorf("%s: %q != %q", example.command, got, example.want)
			}
			nativeTestAction(e, "undo", "")
			got, _, _ = e.Snapshot()
			if got != "a word" {
				t.Errorf("%s undo: %q", example.command, got)
			}
			e.Format(example.command)
			e.Format(example.command)
			got, _, _ = e.Snapshot()
			if got != "a word" {
				t.Errorf("%s double toggle: %q", example.command, got)
			}
		}
		// Superscript and subscript exclude each other.
		e.Load("x")
		nativeTestSelect(e, 0, 1)
		e.Format("sup")
		e.Format("sub")
		got, _, _ = e.Snapshot()
		if got != "~x~" {
			t.Errorf("sup/sub exclusion: %q", got)
		}
		// Empty selection toggles the typing attributes.
		e.Load("")
		e.Format("underline")
		nativeTestAction(e, "type", "typed")
		got, _, _ = e.Snapshot()
		if got != "<u>typed</u>" {
			t.Errorf("typing-attribute underline: %q", got)
		}
	})
}

func TestNativeParityHeadingsAndClear(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for level := 3; level <= 6; level++ {
			e.Load("Title")
			nativeTestSelect(e, 1, 0)
			e.Format("heading" + strconv.Itoa(level))
			got, _, _ := e.Snapshot()
			if want := strings.Repeat("#", level) + " Title"; got != want {
				t.Errorf("heading%d: %q != %q", level, got, want)
			}
			nativeTestAction(e, "undo", "")
			got, _, _ = e.Snapshot()
			if got != "Title" {
				t.Errorf("heading%d undo: %q", level, got)
			}
		}
		original := "##### deep\n\nbody"
		e.Load(original)
		got, _, _ := e.Snapshot()
		if got != original {
			t.Errorf("deep heading roundtrip: %q", got)
		}
		// Clear strips inline traits and resets the paragraph kind.
		e.Load("## Title **bold** <u>under</u>")
		nativeTestSelect(e, 0, 16)
		e.Format("clear")
		got, _, _ = e.Snapshot()
		if got != "Title bold under" {
			t.Errorf("clear selection: %q", got)
		}
		e.Load("# T")
		nativeTestSelect(e, 2, 0)
		e.Format("clear")
		got, _, _ = e.Snapshot()
		if got != "T" {
			t.Errorf("clear caret paragraph: %q", got)
		}
		e.Load("")
		e.Format("bold")
		e.Format("clear")
		nativeTestAction(e, "type", "plain")
		got, _, _ = e.Snapshot()
		if got != "plain" {
			t.Errorf("clear typing attributes: %q", got)
		}
	})
}

func TestNativeParityLinkCommand(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("click here")
		nativeTestSelect(e, 6, 4)
		e.Format("link:https://example.com")
		got, _, _ := e.Snapshot()
		if got != "click [here](<https://example.com>)" {
			t.Errorf("link command: %q", got)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != "click here" {
			t.Errorf("link undo: %q", got)
		}
		// Empty selection is a no-op.
		nativeTestSelect(e, 0, 0)
		e.Format("link:https://example.com")
		got, _, _ = e.Snapshot()
		if got != "click here" {
			t.Errorf("link empty selection: %q", got)
		}
	})
}

func TestNativeParityEmojiRule(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		emoji, ok := mdext.EmojiFor("smile")
		if !ok {
			t.Fatal("smile emoji missing")
		}
		// No word boundary yet: the shortcode stays literal.
		e.Load("")
		nativeTypeCharacters(e, ":smile:")
		got, _, _ := e.Snapshot()
		if got != ":smile:" {
			t.Errorf("emoji before boundary: %q", got)
		}
		// A space closes the shortcode and converts it.
		nativeTestAction(e, "type", " ")
		got, _, _ = e.Snapshot()
		if !strings.HasPrefix(got, emoji) || strings.Contains(got, ":smile:") {
			t.Errorf("emoji on space: %q", got)
		}
		if start, _ := nativeTestSelection(e); start != richtext.UTF16Len(emoji)+1 {
			t.Errorf("caret after emoji: %d", start)
		}
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != "" {
			t.Errorf("emoji undo group: %q", got)
		}
		// Enter is a word boundary too.
		e.Load("")
		nativeTypeCharacters(e, ":smile:")
		nativeTestAction(e, "newline", "")
		got, _, _ = e.Snapshot()
		if !strings.HasPrefix(got, emoji) || strings.Contains(got, ":smile:") {
			t.Errorf("emoji on newline: %q", got)
		}
		// Unknown shortcodes stay literal.
		e.Load("")
		nativeTypeCharacters(e, ":nosuchemoji: ")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, ":nosuchemoji:") {
			t.Errorf("unknown emoji: %q", got)
		}
		// Never inside a code block.
		e.Load("```\nx\n```")
		nativeTestSelect(e, 0, 0)
		nativeTypeCharacters(e, ":smile:")
		nativeTestAction(e, "type", " ")
		got, _, _ = e.Snapshot()
		if strings.Contains(got, emoji) || !strings.Contains(got, ":smile:") {
			t.Errorf("emoji in code block: %q", got)
		}
	})
}

func TestNativeParityAutoPair(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		// Pair insertion and pair deletion. Brackets need no escaping in prose.
		e.Load("")
		nativeTestAction(e, "type", "(")
		got, _, _ := e.Snapshot()
		if got != "()" {
			t.Errorf("pair insert: %q", got)
		}
		if start, length := nativeTestSelection(e); start != 1 || length != 0 {
			t.Errorf("caret inside pair: %d,%d", start, length)
		}
		nativeTestAction(e, "backspace", "")
		got, _, _ = e.Snapshot()
		if got != "" {
			t.Errorf("pair backspace: %q", got)
		}
		// Type-over steps past an existing closer.
		e.Load("")
		nativeTestAction(e, "type", "(")
		nativeTestAction(e, "type", ")")
		got, _, _ = e.Snapshot()
		if got != "()" {
			t.Errorf("type-over: %q", got)
		}
		if start, _ := nativeTestSelection(e); start != 2 {
			t.Errorf("caret after type-over: %d", start)
		}
		// A Markdown marker formats the selection; a bracket wraps it.
		e.Load("hello")
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "type", "*")
		got, _, _ = e.Snapshot()
		if got != "_hello_" {
			t.Errorf("format selection: %q", got)
		}
		if start, length := nativeTestSelection(e); start != 0 || length != 5 {
			t.Errorf("selection kept after format: %d,%d", start, length)
		}
		nativeTestSelect(e, 0, 5)
		nativeTestAction(e, "type", "[")
		got, _, _ = e.Snapshot()
		if got != "_[hello]_" {
			t.Errorf("wrap with bracket: %q", got)
		}
		// Quotes pair on a boundary, type over, and stay single after a word.
		e.Load("")
		nativeTestAction(e, "type", "\"")
		nativeTestAction(e, "type", "x")
		nativeTestAction(e, "type", "\"")
		got, _, _ = e.Snapshot()
		if got != "\"x\"" {
			t.Errorf("quote pair and type-over: %q", got)
		}
		e.Load("don")
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "type", "'")
		got, _, _ = e.Snapshot()
		if got != "don'" {
			t.Errorf("apostrophe after word: %q", got)
		}
		// Never inside code blocks or inline code runs.
		e.Load("```\nx\n```")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "type", "(")
		got, _, _ = e.Snapshot()
		if got != "```\n(x\n```" {
			t.Errorf("pair in code block: %q", got)
		}
		e.Load("`code`")
		// Offsets are visual (backticks hidden): 1 lands after 'c'. The pair
		// must be suppressed inside the code run.
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "type", "(")
		got, _, _ = e.Snapshot()
		if got != "`c(ode`" {
			t.Errorf("pair in inline code: %q", got)
		}
		// Pair insertion is a single undo step.
		e.Load("")
		nativeTestAction(e, "type", "[")
		nativeTestAction(e, "undo", "")
		got, _, _ = e.Snapshot()
		if got != "" {
			t.Errorf("pair undo: %q", got)
		}
	})
}

func TestNativeParitySmartPaste(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		savedName, savedBytes := "", 0
		e.OnSaveImage = func(name string, png []byte) (string, bool) {
			savedName, savedBytes = name, len(png)
			return "![pasted](saved.png)", true
		}
		e.Load("")
		nativeTestAction(e, "pngPaste", "")
		got, _, _ := e.Snapshot()
		if !strings.Contains(got, "![pasted](saved.png)") {
			t.Errorf("image paste: %q", got)
		}
		if savedName != "pasted-image.png" || savedBytes == 0 {
			t.Errorf("image handler args: %q %d bytes", savedName, savedBytes)
		}
		// Without a handler the image paste is declined and changes nothing.
		e.OnSaveImage = nil
		e.Load("")
		nativeTestAction(e, "pngPaste", "")
		got, _, _ = e.Snapshot()
		if got != "" {
			t.Errorf("declined image paste: %q", got)
		}
		// Clipboard HTML converts to Markdown.
		e.Load("")
		nativeTestAction(e, "htmlPaste", "<h1>Title</h1><p>Some <strong>bold</strong> text</p>")
		got, _, _ = e.Snapshot()
		if !strings.Contains(got, "# Title") || !strings.Contains(got, "**bold**") {
			t.Errorf("html paste: %q", got)
		}
		// Copied files import through the drop handler.
		var imported []string
		e.OnImportFiles = func(paths []string) (string, bool) {
			imported = paths
			return "![f](" + paths[0] + ")", true
		}
		e.Load("")
		nativeTestAction(e, "filePaste", "/tmp/a.png|/tmp/b.png")
		got, _, _ = e.Snapshot()
		if len(imported) != 2 || imported[0] != "/tmp/a.png" || imported[1] != "/tmp/b.png" {
			t.Errorf("file paste paths: %v", imported)
		}
		if !strings.Contains(got, "![f](/tmp/a.png)") {
			t.Errorf("file paste insert: %q", got)
		}
	})
}

func TestNativeParityFocusMode(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("one\n\ntwo\n\nthree")
		nativeTestSelect(e, 1, 0)
		e.SetFocusMode(true)
		if !nativeTestHasTraits(e, "alphaAt:4") || !nativeTestHasTraits(e, "alphaAt:8") {
			t.Error("focus mode did not dim the other paragraphs")
		}
		if nativeTestHasTraits(e, "alphaAt:1") {
			t.Error("focus mode dimmed the caret paragraph")
		}
		nativeTestSelect(e, 5, 0)
		if !nativeTestHasTraits(e, "alphaAt:1") || nativeTestHasTraits(e, "alphaAt:5") {
			t.Error("focus mode did not follow the caret")
		}
		// Dimming survives a dark-mode restyle.
		e.SetStyle(1, true, true)
		if !nativeTestHasTraits(e, "alphaAt:1") || nativeTestHasTraits(e, "alphaAt:5") {
			t.Error("focus mode lost after a restyle")
		}
		e.SetFocusMode(false)
		if nativeTestHasTraits(e, "alphaAt:1") || nativeTestHasTraits(e, "alphaAt:5") {
			t.Error("focus mode did not restore colors")
		}
	})
}

func TestNativeParityTypewriter(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 200)
		e.SetVisible(true)
		var sb strings.Builder
		for i := 0; i < 40; i++ {
			sb.WriteString("paragraph\n\n")
		}
		e.Load(sb.String())
		e.SetTypewriter(true)
		nativeTestSelect(e, 0, 0)
		if nativeTestHasTraits(e, "scrolled") {
			t.Error("typewriter scrolled at the top")
		}
		nativeTestSelect(e, 398, 0)
		if !nativeTestHasTraits(e, "scrolled") {
			t.Error("typewriter did not center the caret line")
		}
		e.SetTypewriter(false)
		nativeTestSelect(e, 0, 0)
		if !nativeTestHasTraits(e, "scrolled") {
			t.Error("disabled typewriter should leave scrolling alone")
		}
	})
}

func TestNativeParityRevealHeading(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.SetFrame(0, 0, 720, 200)
		e.SetVisible(true)
		var sb strings.Builder
		sb.WriteString("# One\n\n")
		for i := 0; i < 30; i++ {
			sb.WriteString("filler\n\n")
		}
		sb.WriteString("# Two\n\n")
		for i := 0; i < 30; i++ {
			sb.WriteString("filler\n\n")
		}
		sb.WriteString("# Three")
		e.Load(sb.String())
		e.RevealHeading(2)
		if !nativeTestHasTraits(e, "scrolled") {
			t.Error("reveal did not scroll to the last heading")
		}
		e.RevealHeading(99) // out of range: no-op
		if !nativeTestHasTraits(e, "scrolled") {
			t.Error("out-of-range reveal moved the view")
		}
		e.RevealHeading(0)
		if nativeTestHasTraits(e, "scrolled") {
			t.Error("reveal first heading did not return to the top")
		}
	})
}

func TestNativeParityFindBarAndSpellcheck(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("find me")
		e.ShowFindBar()
		if !nativeTestHasTraits(e, "findbar") {
			t.Error("find bar not enabled")
		}
		// Spellcheck composes with IME marked text.
		e.SetSpellcheck(true)
		nativeTestSelect(e, 7, 0)
		nativeTestAction(e, "mark", "zhong")
		nativeTestAction(e, "commit", "中文")
		got, _, _ := e.Snapshot()
		if got != "find me中文" {
			t.Errorf("spellcheck IME: %q", got)
		}
		e.SetSpellcheck(false)
	})
}

// Block editing keys behave like Typora: lists stay tight, quotes continue,
// Backspace at a block start demotes it, ordered items renumber.
func TestNativeParityBlockKeys(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, c := range []struct {
			name, source string
			caret        int
			actions      []string
			want         string
		}{
			{"bullet enter", "- a", 1, []string{"newline", "", "type", "b"}, "- a\n- b"},
			{"typed list stays tight", "", 0, []string{"type", "- ", "type", "a", "newline", "", "type", "b"}, "- a\n- b"},
			{"ordered insert renumbers", "1. a\n2. b", 1, []string{"newline", "", "type", "x"}, "1. a\n2. x\n3. b"},
			{"empty item leaves list", "- a", 1, []string{"newline", "", "newline", ""}, "- a\n\n"},
			{"quote continues", "> q", 1, []string{"newline", "", "type", "r"}, "> q\n>\n> r"},
			{"empty quote line leaves quote", "> a", 1, []string{"newline", "", "newline", "", "type", "b"}, "> a\n\nb"},
			{"backspace demotes heading", "para\n\n# Title", 5, []string{"backspace", ""}, "para\n\nTitle"},
			{"backspace demotes checklist", "- [ ] t", 0, []string{"backspace", ""}, "t"},
			{"backspace demotes quote paragraph", "> a\n>\n> b", 2, []string{"backspace", ""}, "> a\n\nb"},
			{"enter at heading start", "# Title", 0, []string{"newline", ""}, "\n# Title"},
			{"tab in list is inert", "- item", 0, []string{"tab", ""}, "- item"},
		} {
			e.Load(c.source)
			nativeTestSelect(e, c.caret, 0)
			for i := 0; i+1 < len(c.actions); i += 2 {
				nativeTestAction(e, c.actions[i], c.actions[i+1])
			}
			if got, _, _ := e.Snapshot(); got != c.want {
				t.Errorf("%s: %q != %q", c.name, got, c.want)
			}
		}
		// Demoting the heading is one undo step.
		e.Load("# Title")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "backspace", "")
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != "# Title" {
			t.Errorf("backspace demote undo: %q", got)
		}
	})
}

// TestNativeParityHardBreak proves hard line breaks are WYSIWYG: every spelling
// loads as one editable block, keeps its block identity across a soft visual
// line, and survives the round-trip as Markdown.
func TestNativeParityHardBreak(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		for _, source := range []string{
			"line one  \nline two",
			"line one<br>line two",
			"> quoted a  \n> quoted b",
		} {
			e.Load(source)
			got, dirty, err := e.Snapshot()
			if err != nil || dirty || got != source {
				t.Errorf("hard break roundtrip %q: got %q dirty=%v err=%v", source, got, dirty, err)
			}
		}
		// The break keeps both halves in one paragraph: appending text after it
		// extends the same block instead of starting a new line.
		e.Load("one  \ntwo")
		// Display is "one  <U+2028>two" = 9 UTF-16 units; caret at the end.
		nativeTestSelect(e, 9, 0)
		nativeTestAction(e, "type", "!")
		got, _, _ := e.Snapshot()
		if got != "one  \ntwo!" {
			t.Errorf("type after break: %q", got)
		}
		// Shift+Return authors a hard break inside the current block.
		e.Load("alpha")
		nativeTestSelect(e, 5, 0)
		nativeTestAction(e, "linebreak", "")
		nativeTestAction(e, "type", "beta")
		got, _, _ = e.Snapshot()
		if got != "alpha  \nbeta" {
			t.Errorf("shift+enter authoring: %q", got)
		}
		// In a quote, the authored break stays a single quote block.
		e.Load("> q")
		nativeTestSelect(e, 1, 0)
		nativeTestAction(e, "linebreak", "")
		nativeTestAction(e, "type", "r")
		got, _, _ = e.Snapshot()
		if got != "> q  \n> r" {
			t.Errorf("shift+enter in quote: %q", got)
		}
	})
}

// TestNativeParityFigureRestore proves a rendered figure (a formula) that is
// clicked open for editing re-renders to its figure form once the editor loses
// focus, even when it is the first block.
func TestNativeParityFigureRestore(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("$$\na^2 + b^2 = c^2\n$$")
		if !nativeTestHasTraits(e, "math") {
			t.Fatal("formula did not render on load")
		}
		// Click the figure to edit its source.
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "reveal", "")
		if nativeTestHasTraits(e, "math") || !nativeTestHasTraits(e, "source") {
			t.Fatal("clicking the formula did not reveal its source")
		}
		// Losing focus restores the rendered figure.
		nativeTestAction(e, "blur", "")
		if !nativeTestHasTraits(e, "math") || nativeTestHasTraits(e, "source") {
			t.Fatal("formula did not re-render after losing focus")
		}
		if got, _, _ := e.Snapshot(); got != "$$\na^2 + b^2 = c^2\n$$" {
			t.Errorf("formula round trip: %q", got)
		}
	})
}

// TestNativeParityCodeHighlight proves a fenced code block with a known
// language is syntax-highlighted: keyword/string token classes are tagged on
// its characters, the block still round-trips byte-exact, and an unknown
// language leaves the code untinted.
func TestNativeParityCodeHighlight(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "```go\nfunc main() {\n\tx := \"hi\"\n}\n```"
		e.Load(source)
		if !nativeTestHasTraits(e, "hl:keyword") {
			t.Error("code block keywords not highlighted")
		}
		if !nativeTestHasTraits(e, "hl:string") {
			t.Error("code block strings not highlighted")
		}
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("highlighted code round trip: %q != %q", got, source)
		}
		// An unlabeled fence has no language: nothing to tint.
		e.Load("```\nfunc main() {}\n```")
		if nativeTestHasTraits(e, "hl:keyword") {
			t.Error("unlabeled code block should not be highlighted")
		}
	})
}

// TestNativeParityDiagramRender proves a chart/diagram fence renders as a
// figure on load, reveals its editable source on click, re-renders on blur,
// and round-trips byte-exact as Markdown.
func TestNativeParityDiagramRender(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "```chart\ntype: line\nlabels: [一, 二, 三]\nseries:\n  - name: 销量\n    data: [3, 7, 5]\n```"
		e.Load(source)
		if !nativeTestHasTraits(e, "diagram") {
			t.Fatal("chart did not render as a figure on load")
		}
		// Click the figure to edit its source.
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "reveal", "")
		if nativeTestHasTraits(e, "diagram") || !nativeTestHasTraits(e, "source") {
			t.Fatal("clicking the chart did not reveal its source")
		}
		// Losing focus restores the rendered figure.
		nativeTestAction(e, "blur", "")
		if !nativeTestHasTraits(e, "diagram") || nativeTestHasTraits(e, "source") {
			t.Fatal("chart did not re-render after losing focus")
		}
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("chart round trip: %q != %q", got, source)
		}
	})
}

