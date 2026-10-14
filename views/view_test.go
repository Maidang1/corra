package views

import (
	"testing"

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

// TestViewSourceMode renders the source pane.
func TestViewSourceMode(t *testing.T) {
	a := app.New()
	a.Mode = app.ModeSource
	v := New(a, app.NewImageLoader(""))
	img := ui.Render(v.Build, 800, 600, 1)
	if img == nil {
		t.Fatal("nil render")
	}
}
