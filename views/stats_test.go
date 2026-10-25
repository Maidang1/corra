package views

import "testing"

func TestInspectDocument(t *testing.T) {
	src := "# Title\n\n## Sub\n\nA [link](x.md) and ![img](a.png).\n\n- [ ] task\n- [x] done\n\n```go\n# not a heading\n![not](img.png)\n```\n"
	stats := inspectDocument(src)
	if stats.Headings != 2 {
		t.Errorf("headings = %d", stats.Headings)
	}
	if stats.Links != 1 {
		t.Errorf("links = %d", stats.Links)
	}
	if stats.Images != 1 {
		t.Errorf("images = %d", stats.Images)
	}
	if stats.Tasks != 2 {
		t.Errorf("tasks = %d", stats.Tasks)
	}
	if stats.Code != 1 {
		t.Errorf("code = %d", stats.Code)
	}
	if empty := inspectDocument(""); empty != (documentStats{}) {
		t.Errorf("empty = %+v", empty)
	}
}

func TestDocumentStatsStrip(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.WordCount = true
	tt.Frame()
	if tt.HasText("阅读") {
		t.Fatal("stats strip is open by default")
	}
	// Clicking the status-bar count expands the strip.
	click(t, tt, "文档统计")
	tt.Frame()
	if !v.StatsOpen {
		t.Fatal("stats strip did not open")
	}
	if !tt.HasText("字数") || !tt.HasText("标题") || !tt.HasText("阅读") {
		t.Fatal("stats chips missing")
	}
	// Clicking again collapses it.
	click(t, tt, "文档统计")
	tt.Frame()
	if v.StatsOpen {
		t.Fatal("stats strip did not close")
	}
}
