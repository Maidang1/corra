package render

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
)

func TestPDFMathUsesDocumentContext(t *testing.T) {
	source := "# Document equations\n\n$$\n\\newcommand{\\RR}{\\mathbb{R}}\\RR^2\\label{space}\n$$\n\n$$\n\\eqref{space}: x\\in\\RR\n$$"
	doc := Parse(source)
	doc.MathOptions = MathOptions{Numbering: "all"}
	data, err := ExportPDF(context.Background(), doc, "Document equations")
	if err != nil {
		t.Fatal(err)
	}
	// Both equations must be images; singleton rendering cannot resolve \RR
	// in the second equation and would silently export its raw source.
	if count := bytes.Count(data, []byte("/SMask")); count < 2 {
		t.Errorf("document equations did not both render: %d", count)
	}
	if file := os.Getenv("CORRA_MATH_PDF"); file != "" {
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
		html, err := ExportHTMLOpts(context.Background(), source, "Document equations", false, doc.MathOptions)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file+".html", html, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExportPDF(ctx, doc, "Math"); !errors.Is(err, context.Canceled) {
		t.Errorf("ignored PDF cancellation: %v", err)
	}
}
