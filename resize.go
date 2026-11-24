package main

import (
	"math"

	"mygo-desktop-fish/tank"
)

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

// Window-space hit sizes, not design-space lengths. The 8 DIP edge
// band is the preferred outward/inward depth, and corners stay 16 DIP
// wide along the edge so they are easy to find. At 35% the FEED strip
// is only about 13 DIP tall, so inward depth is shrunk just enough to
// leave the FEED mid and stand mid free; the 16 DIP transparent pad
// still gives a comfortable outward grab. Corners use the same inward
// depth so they cannot swallow FEED or SET.
const (
	resizeEdgeHit   = 8.0
	resizeCornerHit = 16.0
	minFeedHitH     = 24.0
	minSetW         = 36.0
	minSetH         = 22.0
)

type rectWH struct {
	X, Y, W, H float64
}

func (r rectWH) contains(wx, wy float64) bool {
	return r.W > 0 && r.H > 0 && wx >= r.X && wx <= r.X+r.W && wy >= r.Y && wy <= r.Y+r.H
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

func clampNonNeg(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// resizeInwardHit is how far the handle reaches into the visible
// chassis. It never covers the FEED mid or the stand mid.
func resizeInwardHit(scale float64) float64 {
	s := clampScale(scale)
	pan := tank.Layout(designW, designH)
	fx, fy, fw, fh := pan.FrameBounds()
	_, y, _, h := windowFrame(fx, fy, fw, fh, s)
	feedMid := dipY(pan.FY+pan.FH/2, s)
	standMid := dipY(pan.BY+(pan.BH+pan.LegH)/2, s)
	topClear := feedMid - y - 2
	botClear := y + h - standMid - 2
	inward := math.Min(resizeEdgeHit, math.Min(topClear, botClear))
	if inward < 3 {
		return 3
	}
	return inward
}

// resizeHandlePads returns the outward depth (kept outside the native
// 5 DIP frameless inset), the inward depth (FEED/stand mid stay free),
// and how far a corner extends along the edge.
func resizeHandlePads(fx, fy, fw, fh, scale float64) (outward, inward, cornerAlong float64) {
	s := clampScale(scale)
	x, y, w, h := windowFrame(fx, fy, fw, fh, s)
	ww, wh := WindowSize(s)
	inset := float64(nativeResizeInset)
	outward = resizeEdgeHit
	outward = math.Min(outward, clampNonNeg(y-inset))
	outward = math.Min(outward, clampNonNeg(x-inset))
	outward = math.Min(outward, clampNonNeg(float64(ww)-inset-(x+w)))
	outward = math.Min(outward, clampNonNeg(float64(wh)-inset-(y+h)))
	inward = resizeInwardHit(s)
	cornerAlong = resizeCornerHit
	return outward, inward, cornerAlong
}

// feedHitRect is the window-space click target for FEED. At small
// scales the visual strip is shorter than a comfortable tap, so the
// box grows downward (it may slightly overlap the top of the glass)
// starting just below the top resize band. The tank face itself is not
// stretched.
func feedHitRect(pan tank.Panel, scale float64) rectWH {
	s := clampScale(scale)
	_, fy, _, _ := pan.FrameBounds()
	frameTop := dipY(fy, s)
	inward := resizeInwardHit(s)
	visX := dipX(pan.FX, s)
	visW := pan.FW * s
	visTop := dipY(pan.FY, s)
	visBot := visTop + pan.FH*s
	top := math.Max(visTop, frameTop+inward)
	h := visBot - top
	if h < minFeedHitH {
		h = minFeedHitH
	}
	return rectWH{X: visX, Y: top, W: visW, H: h}
}

// hitResizeEdge reports which frame edge or corner (wx, wy) is on, in
// window DIPs. exclude (typically SET) wins over a resize handle.
func hitResizeEdge(wx, wy, fx, fy, fw, fh, scale float64, exclude rectWH) resizeEdge {
	if exclude.contains(wx, wy) {
		return resizeNone
	}
	x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
	if w <= 0 || h <= 0 {
		return resizeNone
	}
	outward, inward, cornerAlong := resizeHandlePads(fx, fy, fw, fh, scale)
	if wx < x-outward || wy < y-outward || wx > x+w+outward || wy > y+h+outward {
		return resizeNone
	}
	nearW := wx >= x-outward && wx <= x+inward
	nearE := wx >= x+w-inward && wx <= x+w+outward
	nearN := wy >= y-outward && wy <= y+inward
	nearS := wy >= y+h-inward && wy <= y+h+outward
	// Corners are wider along the edge, but no deeper than the edge
	// band, so they cannot cover FEED or the stand mid at 35%.
	nearCW := wx >= x-outward && wx <= x+cornerAlong
	nearCE := wx >= x+w-cornerAlong && wx <= x+w+outward
	nearCN := wy >= y-outward && wy <= y+inward
	nearCS := wy >= y+h-inward && wy <= y+h+outward
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
	outward, _, _ := resizeHandlePads(fx, fy, fw, fh, scale)
	pad := int(math.Ceil(outward))
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
