package richtext

import (
	"os"
	"strings"
	"testing"
)

func TestCorpusSourceFallback(t *testing.T) {
	files := strings.Fields(os.Getenv("CORRA_CORPUS"))
	if len(files) == 0 {
		t.Skip()
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		doc := Parse(string(data))
		for _, p := range doc.Paragraphs {
			if p.Kind == "source" || p.Kind == "opaque" {
				line := strings.SplitN(p.Raw, "\n", 2)[0]
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "---") && !strings.HasPrefix(strings.TrimSpace(line), "***") {
					t.Logf("%s: %s %q", f, p.Kind, line)
				}
			}
		}
		if Serialize(doc) != string(data) {
			t.Logf("%s: ROUND TRIP DIFFERS", f)
		}
	}
}

// TestCorpusRewrite re-serializes every paragraph of real notes as if each
// had been edited: the visible text and block kinds must survive, and the
// result must be a fixed point. Run: CORRA_CORPUS="a.md b.md" go test ...
func TestCorpusRewrite(t *testing.T) {
	files := strings.Fields(os.Getenv("CORRA_CORPUS"))
	if len(files) == 1 && strings.HasPrefix(files[0], "@") {
		list, _ := os.ReadFile(files[0][1:]) // CORRA_CORPUS=@paths.txt
		files = strings.Fields(string(list))
	}
	if len(files) == 0 {
		t.Skip()
	}
	shape := func(doc Document) string {
		var b strings.Builder
		for _, p := range doc.Paragraphs {
			if p.Kind == "source" || p.Kind == "opaque" {
				continue
			}
			text := PlainText(p)
			if p.Kind != "table" {
				var runs strings.Builder
				for _, run := range p.Runs {
					if run.Break {
						runs.WriteByte(' ') // a hard break and a wrap both read as a gap
					}
					runs.WriteString(run.Text)
				}
				text = runs.String()
			}
			b.WriteString(p.Kind + ":" + strings.Join(strings.Fields(text), " ") + "\n")
		}
		return b.String()
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		doc := Parse(string(data))
		for i := range doc.Paragraphs {
			p := &doc.Paragraphs[i]
			switch p.Kind {
			case "source", "opaque", "math", "image", "diagram", "toc":
				// These are written from their raw source.
			default:
				p.Raw, p.Baseline, p.ListBaseline = "", "", ""
			}
		}
		saved := Serialize(doc)
		again := Parse(saved)
		if shape(again) != shape(doc) {
			a, b := strings.Split(shape(doc), "\n"), strings.Split(shape(again), "\n")
			for i := range a {
				if i >= len(b) || a[i] != b[i] {
					got := ""
					if i < len(b) {
						got = b[i]
					}
					t.Errorf("%s: paragraph %d\n want %q\n  got %q", f, i, a[i], got)
					break
				}
			}
			continue
		}
		if Serialize(again) != saved {
			t.Errorf("%s: not a fixed point", f)
		}
	}
}
