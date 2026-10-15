package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/color"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestHTMLMathUsesOfflineDocumentContext(t *testing.T) {
	source := "See $\\eqref{energy}$.\n\n$$\nE=mc^2\\tag{1}\\label{energy}\n$$\n\n| Reference |\n| --- |\n| $\\eqref{energy}$ |\n\n`$not_math$`"
	data, err := ExportHTML(source, "Math", false)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var images [][]byte
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" && strings.HasPrefix(attr.Val, "data:image/svg+xml;base64,") {
					decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(attr.Val, "data:image/svg+xml;base64,"))
					if err != nil {
						t.Error(err)
					}
					images = append(images, decoded)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	expected, err := RenderMathBatch([]MathExpression{{TeX: `\eqref{energy}`}, {TeX: `E=mc^2\tag{1}\label{energy}`, Display: true}, {TeX: `\eqref{energy}`}}, MathOptions{}, 16, color.Black)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 3 {
		t.Fatalf("export did not embed three independently readable equations: %d", len(images))
	}
	for i, img := range images {
		if expected[i].Error != nil || !bytes.Equal(img, expected[i].SVG) {
			t.Errorf("equation %d lost shared references: %v", i, expected[i].Error)
		}
	}
	if strings.Contains(string(data), "katex") || strings.Contains(string(data), "renderMathInElement") {
		t.Error("export still needs online math scripts")
	}
	if !strings.Contains(string(data), "<code>$not_math$</code>") {
		t.Error("math export modified code")
	}
}

func TestHTMLMathNumberingAndFailures(t *testing.T) {
	for _, mode := range []string{"none", "ams", "all"} {
		data, err := ExportHTMLOpts(context.Background(), "$$\nx\n$$\n\n$$\nx\n$$", "Math", false, MathOptions{Numbering: mode})
		if err != nil {
			t.Fatal(err)
		}
		if mode == "all" {
			if !bytes.Contains(data, []byte(`data-equation-number="1"`)) || !bytes.Contains(data, []byte(`data-equation-number="2"`)) {
				t.Errorf("numbering mode %s ignored", mode)
			}
		} else if bytes.Contains(data, []byte(`data-equation-number="1"`)) {
			t.Errorf("unrequested number for %s", mode)
		}
	}
	broken := "inline $\\badcommand$\n\n$$\nunclosed"
	data, err := ExportHTML(broken, "Math", false)
	if err != nil || !bytes.Contains(data, []byte(`$\badcommand$`)) || !bytes.Contains(data, []byte("$$\nunclosed")) {
		t.Errorf("failed math source lost: %s %v", data, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExportHTMLOpts(ctx, "$x$", "Math", false, MathOptions{}); !errors.Is(err, context.Canceled) {
		t.Errorf("ignored export cancellation: %v", err)
	}
}

func TestHTMLSingleLineMathKeepsFollowingParagraph(t *testing.T) {
	data, err := ExportHTML("$$x\\tag{1}\\label{a}$$\n\nSee $\\ref{a}$", "Math", false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`alt="x\tag{1}\label{a}"`)) || !bytes.Contains(data, []byte("<p>See ")) || bytes.Count(data, []byte("data:image/svg+xml;base64,")) != 2 {
		t.Fatalf("single-line formula damaged export: %s", data)
	}
}

func TestHTMLMathInImageAltRemainsLiteral(t *testing.T) {
	source := `![alt $\newcommand{\foo}{x}$](img.png)` + "\n\n$$\\foo$$"
	data, err := ExportHTML(source, "Math", false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`alt="alt $\newcommand{\foo}{x}$"`)) || !bytes.Contains(data, []byte(`class="math-error"`)) {
		t.Fatal("image alt math lost its literal source or changed visible equation context")
	}
	if expressions := Parse(source).MathExpressions; len(expressions) != 1 || expressions[0].TeX != `\foo` {
		t.Fatalf("image alt contributed phantom math: %+v", expressions)
	}
}

func TestHTMLMidParagraphDoubleDollarUsesDisplayMath(t *testing.T) {
	data, err := ExportHTMLOpts(context.Background(), "before $$x$$ after", "Math", false, MathOptions{Numbering: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`class="math-block"`)) || !bytes.Contains(data, []byte(`data-equation-number="1"`)) {
		t.Fatal("double-dollar formula lost display mode")
	}
	plan := Parse("before $$x$$ after").MathExpressions
	if len(plan) != 1 || !plan[0].Display {
		t.Fatalf("double-dollar formula lost display context: %+v", plan)
	}
}
