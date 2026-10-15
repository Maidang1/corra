// Package mathjax typesets TeX with a self-contained MathJax bundle in Goja.
// No Node, browser, external process, filesystem, or network is used at runtime.
package mathjax

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dop251/goja"
)

//go:embed mathjax.bundle.js
var bundle string

const (
	MaxExpressionBytes  = 32 * 1024
	MaxDocumentBytes    = 1024 * 1024
	MaxExpressions      = 1024
	MaxSVGBytes         = 8 * 1024 * 1024
	MaxDocumentSVGBytes = 32 * 1024 * 1024
	renderTimeout       = 5 * time.Second
)

type Expression struct {
	TeX     string `json:"tex"`
	Display bool   `json:"display"`
}

type Options struct {
	Macros    map[string]string `json:"macros,omitempty"`
	Numbering string            `json:"numbering,omitempty"`
}

type Result struct {
	SVG    string `json:"svg,omitempty"`
	Number string `json:"number,omitempty"`
	Error  string `json:"error,omitempty"`
}

var (
	gate         = make(chan struct{}, 1)
	runtime      *goja.Runtime
	programOnce  sync.Once
	program      *goja.Program
	programError error
)

// Render shares macros and equation labels within this ordered batch only.
// Syntax errors are reported per expression, while request/runtime errors fail
// the call. A bounded, serialized interpreter reuses compiled code and fonts.
func Render(ctx context.Context, expressions []Expression, options Options) ([]Result, error) {
	if err := validate(expressions, options); err != nil {
		return nil, err
	}
	if len(expressions) == 0 {
		return []Result{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
	case <-ctx.Done():
		return nil, fmt.Errorf("mathjax: %w", ctx.Err())
	}
	programOnce.Do(func() { program, programError = goja.Compile("mathjax.bundle.js", bundle, false) })
	if programError != nil {
		return nil, fmt.Errorf("mathjax bundle: %w", programError)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("mathjax: %w", err)
	}
	vm := runtime
	if vm == nil {
		vm = goja.New()
		vm.SetMaxCallStackSize(2048)
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		_ = vm.Set("console", map[string]any{"log": func(...any) {}, "warn": func(...any) {}, "error": func(...any) {}})
	}
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(stopped); vm.Interrupt(ctx.Err()) })
	defer func() {
		if !stop() {
			<-stopped
		}
		vm.ClearInterrupt()
	}()
	if runtime == nil {
		if _, err := vm.RunProgram(program); err != nil {
			return nil, fmt.Errorf("mathjax initialization: %w", err)
		}
	}
	fn, ok := goja.AssertFunction(vm.Get("corraMathJax"))
	if !ok {
		return nil, fmt.Errorf("mathjax: missing renderer")
	}
	value, err := fn(goja.Undefined(), vm.ToValue(map[string]any{"expressions": expressions, "options": options}))
	if err != nil {
		runtime = nil
		return nil, fmt.Errorf("mathjax: %w", err)
	}
	var results []Result
	if err := vm.ExportTo(value, &results); err != nil {
		runtime = nil
		return nil, fmt.Errorf("mathjax result: %w", err)
	}
	if len(results) != len(expressions) {
		runtime = nil
		return nil, fmt.Errorf("mathjax: result count mismatch")
	}
	total := 0
	for _, result := range results {
		total += len(result.SVG)
		if len(result.SVG) > MaxSVGBytes || total > MaxDocumentSVGBytes {
			runtime = nil
			return nil, fmt.Errorf("mathjax: SVG output limit exceeded")
		}
	}
	runtime = vm
	return results, nil
}

func validate(expressions []Expression, options Options) error {
	if len(expressions) > MaxExpressions {
		return fmt.Errorf("mathjax: too many expressions (maximum %d)", MaxExpressions)
	}
	if options.Numbering != "" && options.Numbering != "none" && options.Numbering != "ams" && options.Numbering != "all" {
		return fmt.Errorf("mathjax: invalid numbering mode")
	}
	total := 0
	for _, expression := range expressions {
		total += len(expression.TeX)
		if len(expression.TeX) > MaxExpressionBytes || total > MaxDocumentBytes {
			return fmt.Errorf("mathjax: TeX input limit exceeded")
		}
		if !utf8.ValidString(expression.TeX) || strings.ContainsRune(expression.TeX, 0) {
			return fmt.Errorf("mathjax: invalid TeX text")
		}
	}
	if len(options.Macros) > 256 {
		return fmt.Errorf("mathjax: too many macros")
	}
	for name, value := range options.Macros {
		if name == "" || strings.ContainsAny(name, "\\{} \t\r\n") || !utf8.ValidString(name) || !utf8.ValidString(value) {
			return fmt.Errorf("mathjax: invalid macro")
		}
		total += len(name) + len(value)
		if len(value) > MaxExpressionBytes || total > MaxDocumentBytes {
			return fmt.Errorf("mathjax: macro input limit exceeded")
		}
	}
	return nil
}

// MarshalOptions returns a stable description for document cache keys.
func MarshalOptions(options Options) string {
	data, _ := json.Marshal(options)
	return string(data)
}
