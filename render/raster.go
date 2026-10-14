package render

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/draw"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"codeberg.org/go-latex/latex/font/liberation"
)

var (
	styleBlockRe = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	dataURIRe    = regexp.MustCompile(`(xlink:href|href)="data:[^"]*"`)
)

var errNoSize = errString("svg: no size")

type errString string

func (e errString) Error() string { return string(e) }

// RasterizeSVG rasterizes an SVG document to an RGBA image, at scale times
// its natural size. Shapes go through oksvg; <text> elements — which oksvg
// skips — are overlaid with an embedded font, so diagram labels from mermaid
// and d2 survive. The background is filled white: diagrams display on a
// light card in both themes, like images on GitHub.
//
// Before parsing, embedded CSS and base64 data URIs (which d2 inlines for
// its fonts) are stripped: oksvg rejects them, and the text overlay does
// the labeling anyway.
func RasterizeSVG(data []byte, scale float64) (image.Image, error) {
	data = styleBlockRe.ReplaceAll(data, nil)
	data = dataURIRe.ReplaceAll(data, []byte(`href=""`))
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, err
	}
	vb := icon.ViewBox
	w, h := vb.W, vb.H
	if w <= 0 || h <= 0 {
		w, h = svgRootSize(data)
	}
	if w <= 0 || h <= 0 {
		return nil, errNoSize
	}
	tw, th := int(math.Ceil(w*scale)), int(math.Ceil(h*scale))
	img := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	icon.SetTarget(0, 0, float64(tw), float64(th))
	scanner := rasterx.NewScannerGV(tw, th, img, img.Bounds())
	dasher := rasterx.NewDasher(tw, th, scanner)
	icon.Draw(dasher, 1.0)
	overlaySVGText(img, data, float64(tw)/w)
	return img, nil
}

// svgRootSize reads width/height attributes of the root <svg> element.
func svgRootSize(data []byte) (w, h float64) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "svg" {
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "width":
					w = parseSVGLen(a.Value)
				case "height":
					h = parseSVGLen(a.Value)
				}
			}
			return
		}
	}
}

func parseSVGLen(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "px")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// svgText is one <text> element worth overlaying.
type svgText struct {
	x, y, size float64
	anchor     string
	fill       string
	content    string
}

// overlaySVGText draws the SVG's <text> elements onto img, at the given
// scale relative to the SVG viewbox.
func overlaySVGText(img draw.Image, data []byte, scale float64) {
	texts := collectSVGTexts(data)
	if len(texts) == 0 {
		return
	}
	faces := map[int]font.Face{}
	defer func() {
		for _, f := range faces {
			f.Close()
		}
	}()
	face := func(px int) font.Face {
		if f, ok := faces[px]; ok {
			return f
		}
		f, err := opentype.NewFace(liberation.Fonts().Rm, &opentype.FaceOptions{
			Size: float64(px), DPI: 72, Hinting: font.HintingNone,
		})
		if err != nil {
			return nil
		}
		faces[px] = f
		return f
	}
	for _, t := range texts {
		size := int(math.Round(t.size * scale))
		if size < 6 {
			size = 6
		}
		f := face(size)
		if f == nil {
			continue
		}
		d := font.Drawer{Dst: img, Src: image.NewUniform(parseSVGColor(t.fill)), Face: f}
		width := fixed.I(int(fontWidth(f, t.content)))
		x := fixed.I(int(math.Round(t.x * scale)))
		switch t.anchor {
		case "middle":
			x = x - width/2
		case "end":
			x = x - width
		}
		d.Dot = fixed.P(x.Round(), int(math.Round(t.y*scale)))
		d.DrawString(t.content)
	}
}

func fontWidth(f font.Face, s string) float64 {
	d := font.Drawer{Face: f}
	return float64(d.MeasureString(s).Ceil())
}

var svgColorNames = map[string]color.RGBA{
	"black": {0, 0, 0, 255}, "white": {255, 255, 255, 255},
	"gray": {128, 128, 128, 255}, "grey": {128, 128, 128, 255},
	"red": {220, 38, 38, 255}, "green": {22, 163, 74, 255},
	"blue": {37, 99, 235, 255}, "none": {0, 0, 0, 0},
}

func parseSVGColor(s string) color.Color {
	s = strings.TrimSpace(s)
	if c, ok := svgColorNames[strings.ToLower(s)]; ok {
		return c
	}
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) >= 6 {
			if v, err := strconv.ParseUint(hex[:6], 16, 32); err == nil {
				return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
			}
		}
	}
	if strings.HasPrefix(s, "rgb(") && strings.HasSuffix(s, ")") {
		parts := strings.Split(s[4:len(s)-1], ",")
		if len(parts) == 3 {
			var c color.RGBA
			c.A = 255
			for i, p := range parts {
				v, _ := strconv.Atoi(strings.TrimSpace(p))
				switch i {
				case 0:
					c.R = uint8(v)
				case 1:
					c.G = uint8(v)
				case 2:
					c.B = uint8(v)
				}
			}
			return c
		}
	}
	return color.RGBA{0x33, 0x33, 0x33, 255}
}

// collectSVGTexts walks the SVG and gathers <text> elements, including the
// text of nested <tspan>s.
func collectSVGTexts(data []byte) []svgText {
	var out []svgText
	dec := xml.NewDecoder(bytes.NewReader(data))
	var cur *svgText
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "text" {
				var nt svgText
				nt.size = 14
				nt.fill = "#333333"
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "x":
						nt.x = firstSVGLen(a.Value)
					case "y":
						nt.y = firstSVGLen(a.Value)
					case "font-size":
						nt.size = parseSVGLen(a.Value)
					case "fill":
						nt.fill = a.Value
					case "text-anchor":
						nt.anchor = a.Value
					}
				}
				out = append(out, nt)
				cur = &out[len(out)-1]
			}
		case xml.EndElement:
			if t.Name.Local == "text" {
				cur = nil
			}
		case xml.CharData:
			if cur != nil {
				cur.content += string(t)
			}
		}
	}
	filtered := out[:0]
	for _, t := range out {
		if strings.TrimSpace(t.content) != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// firstSVGLen parses a possibly space-separated list of lengths, first only.
func firstSVGLen(s string) float64 {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	return parseSVGLen(s)
}
