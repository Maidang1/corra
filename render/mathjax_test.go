package render

import (
	"bytes"
	"context"
	"image/color"
	"math"
	"runtime"
	"strings"
	"testing"
)

func TestMathJaxNativeTextInk(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("CoreText font fallback is native to macOS")
	}
	for _, expr := range []string{`\text{中文}`, `x_{\text{中文}}^{\text{日本語}}`, `\frac{\text{中文}}{\text{日本語}}`} {
		results, err := RenderMathBatch([]MathExpression{{TeX: expr, Display: true}}, MathOptions{}, 20, color.Black)
		if err != nil {
			t.Fatalf("native text %q: %v", expr, err)
		}
		if results[0].Error != nil {
			t.Fatal(results[0].Error)
		}
		img := results[0].Image
		if bytes.Contains(results[0].SVG, []byte("<text")) {
			t.Fatal("shared SVG retains unsupported text")
		}
		ink := 0
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a > 0 {
					ink++
				}
			}
		}
		if ink < 100 {
			t.Fatalf("native text %q has only %d ink pixels", expr, ink)
		}
	}
}

func TestNativeMathTextPreservesTransforms(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("CoreText font fallback is native to macOS")
	}
	const transform = `translate(1700,500) rotate(13) scale(0.5,0.75)`
	data := []byte(`<svg viewBox="0 0 3000 2500"><g transform="translate(100,1800)"><text transform="` + transform + `" font-size="884px">中文</text></g></svg>`)
	outlined, err := outlineMathSVGText(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outlined, []byte(`transform="`+transform+`"`)) || bytes.Contains(outlined, []byte("<text")) {
		t.Fatalf("transform not preserved: %s", outlined)
	}
	img, err := rasterizeMathSVG(outlined, 20, color.Black)
	if err != nil {
		t.Fatal(err)
	}
	ink := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				ink++
				if x < 65 || y < 55 {
					t.Fatalf("ink escaped transformed area at %d,%d", x, y)
				}
			}
		}
	}
	if ink < 100 {
		t.Fatalf("transformed glyphs have only %d ink pixels", ink)
	}
}

func TestMathBatchRendersNumbersAndReferences(t *testing.T) {
	results, err := RenderMathBatch([]MathExpression{{TeX: `\eqref{energy}`}, {TeX: `E=mc^2\label{energy}`, Display: true}, {TeX: `\eqref{energy}`}}, MathOptions{Numbering: "all"}, 16, color.Black)
	if err != nil {
		t.Fatal(err)
	}
	for i, result := range results {
		if result.Error != nil || result.Image == nil {
			t.Fatalf("item %d: %+v", i, result)
		}
		ink := 0
		for y := 0; y < result.Image.Bounds().Dy(); y++ {
			for x := 0; x < result.Image.Bounds().Dx(); x++ {
				_, _, _, a := result.Image.At(x, y).RGBA()
				if a > 0 {
					ink++
				}
			}
		}
		if ink < 10 {
			t.Fatalf("item %d has no rendered ink", i)
		}
	}
	if results[1].Number != "1" || !bytes.Equal(results[0].SVG, results[2].SVG) {
		t.Fatal("numbering/references disagree")
	}
}

func TestMathRasterLimits(t *testing.T) {
	for _, size := range []float64{0, -1, math.NaN(), math.Inf(1), 513} {
		if _, err := RenderMath("x", false, size, color.Black); err == nil {
			t.Fatalf("accepted size %v", size)
		}
	}
	for _, data := range []string{`<svg viewBox="0 0 1e100 20"></svg>`, `<svg viewBox="0 0 NaN 20"></svg>`, `<svg viewBox="0 0 -1 20"></svg>`, strings.Repeat("x", 8*1024*1024+1)} {
		if _, err := rasterizeMathSVG([]byte(data), 16, color.Black); err == nil {
			t.Fatal("accepted invalid SVG dimensions/size")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RenderMathBatchContext(ctx, []MathExpression{{TeX: "x"}}, MathOptions{}, 16, color.Black); err == nil {
		t.Fatal("accepted cancelled render")
	}
}

func TestMathJaxAdvancedExpressions(t *testing.T) {
	for name, expr := range map[string]string{
		"matrix":    `\begin{pmatrix}a&b\\c&d\end{pmatrix}`,
		"aligned":   `\begin{aligned}a&=b+c\\x&=\frac{1}{\sqrt{y}}\end{aligned}`,
		"macro":     `\newcommand{\RR}{\mathbb{R}}\RR^2\to\RR`,
		"chemistry": `\ce{2H2 + O2 -> 2H2O}`,
		"physics":   `\dv{f}{x}+\bra{\psi}\ket{\phi}`,
		"tag":       `E=mc^2\tag{energy}`,
	} {
		t.Run(name, func(t *testing.T) {
			img, err := RenderMath(expr, true, 16, color.RGBA{R: 40, G: 60, B: 80, A: 255})
			if err != nil {
				t.Fatalf("advanced TeX failed: %v", err)
			}
			if img.Bounds().Dx() < 2 || img.Bounds().Dy() < 2 || img.Bounds().Dx() > 4096 || img.Bounds().Dy() > 4096 {
				t.Fatalf("unexpected math bounds: %v", img.Bounds())
			}
			visible, transparent := false, false
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					_, _, _, a := img.At(x, y).RGBA()
					visible = visible || a != 0
					transparent = transparent || a == 0
				}
			}
			if !visible || !transparent {
				t.Fatalf("math must have visible glyphs on transparency: visible=%v transparent=%v", visible, transparent)
			}
		})
	}
}
