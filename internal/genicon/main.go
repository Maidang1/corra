// Command genicon draws the app icon (1024×1024 PNG) deterministically in
// pure Go, so the repository carries no binary assets. Run before
// packaging: go run ./internal/genicon
//
// The mark: an ink-blue macOS squircle with a fountain-pen nib — Corra is
// a writing tool. The nib's slit and breather hole are punched out with an
// even-odd fill, so no masks or secondary passes are needed.
package main

import (
	"image/color"
	"os"

	"git.sr.ht/~sbinet/gg"
)

const (
	size   = 1024.0
	radius = 228.0 // macOS icon corner, ~22.3% of the side
)

func main() {
	dc := gg.NewContext(int(size), int(size))

	// Squircle tile: a vertical ink gradient, light at the top like a
	// backlit key, deep navy at the foot.
	tile := gg.NewLinearGradient(0, 0, 0, size)
	tile.AddColorStop(0, color.RGBA{86, 141, 255, 255})
	tile.AddColorStop(0.55, color.RGBA{38, 92, 235, 255})
	tile.AddColorStop(1, color.RGBA{13, 46, 138, 255})
	dc.DrawRoundedRectangle(0, 0, size, size, radius)
	dc.SetFillStyle(tile)
	dc.Fill()

	// Top sheen: a soft radial light falling off by the icon's middle.
	dc.DrawRoundedRectangle(0, 0, size, size, radius)
	dc.Clip()
	sheen := gg.NewRadialGradient(size/2, -140, 60, size/2, -140, 760)
	sheen.AddColorStop(0, color.RGBA{255, 255, 255, 44})
	sheen.AddColorStop(1, color.RGBA{255, 255, 255, 0})
	dc.SetFillStyle(sheen)
	dc.DrawRectangle(0, 0, size, size)
	dc.Fill()

	// Nib glyph, warm paper white with a hint of blue at the foot.
	nib := gg.NewLinearGradient(0, 220, 0, 800)
	nib.AddColorStop(0, color.RGBA{255, 255, 255, 255})
	nib.AddColorStop(1, color.RGBA{214, 228, 252, 255})
	dc.SetFillStyle(nib)
	dc.SetFillRule(gg.FillRuleEvenOdd)
	nibPath(dc)
	dc.Fill()
	dc.ResetClip()

	// Hairline rim inside the tile edge, so the icon holds together on
	// both light and dark desktops.
	dc.DrawRoundedRectangle(1.5, 1.5, size-3, size-3, radius-1.5)
	dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{255, 255, 255, 40}))
	dc.SetLineWidth(3)
	dc.Stroke()

	if err := os.MkdirAll("resources", 0o755); err != nil {
		panic(err)
	}
	if err := dc.SavePNG("resources/icon.png"); err != nil {
		panic(err)
	}
}

// nibPath traces the fountain-pen nib centered on the tile: the outline,
// then the slit and the breather hole as subpaths the even-odd rule
// punches out.
func nibPath(dc *gg.Context) {
	// Outline: a gently domed heel at the top, full shoulders, and the
	// sides tapering to the sharp writing tip at the bottom.
	dc.MoveTo(384, 296)
	dc.QuadraticTo(512, 238, 640, 296) // heel dome
	dc.CubicTo(694, 388, 708, 486, 664, 596)
	dc.CubicTo(628, 688, 560, 760, 512, 800) // right side to the tip
	dc.CubicTo(464, 760, 396, 688, 360, 596)
	dc.CubicTo(316, 486, 330, 388, 384, 296)
	dc.ClosePath()

	// Slit: a narrow wedge from under the breather hole to the tip.
	dc.MoveTo(501, 508)
	dc.LineTo(523, 508)
	dc.LineTo(516, 770)
	dc.QuadraticTo(512, 784, 508, 770)
	dc.ClosePath()

	// Breather hole.
	dc.DrawCircle(512, 458, 34)
	dc.ClosePath()
}
