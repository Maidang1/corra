package views

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Maidang1/corra/app"
)

func TestLinkCandidates(t *testing.T) {
	dir := t.TempDir()
	notes := []app.Note{
		{Path: filepath.Join(dir, "notes", "b.md"), Title: "Beta"},
		{Path: filepath.Join(dir, "notes", "sub", "c.md"), Title: "Gamma"},
		{Path: filepath.Join(dir, "notes", "a.md"), Title: "Self"},
	}
	v := &View{App: &app.App{Path: filepath.Join(dir, "notes", "a.md"), Lib: &app.Library{Notes: notes}}}

	type candidate struct{ Title, Path string }
	decode := func(raw string) []candidate {
		var out []candidate
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatalf("bad JSON %q: %v", raw, err)
		}
		return out
	}

	// Every note except the open one, with paths relative to it.
	all := decode(v.LinkCandidates(""))
	if len(all) != 2 || all[0].Title != "Beta" || all[0].Path != "b.md" ||
		all[1].Title != "Gamma" || all[1].Path != "sub/c.md" {
		t.Fatalf("candidates = %+v", all)
	}
	if got := decode(v.LinkCandidates("gam")); len(got) != 1 || got[0].Title != "Gamma" {
		t.Fatalf("filtered = %+v", got)
	}
	if got := decode(v.LinkCandidates("  GAM ")); len(got) != 1 {
		t.Fatalf("case/space filter = %+v", got)
	}
	if got := v.LinkCandidates(""); got == "" {
		t.Fatal("empty result encoded as an empty string")
	}
	if got := (&View{App: &app.App{}}).LinkCandidates("x"); got != "[]" {
		t.Fatalf("missing library = %q", got)
	}
}
