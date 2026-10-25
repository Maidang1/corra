package render

import (
	"strings"
	"testing"
)

func TestExportHTMLColorSpan(t *testing.T) {
	out, err := ExportHTML(`a <span style="color: #D44C47">red</span> b`, "t", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `<span style="color:#d44c47">red</span>`) {
		t.Fatalf("export lost the color span:\n%s", out)
	}
}
