package main

import (
	"math"

	"github.com/egoist/mygo/ui"
	"mygo-desktop-fish/tank"
)

// Both measuring and drawing must use the same family, DIP size and weight.
// Generic monospace resolves to different native fonts on different platforms;
// a character-count width or the simulation's 108x28 adult box is not a glyph
// measurement. Font size/weight remain the existing 35%-125% typography policy.
func tankGlyphFont(size float32, scale float64) ui.Font {
	weight := 400
	if tankTextBold(float64(size), scale) {
		weight = 700
	}
	return ui.Font{Family: "monospace", Size: tankFontSize(float64(size), scale), Weight: weight}
}

// fishTextPosition contains the complete measured run, including space for its
// glow, inside the interior. Growth and temporary birth swelling can exceed
// the simulation's stable adult collision box. That box must not be mistaken
// for the visual bounds. Correct only an overflowing draw position: do not
// shrink the fish, clip characters, change physics/feeding/hover coordinates,
// or feed UI font measurements back into lifecycle state.
//
// ShapeText uses the current OS text engine with width=0 (one whole line).
// Deliberately do not introduce a size-only application cache: native font
// fallback and glyph rasterization/DPI remain owned by MyGo's text engine.
func fishTextPosition(sprite string, size float32, x, y, scale float64, p tank.Panel) (float64, float64) {
	w, h := ui.ShapeText(sprite, tankGlyphFont(size, scale), 0).Size()
	// Round outwards: layout may round a fractional font advance or line height.
	// The extra DIP covers antialiased edge coverage beyond the advance box.
	pad := float64(glowStroke(scale)) + 1
	left, top := dipX(p.IX, scale)+pad, dipY(p.IY, scale)+pad
	right := dipX(p.IX+p.IW, scale) - pad - math.Ceil(float64(w))
	bottom := dipY(p.FloorY(), scale) - pad - math.Ceil(float64(h))
	px, py := dipX(x, scale), dipY(y, scale)
	px = math.Max(left, math.Min(px, math.Max(left, right)))
	py = math.Max(top, math.Min(py, math.Max(top, bottom)))
	return windowToDesign(px, py, scale)
}
