package main

import "math"

const (
	designW      = 420
	designH      = 520
	defaultScale = 0.75
	minScale     = 0.35
	maxScale     = 1.25
	// windowPad is a transparent DIP margin around the tank. MyGo's
	// Linux backend intercepts the outer 5 page pixels of a
	// *resizable* frameless window for gtkWindowBeginResizeDrag
	// (resizeInset). The visible tank edge used to sit about 4 DIP
	// inside the window, so edge-drag never reached our handler even
	// though hover could still show a resize cursor. The pad keeps
	// that edge outside the native inset at every legal scale.
	windowPad         = 16
	nativeResizeInset = 5
)

// WindowSize is the frameless tank window in DIPs for a user scale.
// The tank content is designW×designH at scale 1.0; the window adds
// windowPad on each side.
func WindowSize(scale float64) (width, height int) {
	cw, ch := contentSize(scale)
	return cw + 2*windowPad, ch + 2*windowPad
}

func contentSize(scale float64) (width, height int) {
	scale = clampScale(scale)
	return int(math.Round(designW * scale)), int(math.Round(designH * scale))
}

func contentSizeOfWindow(winW, winH int) (width, height int) {
	w, h := winW-2*windowPad, winH-2*windowPad
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// dipX/dipY map a design-space point into the padded window.
func dipX(v, scale float64) float64 { return v*clampScale(scale) + float64(windowPad) }
func dipY(v, scale float64) float64 { return v*clampScale(scale) + float64(windowPad) }

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
	return (x - float64(windowPad)) / s, (y - float64(windowPad)) / s
}

func designToWindow(x, y, scale float64) (float64, float64) {
	s := clampScale(scale)
	return x*s + float64(windowPad), y*s + float64(windowPad)
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

// controlFontSize is for FEED/SET labels only. Below the previous 55%
// minimum the glyphs would shrink to a few DIP; keep a 10 DIP floor so
// they stay readable without changing the 75% default look or stretching
// the tank face.
func controlFontSize(size, scale float64) float32 {
	s := scaledFontSize(size, scale)
	if clampScale(scale) < 0.55 && s < 10 {
		return 10
	}
	return s
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
	return int(math.Floor(x*s)) + windowPad, int(math.Floor(y*s)) + windowPad, int(math.Ceil(w * s)), int(math.Ceil(h * s))
}
