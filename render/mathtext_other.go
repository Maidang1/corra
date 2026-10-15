//go:build !darwin || !cgo

package render

import "fmt"

func nativeMathTextPath(text string, size float64, bold, italic bool) (string, error) {
	return "", fmt.Errorf("math: native font fallback requires macOS and cgo")
}
