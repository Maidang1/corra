package render

import (
	"image/color"
	"strings"
	"testing"
)

func TestMathCacheKeepsContextualExpressions(t *testing.T) {
	batch := []MathExpression{
		{TeX: `x^2`},
		{TeX: `\begin{equation}a=b\label{eq:one}\end{equation}`, Display: true},
		{TeX: `\eqref{eq:one}`},
		{TeX: `x^2`},
	}
	for round := 0; round < 2; round++ {
		results, err := RenderMathBatch(batch, MathOptions{Numbering: "ams"}, 15, color.Black)
		if err != nil {
			t.Fatal(err)
		}
		if results[1].Number != "1" {
			t.Fatalf("round %d: number = %q", round, results[1].Number)
		}
		if results[0].Image == nil || results[3].Image == nil || string(results[0].SVG) != string(results[3].SVG) {
			t.Fatalf("round %d: duplicate inline formulas differ", round)
		}
		if results[2].Error != nil || !strings.Contains(string(results[2].SVG), "<svg") {
			t.Fatalf("round %d: reference failed: %v", round, results[2].Error)
		}
	}
}
