package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRebaseAttachmentPaths(t *testing.T) {
	root := t.TempDir()
	oldDir, newDir := root, filepath.Join(root, "Work")
	for _, tc := range []struct{ name, source, want string }{
		{"inline image", "![a](assets/a.png)", "![a](../assets/a.png)"},
		{"angle and title", "![a](<assets/a b.png> \"caption\")", "![a](<../assets/a b.png> \"caption\")"},
		{"link suffix", "[other](other.md?view=1#heading)", "[other](../other.md?view=1#heading)"},
		{"nested label", "[**other**](other.md)", "[**other**](../other.md)"},
		{"reference", "![a][pic]\n\n[pic]: assets/a.png \"caption\"\n", "![a][pic]\n\n[pic]: ../assets/a.png \"caption\"\n"},
		{"escaped destination", "![a](assets/a\\(1\\).png)", "![a](../assets/a%281%29.png)"},
		{"unchanged external", "[web](https://example.test/a) ![d](data:image/png;base64,AAA) [anchor](#part) [abs](/tmp/a.png) [net](//example.test/a)", "[web](https://example.test/a) ![d](data:image/png;base64,AAA) [anchor](#part) [abs](/tmp/a.png) [net](//example.test/a)"},
		{"code and prose", "assets/a.png `![a](assets/a.png)`\n\n```md\n![a](assets/a.png)\n```\n\n    [a](other.md)\n", "assets/a.png `![a](assets/a.png)`\n\n```md\n![a](assets/a.png)\n```\n\n    [a](other.md)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := rebaseAttachmentPaths(tc.source, oldDir, newDir); got != tc.want {
				t.Fatalf("rebased source = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMoveNoteAttachmentRoundTrip(t *testing.T) {
	a := New()
	a.LibraryRoot = t.TempDir()
	path := writeNote(t, a.LibraryRoot, "note.md", "![a](assets/a.png)\n[other](other.md)\n", 0)
	if err := a.OpenNote(path); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateFolder("Work"); err != nil {
		t.Fatal(err)
	}
	if err := a.MoveSelectedNote("Work"); err != nil {
		t.Fatal(err)
	}
	want := "![a](../assets/a.png)\n[other](../other.md)\n"
	assertNoteSource(t, a.Path, want)
	if a.Source != want || a.Dirty {
		t.Fatalf("moved editor source = %q dirty=%v", a.Source, a.Dirty)
	}
	if err := a.DeleteSelectedNote(); err != nil {
		t.Fatal(err)
	}
	a.SelFolder = TrashName
	notes := a.VisibleNotes()
	if len(notes) != 1 {
		t.Fatalf("trash notes = %d", len(notes))
	}
	if err := a.OpenNote(notes[0].Path); err != nil {
		t.Fatal(err)
	}
	assertNoteSource(t, a.Path, want)
	if err := a.RestoreSelectedNote(); err != nil {
		t.Fatal(err)
	}
	assertNoteSource(t, a.Path, "![a](assets/a.png)\n[other](other.md)\n")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
