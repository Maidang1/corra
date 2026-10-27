package views

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Maidang1/corra/app"
)

func TestMetadataTagsReplaceFrontMatterInRichBody(t *testing.T) {
	v, tt := notesFixture(t)
	prefix := "---\r\n# keep comment\r\ntitle: 图解 Transformer\r\nauthor: Jay Alammar\r\nsource: https://jalammar.github.io/illustrated-transformer/\r\npublished: 2018-06-27\r\nlicense: CC BY-NC-SA 4.0\r\nfetched: 2026-10-26\r\ntags: [AI, 阅读]\r\n---\r\n\r\n"
	v.App.Loaded(v.App.Path, prefix+"# 正文\r\n\r\n内容\r\n")
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	if strings.Contains(e.source, "title:") || !strings.HasPrefix(e.source, "# 正文") {
		t.Fatalf("front matter still in editor body: %q", e.source)
	}
	for _, label := range []string{"标题：图解 Transformer", "作者：Jay Alammar", "发布：2018-06-27", "#AI", "#阅读"} {
		if !tt.HasText(label) {
			t.Errorf("metadata tag missing: %s", label)
		}
	}
	if dir := os.Getenv("CORRA_SCREENSHOTS"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, theme := range []app.ThemeMode{app.ThemeLight, app.ThemeDark} {
			v.App.ThemeMode = theme
			tt.Frame()
			name := "metadata-light.png"
			if theme == app.ThemeDark {
				name = "metadata-dark.png"
			}
			file, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(file, tt.Image())
			_ = file.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	e.source = "# 改动\r\n"
	e.changed = true
	if !v.FlushAutosave() {
		t.Fatal(v.SaveError)
	}
	if got := v.App.CurrentSource(); got != prefix+"# 改动\r\n" {
		t.Fatalf("metadata bytes lost on save: %q", got)
	}
	if data, err := os.ReadFile(v.App.Path); err != nil || string(data) != prefix+"# 改动\r\n" {
		t.Fatalf("saved file lost front matter: %q %v", data, err)
	}
	v.EditSource()
	tt.Frame()
	if v.App.Mode != app.ModeEdit || v.edState.Buffer.String() != strings.ReplaceAll(prefix+"# 改动\r\n", "\r\n", "\n") || v.App.Source != prefix+"# 改动\r\n" {
		t.Fatalf("source mode did not retain full front matter: mode=%v buffer=%q source=%q", v.App.Mode, v.edState.Buffer.String(), v.App.Source)
	}
}

func TestMetadataRefreshesAfterSourceEditAndDocumentSwitch(t *testing.T) {
	v, tt := notesFixture(t)
	v.App.Loaded(v.App.Path, "---\nauthor: First\n---\n\n# Body")
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	v.EditSource()
	v.edState.Select(0, v.edState.Buffer.Len())
	v.edState.Insert("---\nauthor: Second\n---\n\n# Body")
	v.RichAction()
	tt.Frame()
	if !tt.HasText("作者：Second") || tt.HasText("作者：First") || e.source != "# Body" {
		t.Fatal("source edit did not refresh metadata and native body")
	}
	v.App.Loaded(v.App.Path, "# No metadata")
	tt.Frame()
	e.source, e.changed = "# Changed", true
	if !v.syncEditor() || v.App.Source != "# Changed" || tt.HasText("作者：Second") {
		t.Fatal("metadata leaked into a document without front matter")
	}
}

func TestMalformedMetadataRemainsVisible(t *testing.T) {
	for _, src := range []string{"---\ntitle: [broken\n---\n\nbody", "---\n- a\n- b\n---\nbody", "---\ntitle: unclosed\nbody"} {
		v, tt := notesFixture(t)
		v.App.Loaded(v.App.Path, src)
		e := &fakeRichEditor{}
		v.AttachRichEditor(e)
		tt.Frame()
		if e.source != src {
			t.Fatalf("unrecognized metadata hidden: %q", e.source)
		}
	}
}

func TestMetadataOnlyDocumentCanGainBody(t *testing.T) {
	v, tt := notesFixture(t)
	src := "---\ntitle: Empty\n---"
	v.App.Loaded(v.App.Path, src)
	e := &fakeRichEditor{}
	v.AttachRichEditor(e)
	tt.Frame()
	if e.source != "" {
		t.Fatalf("metadata-only body = %q", e.source)
	}
	e.source = "new body"
	e.changed = true
	if !v.syncEditor() {
		t.Fatal(v.SaveError)
	}
	if v.App.Source != src+"\n\nnew body" {
		t.Fatalf("missing boundary after metadata: %q", v.App.Source)
	}
}
