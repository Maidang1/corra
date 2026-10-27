package app

import (
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Missing file: system appearance and default fonts.
	if s := LoadSettings(dir); s.ThemeMode != ThemeSystem || s.BodyFont != "" || s.MonoFont != "" {
		t.Fatalf("zero settings expected, got %+v", s)
	}
	want := Settings{ThemeMode: ThemeDark, BodyFont: "Songti SC", MonoFont: "Menlo", LibraryRoot: filepath.Join(t.TempDir(), "文档")}
	if err := SaveSettings(dir, want); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings(dir); got != want {
		t.Fatalf("settings round trip: got %+v, want %+v", got, want)
	}
}

func TestLoadSettingsClampsThemeMode(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSettings(dir, Settings{ThemeMode: ThemeMode(99)}); err != nil {
		t.Fatal(err)
	}
	// A mode out of range falls back to following the system.
	if s := LoadSettings(dir); s.ThemeMode != ThemeSystem {
		t.Fatalf("theme mode = %v, want ThemeSystem", s.ThemeMode)
	}
}

func TestSettingsFileLocation(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSettings(dir, Settings{ThemeMode: ThemeLight}); err != nil {
		t.Fatal(err)
	}
	if _, err := filepath.Glob(filepath.Join(dir, "settings.json")); err != nil || len([]string{}) > 0 {
		t.Fatal(err)
	}
	if s := LoadSettings(dir); s.ThemeMode != ThemeLight {
		t.Fatalf("theme mode = %v, want ThemeLight", s.ThemeMode)
	}
}
