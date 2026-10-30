package views

import (
	"strings"
	"testing"
)

const mygoCorpus = "# 标题\n\n正文 **粗** ==高亮== x^2^\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n$$\nx^2\n$$\n\n> [!NOTE]\n> hi\n\n---\n\n```go\nfmt.Println()\n```\n"

func TestMygoRichRoundTrip(t *testing.T) {
	m := NewMygoRich(func() {}, nil, func() {})
	m.Load(mygoCorpus)
	if _, changed, _ := m.Snapshot(); changed {
		t.Fatal("unchanged doc reported changed")
	}
	m.doc.SetSelection(0, 2)
	m.Format("bold")
	out, changed, _ := m.Snapshot()
	if !changed {
		t.Fatal("bold did not change doc")
	}
	for _, w := range []string{"| a | b |", "$$\nx^2\n$$", "[!NOTE]", "```go", "==高亮=="} {
		if !strings.Contains(out, w) {
			t.Errorf("lost %q in:\n%s", w, out)
		}
	}
}

func TestMygoRichInsertAndFind(t *testing.T) {
	n := 0
	m := NewMygoRich(func() { n++ }, nil, func() {})
	m.Load("hello world hello")
	m.InsertMarkdown(tableTemplate)
	out, _, _ := m.Snapshot()
	if !strings.Contains(out, "| 列 1 | 列 2 |") {
		t.Fatalf("table missing: %q", out)
	}
	m.find.query = "hello"
	m.find.repl = "hi"
	m.replaceAll()
	out, _, _ = m.Snapshot()
	if strings.Contains(out, "hello") || !strings.Contains(out, "hi world hi") {
		t.Fatalf("replace failed: %q", out)
	}
	if n == 0 {
		t.Fatal("onChange not called")
	}
}
