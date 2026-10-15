package app

import (
	"image/color"
	"testing"
	"time"

	"github.com/Maidang1/corra/render"
)

func TestMathImagesShareContextAndReplaceStaleState(t *testing.T) {
	a := New()
	ready := make(chan struct{}, 4)
	a.invalidate = func() { ready <- struct{}{} }
	expressions := []render.MathExpression{{TeX: `\eqref{a}`}, {TeX: `x\label{a}`, Display: true}, {TeX: `y`, Display: true}}
	_, done, _ := a.MathImages(expressions, render.MathOptions{Numbering: "all"}, 16, color.Black)
	if done {
		t.Fatal("first batch should run asynchronously")
	}
	select {
	case <-ready:
	case <-time.After(15 * time.Second):
		t.Fatal("math batch did not finish")
	}
	results, done, err := a.MathImages(expressions, render.MathOptions{Numbering: "all"}, 16, color.Black)
	if err != nil || !done || len(results) != 3 || results[0].Error != nil || results[1].Number != "1" || results[2].Number != "2" {
		t.Fatalf("batch context: %+v %v %v", results, done, err)
	}
	_, _, _ = a.MathImages([]render.MathExpression{{TeX: `z`, Display: true}}, render.MathOptions{Numbering: "all"}, 16, color.Black)
	select {
	case <-ready:
	case <-time.After(15 * time.Second):
		t.Fatal("new document batch did not finish")
	}
	results, done, err = a.MathImages([]render.MathExpression{{TeX: `z`, Display: true}}, render.MathOptions{Numbering: "all"}, 16, color.Black)
	if err != nil || !done || len(results) != 1 || results[0].Number != "1" {
		t.Fatalf("stale document math: %+v %v %v", results, done, err)
	}
}
