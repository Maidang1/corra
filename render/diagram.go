package render

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bep/goat"
	"github.com/xo/mermaid"
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// Diagram renders diagram source (mermaid, d2, goat, flow, sequence) to an SVG
// document. The returned bytes are the SVG; views decide how to display them.

var (
	mermaidOnce   sync.Once
	mermaidRender *mermaid.Renderer
	mermaidErr    error
	// errUnsupportedDiagram means the source is not valid for its engine. The
	// caller shows the source instead of a wrong diagram.
	errUnsupportedDiagram = errors.New("diagram: unsupported syntax")
)

// MermaidRenderer returns the shared mermaid renderer, compiling the
// WebAssembly module at most once per process.
func MermaidRenderer(ctx context.Context) (*mermaid.Renderer, error) {
	mermaidOnce.Do(func() {
		mermaidRender, mermaidErr = mermaid.New(ctx)
	})
	return mermaidRender, mermaidErr
}

// RenderDiagram renders a diagram of the given kind to SVG bytes.
func RenderDiagram(ctx context.Context, kind DiagramKind, src string) (out []byte, err error) {
	defer func() {
		// Diagram engines can panic on malformed input; degrade, never
		// crash the editor.
		if r := recover(); r != nil {
			out, err = nil, fmt.Errorf("diagram: %v", r)
		}
	}()
	switch kind {
	case DiagramMermaid:
		r, err := MermaidRenderer(ctx)
		if err != nil {
			return nil, fmt.Errorf("mermaid renderer: %w", err)
		}
		svg, err := r.Render(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("mermaid: %w", err)
		}
		return []byte(svg), nil
	case DiagramD2:
		return renderD2(ctx, src)
	case DiagramGoAT:
		return renderGoAT(src)
	case DiagramFlowchartJS, DiagramSequenceJS:
		// Both grammars translate to mermaid and reuse its renderer, so the
		// layout is mermaid's rather than a pixel copy of the original engine.
		var translated string
		var err error
		if kind == DiagramFlowchartJS {
			translated, err = flowchartToMermaid(src)
		} else {
			translated, err = sequenceToMermaid(src)
		}
		if err != nil {
			return nil, err
		}
		return RenderDiagram(ctx, DiagramMermaid, translated)
	}
	return nil, errors.New("unknown diagram kind")
}

func renderGoAT(src string) ([]byte, error) {
	var buf bytes.Buffer
	goat.BuildAndWriteSVG(strings.NewReader(src), &buf)
	if buf.Len() == 0 {
		return nil, errors.New("goat: empty output")
	}
	return buf.Bytes(), nil
}

// renderD2 compiles D2 source to SVG through the d2 library.
func renderD2(ctx context.Context, src string) ([]byte, error) {
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, fmt.Errorf("d2 ruler: %w", err)
	}
	compileOpts := &d2lib.CompileOptions{
		Ruler: ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}
	diagram, _, err := d2lib.Compile(ctx, src, compileOpts, nil)
	if err != nil {
		return nil, fmt.Errorf("d2 compile: %w", err)
	}
	pad := int64(24)
	out, err := d2svg.Render(diagram, &d2svg.RenderOpts{Pad: &pad})
	if err != nil {
		return nil, fmt.Errorf("d2 render: %w", err)
	}
	return out, nil
}
