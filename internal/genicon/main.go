// Command genicon draws the app icon (1024x102 PNG) deterministically in
// pure Go, so the repository carries no binary assets. Run before
// packaging: go run ./internal/genicon
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Vertical gradient background, deep ink, with rounded corners.
	top := color.RGBA{24, 26, 32, 255}
	bot := color.RGBA{38, 42, 54, 255}
	md := &rounded{rect: image.Rect(0, 0, size, size), radius: 230}
	draw.DrawMask(img, img.Bounds(), &gradientImage{top: top, bot: bot}, image.Point{}, md, image.Point{}, draw.Over)

	// Accent ring.
	ring := color.RGBA{94, 141, 255, 255}
	for a := 0; a < 360*4; a++ {
		ang := float64(a) / 4 * math.Pi / 180
		for w := 0; w < 26; w++ {
			r := 448 + float64(w)
			x := size/2 + int(r*math.Cos(ang))
			y := size/2 + int(r*math.Sin(ang))
			img.SetRGBA(x, y, ring)
		}
	}

	// "C" glyph, centered.
	drawBigC(img, size/2, size/2, 16, color.RGBA{240, 242, 246, 255})

	if err := os.MkdirAll("resources", 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create("resources/icon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// rounded is a rounded-rectangle alpha mask.
type rounded struct {
	rect   image.Rectangle
	radius int
}

func (r *rounded) ColorModel() color.Model { return color.AlphaModel }

func (r *rounded) Bounds() image.Rectangle { return r.rect }

func (r *rounded) At(x, y int) color.Color {
	dx, dy := 0, 0
	if x < r.rect.Min.X+r.radius {
		dx = r.rect.Min.X + r.radius - x
	} else if x > r.rect.Max.X-1-r.radius {
		dx = x - (r.rect.Max.X - 1 - r.radius)
	}
	if y < r.rect.Min.Y+r.radius {
		dy = r.rect.Min.Y + r.radius - y
	} else if y > r.rect.Max.Y-1-r.radius {
		dy = y - (r.rect.Max.Y - 1 - r.radius)
	}
	if dx == 0 && dy == 0 {
		return color.Alpha{255}
	}
	if dx*dx+dy*dy <= r.radius*r.radius {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

type gradientImage struct{ top, bot color.RGBA }

func (g *gradientImage) ColorModel() color.Model { return color.RGBAModel }
func (g *gradientImage) Bounds() image.Rectangle  { return image.Rect(0, 0, size, size) }
func (g *gradientImage) At(x, y int) color.Color {
	t := float64(y) / (size - 1)
	return color.RGBA{
		R: uint8(float64(g.top.R) + t*float64(g.bot.R-g.top.R)),
		G: uint8(float64(g.top.G) + t*float64(g.bot.G-g.top.G)),
		B: uint8(float64(g.top.B) + t*float64(g.bot.B-g.top.B)),
		A: 255,
	}
}

// drawBigC rasterizes a blocky "C" at the given scale.
func drawBigC(img *image.RGBA, cx, cy, scale int, c color.RGBA) {
	// 7x13 bitmap for "C" in basicfont, scaled up.
	glyph := []string{
		"0111110",
		"1111111",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1110000",
		"1111111",
		"0111110",
	}
	w := 7 * scale
	h := 13 * scale
	x0 := cx - w/2
	y0 := cy - h/2
	for row, line := range glyph {
		for col, ch := range line {
			if ch != '1' {
				continue
			}
			draw.Draw(img, image.Rect(x0+col*scale, y0+row*scale, x0+(col+1)*scale, y0+(row+1)*scale),
				image.NewUniform(c), image.Point{}, draw.Src)
		}
	}
}
