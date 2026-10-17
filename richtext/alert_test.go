package richtext

import "testing"

func TestQuoteAlertLabel(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want string
	}{
		{"", ""},
		{"plain quote", ""},
		{"> [!NOTE]\n> body", "NOTE"},
		{"> [!TIP] body", "TIP"},
		{"> [!important]\n> body", "IMPORTANT"},
		{"> [!Warning] body", "WARNING"},
		{"> [!caution]\n> body", "CAUTION"},
		{"> [!NOTE] only", "NOTE"},
		// Not a marker: missing bracket, unknown kind, or glued text.
		{"> [!NOTES] body", ""},
		{"> [!HINT] body", ""},
		{"> [!NOTE]body", ""},
		{"> note body", ""},
		// Nested quotes carry the marker too.
		{"> > [!NOTE]\n> > body", "NOTE"},
		// A callout may hold a list; code inside a quote is still source.
		{"> [!TIP]\n> - item", "TIP"},
		{"> [!TIP]\n> ```\n> x\n> ```", ""},
	} {
		got := ""
		for _, p := range Parse(tc.src).Paragraphs {
			if p.Alert != "" {
				got = p.Alert
			}
		}
		if got != tc.want {
			t.Errorf("Parse(%q): alert = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestQuoteAlertGroupSpansBlankMarkerLine(t *testing.T) {
	doc := Parse("> [!NOTE]\n> first\n>\n> second")
	if len(doc.Paragraphs) != 2 {
		t.Fatalf("paragraphs = %d, want 2", len(doc.Paragraphs))
	}
	for i, p := range doc.Paragraphs {
		if p.Alert != "NOTE" {
			t.Errorf("paragraph %d: alert = %q, want NOTE", i, p.Alert)
		}
	}
	// A blank source line starts a new blockquote, which is not an alert.
	doc = Parse("> [!NOTE]\n> first\n\n> second")
	if doc.Paragraphs[0].Alert != "NOTE" || doc.Paragraphs[1].Alert != "" {
		t.Errorf("alerts = %q, %q; want NOTE, \"\"",
			doc.Paragraphs[0].Alert, doc.Paragraphs[1].Alert)
	}
}

func TestQuoteAlertRoundTrip(t *testing.T) {
	for _, src := range []string{
		"> [!NOTE]\n> body text",
		"> [!NOTE]\n> body\n> more",
		"> [!NOTE] only",
		"> [!NOTE]\n> first\n>\n> second",
		"> > [!WARNING]\n> > body",
		"> [!NOTE]\n> body with *emphasis* and `code`",
		"> [!TIP]\r\n> body text\r\n",
	} {
		if got := Serialize(Parse(src)); got != src {
			t.Errorf("Serialize(Parse(%q)) = %q", src, got)
		}
	}
}
