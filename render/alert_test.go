package render

import (
	"context"
	"strings"
	"testing"
)

// A callout is a blockquote whose first line is "[!LABEL]". The HTML export
// turns it into a tinted card with the label as its title, and the marker must
// not be printed next to that title.
func TestExportHTMLAlerts(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{"note", "> [!NOTE]\n>\n> body", "<div class=\"alert alert-note\">\n<p class=\"alert-title\">Note</p>\n<p>body</p>\n</div>"},
		{"tip", "> [!TIP] inline body", "<div class=\"alert alert-tip\">\n<p class=\"alert-title\">Tip</p>\n<p>inline body</p>\n</div>"},
		{"important", "> [!IMPORTANT]\n>\n> body", "<div class=\"alert alert-important\">\n<p class=\"alert-title\">Important</p>\n<p>body</p>\n</div>"},
		{"warning", "> [!WARNING] body", "<div class=\"alert alert-warning\">\n<p class=\"alert-title\">Warning</p>\n<p>body</p>\n</div>"},
		{"caution", "> [!caution]\n>\n> body", "<div class=\"alert alert-caution\">\n<p class=\"alert-title\">Caution</p>\n<p>body</p>\n</div>"},
		{"several paragraphs", "> [!NOTE]\n>\n> first\n>\n> second", "<div class=\"alert alert-note\">\n<p class=\"alert-title\">Note</p>\n<p>first</p>\n<p>second</p>\n</div>"},
		{"inline styles", "> [!NOTE] **bold** and `code`", "<div class=\"alert alert-note\">\n<p class=\"alert-title\">Note</p>\n<p><strong>bold</strong> and <code>code</code></p>\n</div>"},
		{"list body", "> [!NOTE]\n>\n> - item\n> - item", "<div class=\"alert alert-note\">\n<p class=\"alert-title\">Note</p>\n<ul>\n<li>item</li>\n<li>item</li>\n</ul>\n</div>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := ExportHTML(tc.source, "T", false)
			if err != nil {
				t.Fatal(err)
			}
			html := string(out)
			if !strings.Contains(html, tc.want) {
				t.Errorf("missing callout card\nwant: %s\ngot:\n%s", tc.want, mainBody(html))
			}
			if strings.Contains(html, "[!") || strings.Contains(html, "<blockquote>") {
				t.Errorf("marker or plain quote leaked into\n%s", mainBody(html))
			}
			if !strings.Contains(html, ".alert-note{") {
				t.Error("missing callout CSS")
			}
		})
	}
}

// mainBody trims the HTML export down to what it rendered, so a failure shows
// the document instead of the whole style sheet.
func mainBody(html string) string {
	body := html
	if i := strings.Index(body, "<main"); i >= 0 {
		body = body[i:]
		if j := strings.Index(body, "\n</main>"); j >= 0 {
			body = body[:j]
		}
	}
	return strings.TrimSpace(body)
}

// A quote without a marker, and a marker that is not a GitHub kind, stay plain
// blockquotes with their text untouched.
func TestExportHTMLPlainQuotes(t *testing.T) {
	for _, source := range []string{"> quoted", "> [!HINT]\n>\n> body", "> [!NOTES] body"} {
		out, err := ExportHTML(source, "T", false)
		if err != nil {
			t.Fatal(err)
		}
		html := mainBody(string(out))
		if !strings.Contains(html, "<blockquote>") {
			t.Errorf("plain quote lost its element: %s", html)
		}
		if strings.Contains(html, "alert-title") {
			t.Errorf("non-callout rendered as a callout: %s", html)
		}
		if strings.HasPrefix(source, "> [!") && !strings.Contains(html, "[!") {
			t.Errorf("unknown marker was dropped: %s", html)
		}
	}
}

// The block IR keeps the quote's own source untouched and reports the label,
// with the callout body in the items and the marker out of them.
func TestParseAlertBlocks(t *testing.T) {
	doc := Parse("> [!WARNING]\n>\n> body **bold**\n\nplain\n\n> quoted")
	var alerts, quotes int
	for _, b := range doc.Blocks {
		switch {
		case b.Kind != KindQuote:
		case b.Alert != "":
			alerts++
			if !strings.HasPrefix(b.Source, "> [!WARNING]") {
				t.Errorf("alert lost its source: %q", b.Source)
			}
			if len(b.Items) != 1 || strings.Contains(b.Items[0].Source, "[!") || !strings.Contains(b.Items[0].Source, "bold") {
				t.Errorf("alert items wrong: %+v", b.Items)
			}
		default:
			quotes++
			if !strings.Contains(b.Items[0].Source, "quoted") {
				t.Errorf("plain quote items wrong: %+v", b.Items)
			}
		}
	}
	if alerts != 1 || quotes != 1 {
		t.Errorf("found %d alerts and %d plain quotes", alerts, quotes)
	}
}

// A callout exports to PDF with its label as a title. Quotes used to export
// nothing at all, because a container block owns no source lines.
func TestExportPDFAlert(t *testing.T) {
	doc := Parse("> [!NOTE]\n>\n> body text\n\n> quoted text")
	data, err := ExportPDF(context.Background(), doc, "T")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 10000 {
		t.Errorf("callout PDF too small to hold the quote text: %d bytes", len(data))
	}
}
