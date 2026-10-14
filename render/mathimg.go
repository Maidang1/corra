package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"codeberg.org/go-latex/latex/drawtex/drawimg"
	"codeberg.org/go-latex/latex/font/lm"
	"codeberg.org/go-latex/latex/mtex"
)

// mathDPI renders at twice the display resolution for crisp scaling.
const mathDPI = 144

// RenderMath renders a TeX math expression to an RGBA image with
// go-latex (pure Go, Latin Modern fonts). fg tints the glyphs so math
// follows the document theme.
//
// go-latex's mtex cannot yet typeset TeX scripts (^ and _): the parser
// produces ast.Sup/ast.Sub nodes no visitor handles. So simple scripts are
// first transliterated to Unicode super/subscripts (E=mc^2 -> E=mc²), which
// covers most formulas in notes; anything still failing returns an error
// and the view degrades to showing the source.
func RenderMath(expr string, display bool, fontSize float64, fg color.Color) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("math: %v", r)
		}
	}()
	size := fontSize
	if display {
		size = fontSize * 1.15
	}
	for _, candidate := range []string{expr, transliterateScripts(expr)} {
		im, err := renderMathOnce(candidate, size)
		if err == nil {
			return tintImage(im, fg), nil
		}
	}
	return nil, fmt.Errorf("math: unsupported expression %q", expr)
}

func renderMathOnce(expr string, size float64) (image.Image, error) {
	var buf bytes.Buffer
	r := drawimg.NewRenderer(&buf)
	// mtex only enters math mode inside math delimiters; without them the
	// TeX markup hits nodes it cannot typeset.
	wrapped := expr
	trimmed := strings.TrimSpace(expr)
	if !strings.HasPrefix(trimmed, "$") && !strings.HasPrefix(trimmed, "\\(") && !strings.HasPrefix(trimmed, "\\[") {
		wrapped = "$" + expr + "$"
	}
	if err := mtex.Render(r, wrapped, size, mathDPI, lm.Fonts()); err != nil {
		return nil, err
	}
	return png.Decode(&buf)
}

var (
	superMap = map[rune]rune{
		'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴', '5': '⁵',
		'6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹', '+': '⁺', '-': '⁻',
		'=': '⁼', '(': '⁽', ')': '⁾', 'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ',
		'd': 'ᵈ', 'e': 'ᵉ', 'f': 'ᶠ', 'g': 'ᵍ', 'h': 'ʰ', 'i': 'ⁱ',
		'j': 'ʲ', 'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ', 'n': 'ⁿ', 'o': 'ᵒ',
		'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ', 'v': 'ᵛ',
		'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
	}
	subMap = map[rune]rune{
		'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄', '5': '₅',
		'6': '₆', '7': '₇', '8': '₈', '9': '₉', '+': '₊', '-': '₋',
		'(': '₍', ')': '₎', 'a': 'ₐ', 'e': 'ₑ', 'h': 'ₕ', 'i': 'ᵢ',
		'j': 'ⱼ', 'k': 'ₖ', 'l': 'ₗ', 'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ',
		'p': 'ₚ', 'r': 'ᵣ', 's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ', 'v': 'ᵥ',
		'x': 'ₓ',
	}
)

// transliterateScripts rewrites simple TeX scripts to Unicode super/
// subscripts: x^2 -> x², x_{max} stays (not mappable) — unmappable
// scripts make the whole expression untransformed, so the caller's first
// attempt still handles complex cases through the real renderer.
func transliterateScripts(expr string) string {
	rs := []rune(strings.TrimSpace(expr))
	var out []rune
	ok := true
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r != '^' && r != '_' || i+1 >= len(rs) {
			out = append(out, r)
			continue
		}
		var group []rune
		if rs[i+1] == '{' {
			j := i + 2
			for j < len(rs) && rs[j] != '}' {
				group = append(group, rs[j])
				j++
			}
			if j >= len(rs) {
				return expr // unbalanced
			}
			i = j
		} else {
			group = []rune{rs[i+1]}
			i++
		}
		m := superMap
		if r == '_' {
			m = subMap
		}
		for _, g := range group {
			u, found := m[g]
			if !found {
				ok = false
				break
			}
			out = append(out, u)
		}
		if !ok {
			return expr
		}
	}
	return string(out)
}

// tintImage replaces dark (glyph) pixels with fg, preserving anti-aliasing
// through the source alpha; light pixels become transparent.
func tintImage(src image.Image, fg color.Color) image.Image {
	f := color.RGBAModel.Convert(fg).(color.RGBA)
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			_, _, _, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if a == 0 {
				continue
			}
			af := float64(a) / 65535.0
			dst.SetRGBA(x, y, color.RGBA{f.R, f.G, f.B, uint8(af * 255)})
		}
	}
	return dst
}
