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

// resizeInwardHit is how far a handle reaches into a visible part
// (the glass plate or the stand). The stand mid and FEED mid stay
// free because those points are not on the part edges.
func resizeInwardHit(scale float64) float64 {
	_ = scale
	return resizeEdgeHit
}

// visibleGlassRect is the window-space box of the plate the user sees,
// including the 3D top lip and right-side thickness. This is the edge
// people grab; it is inset from the stand, so a bounding box around
// the whole chassis (used previously) misses the glass face.
func visibleGlassRect(scale float64) rectWH {
	s := clampScale(scale)
	pan := tank.Layout(designW, designH)
	top := pan.GY - pan.Thickness*0.35
	if top < 0 {
		top = 0
	}
	return rectWH{
		X: dipX(pan.GX, s),
		Y: dipY(top, s),
		W: (pan.GW + pan.Thickness) * s,
		H: (pan.GY + pan.GH - top) * s,
	}
}

// visibleStandRect is the platform and legs in window space.
func visibleStandRect(scale float64) rectWH {
	s := clampScale(scale)
	pan := tank.Layout(designW, designH)
	return rectWH{
		X: dipX(pan.BX, s),
		Y: dipY(pan.BY, s),
		W: pan.BW * s,
		H: (pan.BH + pan.LegH) * s,
	}
}

func productionPanel() tank.Panel {
	return tank.Layout(designW, designH)
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

// feedHitRect is the window-space click target for FEED. The visual
// strip can be shorter than a comfortable tap at 35%, so the box may
// grow downward slightly into the gap above the glass. The plate
// itself is not stretched. North resize uses the glass top and the
// pad above FEED, not the strip interior, so the mid of FEED stays
// a feed click.
func feedHitRect(pan tank.Panel, scale float64) rectWH {
	s := clampScale(scale)
	visX := dipX(pan.FX, s)
	visW := pan.FW * s
	visTop := dipY(pan.FY, s)
	visBot := visTop + pan.FH*s
	h := visBot - visTop
	if h < minFeedHitH {
		h = minFeedHitH
	}
	// Do not cover the glass top lip: that visible edge must stay a
	// north resize handle, and at 35% a 24 DIP feed box would swallow it.
	glassTop := visibleGlassRect(s).Y
	if visTop+h > glassTop-1 && glassTop-1 > visTop {
		h = glassTop - 1 - visTop
	}
	return rectWH{X: visX, Y: visTop, W: visW, H: h}
}

type edgeHit struct {
	n, s, e, w     bool
	cn, cs, ce, cw bool
}

func (h *edgeHit) add(o edgeHit) {
	h.n, h.s, h.e, h.w = h.n || o.n, h.s || o.s, h.e || o.e, h.w || o.w
	h.cn, h.cs, h.ce, h.cw = h.cn || o.cn, h.cs || o.cs, h.ce || o.ce, h.cw || o.cw
}

func (h edgeHit) edge() resizeEdge {
	// The plate sits on the stand: glass south and stand north are the
	// same visible seam. Treat it as a south handle, not a canceling
	// N+S pair or a false NE/NW on the stand top.
	if h.n && h.s {
		h.n, h.cn = false, false
	}
	if h.e && h.w {
		h.w, h.cw = false, false
	}
	switch {
	case h.cw && h.cn:
		return resizeNW
	case h.ce && h.cn:
		return resizeNE
	case h.cw && h.cs:
		return resizeSW
	case h.ce && h.cs:
		return resizeSE
	case h.n:
		return resizeN
	case h.s:
		return resizeS
	case h.w:
		return resizeW
	case h.e:
		return resizeE
	default:
		return resizeNone
	}
}

// hitRectEdge reports edge proximity to one visible rectangle.
// inwardN of 0 means that side only accepts outward grabs (used for
// the pad above the FEED strip so the strip itself stays a feed click).
func hitRectEdge(wx, wy float64, r rectWH, outward, cornerAlong, inwardN, inwardS, inwardE, inwardW float64) edgeHit {
	var h edgeHit
	if r.W <= 0 {
		return h
	}
	x, y, w, ht := r.X, r.Y, r.W, r.H
	if ht < 0 {
		ht = 0
	}
	if wx < x-outward || wy < y-outward || wx > x+w+outward || wy > y+ht+outward {
		return h
	}
	h.w = wx >= x-outward && wx <= x+inwardW && wy >= y && wy <= y+ht
	h.e = wx >= x+w-inwardE && wx <= x+w+outward && wy >= y && wy <= y+ht
	h.n = wy >= y-outward && wy <= y+inwardN && wx >= x && wx <= x+w
	h.s = wy >= y+ht-inwardS && wy <= y+ht+outward && wx >= x && wx <= x+w
	h.cw = wx >= x-outward && wx <= x+cornerAlong
	h.ce = wx >= x+w-cornerAlong && wx <= x+w+outward
	h.cn = wy >= y-outward && wy <= y+inwardN
	h.cs = wy >= y+ht-inwardS && wy <= y+ht+outward
	// A corner requires both axis bands; drop a lone corner flag.
	if !(h.cw && h.cn) && !(h.ce && h.cn) && !(h.cw && h.cs) && !(h.ce && h.cs) {
		h.cw, h.ce, h.cn, h.cs = false, false, false, false
	}
	return h
}

// hitResizeEdge reports which visible chassis edge or corner (wx, wy)
// is on, in window DIPs. The user-facing target is the glass plate
// (front face plus thickness) and the stand, not the stand-wide
// bounding box. exclude (typically SET) and the FEED hit box win over
// a resize handle. fx,fy,fw,fh remain the overall chassis box and are
// used only for pad clearance against the native inset.
func hitResizeEdge(wx, wy, fx, fy, fw, fh, scale float64, exclude rectWH) resizeEdge {
	if exclude.contains(wx, wy) {
		return resizeNone
	}
	s := clampScale(scale)
	pan := productionPanel()
	if feedHitRect(pan, s).contains(wx, wy) {
		return resizeNone
	}
	outward, inward, cornerAlong := resizeHandlePads(fx, fy, fw, fh, s)
	var acc edgeHit
	acc.add(hitRectEdge(wx, wy, visibleGlassRect(s), outward, cornerAlong, inward, inward, inward, inward))
	acc.add(hitRectEdge(wx, wy, visibleStandRect(s), outward, cornerAlong, inward, inward, inward, inward))
	// The pad above FEED is a north handle only (no inward bite into
	// the strip). This lets a user grab the top of the widget without
	// fighting the FEED label.
	feed := rectWH{X: dipX(pan.FX, s), Y: dipY(pan.FY, s), W: pan.FW * s, H: 0}
	acc.add(hitRectEdge(wx, wy, feed, outward, cornerAlong, 0, 0, 0, 0))
	return acc.edge()
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
