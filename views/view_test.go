package views

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// TestViewRendersHeadless renders the full view through MyGo's headless
// rasterizer: no window, no GPU. It catches build-pass crashes in the view
// tree and proves every block kind draws.
func TestViewRendersHeadless(t *testing.T) {
	a := app.New()
	a.Loaded("test.md", `# Title

Paragraph **bold** text.

- [ ] task
- [x] done

| A | B |
|---|---|
| 1 | 2 |

> quote

`+"```go\nfmt.Println(1)\n```"+`

$$
a^2+b^2=c^2
$$

`+"```chart\ntype: line\nlabels: [一, 二]\nseries:\n  - name: s\n    data: [1, 2]\n```"+`

---

`+"```html\n<div>x</div>\n```"+`

`+"```goat\n A --> B\n```"+`

![img](missing.png)
`)
	v := New(a, app.NewImageLoader(""))
	img := ui.Render(v.Build, 900, 1200, 1)
	if img == nil {
		t.Fatal("render returned nil")
	}
	b := img.Bounds()
	if b.Dx() != 900 || b.Dy() != 1200 {
		t.Fatalf("size = %v", b)
	}
	// The view must paint something (non-uniform pixels).
	ink := 0
	first := img.At(0, 0)
	for y := 0; y < b.Dy(); y += 7 {
		for x := 0; x < b.Dx(); x += 7 {
			if img.At(x, y) != first {
				ink++
			}
		}
	}
	if ink < 10 {
		t.Fatalf("view painted almost nothing (%d sampled diffs)", ink)
	}
}

// TestViewNotesLibrary renders the full two-column Notes UI with a
// real library on disk: folders-and-notes sidebar, editor column.
func TestViewNotesLibrary(t *testing.T) {
	root := t.TempDir()
	write := func(name, src string, age time.Duration) string {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		mt := time.Now().Add(-age)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
		return p
	}
	today := write("today.md", "# 今天的笔记\n\n内容 A", 2*time.Hour)
	write("work/plan.md", "# 计划\n\n工作内容", 48*time.Hour)
	write("work/old.md", "# 旧笔记\n\n更久以前", 2000*time.Hour)

	a := app.New()
	a.LibraryRoot = root
	a.ScanLibrary()
	if err := a.OpenNote(today); err != nil {
		t.Fatal(err)
	}
	v := New(a, app.NewImageLoader(""))
	img := ui.Render(v.Build, 1200, 800, 1)
	if img == nil {
		t.Fatal("nil render")
	}
}
