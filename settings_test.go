package main

import (
	"path/filepath"
	"testing"
)

func TestDefaultSettingsAreTrulyClear(t *testing.T) {
	s := DefaultSettings()
	g := s.Resolve(false)
	if s.Mode != GlassClear || g.FillA != 0 || g.Tint != "clear" {
		t.Fatalf("default should be true clear, got mode=%s fill=%v tint=%s", s.Mode, g.FillA, g.Tint)
	}
	if s.Highlight < 0.5 {
		t.Fatalf("default highlight %v should stay visible", s.Highlight)
	}
}

func TestResolveLightDarkAndAuto(t *testing.T) {
	s := DefaultSettings()
	s.SetMode(GlassLight)
	if s.Opacity != defaultTintOpacity {
		t.Fatalf("leaving clear should bump opacity, got %v", s.Opacity)
	}
	light := s.Resolve(true)
	if light.Tint != "light" || light.FillA != defaultTintOpacity {
		t.Fatalf("light preset: %+v", light)
	}
	s.SetMode(GlassDark)
	dark := s.Resolve(false)
	if dark.Tint != "dark" || dark.FillA != defaultTintOpacity {
		t.Fatalf("dark preset: %+v", dark)
	}
	s.SetMode(GlassAuto)
	autoDarkDesk := s.Resolve(true)
	if autoDarkDesk.Tint != "light" {
		t.Fatalf("auto on a dark desktop should use the light tint, got %s", autoDarkDesk.Tint)
	}
	autoLightDesk := s.Resolve(false)
	if autoLightDesk.Tint != "dark" {
		t.Fatalf("auto on a light desktop should use the dark tint, got %s", autoLightDesk.Tint)
	}
	s.SetMode(GlassClear)
	if s.Opacity != 0 || s.Resolve(true).FillA != 0 {
		t.Fatal("clear must force true transparency")
	}
}

func TestOpacityAndHighlightRange(t *testing.T) {
	s := Settings{Mode: GlassLight, Opacity: 2, Highlight: -1}
	s.Clamp()
	if s.Opacity != 1 || s.Highlight != 0 {
		t.Fatalf("clamp failed: %+v", s)
	}
	s.Opacity, s.Highlight = 0, 0
	g := s.Resolve(false)
	if g.FillA != 0 {
		t.Fatalf("opacity 0 should be no fill, got %v", g.FillA)
	}
	s.Opacity, s.Highlight = 1, 1
	g = s.Resolve(false)
	if g.FillA != 1 {
		t.Fatalf("opacity 1 should be fully tinted, got %v", g.FillA)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, settingsFileName)
	s := Settings{Mode: GlassDark, Opacity: 0.4, Highlight: 0.2}
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != GlassDark || got.Opacity != 0.4 || got.Highlight != 0.2 {
		t.Fatalf("round trip %+v", got)
	}
	missing, err := LoadSettings(filepath.Join(dir, "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Mode != GlassClear {
		t.Fatalf("missing file should yield defaults, got %+v", missing)
	}
}
