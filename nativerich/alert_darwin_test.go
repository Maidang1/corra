//go:build darwin && cgo

package nativerich

import (
	"strings"
	"testing"
)

// A callout is a quote whose first line is "[!LABEL]". The marker is ordinary
// text, so the document must round-trip byte for byte while the label rides
// along as paragraph metadata the renderer paints.
func TestNativeAlertRoundTrip(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		source := "> [!NOTE]\r\n>\r\n> body **bold**\r\n\r\nplain\r\n\r\n> quoted\r\n\r\n> [!TIP] one line\r\n\r\n> [!HINT] unknown"
		e.Load(source)
		if !nativeTestHasTraits(e, "quote") {
			t.Error("quotes were not recognised")
			return
		}
		for _, label := range []string{"NOTE", "TIP"} {
			if !nativeTestHasTraits(e, "alert:"+label) {
				t.Errorf("missing %s callout label", label)
			}
		}
		for _, label := range []string{"WARNING", "IMPORTANT"} {
			if nativeTestHasTraits(e, "alert:"+label) {
				t.Errorf("unexpected %s callout label", label)
			}
		}
		// "[!HINT]" is not a GitHub kind, so it stays plain quote text.
		if !strings.Contains(nativeTestText(e), "[!HINT] unknown") {
			t.Errorf("unknown marker was rewritten: %q", nativeTestText(e))
		}
		if got, dirty, err := e.Snapshot(); got != source || dirty || err != nil {
			t.Errorf("load changed source: %q dirty=%v err=%v", got, dirty, err)
		}
	})
}

// Typing a marker must produce the label, and deleting it must take the label
// away again without touching the rest of the paragraph. A quote's ">" markers
// are not part of the text, so offset 0 is the start of its content.
func TestNativeAlertMarkerEditing(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("> quoted body\r\n\r\nafter")
		if nativeTestHasTraits(e, "alert:NOTE") {
			t.Error("a plain quote reported a callout label")
			return
		}
		nativeTestSelect(e, 0, 0)
		nativeTestAction(e, "type", "[!NOTE] ")
		if !nativeTestHasTraits(e, "alert:NOTE") {
			t.Errorf("typing a marker did not register: %q", nativeTestText(e))
		}
		if got, _, _ := e.Snapshot(); got != "> [!NOTE] quoted body\r\n\r\nafter" {
			t.Errorf("marker edit lost structure: %q", got)
		}
		nativeTestSelect(e, 0, len("[!NOTE] "))
		nativeTestAction(e, "type", "")
		if nativeTestHasTraits(e, "alert:NOTE") {
			t.Errorf("deleting the marker left the label behind: %q", nativeTestText(e))
		}
		if got, _, _ := e.Snapshot(); got != "> quoted body\r\n\r\nafter" {
			t.Errorf("marker removal changed the paragraph: %q", got)
		}
	})
}

// The insert command rewrites the marker in place: asking for a second kind
// replaces the first, and an empty label removes it.
func TestNativeAlertInsertReplacesMarker(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		e.Load("body\r\n\r\nafter")
		nativeTestSelect(e, 0, 0)
		e.Format("quote-alert:WARNING")
		if !nativeTestHasTraits(e, "quote", "alert:WARNING") {
			t.Errorf("insert did not create the callout: %q", nativeTestText(e))
			return
		}
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "> [!WARNING] body") {
			t.Errorf("insert wrote the wrong marker: %q", got)
		}
		e.Format("quote-alert:CAUTION")
		if !nativeTestHasTraits(e, "alert:CAUTION") || nativeTestHasTraits(e, "alert:WARNING") {
			t.Errorf("second kind did not replace the first: %q", nativeTestText(e))
		}
		got, _, _ := e.Snapshot()
		if !strings.HasPrefix(got, "> [!CAUTION] body") || strings.Contains(got, "WARNING") {
			t.Errorf("replacement left the old marker: %q", got)
		}
		e.Format("quote-alert:")
		if nativeTestHasTraits(e, "alert:CAUTION") {
			t.Errorf("removal left the label behind: %q", nativeTestText(e))
		}
		if got, _, _ := e.Snapshot(); !strings.HasPrefix(got, "> body") {
			t.Errorf("removal changed the paragraph: %q", got)
		}
	})
}
