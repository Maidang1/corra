package mathjax

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRenderProducesSelfContainedSVG(t *testing.T) {
	results, err := Render(context.Background(), []Expression{{TeX: `x^2`, Display: true}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !strings.HasPrefix(results[0].SVG, "<svg") || !strings.Contains(results[0].SVG, "viewBox=") || !strings.Contains(results[0].SVG, "<path") || strings.Contains(results[0].SVG, "<use") {
		t.Fatalf("invalid SVG result: %+v", results)
	}
	t.Log(results[0].SVG[:min(500, len(results[0].SVG))])
}

func TestDocumentMacrosAndNumbering(t *testing.T) {
	expressions := []Expression{{TeX: `\newcommand{\localvalue}{x}\localvalue=\RR`, Display: true}, {TeX: `\localvalue^2`, Display: true}, {TeX: `\begin{equation}y=2\end{equation}`, Display: true}, {TeX: `z=3`, Display: false}}
	for mode, want := range map[string][]string{"none": {"", "", "", ""}, "ams": {"", "", "1", ""}, "all": {"1", "2", "3", ""}} {
		t.Run(mode, func(t *testing.T) {
			results, err := Render(context.Background(), expressions, Options{Macros: map[string]string{"RR": `\mathbb{R}`}, Numbering: mode})
			if err != nil {
				t.Fatal(err)
			}
			for i, result := range results {
				if result.Error != "" || result.SVG == "" || result.Number != want[i] {
					t.Fatalf("item %d: number=%q error=%q want=%q", i, result.Number, result.Error, want[i])
				}
			}
		})
	}
	results, err := Render(context.Background(), []Expression{{TeX: `\localvalue+\RR`}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Error == "" {
		t.Fatal("document macros leaked into another batch")
	}
	results, err = Render(context.Background(), []Expression{{TeX: `x`, Display: true}}, Options{Numbering: "all"})
	if err != nil || results[0].Number != "1" {
		t.Fatalf("counter leaked: %+v %v", results, err)
	}
}

func TestForwardAndBackwardReferences(t *testing.T) {
	results, err := Render(context.Background(), []Expression{{TeX: `\eqref{later}`, Display: false}, {TeX: `x=1\label{later}`, Display: true}, {TeX: `\eqref{later}`, Display: false}}, Options{Numbering: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Error != "" || results[2].Error != "" || results[0].SVG != results[2].SVG || strings.Contains(results[0].SVG, "3F") || results[1].Number != "1" {
		t.Fatalf("unresolved or inconsistent references: %+v", results)
	}
	isolated, err := Render(context.Background(), []Expression{{TeX: `\eqref{later}`}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if isolated[0].SVG == results[0].SVG {
		t.Fatal("label leaked into a new batch")
	}
}

func TestInvalidTeXIsPerExpressionError(t *testing.T) {
	for _, bad := range []string{`\notARealTeXCommand`, `\frac{1}`, `\begin{matrix}a\end{aligned}`, `{`} {
		results, err := Render(context.Background(), []Expression{{TeX: bad}, {TeX: `x+1`}}, Options{})
		if err != nil {
			t.Fatalf("%q failed entire document: %v", bad, err)
		}
		if results[0].Error == "" || results[0].SVG != "" || results[1].Error != "" || results[1].SVG == "" {
			t.Fatalf("%q: %+v", bad, results)
		}
	}
}

func TestCancellationAndConcurrentIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Render(ctx, []Expression{{TeX: `x`}}, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	if _, err := Render(ctx, []Expression{{TeX: `x`}}, Options{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := Render(context.Background(), []Expression{{TeX: `\newcommand{\foo}{x}\foo\label{one}`, Display: true}}, Options{Numbering: "all"})
			if err != nil || result[0].Error != "" || result[0].Number != "1" {
				t.Errorf("concurrent batch: %+v %v", result, err)
			}
		}()
	}
	wg.Wait()
}

func TestInputLimits(t *testing.T) {
	for name, request := range map[string]struct {
		expressions []Expression
		options     Options
	}{
		"expression": {expressions: []Expression{{TeX: strings.Repeat("x", MaxExpressionBytes+1)}}},
		"count":      {expressions: make([]Expression, MaxExpressions+1)},
		"document": {expressions: func() []Expression {
			e := make([]Expression, 34)
			for i := range e {
				e[i].TeX = strings.Repeat("x", MaxExpressionBytes)
			}
			return e
		}()},
		"numbering": {options: Options{Numbering: "invalid"}},
		"macro":     {options: Options{Macros: map[string]string{"bad name": "x"}}},
		"utf8":      {expressions: []Expression{{TeX: string([]byte{255})}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Render(context.Background(), request.expressions, request.options); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
}

func TestActiveCancellationLeavesReusableEngine(t *testing.T) {
	if _, err := Render(context.Background(), []Expression{{TeX: "x"}}, Options{}); err != nil {
		t.Fatal(err)
	}
	expressions := make([]Expression, 300)
	for i := range expressions {
		expressions[i] = Expression{TeX: `\begin{pmatrix}a&b\\c&d\end{pmatrix}`, Display: true}
	}
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(10*time.Millisecond, cancel)
	defer timer.Stop()
	if _, err := Render(ctx, expressions, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("active cancellation: %v", err)
	}
	results, err := Render(context.Background(), []Expression{{TeX: "y", Display: true}}, Options{Numbering: "all"})
	if err != nil || results[0].Error != "" || results[0].Number != "1" {
		t.Fatalf("engine after interrupt: %+v %v", results, err)
	}
}
