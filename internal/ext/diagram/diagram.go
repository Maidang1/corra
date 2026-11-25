// Package diagram previews code blocks in diagram languages (mermaid, d2,
// goat, flowchart.js, sequence) and ```chart data under their source.
package diagram

import (
	"context"
	"image"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/render"
)

var kinds = map[string]render.DiagramKind{
	"mermaid": render.DiagramMermaid, "d2": render.DiagramD2, "goat": render.DiagramGoAT,
	"flow": render.DiagramFlowchartJS, "sequence": render.DiagramSequenceJS,
}

// Extension adds the previews to the code block's view; the code itself
// stays a code block, so its Markdown is untouched.
func Extension() ext.Extension {
	base := editor.DefaultViews()["code_block"].Text
	return ext.Extension{
		Name:     "diagram",
		Priority: 1,
		Views: map[string]editor.ViewSpec{"code_block": {Text: func(n *doc.Node, env *editor.Env) editor.TextLook {
			l := base(n, env)
			lang := n.Attrs().String("language")
			if _, ok := kinds[lang]; !ok && lang != "chart" {
				return l
			}
			src := n.Inline().String()
			pv := preview(lang, src, env.Refresh)
			l.Below = func(width float32) (float32, func(*ui.Painter, ui.Rect, *editor.Env)) {
				bmp, err, done := pv.result()
				switch {
				case bmp != nil:
					w, h := bmp.Size()
					fw, fh := float32(w)/2, float32(h)/2 // rasterized at 2x
					if fw > width-16 {
						fh, fw = fh*(width-16)/fw, width-16
					}
					return fh + 16, func(p *ui.Painter, r ui.Rect, env *editor.Env) {
						p.Fill(ui.Rect{X: r.X, Y: r.Y + 4, W: r.W, H: r.H - 8}, ui.Hex("#ffffff"), 6)
						p.Image(bmp, ui.Rect{X: r.X + (r.W-fw)/2, Y: r.Y + 8, W: fw, H: fh}, ui.Contain)
					}
				case done && err != nil:
					msg := err.Error()
					return env.Font.Size * 1.6, func(p *ui.Painter, r ui.Rect, env *editor.Env) {
						p.Text(r.X+8, r.Y+env.Font.Size*1.1, "预览失败："+msg, env.Font.Size*0.75, env.Theme.Danger)
					}
				default:
					return env.Font.Size * 1.6, func(p *ui.Painter, r ui.Rect, env *editor.Env) {
						p.Text(r.X+8, r.Y+env.Font.Size*1.1, "正在生成预览…", env.Font.Size*0.75, env.Theme.TextMuted)
					}
				}
			}
			return l
		}}},
	}
}

// previews are rendered once per language and source, in the background,
// and shared.
type rendering struct {
	mu   sync.Mutex
	done bool
	err  error
	img  image.Image
	bmp  *ui.Bitmap
}

func (r *rendering) result() (*ui.Bitmap, error, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done && r.err == nil && r.bmp == nil && r.img != nil {
		r.bmp = ui.NewBitmap(r.img)
	}
	return r.bmp, r.err, r.done
}

var (
	cacheMu sync.Mutex
	cache   = map[string]*rendering{}
)

func preview(lang, src string, refresh editor.Refresher) *rendering {
	key := lang + "\x00" + src
	cacheMu.Lock()
	if r, ok := cache[key]; ok {
		cacheMu.Unlock()
		return r
	}
	r := &rendering{}
	cache[key] = r
	cacheMu.Unlock()
	go func() {
		var img image.Image
		var err error
		if lang == "chart" {
			var spec *render.ChartSpec
			if spec, err = render.ParseChart(src); err == nil {
				img, err = render.RenderChartImage(spec, 2)
			}
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			var svg []byte
			if svg, err = render.RenderDiagram(ctx, kinds[lang], src); err == nil {
				img, err = render.RasterizeSVG(svg, 2)
			}
			cancel()
		}
		r.mu.Lock()
		r.img, r.err, r.done = img, err, true
		r.mu.Unlock()
		if refresh != nil {
			refresh.Refresh()
		}
	}()
	return r
}
