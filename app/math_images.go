package app

import (
	"context"
	"fmt"
	"image/color"
	"maps"
	"slices"

	"github.com/Maidang1/corra/render"
)

type mathImageBatch struct {
	key     string
	cancel  context.CancelFunc
	ready   bool
	results []render.MathResult
	err     error
}

// MathImages returns one complete document batch. A changed snapshot cancels
// obsolete work; late results never replace the current document or its theme.
func (a *App) MathImages(expressions []render.MathExpression, options render.MathOptions, fontSize float64, fg color.Color) ([]render.MathResult, bool, error) {
	if fg == nil {
		fg = color.Black
	}
	r, g, b, alpha := fg.RGBA()
	key := fmt.Sprintf("%s|%g|%x|%x|%x|%x", render.MathKey(expressions, options), fontSize, r, g, b, alpha)
	a.mu.Lock()
	if len(expressions) == 0 {
		if a.mathImages != nil {
			a.mathImages.cancel()
			a.mathImages = nil
		}
		a.mu.Unlock()
		return nil, true, nil
	}
	entry := a.mathImages
	if entry != nil && entry.key == key {
		results, ready, err := entry.results, entry.ready, entry.err
		a.mu.Unlock()
		return results, ready, err
	}
	if entry != nil {
		entry.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry = &mathImageBatch{key: key, cancel: cancel}
	a.mathImages = entry
	inputs := slices.Clone(expressions)
	options.Macros = maps.Clone(options.Macros)
	a.mu.Unlock()
	go func() {
		defer cancel()
		results, err := render.RenderMathBatchContext(ctx, inputs, options, fontSize, color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(alpha)})
		a.mu.Lock()
		current := a.mathImages == entry
		if current {
			entry.results, entry.err, entry.ready = results, err, true
		}
		a.mu.Unlock()
		if current {
			a.InvalidateUI()
		}
	}()
	return nil, false, nil
}
