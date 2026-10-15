//go:build darwin && cgo

package nativerich

import (
	"encoding/json"
	"testing"
)

func TestNativeCodeHeaderSurvivesZoomAndResize(t *testing.T) {
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, nil, nil)
		defer e.Close()
		check := func(want float64) {
			var state struct {
				InsetY float64 `json:"insetY"`
			}
			if err := json.Unmarshal([]byte(nativeTestCodeControls(e)), &state); err != nil || state.InsetY != want {
				t.Errorf("code header inset=%v want=%v err=%v", state.InsetY, want, err)
			}
		}
		e.Load("```go\nfirst\n```")
		e.SetStyle(1.5, false, true)
		check(63)
		e.SetFrame(0, 0, 400, 350)
		check(63)
		e.SetFrame(0, 0, 720, 500)
		check(63)
		e.Load("prose\n\n```go\nfirst\n```")
		check(33)
		e.SetFrame(0, 0, 500, 400)
		check(33)
	})
}
