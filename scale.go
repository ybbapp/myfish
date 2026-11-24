package main

import "math"

const (
	designW      = 420
	designH      = 520
	defaultScale = 0.75
	minScale     = 0.55
	maxScale     = 1.25
)

// WindowSize is the frameless tank window in DIPs for a user scale.
// 1.0 matches the original 420×520 preview size.
func WindowSize(scale float64) (width, height int) {
	scale = clampScale(scale)
	return int(math.Round(designW * scale)), int(math.Round(designH * scale))
}

func clampScale(scale float64) float64 {
	if scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return defaultScale
	}
	if scale < minScale {
		return minScale
	}
	if scale > maxScale {
		return maxScale
	}
	return scale
}

func windowToDesign(x, y, scale float64) (float64, float64) {
	s := clampScale(scale)
	return x / s, y / s
}

func designToWindow(x, y, scale float64) (float64, float64) {
	s := clampScale(scale)
	return x * s, y * s
}

func scaleLength(v, scale float64) float64 {
	return v * clampScale(scale)
}

// scaledFontSize is the DIP size sent to the text layout. Glyphs are
// rasterized at this size (and the device pixel ratio), not a stretched
// low-resolution snapshot of the default tank.
func scaledFontSize(size, scale float64) float32 {
	s := size * clampScale(scale)
	if s < 1 {
		return 1
	}
	return float32(s)
}

type workArea struct {
	X, Y, Width, Height int
}

// clampPositionToWorkArea keeps a w×h window as visible as possible
// inside the given work area. It moves the window; it does not change
// the chosen scale.
func clampPositionToWorkArea(x, y, w, h int, area workArea) (int, int) {
	if area.Width <= 0 || area.Height <= 0 {
		return x, y
	}
	if x+w > area.X+area.Width {
		x = area.X + area.Width - w
	}
	if y+h > area.Y+area.Height {
		y = area.Y + area.Height - h
	}
	if x < area.X {
		x = area.X
	}
	if y < area.Y {
		y = area.Y
	}
	return x, y
}

func scaleFrameBounds(x, y, w, h, scale float64) (int, int, int, int) {
	s := clampScale(scale)
	return int(math.Floor(x * s)), int(math.Floor(y * s)), int(math.Ceil(w * s)), int(math.Ceil(h * s))
}
