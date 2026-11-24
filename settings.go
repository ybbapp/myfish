package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

// GlassMode is the manual or automatic tint of the plate.
// Auto follows the desktop light/dark appearance via mygo.Theme,
// not the wallpaper.
type GlassMode string

const (
	GlassClear GlassMode = "clear"
	GlassLight GlassMode = "light"
	GlassDark  GlassMode = "dark"
	GlassAuto  GlassMode = "auto"

	defaultTintOpacity = 0.18
	settingsFileName   = "settings.json"
)

// Settings is the user-adjustable glass look, saved next to the app data.
type Settings struct {
	Mode      GlassMode `json:"mode"`
	Opacity   float64   `json:"opacity"`   // 0–1 fill alpha; 0 is no fill
	Highlight float64   `json:"highlight"` // 0–1 edge highlight strength
	Scale     float64   `json:"scale"`     // 1.0 = original 420×520; default is smaller
}

// DefaultSettings is true transparency with a moderate edge highlight.
func DefaultSettings() Settings {
	return Settings{Mode: GlassClear, Opacity: 0, Highlight: 0.70, Scale: defaultScale}
}

func (s *Settings) Clamp() {
	if s.Opacity < 0 {
		s.Opacity = 0
	}
	if s.Opacity > 1 {
		s.Opacity = 1
	}
	if s.Highlight < 0 {
		s.Highlight = 0
	}
	if s.Highlight > 1 {
		s.Highlight = 1
	}
	s.Scale = clampScale(s.Scale)
	switch s.Mode {
	case GlassClear, GlassLight, GlassDark, GlassAuto:
	default:
		s.Mode = GlassClear
	}
}

// SetMode switches the preset. Leaving "clear" with a zero opacity bumps
// the fill so the light/dark/auto tint is actually visible. Choosing
// "clear" forces opacity back to 0 (true transparency).
func (s *Settings) SetMode(m GlassMode) {
	s.Mode = m
	if m == GlassClear {
		s.Opacity = 0
	} else if s.Opacity < 0.04 {
		s.Opacity = defaultTintOpacity
	}
	s.Clamp()
}

// GlassStyle is the resolved drawing colors for the plate.
type GlassStyle struct {
	FillA float64
	Fill  ui.Color
	Edge  ui.Color
	Side  ui.Color
	Top   ui.Color
	Hi    ui.Color
	Tint  string // "clear", "light" or "dark" after resolving auto
}

// Resolve turns settings and the system light/dark flag into colors.
// systemDark is only meaningful when mode is Auto; it is the desktop
// appearance, not a reading of the wallpaper.
func (s Settings) Resolve(systemDark bool) GlassStyle {
	s.Clamp()
	tint := s.Mode
	if tint == GlassAuto {
		if systemDark {
			tint = GlassLight
		} else {
			tint = GlassDark
		}
	}
	hi := s.Highlight
	st := GlassStyle{Tint: string(tint)}
	switch tint {
	case GlassLight:
		st.FillA = s.Opacity
		st.Fill = ui.RGBA(230, 242, 248, float32(st.FillA))
	case GlassDark:
		st.FillA = s.Opacity
		st.Fill = ui.RGBA(28, 42, 52, float32(st.FillA))
	default:
		st.Tint = "clear"
		st.FillA = 0
		st.Fill = ui.RGBA(0, 0, 0, 0)
	}
	// Thickness stays a light glass edge even on the dark tint, scaled
	// by the highlight slider (0 hides the highlights).
	st.Edge = ui.RGBA(200, 222, 232, float32(0.85*hi))
	st.Side = ui.RGBA(228, 238, 244, float32(0.20*hi))
	st.Top = ui.RGBA(245, 250, 252, float32(0.22*hi))
	st.Hi = ui.RGBA(255, 255, 255, float32(0.70*hi))
	if tint == GlassDark && st.FillA > 0.04 {
		st.Edge = ui.RGBA(170, 200, 214, float32(0.90*hi))
		st.Hi = ui.RGBA(220, 236, 244, float32(0.65*hi))
	}
	return st
}

func LoadSettings(path string) (Settings, error) {
	s := DefaultSettings()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return DefaultSettings(), err
	}
	s.Clamp()
	return s, nil
}

func SaveSettings(path string, s Settings) error {
	s.Clamp()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func settingsPath(dir string) string {
	return filepath.Join(dir, settingsFileName)
}

// ClampedScale is the persisted tank scale, or the default if unset.
func (s Settings) ClampedScale() float64 {
	s.Clamp()
	return s.Scale
}

// RestoreDefaultScale returns the recommended smaller size, not the
// original 420×520 preview size.
func (s *Settings) RestoreDefaultScale() {
	s.Scale = defaultScale
	s.Clamp()
}
