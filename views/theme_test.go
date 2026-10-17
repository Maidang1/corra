package views

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestThemeTextContrast(t *testing.T) {
	tests := []struct {
		name       string
		foreground ui.Color
		background ui.Color
	}{
		{"light selected note", light.AccentText, light.AccentPressed},
		{"dark selected note", dark.AccentText, dark.AccentPressed},
		{"light selected folder", light.Text, light.SurfacePressed},
		{"dark selected folder", dark.Text, dark.SurfacePressed},
		{"light muted surface", light.TextMuted, light.Surface},
		{"dark muted surface", dark.TextMuted, dark.Surface},
		{"light muted pressed", light.TextMuted, light.SurfacePressed},
		{"dark muted pressed", dark.TextMuted, dark.SurfacePressed},
		{"light muted sidebar", light.TextMuted, sidebarBackground(light)},
		{"dark muted sidebar", dark.TextMuted, sidebarBackground(dark)},
		{"light muted sidebar pressed", light.TextMuted, sidebarPressed(light).Over(sidebarBackground(light))},
		{"dark muted sidebar pressed", dark.TextMuted, sidebarPressed(dark).Over(sidebarBackground(dark))},
		{"light danger", light.Danger, light.Surface},
		{"dark danger", dark.Danger, dark.Surface},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ratio := wcagContrast(tt.foreground, tt.background); ratio < 4.5 {
				t.Fatalf("contrast = %.2f:1, want at least 4.50:1", ratio)
			}
		})
	}
}

func wcagContrast(a, b ui.Color) float64 {
	la, lb := wcagLuminance(a), wcagLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func wcagLuminance(c ui.Color) float64 {
	return 0.2126*wcagLinear(c.R) + 0.7152*wcagLinear(c.G) + 0.0722*wcagLinear(c.B)
}

func wcagLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
