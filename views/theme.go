// Package views renders the App state into MyGo native UI and turns user
// input back into App mutations.
package views

import (
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// Design tokens: a quiet, paper-like light theme and a low-contrast dark
// theme, sharing one accent. Spacing is the base unit MyGo widgets size by.
var (
	light = &ui.Theme{
		Dark:           false,
		Background:     ui.Hex("#fbfbf9"),
		Surface:        ui.Hex("#ffffff"),
		SurfaceHover:   ui.Hex("#f2f2ef"),
		SurfacePressed: ui.Hex("#e9e9e5"),
		Border:         ui.Hex("#e3e3de"),
		Text:           ui.Hex("#2b2f36"),
		TextMuted:      ui.Hex("#8a8f98"),
		Accent:         ui.Hex("#3b6fe0"),
		AccentHover:    ui.Hex("#2f5fc9"),
		AccentPressed:  ui.Hex("#2a54b3"),
		AccentText:     ui.Hex("#ffffff"),
		Danger:         ui.Hex("#d64545"),
		Warning:        ui.Hex("#c98a1b"),
		Success:        ui.Hex("#3f9e5f"),
		Selection:      ui.Hex("#d6e4ff"),
		Focus:          ui.Hex("#3b6fe0"),
		Inverse:        ui.Hex("#22242a"),
		InverseText:    ui.Hex("#f2f3f5"),
		Scrollbar:      ui.Hex("#c9c9c4"),
		Radius:         8,
	}
	dark = &ui.Theme{
		Dark:           true,
		Background:     ui.Hex("#16171b"),
		Surface:        ui.Hex("#1e2025"),
		SurfaceHover:   ui.Hex("#26282e"),
		SurfacePressed: ui.Hex("#2d3037"),
		Border:         ui.Hex("#2b2e35"),
		Text:           ui.Hex("#e7e9ee"),
		TextMuted:      ui.Hex("#8b909a"),
		Accent:         ui.Hex("#6c9bff"),
		AccentHover:    ui.Hex("#82abff"),
		AccentPressed:  ui.Hex("#5b8bf2"),
		AccentText:     ui.Hex("#101114"),
		Danger:         ui.Hex("#e06c6c"),
		Warning:        ui.Hex("#d9a13b"),
		Success:        ui.Hex("#5cb87e"),
		Selection:      ui.Hex("#2e4a86"),
		Focus:          ui.Hex("#6c9bff"),
		Inverse:        ui.Hex("#eceef2"),
		InverseText:    ui.Hex("#1a1b1f"),
		Scrollbar:      ui.Hex("#3a3d45"),
		Radius:         8,
	}

	// Document typography.
	fontUI   = "Segoe UI, Inter, SF Pro Text, PingFang SC, Microsoft YaHei, Noto Sans CJK SC, sans-serif"
	fontMono = "Cascadia Code, JetBrains Mono, SF Mono, Menlo, Consolas, Noto Sans Mono CJK SC, monospace"
)

// theme resolves the effective theme for the app's theme mode.
func (v *View) theme(c *ui.Context) *ui.Theme {
	darkNow := v.App.SysDark
	if !v.App.HasSysDark {
		darkNow = false // headless or pre-startup: default to light
	}
	switch v.App.ThemeMode {
	case app.ThemeLight:
		darkNow = false
	case app.ThemeDark:
		darkNow = true
	}
	t := light
	if darkNow {
		t = dark
	}
	// Zoom scales the base text and, gently, the spacing unit.
	zoom := v.App.Zoom
	if zoom <= 0 {
		zoom = 1
	}
	tt := *t
	tt.FontSize = 13.5 * zoom
	tt.Font = fontUI
	tt.Spacing = 4 * minZoom(zoom)
	return &tt
}

func minZoom(z float32) float32 {
	if z < 0.85 {
		return 0.85
	}
	if z > 1.3 {
		return 1.3
	}
	return z
}

// codeColors maps highlight classes to theme colors.
func codeColors(t *ui.Theme) map[string]ui.Color {
	muted := t.TextMuted
	if t.Dark {
		return map[string]ui.Color{
			"text":     t.Text,
			"keyword":  ui.Hex("#c792ea"),
			"string":   ui.Hex("#98c379"),
			"number":   ui.Hex("#d19a66"),
			"comment":  muted,
			"operator": ui.Hex("#56b6c2"),
			"type":     ui.Hex("#e5c07b"),
			"literal":  ui.Hex("#d19a66"),
			"name":     ui.Hex("#61afef"),
		}
	}
	return map[string]ui.Color{
		"text":     t.Text,
		"keyword":  ui.Hex("#a626a4"),
		"string":   ui.Hex("#50a14f"),
		"number":   ui.Hex("#b76b01"),
		"comment":  muted,
		"operator": ui.Hex("#0184bc"),
		"type":     ui.Hex("#c18401"),
		"literal":  ui.Hex("#b76b01"),
		"name":     ui.Hex("#4078f2"),
	}
}
