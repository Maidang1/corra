package render

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Maidang1/corra/internal/mathjax"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// mathDPI renders at twice the display resolution for crisp native scaling.
const mathDPI = 144

// MathExpression is TeX source without delimiters and its inline/display mode.
type MathExpression = mathjax.Expression

// MathOptions controls document-local macros and equation numbering. Numbering
// is "none" (default), "ams", or "all"; the batch shares labels and definitions.
type MathOptions = mathjax.Options

type MathResult struct {
	Image  image.Image
	SVG    []byte
	Number string
	Error  error
}

// RenderMath typesets TeX offline with embedded MathJax and returns transparent
// pixels tinted with fg. Errors remain explicit so native views can show source.
func RenderMath(expr string, display bool, fontSize float64, fg color.Color) (image.Image, error) {
	results, err := RenderMathBatch([]MathExpression{{TeX: expr, Display: display}}, MathOptions{}, fontSize, fg)
	if err != nil {
		return nil, err
	}
	return results[0].Image, results[0].Error
}

// RenderMathBatch typesets one ordered document snapshot. New TeX state per
// batch prevents macros, labels, and counters from leaking between documents.
// Forward references resolve before SVG generation. Syntax and raster errors
// are reported per equation; request/runtime failures return a top-level error.
func RenderMathBatch(expressions []MathExpression, options MathOptions, fontSize float64, fg color.Color) ([]MathResult, error) {
	return RenderMathBatchContext(context.Background(), expressions, options, fontSize, fg)
}

// RenderMathBatchContext permits stale document rendering to be cancelled.
func RenderMathBatchContext(ctx context.Context, expressions []MathExpression, options MathOptions, fontSize float64, fg color.Color) ([]MathResult, error) {
	if math.IsNaN(fontSize) || math.IsInf(fontSize, 0) || fontSize <= 0 || fontSize > 512 {
		return nil, fmt.Errorf("math: font size must be finite and between 0 and 512")
	}
	normalized := make([]MathExpression, len(expressions))
	copy(normalized, expressions)
	for i := range normalized {
		normalized[i].TeX = stripMathDelimiters(normalized[i].TeX)
	}
	rendered, err := cachedMathJax(ctx, normalized, options)
	if err != nil {
		return nil, err
	}
	results := make([]MathResult, len(rendered))
	totalPixels := 0
	for i, result := range rendered {
		results[i].Number = result.Number
		if result.Error != "" {
			results[i].Error = fmt.Errorf("math: %s", result.Error)
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		size := fontSize
		if normalized[i].Display {
			size *= 1.15
		}
		data, img, err := cachedMathRaster(result.SVG, size, fg)
		if err != nil {
			results[i].Error = err
			continue
		}
		totalPixels += img.Bounds().Dx() * img.Bounds().Dy()
		if totalPixels > 32*1024*1024 {
			return nil, fmt.Errorf("math: document raster pixel limit exceeded")
		}
		results[i].Image, results[i].SVG = img, data
	}
	return results, nil
}

func stripMathDelimiters(expression string) string {
	trimmed := strings.TrimSpace(expression)
	for _, pair := range [][2]string{{"$$", "$$"}, {"\\[", "\\]"}, {"\\(", "\\)"}, {"$", "$"}} {
		if len(trimmed) >= len(pair[0])+len(pair[1]) && strings.HasPrefix(trimmed, pair[0]) && strings.HasSuffix(trimmed, pair[1]) {
			return trimmed[len(pair[0]) : len(trimmed)-len(pair[1])]
		}
	}
	return expression
}

// MathJax's SVG viewBox is in 1000 units/em, regardless of width="...ex".
// Validate dimensions before converting to ints or allocating any raster.
func rasterizeMathSVG(data []byte, fontSize float64, fg color.Color) (image.Image, error) {
	if len(data) > mathjax.MaxSVGBytes {
		return nil, fmt.Errorf("math: SVG output limit exceeded")
	}
	viewBox, err := mathSVGViewBox(data)
	if err != nil {
		return nil, err
	}
	scale := fontSize * (mathDPI / 72.0) / 1000
	width, height := math.Ceil(viewBox[2]*scale), math.Ceil(viewBox[3]*scale)
	if math.IsNaN(width) || math.IsNaN(height) || math.IsInf(width, 0) || math.IsInf(height, 0) || width < 0 || height < 0 || width > 8192 || height > 8192 || width*height > 16*1024*1024 {
		return nil, fmt.Errorf("math: raster dimensions exceed limit")
	}
	// oksvg stops processing root attributes at ex/% sizes, before viewBox.
	// Strip those root dimensions; glyph rect dimensions remain unchanged.
	rootEnd := bytes.IndexByte(data, '>')
	if rootEnd < 0 {
		return nil, fmt.Errorf("math: missing SVG root")
	}
	data = append(mathRootSizes.ReplaceAll(data[:rootEnd], nil), data[rootEnd:]...)
	data = bytes.ReplaceAll(data, []byte("currentColor"), []byte("black"))
	// oksvg's one-argument scale implementation collapses the Y axis.
	data = mathSingleScale.ReplaceAll(data, []byte(`scale($1,$1)`))
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, fmt.Errorf("math SVG: %w", err)
	}
	w, h := max(1, int(width)), max(1, int(height))
	pixels := image.NewRGBA(image.Rect(0, 0, w, h))
	// Scale the viewBox translation as well; MathJax has a negative min-Y.
	icon.Transform = rasterx.Identity.Scale(float64(w)/max(1, viewBox[2]), float64(h)/max(1, viewBox[3])).Translate(-viewBox[0], -viewBox[1])
	scanner := rasterx.NewScannerGV(w, h, pixels, pixels.Bounds())
	icon.Draw(rasterx.NewDasher(w, h, scanner), 1)
	return tintImage(pixels, fg), nil
}

var mathRootSizes = regexp.MustCompile(`\s(?:width|height)="[^"]*"`)
var mathSingleScale = regexp.MustCompile(`\bscale\(\s*([-+0-9.eE]+)\s*\)`)

// Convert only text leaves to native font outlines. Keeping their original
// opening attributes on a group preserves every ancestor/local SVG transform.
func outlineMathSVGText(data []byte) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	last := 0
	for {
		start := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("math SVG: %w", err)
		}
		element, ok := token.(xml.StartElement)
		if !ok || element.Name.Local != "text" {
			continue
		}
		openingEnd := int(decoder.InputOffset())
		size, bold, italic := 0.0, false, false
		for _, attr := range element.Attr {
			switch attr.Name.Local {
			case "font-size":
				size, err = strconv.ParseFloat(strings.TrimSuffix(attr.Value, "px"), 64)
			case "font-weight":
				bold = attr.Value == "bold" || attr.Value == "700"
			case "font-style":
				italic = attr.Value == "italic"
			}
			if err != nil {
				return nil, fmt.Errorf("math: invalid native text size")
			}
		}
		// MathJax emits text leaves, never nested spans. Reject unexpected child
		// elements instead of dropping any part of an unsupported glyph silently.
		var content strings.Builder
		for {
			next, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("math SVG text: %w", err)
			}
			if _, ok := next.(xml.EndElement); ok {
				break
			}
			if chars, ok := next.(xml.CharData); ok {
				content.Write(chars)
			} else {
				return nil, fmt.Errorf("math: unsupported nested SVG text")
			}
		}
		path := ""
		if content.Len() > 0 {
			if size <= 0 || size > 100000 || math.IsNaN(size) || math.IsInf(size, 0) {
				return nil, fmt.Errorf("math: invalid native text size")
			}
			path, err = nativeMathTextPath(content.String(), size, bold, italic)
			if err != nil {
				return nil, err
			}
		}
		output.Write(data[last:start])
		output.Write(bytes.Replace(data[start:openingEnd], []byte("<text"), []byte("<g"), 1))
		output.WriteString(`<path d="`)
		output.WriteString(path)
		output.WriteString(`"></path></g>`)
		last = int(decoder.InputOffset())
		if output.Len() > mathjax.MaxSVGBytes {
			return nil, fmt.Errorf("math: native SVG output limit exceeded")
		}
	}
	if last == 0 {
		return data, nil
	}
	output.Write(data[last:])
	if output.Len() > mathjax.MaxSVGBytes {
		return nil, fmt.Errorf("math: native SVG output limit exceeded")
	}
	return output.Bytes(), nil
}

func mathSVGViewBox(data []byte) ([4]float64, error) {
	var viewBox [4]float64
	decoder := xml.NewDecoder(bytes.NewReader(data))
	textDepth, depth := 0, 0
	root := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return viewBox, fmt.Errorf("math SVG: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if !root {
				if value.Name.Local != "svg" {
					return viewBox, fmt.Errorf("math: expected SVG root")
				}
				root = true
				for _, attr := range value.Attr {
					if attr.Name.Local != "viewBox" {
						continue
					}
					fields := strings.Fields(attr.Value)
					if len(fields) != 4 {
						return viewBox, fmt.Errorf("math: invalid SVG viewBox")
					}
					for i, field := range fields {
						v, err := strconv.ParseFloat(field, 64)
						if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
							return viewBox, fmt.Errorf("math: invalid SVG viewBox")
						}
						viewBox[i] = v
					}
				}
			}
			if value.Name.Local == "text" {
				textDepth = depth
			}
		case xml.EndElement:
			if depth == textDepth {
				textDepth = 0
			}
			depth--
		case xml.CharData:
			if textDepth > 0 && len(bytes.TrimSpace(value)) > 0 {
				return viewBox, fmt.Errorf("math: glyph requires a font outside the embedded MathJax font")
			}
		}
	}
	if !root || viewBox[2] < 0 || viewBox[3] < 0 {
		return viewBox, fmt.Errorf("math: invalid SVG dimensions")
	}
	return viewBox, nil
}

func tintImage(src image.Image, fg color.Color) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if fg == nil {
		fg = color.Black
	}
	draw.DrawMask(dst, dst.Bounds(), image.NewUniform(fg), image.Point{}, src, bounds.Min, draw.Src)
	return dst
}

// mathRasterCache keeps outlined SVG and raster per MathJax output, size and
// colour: reopening a note or editing next to a formula must not re-raster
// every formula in the document. Bounded, oldest entries evicted first.
var mathRasterCache = struct {
	sync.Mutex
	entries map[string]mathRaster
	order   []string
}{entries: map[string]mathRaster{}}

type mathRaster struct {
	svg []byte
	img image.Image
}

const mathRasterCacheMax = 1024

func cachedMathRaster(svg string, size float64, fg color.Color) ([]byte, image.Image, error) {
	r, g, b, a := fg.RGBA()
	key := fmt.Sprintf("%g|%d,%d,%d,%d|%s", size, r, g, b, a, svg)
	mathRasterCache.Lock()
	hit, ok := mathRasterCache.entries[key]
	mathRasterCache.Unlock()
	if ok {
		return hit.svg, hit.img, nil
	}
	data, err := outlineMathSVGText([]byte(svg))
	if err != nil {
		return nil, nil, err
	}
	img, err := rasterizeMathSVG(data, size, fg)
	if err != nil {
		return nil, nil, err
	}
	mathRasterCache.Lock()
	if _, ok := mathRasterCache.entries[key]; !ok {
		mathRasterCache.entries[key] = mathRaster{data, img}
		mathRasterCache.order = append(mathRasterCache.order, key)
		for len(mathRasterCache.order) > mathRasterCacheMax {
			delete(mathRasterCache.entries, mathRasterCache.order[0])
			mathRasterCache.order = mathRasterCache.order[1:]
		}
	}
	mathRasterCache.Unlock()
	return data, img, nil
}

// mathJaxCache memoizes MathJax output per context-free expression. Anything
// that defines macros, labels or references, or a display equation that may be
// numbered, depends on its neighbours and always renders in the ordered batch.
var mathJaxCache = struct {
	sync.Mutex
	entries map[string]mathjax.Result
	order   []string
}{entries: map[string]mathjax.Result{}}

var mathContextual = []string{`\def`, `\gdef`, `\edef`, `\let`, `\newcommand`, `\renewcommand`, `\DeclareMathOperator`, `\newenvironment`, `\label`, `\ref`, `\eqref`, `\tag`, `\notag`, `\nonumber`}

func mathCacheable(expression mathjax.Expression, options mathjax.Options) bool {
	for _, token := range mathContextual {
		if strings.Contains(expression.TeX, token) {
			return false
		}
	}
	return !expression.Display || options.Numbering == "" || options.Numbering == "none"
}

func cachedMathJax(ctx context.Context, expressions []mathjax.Expression, options mathjax.Options) ([]mathjax.Result, error) {
	for _, expression := range expressions {
		if strings.Contains(expression.TeX, `\def`) || strings.Contains(expression.TeX, `\newcommand`) || strings.Contains(expression.TeX, `\let`) || strings.Contains(expression.TeX, `\renewcommand`) || strings.Contains(expression.TeX, `\DeclareMathOperator`) {
			// Macro definitions change how later expressions parse.
			return mathjax.Render(ctx, expressions, options)
		}
	}
	macros := mathjax.MarshalOptions(mathjax.Options{Macros: options.Macros})
	keys := make([]string, len(expressions))
	results := make([]mathjax.Result, len(expressions))
	var pending []mathjax.Expression
	var slots []int
	queued := map[string]int{}
	mathJaxCache.Lock()
	for i, expression := range expressions {
		if !mathCacheable(expression, options) {
			pending, slots = append(pending, expression), append(slots, i)
			continue
		}
		keys[i] = fmt.Sprintf("%t|%s|%s", expression.Display, macros, expression.TeX)
		if hit, ok := mathJaxCache.entries[keys[i]]; ok {
			results[i] = hit
			continue
		}
		if _, ok := queued[keys[i]]; ok {
			continue
		}
		queued[keys[i]] = len(pending)
		pending, slots = append(pending, expression), append(slots, i)
	}
	mathJaxCache.Unlock()
	if len(pending) > 0 {
		rendered, err := mathjax.Render(ctx, pending, options)
		if err != nil {
			return nil, err
		}
		mathJaxCache.Lock()
		for j, result := range rendered {
			i := slots[j]
			results[i] = result
			if keys[i] == "" || result.Error != "" {
				continue
			}
			if _, ok := mathJaxCache.entries[keys[i]]; !ok {
				mathJaxCache.entries[keys[i]] = result
				mathJaxCache.order = append(mathJaxCache.order, keys[i])
				for len(mathJaxCache.order) > mathRasterCacheMax {
					delete(mathJaxCache.entries, mathJaxCache.order[0])
					mathJaxCache.order = mathJaxCache.order[1:]
				}
			}
		}
		mathJaxCache.Unlock()
		for i, key := range keys {
			if j, ok := queued[key]; ok && key != "" && slots[j] != i {
				results[i] = rendered[j]
			}
		}
	}
	return results, nil
}
