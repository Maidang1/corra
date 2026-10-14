package render

import (
	"fmt"
	"image"
	"image/color"
	"math"

	gg "git.sr.ht/~sbinet/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// chartPalette cycles through distinguishable, theme-neutral colors.
var chartPalette = []color.RGBA{
	{0x5b, 0x8d, 0xee, 0xff}, // blue
	{0xef, 0x8f, 0x5a, 0xff}, // orange
	{0x5c, 0xb8, 0x7c, 0xff}, // green
	{0xc9, 0x7a, 0xdb, 0xff}, // purple
	{0x5f, 0xc2, 0xc9, 0xff}, // teal
	{0xd9, 0x6a, 0x7c, 0xff}, // rose
}

const (
	chartW        = 640
	chartH        = 360
	chartFontSize = 12
)

var chartFaces = map[int]font.Face{}

// notoFace returns a cached Noto Sans SC face (embedded subset with CJK).
func notoFace(size float64) font.Face {
	key := int(size * 10)
	if f, ok := chartFaces[key]; ok {
		return f
	}
	f, err := opentype.NewFace(notoFont, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		panic(err)
	}
	chartFaces[key] = f
	return f
}

// RenderChartImage draws a ChartSpec to an image with gg (pure Go), shared
// by the preview pane (bitmap) and the PDF exporter.
func RenderChartImage(spec *ChartSpec, scale float64) (image.Image, error) {
	w, h := int(chartW*scale), int(chartH*scale)
	dc := gg.NewContext(w, h)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	dc.Scale(scale, scale)
	dc.SetFontFace(notoFace(chartFontSize))

	padL, padR, padT, padB := 52.0, 18.0, 30.0, 42.0
	if spec.Title != "" {
		padT += 22
	}
	cw, ch := float64(chartW)-padL-padR, float64(chartH)-padT-padB

	switch spec.Type {
	case "line":
		chartAxes(dc, spec, padL, padT, cw, ch)
		chartLine(dc, spec, padL, padT, cw, ch)
	case "bar", "hbar":
		chartAxes(dc, spec, padL, padT, cw, ch)
		chartBars(dc, spec, padL, padT, cw, ch, spec.Type == "hbar")
	case "pie":
		chartPie(dc, spec, padL, padT, cw, ch)
	}

	if spec.Title != "" {
		dc.SetRGB(0.15, 0.16, 0.19)
		dc.DrawStringAnchored(spec.Title, chartW/2, 18, 0.5, 0.5)
	}
	return dc.Image(), nil
}

// chartMax is the largest value across all series (at least 1).
func chartMax(spec *ChartSpec) float64 {
	m := 1.0
	for _, s := range spec.Series {
		for _, v := range s.Data {
			if v > m {
				m = v
			}
		}
	}
	return m * 1.1
}

func niceTicks(max float64, n int) []float64 {
	step := max / float64(n)
	mag := math.Pow(10, math.Floor(math.Log10(step)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if step/mag <= m {
			step = m * mag
			break
		}
	}
	var ticks []float64
	for v := step; v < max; v += step {
		ticks = append(ticks, v)
	}
	return ticks
}

// chartAxes draws the y gridlines, y tick labels and x labels.
func chartAxes(dc *gg.Context, spec *ChartSpec, padL, padT, cw, ch float64) {
	max := chartMax(spec)
	dc.SetRGB(0.88, 0.89, 0.91)
	dc.SetLineWidth(1)
	for _, t := range niceTicks(max, 5) {
		y := padT + ch - ch*t/max
		dc.DrawLine(padL, y, padL+cw, y)
		dc.Stroke()
		dc.SetRGB(0.45, 0.47, 0.52)
		dc.DrawStringAnchored(fmt.Sprintf("%g", t), padL-6, y, 1, 0.35)
		dc.SetRGB(0.88, 0.89, 0.91)
	}
	n := float64(len(spec.Labels))
	dc.SetRGB(0.45, 0.47, 0.52)
	for i, l := range spec.Labels {
		x := padL + cw*(float64(i)+0.5)/n
		dc.DrawStringAnchored(l, x, padT+ch+16, 0.5, 0.5)
	}
}

// chartLine draws one polyline per series with point markers and a legend.
func chartLine(dc *gg.Context, spec *ChartSpec, padL, padT, cw, ch float64) {
	max := chartMax(spec)
	n := float64(len(spec.Labels))
	for si, s := range spec.Series {
		c := chartPalette[si%len(chartPalette)]
		dc.SetRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0.95)
		dc.SetLineWidth(2)
		dc.MoveTo(padL+cw*0.5/n, padT+ch-ch*s.Data[0]/max)
		for i, v := range s.Data {
			x := padL + cw*(float64(i)+0.5)/n
			y := padT + ch - ch*v/max
			dc.LineTo(x, y)
		}
		dc.Stroke()
		for i, v := range s.Data {
			x := padL + cw*(float64(i)+0.5)/n
			y := padT + ch - ch*v/max
			dc.DrawArc(x, y, 3, 0, 2*math.Pi)
			dc.Fill()
		}
	}
	chartLegend(dc, spec, padL+cw, padT)
}

// chartBars draws grouped bars (vertical or horizontal).
func chartBars(dc *gg.Context, spec *ChartSpec, padL, padT, cw, ch float64, horiz bool) {
	max := chartMax(spec)
	n := float64(len(spec.Labels))
	m := float64(len(spec.Series))
	gap := 0.25
	for si, s := range spec.Series {
		c := chartPalette[si%len(chartPalette)]
		dc.SetRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0.92)
		for i, v := range s.Data {
			if horiz {
				bh := ch / n / (m + gap)
				x := padL
				y := padT + ch*float64(i)/n + bh*(float64(si)+gap/2)
				w := cw * v / max
				dc.DrawRectangle(x, y, w, bh*0.9)
			} else {
				bw := cw / n / (m + gap)
				x := padL + cw*float64(i)/n + bw*(float64(si)+gap/2)
				y := padT + ch - ch*v/max
				h := ch * v / max
				dc.DrawRectangle(x, y, bw*0.9, h)
			}
			dc.Fill()
		}
	}
	chartLegend(dc, spec, padL+cw, padT)
}

// chartPie draws a pie chart of the first series with percentages.
func chartPie(dc *gg.Context, spec *ChartSpec, padL, padT, cw, ch float64) {
	s := spec.Series[0]
	total := 0.0
	for _, v := range s.Data {
		total += v
	}
	if total <= 0 {
		return
	}
	cx, cy := padL+cw/2-60, padT+ch/2
	r := math.Min(cw, ch)/2 - 8
	angle := -math.Pi / 2
	for i, v := range s.Data {
		c := chartPalette[i%len(chartPalette)]
		dc.SetRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0.95)
		sweep := 2 * math.Pi * v / total
		dc.MoveTo(cx, cy)
		dc.DrawArc(cx, cy, r, angle, angle+sweep)
		dc.ClosePath()
		dc.Fill()
		angle += sweep
	}
	lx := cx + r + 18
	ly := cy - float64(len(s.Data))*10 + 6
	for i, v := range s.Data {
		c := chartPalette[i%len(chartPalette)]
		dc.SetRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1)
		dc.DrawRectangle(lx, ly+float64(i)*20-5, 10, 10)
		dc.Fill()
		dc.SetRGB(0.25, 0.26, 0.3)
		label := ""
		if i < len(spec.Labels) {
			label = spec.Labels[i]
		} else {
			label = fmt.Sprintf("%s %d", s.Name, i+1)
		}
		dc.DrawString(fmt.Sprintf("%s (%.0f%%)", label, 100*v/total), lx+16, ly+float64(i)*20+4)
	}
}

// chartLegend draws a color-chip legend at the top right.
func chartLegend(dc *gg.Context, spec *ChartSpec, right, top float64) {
	if len(spec.Series) < 2 {
		return
	}
	lx := right - 90
	ly := top + 6
	for si, s := range spec.Series {
		c := chartPalette[si%len(chartPalette)]
		dc.SetRGBA(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1)
		dc.DrawRectangle(lx, ly+float64(si)*18-5, 10, 10)
		dc.Fill()
		dc.SetRGB(0.25, 0.26, 0.3)
		dc.DrawString(s.Name, lx+16, ly+float64(si)*18+4)
	}
}
