package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings is the persisted preferences file (settings.json in the config
// directory): selected library, appearance and editor typography. Other toggles
// stay session-only, as before.
type Settings struct {
	LibraryRoot string    `json:"libraryRoot,omitempty"`
	ThemeMode   ThemeMode `json:"themeMode"`
	BodyFont    string    `json:"bodyFont"` // rich editor body family, "" = system default
	MonoFont    string    `json:"monoFont"` // source editor / code family, "" = system default
}

// BodyFonts lists the rich editor's body typefaces. macOS families only:
// the native editor resolves them with NSFont; the empty value keeps the
// system font.
var BodyFonts = []FontChoice{
	{Value: "", Label: "系统默认"},
	{Value: "Songti SC", Label: "宋体"},
	{Value: "Kaiti SC", Label: "楷体"},
	{Value: "Yuanti SC", Label: "圆体"},
	{Value: "New York", Label: "New York"},
}

// MonoFonts lists the typefaces for the source editor and code spans.
var MonoFonts = []FontChoice{
	{Value: "", Label: "系统默认"},
	{Value: "Menlo", Label: "Menlo"},
	{Value: "JetBrains Mono", Label: "JetBrains Mono"},
	{Value: "Cascadia Code", Label: "Cascadia Code"},
	{Value: "Courier New", Label: "Courier New"},
}

// FontChoice is one entry of a font picker: the family name passed to the
// renderers and its display label.
type FontChoice struct {
	Value string
	Label string
}

// LoadSettings reads settings.json, returning the zero value (system
// appearance, default fonts) when the file is missing or unreadable.
func LoadSettings(dir string) Settings {
	var s Settings
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}
	}
	if s.ThemeMode < ThemeSystem || s.ThemeMode > ThemeDark {
		s.ThemeMode = ThemeSystem
	}
	return s
}

// SaveSettings writes settings.json.
func SaveSettings(dir string, s Settings) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o600)
}
