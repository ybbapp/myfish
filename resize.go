package main

import "math"

// Custom (not native OS) edge-resize of the frameless tank. The window
// keeps DisableResize so the WM will not stretch width and height
// independently. Dragging an edge or corner changes the persisted
// proportional Scale and is clamped to minScale–maxScale.

type resizeEdge int

const (
	resizeNone resizeEdge = iota
	resizeN
	resizeS
	resizeE
	resizeW
	resizeNE
	resizeNW
	resizeSE
	resizeSW
)

// Window-space hit sizes, not design-space lengths: they stay easy to
// hit at the smallest tank. 8 DIP is thinner than half the FEED strip
// at 55%, so the middle of FEED still feeds.
const (
	resizeEdgeHit   = 8.0
	resizeCornerHit = 16.0
)

type rectWH struct {
	X, Y, W, H float64
}

func (e resizeEdge) String() string {
	switch e {
	case resizeN:
		return "n"
	case resizeS:
		return "s"
	case resizeE:
		return "e"
	case resizeW:
		return "w"
	case resizeNE:
		return "ne"
	case resizeNW:
		return "nw"
	case resizeSE:
		return "se"
	case resizeSW:
		return "sw"
	default:
		return "none"
	}
}

func (e resizeEdge) hasN() bool {
	return e == resizeN || e == resizeNE || e == resizeNW
}
func (e resizeEdge) hasS() bool {
	return e == resizeS || e == resizeSE || e == resizeSW
}
func (e resizeEdge) hasE() bool {
	return e == resizeE || e == resizeNE || e == resizeSE
}
func (e resizeEdge) hasW() bool {
	return e == resizeW || e == resizeNW || e == resizeSW
}

func windowFrame(fx, fy, fw, fh, scale float64) (x, y, w, h float64) {
	s := clampScale(scale)
	return dipX(fx, s), dipY(fy, s), fw * s, fh * s
}

// hitResizeEdge reports which frame edge or corner (wx, wy) is on, in
// window DIPs. exclude (typically SET) wins over a resize handle.
func hitResizeEdge(wx, wy, fx, fy, fw, fh, scale float64, exclude rectWH) resizeEdge {
	if exclude.W > 0 && exclude.H > 0 {
		if wx >= exclude.X && wx <= exclude.X+exclude.W && wy >= exclude.Y && wy <= exclude.Y+exclude.H {
			return resizeNone
		}
	}
	x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
	if w <= 0 || h <= 0 {
		return resizeNone
	}
	pad := resizeEdgeHit
	corner := resizeCornerHit
	if wx < x-pad || wy < y-pad || wx > x+w+pad || wy > y+h+pad {
		return resizeNone
	}
	nearW := wx >= x-pad && wx <= x+pad
	nearE := wx >= x+w-pad && wx <= x+w+pad
	nearN := wy >= y-pad && wy <= y+pad
	nearS := wy >= y+h-pad && wy <= y+h+pad
	// Corners use a larger box so they are easier to find.
	nearCW := wx >= x-pad && wx <= x+corner
	nearCE := wx >= x+w-corner && wx <= x+w+pad
	nearCN := wy >= y-pad && wy <= y+corner
	nearCS := wy >= y+h-corner && wy <= y+h+pad
	switch {
	case nearCW && nearCN:
		return resizeNW
	case nearCE && nearCN:
		return resizeNE
	case nearCW && nearCS:
		return resizeSW
	case nearCE && nearCS:
		return resizeSE
	case nearN && wx >= x && wx <= x+w:
		return resizeN
	case nearS && wx >= x && wx <= x+w:
		return resizeS
	case nearW && wy >= y && wy <= y+h:
		return resizeW
	case nearE && wy >= y && wy <= y+h:
		return resizeE
	default:
		return resizeNone
	}
}

// scaleFromResizeDrag maps a screen-space pointer delta to a new scale.
// Opposite edges stay put; aspect ratio stays locked.
func scaleFromResizeDrag(edge resizeEdge, startScale float64, startW, startH int, dx, dy float64) float64 {
	if edge == resizeNone {
		return clampScale(startScale)
	}
	var best, bestDelta float64
	found := false
	consider := func(candidate float64) {
		candidate = clampScale(candidate)
		delta := math.Abs(candidate - startScale)
		if !found || delta > bestDelta {
			best, bestDelta, found = candidate, delta, true
		}
	}
	cw, ch := contentSizeOfWindow(startW, startH)
	if edge.hasE() {
		consider((float64(cw) + dx) / designW)
	}
	if edge.hasW() {
		consider((float64(cw) - dx) / designW)
	}
	if edge.hasS() {
		consider((float64(ch) + dy) / designH)
	}
	if edge.hasN() {
		consider((float64(ch) - dy) / designH)
	}
	if !found {
		return clampScale(startScale)
	}
	return best
}

// positionAfterResize keeps the opposite edge of the starting window
// fixed when the size changes.
func positionAfterResize(edge resizeEdge, startX, startY, startW, startH, newW, newH int) (x, y int) {
	x, y = startX, startY
	if edge.hasW() {
		x = startX + startW - newW
	}
	if edge.hasN() {
		y = startY + startH - newH
	}
	return x, y
}

func resizeInputBounds(fx, fy, fw, fh, scale float64) (int, int, int, int) {
	sx, sy, sw, sh := scaleFrameBounds(fx, fy, fw, fh, scale)
	pad := int(math.Ceil(resizeEdgeHit))
	x, y := sx-pad, sy-pad
	w, h := sw+2*pad, sh+2*pad
	ww, wh := WindowSize(scale)
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > ww {
		w = ww - x
	}
	if y+h > wh {
		h = wh - y
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return x, y, w, h
}
