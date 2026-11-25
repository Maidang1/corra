package editor_test

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/ext/std"
)

type fakeClipboard struct{ clip editor.Clip }

func (f *fakeClipboard) Write(c editor.Clip) { f.clip = c }
func (f *fakeClipboard) Read() editor.Clip   { return f.clip }

type harness struct {
	t    *testing.T
	kit  *ext.Kit
	e    *editor.Editor
	tt   *ui.Tester
	clip *fakeClipboard
}

func newHarness(t *testing.T, md string) *harness {
	t.Helper()
	k := std.Kit()
	cfg := k.EditorConfig()
	clip := &fakeClipboard{}
	cfg.Clipboard = clip
	e := editor.New(cfg, k.Codec.Parse(md))
	env := editor.Env{Font: ui.Font{Size: 16}, Mono: "monospace"}
	tt := ui.NewTester(func(c *ui.Context) { e.View(c, env) }, 640, 480)
	tt.Frame()
	tt.Frame()
	h := &harness{t: t, kit: k, e: e, tt: tt, clip: clip}
	h.focusEnd()
	return h
}

// focusEnd clicks the editor, which focuses it, and puts the caret at the
// end of the document.
func (h *harness) focusEnd() {
	h.tt.ClickAt(300, 470)
	h.tt.Frame()
	s := h.e.State()
	last := s.LastLeaf(s.Doc.ID())
	h.e.SetSelection(doc.Caret(doc.Point{Leaf: last.ID(), Offset: last.TextLen()}))
	h.tt.Frame()
}

func (h *harness) key(mods ui.Modifiers, k ui.Key) {
	h.tt.Key(mods, k)
	h.tt.Frame()
}

func (h *harness) typ(s string) {
	for _, r := range s {
		h.tt.Type(string(r))
	}
	h.tt.Frame()
}

func (h *harness) md() string { return h.kit.Codec.Write(h.e.Doc()) }

func (h *harness) want(md string) {
	h.t.Helper()
	if got := h.md(); got != md {
		h.t.Fatalf("got  %q\nwant %q", got, md)
	}
}

func TestTypeAtEnd(t *testing.T) {
	h := newHarness(t, "Hello\n")
	h.typ(" world")
	h.want("Hello world\n")
}

func TestEnterAndBackspace(t *testing.T) {
	h := newHarness(t, "Hello\n")
	h.key(0, ui.KeyLeft)
	h.key(0, ui.KeyLeft)
	h.key(0, ui.KeyEnter)
	h.want("Hel\n\nlo\n")
	h.key(0, ui.KeyBackspace)
	h.want("Hello\n")
	h.key(0, ui.KeyBackspace)
	h.want("Helo\n")
}

func TestCompositionIsOneUndoStep(t *testing.T) {
	h := newHarness(t, "中文\n")
	h.tt.Compose("ni", 2)
	h.tt.Frame()
	if _, ok := h.e.MarkedRange(); !ok {
		t.Fatal("no marked range")
	}
	h.tt.Compose("你", 1)
	h.tt.Type("你")
	h.tt.Frame()
	h.want("中文你\n")
	if _, ok := h.e.MarkedRange(); ok {
		t.Fatal("still composing")
	}
	h.key(ui.Cmd, ui.KeyZ)
	h.want("中文\n")
}

func TestHeadingRuleAndBackspaceUndoesIt(t *testing.T) {
	h := newHarness(t, "text\n")
	h.key(0, ui.KeyEnter)
	h.typ("## ")
	h.want("text\n\n##\n")
	h.typ("Title")
	h.want("text\n\n## Title\n")
	// Right after a rule, Backspace gives the typed syntax back.
	h = newHarness(t, "text\n")
	h.key(0, ui.KeyEnter)
	h.typ("# ")
	h.key(0, ui.KeyBackspace)
	h.typ("x")
	h.want("text\n\n\\# x\n")
}

func TestMarkRules(t *testing.T) {
	h := newHarness(t, "a\n")
	h.typ(" **bold** and *it* `c` ~~s~~ x")
	h.want("a **bold** and *it* `c` ~~s~~ x\n")
	if !strings.Contains(h.md(), "**bold**") {
		t.Fatal("bold rule")
	}
}

func TestBoldKeyOnSelection(t *testing.T) {
	h := newHarness(t, "Hello\n")
	h.key(ui.Cmd, ui.KeyA)
	h.key(ui.Cmd, ui.KeyB)
	h.want("**Hello**\n")
	h.key(ui.Cmd, ui.KeyZ)
	h.want("Hello\n")
	h.key(ui.Cmd|ui.Shift, ui.KeyZ)
	h.want("**Hello**\n")
}

func TestListTyping(t *testing.T) {
	h := newHarness(t, "intro\n")
	h.key(0, ui.KeyEnter)
	h.typ("- one")
	h.key(0, ui.KeyEnter)
	h.typ("two")
	h.want("intro\n\n- one\n- two\n")
	h.key(0, ui.KeyTab)
	h.want("intro\n\n- one\n  - two\n")
	h.key(ui.Shift, ui.KeyTab)
	h.want("intro\n\n- one\n- two\n")
	h.key(0, ui.KeyEnter)
	h.key(0, ui.KeyEnter) // an empty item leaves the list
	h.typ("after")
	h.want("intro\n\n- one\n- two\n\nafter\n")
}

func TestCopyPaste(t *testing.T) {
	h := newHarness(t, "# Title\n\nbody **b**\n")
	h.key(ui.Cmd, ui.KeyA)
	h.key(ui.Cmd, ui.KeyC)
	if h.clip.clip.Markdown != "# Title\n\nbody **b**\n" || h.clip.clip.Plain != "Title\nbody b" {
		t.Fatalf("clip %+v", h.clip.clip)
	}
	h.key(0, ui.KeyRight)
	h.key(0, ui.KeyEnter)
	h.key(ui.Cmd, ui.KeyV)
	h.want("# Title\n\nbody **b**\n\n# Title\n\nbody **b**\n")
	// Markdown from elsewhere is read as Markdown.
	h.clip.clip = editor.Clip{Plain: "- a\n- b"}
	h.key(0, ui.KeyEnter)
	h.key(ui.Cmd, ui.KeyV)
	if !strings.HasSuffix(h.md(), "- a\n- b\n") {
		t.Fatalf("got %q", h.md())
	}
}

func TestFindReplaceAll(t *testing.T) {
	h := newHarness(t, "cat Cat\n\n- the cat\n")
	if n := h.e.Find("cat"); n != 3 {
		t.Fatalf("matches %d", n)
	}
	if n := h.e.ReplaceAll("dog"); n != 3 {
		t.Fatalf("replaced %d", n)
	}
	h.want("dog dog\n\n- the dog\n")
	h.key(ui.Cmd, ui.KeyZ)
	h.want("cat Cat\n\n- the cat\n")
}

func TestClickCheckbox(t *testing.T) {
	h := newHarness(t, "- [ ] todo\n")
	// The checkbox sits left of the item's text, in the list's indent.
	h.tt.ClickAt(24+16*1.6-14-8+7, 24+12)
	h.tt.Frame()
	h.want("- [x] todo\n")
}

func TestVerticalMovement(t *testing.T) {
	h := newHarness(t, "first\n\nsecond line\n\nthird\n")
	h.key(0, ui.KeyUp)
	h.key(0, ui.KeyUp)
	h.typ("X")
	if !strings.HasPrefix(h.md(), "first") || strings.Count(h.md(), "X") != 1 || strings.Contains(h.md(), "thirdX") {
		t.Fatalf("got %q", h.md())
	}
}

func TestReloadIsOneUndoStepAndKeepsBlocks(t *testing.T) {
	h := newHarness(t, "# Title\n\nkeep\n\nold\n")
	keep := h.e.Doc().Child(1)
	h.e.Reload(h.kit.Codec.Parse("# Title\n\nkeep\n\nnew text\n"))
	h.want("# Title\n\nkeep\n\nnew text\n")
	if h.e.Doc().Child(1) != keep {
		t.Fatal("unchanged block was replaced")
	}
	h.key(ui.Cmd, ui.KeyZ)
	h.want("# Title\n\nkeep\n\nold\n")
}

func TestTableEditing(t *testing.T) {
	h := newHarness(t, "| a | b |\n| --- | --- |\n| 1 | 2 |\n")
	h.typ("x")
	h.want("| a | b |\n| --- | --- |\n| 1 | 2x |\n")
	h.key(0, ui.KeyTab) // past the last cell: a new row
	h.typ("y")
	h.want("| a | b |\n| --- | --- |\n| 1 | 2x |\n| y |  |\n")
	h.key(ui.Shift, ui.KeyTab) // back to the previous cell, selected
	h.typ("z")
	h.want("| a | b |\n| --- | --- |\n| 1 | z |\n| y |  |\n")
	// Clicking a header cell puts the caret there.
	h.tt.ClickAt(24+20, 24+10)
	h.tt.Frame()
	h.typ("!")
	if !strings.Contains(h.md(), "!") || !strings.HasPrefix(h.md(), "| a") {
		t.Fatalf("got %q", h.md())
	}
}

func TestDiagramPreviewAppears(t *testing.T) {
	src := "```chart\ntype: line\nlabels: [a, b, c]\nseries:\n  - name: s\n    data: [3, 7, 5]\n```\n"
	h := newHarness(t, src)
	// The code block alone is about 160 tall; the chart adds hundreds.
	for i := 0; i < 200 && h.e.Height() < 300; i++ {
		time.Sleep(10 * time.Millisecond)
		h.tt.Frame()
	}
	if h.e.Height() < 300 {
		t.Fatalf("no preview: height %v", h.e.Height())
	}
	// The source is untouched.
	h.want(src)
}

func TestSlashMenu(t *testing.T) {
	h := newHarness(t, "intro\n")
	h.key(0, ui.KeyEnter)
	h.typ("/")
	h.typ("tab") // filters to the table entry
	h.key(0, ui.KeyEnter)
	h.want("intro\n\n|  |  |  |\n| --- | --- | --- |\n|  |  |  |\n|  |  |  |\n")
	// Escape closes the menu, leaving the typed text.
	h = newHarness(t, "intro\n")
	h.key(0, ui.KeyEnter)
	h.typ("/h")
	h.key(0, ui.KeyEscape)
	h.key(0, ui.KeyEnter)
	h.want("intro\n\n/h\n")
	// Choosing a block type turns the line into it, as one undo step.
	h = newHarness(t, "intro\n")
	h.key(0, ui.KeyEnter)
	h.typ("/二级")
	h.key(0, ui.KeyEnter)
	h.typ("Title")
	h.want("intro\n\n## Title\n")
}

func TestFormatBar(t *testing.T) {
	h := newHarness(t, "Hello\n")
	h.key(ui.Cmd, ui.KeyA)
	r, buttons, ok := h.e.FormatBarForTest()
	if !ok || buttons[0] != "B" {
		t.Fatalf("no bar: %v %v", ok, buttons)
	}
	h.tt.ClickAt(r.X+10, r.Y+r.H/2)
	h.tt.Frame()
	h.want("**Hello**\n")
}

func TestFocusAndTypewriterModes(t *testing.T) {
	var md strings.Builder
	for i := 0; i < 60; i++ {
		md.WriteString("line\n\n")
	}
	h := newHarness(t, md.String())
	h.e.SetTypewriter(true)
	h.e.SetFocusMode(true)
	h.key(ui.Cmd, ui.KeyUp)
	h.key(0, ui.KeyDown)
	h.key(0, ui.KeyDown)
	h.tt.Frame()
	h.typ("x") // still edits normally
	if !strings.Contains(h.md(), "linex") && !strings.Contains(h.md(), "xline") {
		t.Fatalf("typing in focus mode: %q", h.md()[:40])
	}
}

func TestPasteAndDropImages(t *testing.T) {
	k := std.Kit()
	cfg := k.EditorConfig()
	clip := &fakeClipboard{}
	cfg.Clipboard = clip
	cfg.OnPasteImage = func(png []byte) (string, error) { return "![pasted](assets/p.png)", nil }
	cfg.OnDropFiles = func(paths []string) string { return "![dropped](assets/" + paths[0] + ")" }
	e := editor.New(cfg, k.Codec.Parse("text\n"))
	env := editor.Env{Font: ui.Font{Size: 16}, Mono: "monospace"}
	tt := ui.NewTester(func(c *ui.Context) { e.View(c, env) }, 640, 480)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(300, 400)
	tt.Frame()
	s := e.State()
	last := s.LastLeaf(s.Doc.ID())
	e.SetSelection(doc.Caret(doc.Point{Leaf: last.ID(), Offset: last.TextLen()}))
	clip.clip = editor.Clip{Image: []byte{1, 2, 3}}
	tt.Key(ui.Cmd, ui.KeyV)
	tt.Frame()
	if got := k.Codec.Write(e.Doc()); got != "text![pasted](assets/p.png)\n" {
		t.Fatalf("paste: %q", got)
	}
	if !tt.DropFiles(300, 200, "d.png") {
		t.Fatal("drop not taken")
	}
	tt.Frame()
	tt.Frame()
	if got := k.Codec.Write(e.Doc()); !strings.Contains(got, "![dropped](assets/d.png)") {
		t.Fatalf("drop: %q", got)
	}
}

func TestDragBlockByHandle(t *testing.T) {
	h := newHarness(t, "one\n\ntwo\n\nthree\n")
	// The first block's handle sits in the left margin of its first line.
	h.tt.Move(10, 30)
	h.tt.Press(24-14-6+7, 24+8)
	h.tt.Move(24-14-6+7, 200)
	h.tt.Release(24-14-6+7, 200)
	h.tt.Frame()
	h.want("two\n\nthree\n\none\n")
	h.key(ui.Cmd, ui.KeyZ)
	h.want("one\n\ntwo\n\nthree\n")
}

func TestOrdinalsFollowDocumentOrder(t *testing.T) {
	h := newHarness(t, "$$\na\n$$\n\ntext\n\n$$\nb\n$$\n")
	d := h.e.Doc()
	second := d.Child(2)
	if h.e.Ordinal(d.Child(0)) != 1 || h.e.Ordinal(second) != 2 {
		t.Fatalf("ordinals %d %d", h.e.Ordinal(d.Child(0)), h.e.Ordinal(second))
	}
	// A new formula before the second one renumbers it.
	h.e.SetSelection(doc.Caret(doc.Point{Leaf: d.Child(1).ID(), Offset: 4}))
	h.e.Run(doc.InsertBlocks(h.kit.Codec.Parse("$$\nc\n$$\n").Child(0)), "structure")
	h.tt.Frame()
	if got := h.e.Ordinal(second); got != 3 {
		t.Fatalf("second formula is now %d, want 3", got)
	}
}

type wordList map[string]bool

// Misspelled rejects words not in the list.
func (w wordList) Misspelled(text string) []ui.TextInputRange {
	var out []ui.TextInputRange
	off := 0
	for _, f := range strings.Split(text, " ") {
		if f != "" && !w[f] {
			out = append(out, ui.TextInputRange{Start: off, End: off + len(f)})
		}
		off += len(f) + 1
	}
	return out
}

func TestSpellcheckChecksEachBlockOnce(t *testing.T) {
	k := std.Kit()
	calls := 0
	cfg := k.EditorConfig()
	words := wordList{"good": true, "text": true}
	cfg.Spell = spellCounter{words, &calls}
	e := editor.New(cfg, k.Codec.Parse("good text\n\nbadd text\n"))
	e.SetSpellcheck(true)
	env := editor.Env{Font: ui.Font{Size: 16}, Mono: "monospace"}
	tt := ui.NewTester(func(c *ui.Context) { e.View(c, env) }, 640, 480)
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	if calls != 2 {
		t.Fatalf("checked %d times, want once per block", calls)
	}
	if got := e.MisspelledForTest(1); len(got) != 1 || got[0].Start != 0 || got[0].End != 4 {
		t.Fatalf("misspelled %v", got)
	}
}

type spellCounter struct {
	w     wordList
	calls *int
}

func (s spellCounter) Misspelled(text string) []ui.TextInputRange {
	*s.calls++
	return s.w.Misspelled(text)
}
