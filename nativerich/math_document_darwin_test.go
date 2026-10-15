//go:build darwin && cgo

package nativerich

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/richtext"
)

func TestNativeDocumentMathNumbering(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "See $\\eqref{energy}$.\n\n$$\nE=mc^2\\label{energy}\n$$\n\n$$\nx+y\n$$"
		e.Load(source)
		e.Format("math-numbering:all")
		if !nativeTestHasTraits(e, "math-number:1", "math-number:2") {
			t.Error("document equations have no native numbered images")
		}
		if got, changed, err := e.Snapshot(); err != nil || changed || got != source {
			t.Errorf("numbering changed Markdown: %q %v %v", got, changed, err)
		}
		if path := os.Getenv("CORRA_MATH_SCREENSHOT"); path != "" {
			e.SetFrame(0, 0, 720, 500)
			e.SetVisible(true)
			if !nativeTestScreenshot(e, path) {
				t.Error("failed to capture native math")
			}
		}
		nativeTestSelect(e, 0, 3)
		e.Format("bold")
		e.SetStyle(1.3, true, true)
		e.Format("math-numbering:none")
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("math presentation reset native undo: %q", got)
		}
		e.Format("math-numbering:all")
		e.Load("$$\nz\n$$")
		if !nativeTestHasTraits(e, "math-number:1") || nativeTestHasTraits(e, "math-number:2") {
			t.Error("equation numbering leaked from the preceding document")
		}
	})
}

func TestNativeMathDraftKeepsDocumentNumbering(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Format("math-numbering:all")
		source := "$$\nx\n$$\n\n$$\nx\n$$\n\nend"
		e.Load(source)
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "reveal", "")
		if !nativeTestHasTraits(e, "source", "math-number:2") || nativeTestHasTraits(e, "math-number:1") {
			t.Error("revealing equation one renumbered the identical second equation")
		}
		if got, _, _ := e.Snapshot(); got != source {
			t.Errorf("reveal changed source: %q", got)
		}
		nativeTestAction(e, "key", "cmd-return")
		if !nativeTestHasTraits(e, "math-number:1", "math-number:2") {
			t.Error("leaving source did not restore both numbered equations")
		}
		nativeTestSelect(e, 0, 0)
		e.InsertMarkdown("$$\ny\n$$")
		if !nativeTestHasTraits(e, "math-number:3") {
			t.Error("inserting equation did not update following numbers")
		}
		nativeTestAction(e, "undo", "")
		if !nativeTestHasTraits(e, "math-number:2") || nativeTestHasTraits(e, "math-number:3") {
			t.Error("undoing equation insertion did not update numbering")
		}
	})
}

func TestNativeMathDocumentRendersSharedReferences(t *testing.T) {
	// Check the actual PNGs supplied to AppKit, including forward references,
	// macros shared across paragraphs, and a formula in a native table cell.
	source := "See $\\eqref{energy}$.\n\n$$\nE=mc^2\\label{energy}\n$$\n\n| Math |\n| --- |\n| $\\eqref{energy}$ |"
	doc := richtext.Parse(source)
	e := &Editor{}
	var results []nativeMathResult
	if err := json.Unmarshal([]byte(e.documentMath(doc, false, 1, "all")), &results); err != nil {
		t.Fatal(err)
	}
	expected, err := render.RenderMathBatch([]render.MathExpression{{TeX: `\eqref{energy}`}, {TeX: `E=mc^2\label{energy}`, Display: true}, {TeX: `\eqref{energy}`}}, render.MathOptions{Numbering: "all"}, 15, color.RGBA{R: 41, G: 41, B: 41, A: 255})
	if err != nil || len(results) != len(expected) || len(results) != 3 {
		t.Fatalf("formula collection: %d %d %v", len(results), len(expected), err)
	}
	for i, result := range results {
		var pngBytes bytes.Buffer
		if expected[i].Error != nil {
			t.Fatal(expected[i].Error)
		}
		if err := png.Encode(&pngBytes, expected[i].Image); err != nil {
			t.Fatal(err)
		}
		actual, err := base64.StdEncoding.DecodeString(result.Image)
		if err != nil || result.Error != "" || !bytes.Equal(actual, pngBytes.Bytes()) || result.Number != expected[i].Number {
			t.Errorf("formula %d lost document context: number=%q error=%q decode=%v", i, result.Number, result.Error, err)
		}
	}
	key := e.mathKey
	doc.Paragraphs[0].Runs[0].Text = "unrelated prose "
	e.documentMath(doc, false, 1, "all")
	if e.mathKey != key {
		t.Error("prose-only editing invalidated equation rendering")
	}
}

func TestNativeMathErrorEditingAndMacroContext(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load(`before $\badcommand$ after`)
		nativeTestSelect(e, 9, 1)
		nativeTestAction(e, "type", "X")
		if got, _, _ := e.Snapshot(); got != `before $\Xadcommand$ after` {
			t.Errorf("editing formula error lost source: %q", got)
		}
		e.Format("math-numbering:all")
		e.Load("$$\n\\newcommand{\\foo}{x}\n$$\n\n$$\n\\foo\n$$")
		nativeTestSelect(e, 2, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestAction(e, "blur", "")
		if nativeTestHasTraits(e, "source") || !nativeTestHasTraits(e, "math-number:2") {
			t.Error("re-rendering equation lost the document macro context")
		}
	})
}

func TestNativeAdjacentMathRefresh(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("$x$**$x$**\n\n$$\ny\n$$")
		e.Format("math-numbering:all")
		if !nativeTestHasTraits(e, "math-number:1") {
			t.Error("adjacent identical inline equations blocked document refresh")
		}
	})
}

func TestNativeInlineMathSourceEdit(t *testing.T) {
	var window uintptr
	var e *Editor
	nativeTestMain(func() {
		window = nativeTestWindow()
		e = New(window, nil, nil)
		e.Load("before $x$ after")
		nativeTestSelect(e, 7, 0)
		nativeTestAction(e, "reveal", "")
		if !nativeTestHasTraits(e, "inline-source") {
			t.Error("inline formula did not expose editable source")
		}
		nativeTestSelect(e, 0, 12)
		if a, b := nativeTestSelection(e); a != 0 || b != 12 || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("selection across expanded formula collapsed or moved: %d %d", a, b)
		}
		nativeTestSelect(e, 8, 1)
		nativeTestAction(e, "type", "y")
		if got, _, _ := e.Snapshot(); got != "before $y$ after" {
			t.Errorf("expanded inline edit lost source: %q", got)
		}
	})
	defer nativeTestMain(func() { e.Close(); nativeTestCloseWindow(window) })
	if !nativeTestFinishEvent(e) {
		t.Fatal("AppKit did not finish inline typing event")
	}
	nativeTestMain(func() {
		nativeTestAction(e, "blur", "")
		if !nativeTestHasTraits(e, "inline-math") || nativeTestHasTraits(e, "inline-source") {
			t.Error("inline source did not collapse after editing")
		}
	})
	if !nativeTestFinishEvent(e) {
		t.Fatal("AppKit did not finish inline rendering event")
	}
	nativeTestMain(func() {
		nativeTestAction(e, "undo", "")
		if got, _, _ := e.Snapshot(); got != "before $y$ after" || !nativeTestHasTraits(e, "inline-source") {
			t.Errorf("inline render undo lost the edited source: %q", got)
		}
		e.Load(`before $\badcommand$ after`)
		nativeTestSelect(e, 7, 13)
		nativeTestAction(e, "type", "$x$")
		nativeTestAction(e, "blur", "")
		if got, _, _ := e.Snapshot(); got != "before $x$ after" || !nativeTestHasTraits(e, "inline-math") {
			t.Errorf("corrected formula did not recover: %q", got)
		}
		e.Load("$x$")
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 3, 0)
		nativeTestAction(e, "key", "cmd-return")
		if !nativeTestHasTraits(e, "inline-math") || nativeTestHasTraits(e, "inline-source") {
			t.Error("Cmd+Return at the end of inline source did not finish editing")
		}
	})
}

func TestNativeTableMathSourceKeepsPipeEscapes(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("| M | Other |\n| --- | --- |\n| $\\|x\\|$ | ok |")
		nativeTestSelect(e, 8, 0)
		nativeTestAction(e, "reveal", "")
		nativeTestSelect(e, 11, 1)
		nativeTestAction(e, "type", "y")
		nativeTestAction(e, "blur", "")
		got, _, err := e.Snapshot()
		if err != nil || got != "| M | Other |\n| --- | --- |\n| $\\|y\\|$ | ok |" || !nativeTestHasTraits(e, "inline-math") {
			t.Errorf("editing cell math changed table/source: %q %v", got, err)
			return
		}
		parsed := richtext.Parse(got)
		if expression := parsed.Paragraphs[0].Rows[1][0].Runs[0].Math; expression != "|y|" {
			t.Errorf("Markdown pipe escapes leaked into TeX: %q", expression)
		}
		nativeTestSelect(e, 8, 0)
		e.Format("table:row-below")
		got, _, _ = e.Snapshot()
		parsed = richtext.Parse(got)
		if len(parsed.Paragraphs) != 1 || len(parsed.Paragraphs[0].Rows) != 3 || parsed.Paragraphs[0].Rows[1][1].Runs[0].Text != "ok" {
			t.Errorf("table rebuild damaged neighboring cell: %q", got)
		}
	})
}
