package main

import (
	"math"
	"testing"
)

func TestHitResizeEdgesAndCorners(t *testing.T) {
	scale := defaultScale
	fx, fy, fw, fh := 24.0, 8.0, 372.0, 456.0
	x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
	none := rectWH{}
	cases := []struct {
		name   string
		wx, wy float64
		want   resizeEdge
	}{
		{"inside", x + w/2, y + h/2, resizeNone},
		{"n mid", x + w/2, y, resizeN},
		{"s mid", x + w/2, y + h, resizeS},
		{"w mid", x, y + h/2, resizeW},
		{"e mid", x + w, y + h/2, resizeE},
		{"nw", x, y, resizeNW},
		{"ne", x + w, y, resizeNE},
		{"sw", x, y + h, resizeSW},
		{"se", x + w, y + h, resizeSE},
		{"outside pad", x + w/2, y - resizeEdgeHit - 1, resizeNone},
		{"outward n still hits", x + w/2, y - resizeEdgeHit + 0.5, resizeN},
	}
	for _, tc := range cases {
		if got := hitResizeEdge(tc.wx, tc.wy, fx, fy, fw, fh, scale, none); got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestHitResizeExcludesSetButton(t *testing.T) {
	scale := minScale
	fx, fy, fw, fh := 24.0, 8.0, 372.0, 456.0
	x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
	set := rectWH{X: x + w - 16, Y: y + h - 16, W: 20, H: 20}
	wx, wy := set.X+4, set.Y+4
	if got := hitResizeEdge(wx, wy, fx, fy, fw, fh, scale, set); got != resizeNone {
		t.Fatalf("SET should win over resize, got %s", got)
	}
	if got := hitResizeEdge(x+w, y+h, fx, fy, fw, fh, scale, set); got != resizeNone {
		t.Fatalf("corner under SET should not resize, got %s", got)
	}
	// Mid-right edge above SET stays a resize handle.
	if got := hitResizeEdge(x+w, y+h/2, fx, fy, fw, fh, scale, set); got != resizeE {
		t.Fatalf("free E edge: got %s", got)
	}
}

func TestHitResizeLeavesFeedAndStandInteriors(t *testing.T) {
	// Use the production chassis geometry so the middle of FEED and the
	// middle of the stand stay free at every legal scale.
	pan := newApp().tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	none := rectWH{}
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		feedWX := dipX(pan.FX+pan.FW/2, scale)
		feedWY := dipY(pan.FY+pan.FH/2, scale)
		if got := hitResizeEdge(feedWX, feedWY, fx, fy, fw, fh, scale, none); got != resizeNone {
			t.Fatalf("FEED mid at scale %.2f is %s, want none (feed mid=%.1f,%.1f frame=%.1f,%.1f %.1fx%.1f)",
				scale, got, feedWX, feedWY, fx*scale, fy*scale, fw*scale, fh*scale)
		}
		standWX := dipX(pan.BX+pan.BW/2, scale)
		standWY := dipY(pan.BY+(pan.BH+pan.LegH)/2, scale)
		if got := hitResizeEdge(standWX, standWY, fx, fy, fw, fh, scale, none); got != resizeNone {
			t.Fatalf("stand mid at scale %.2f is %s, want none", scale, got)
		}
	}
}

func TestScaleFromResizeDragClampsAndLocksAspect(t *testing.T) {
	start := defaultScale
	sw, sh := WindowSize(start)
	// Drag the east edge far enough to exceed 125%.
	got := scaleFromResizeDrag(resizeE, start, sw, sh, 2000, 0)
	if got != maxScale {
		t.Fatalf("E oversize: got %v want max", got)
	}
	got = scaleFromResizeDrag(resizeW, start, sw, sh, 2000, 0) // dragging right on west shrinks
	if got != minScale {
		t.Fatalf("W shrink past min: got %v want min", got)
	}
	cw, ch := contentSizeOfWindow(sw, sh)
	// South drag by 20% of the content height.
	got = scaleFromResizeDrag(resizeS, start, sw, sh, 0, 0.20*float64(ch))
	want := clampScale(start + 0.20*float64(ch)/designH)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("S +20%% height: got %v want %v", got, want)
	}
	// Corner: prefer the axis with the larger implied change.
	got = scaleFromResizeDrag(resizeSE, start, sw, sh, 0.10*float64(cw), 0.30*float64(ch))
	want = clampScale(start + 0.30*float64(ch)/designH)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("SE mixed: got %v want %v (larger dy)", got, want)
	}
	if got := scaleFromResizeDrag(resizeNone, start, sw, sh, 40, 40); got != start {
		t.Fatalf("none edge should not scale, got %v", got)
	}
}

func TestPositionAfterResizeKeepsOppositeEdge(t *testing.T) {
	sx, sy, sw, sh := 100, 80, 315, 390
	nw, nh := 231, 286
	x, y := positionAfterResize(resizeE, sx, sy, sw, sh, nw, nh)
	if x != sx || y != sy {
		t.Fatalf("E should keep NW origin, got %d,%d", x, y)
	}
	x, y = positionAfterResize(resizeW, sx, sy, sw, sh, nw, nh)
	if x != sx+sw-nw || y != sy {
		t.Fatalf("W should keep east edge, got %d,%d want %d,%d", x, y, sx+sw-nw, sy)
	}
	x, y = positionAfterResize(resizeN, sx, sy, sw, sh, nw, nh)
	if x != sx || y != sy+sh-nh {
		t.Fatalf("N should keep south edge, got %d,%d", x, y)
	}
	x, y = positionAfterResize(resizeSE, sx, sy, sw, sh, nw, nh)
	if x != sx || y != sy {
		t.Fatalf("SE should keep NW origin, got %d,%d", x, y)
	}
	x, y = positionAfterResize(resizeNW, sx, sy, sw, sh, nw, nh)
	if x != sx+sw-nw || y != sy+sh-nh {
		t.Fatalf("NW should keep SE corner, got %d,%d want %d,%d", x, y, sx+sw-nw, sy+sh-nh)
	}
}

func TestResizeInputBoundsPadTheFrame(t *testing.T) {
	fx, fy, fw, fh := 24.0, 8.0, 372.0, 456.0
	sx, sy, sw, sh := scaleFrameBounds(fx, fy, fw, fh, defaultScale)
	rx, ry, rw, rh := resizeInputBounds(fx, fy, fw, fh, defaultScale)
	if rx >= sx || ry >= sy || rx+rw <= sx+sw || ry+rh <= sy+sh {
		t.Fatalf("input region %d,%d %dx%d should pad frame %d,%d %dx%d", rx, ry, rw, rh, sx, sy, sw, sh)
	}
}

func TestSettingsScaleFollowsResizeThenPersists(t *testing.T) {
	dir := t.TempDir()
	s := DefaultSettings()
	path := dir + "/settings.json"
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	startW, startH := WindowSize(s.Scale)
	s.Scale = scaleFromResizeDrag(resizeE, s.Scale, startW, startH, 0.15*float64(startW), 0)
	s.Clamp()
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Scale-s.Scale) > 1e-9 {
		t.Fatalf("persisted scale %v want %v", got.Scale, s.Scale)
	}
	if got.Scale == defaultScale {
		t.Fatal("expected resize to change the default scale")
	}
}

func TestProductionSetButtonWinsOverResize(t *testing.T) {
	a := newApp()
	pan := a.tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		set := a.setButtonRect(scale)
		cx, cy := set.X+set.W/2, set.Y+set.H/2
		if got := hitResizeEdge(cx, cy, fx, fy, fw, fh, scale, set); got != resizeNone {
			t.Fatalf("SET mid at scale %.2f is %s, want none", scale, got)
		}
	}
}

func TestFrameEdgeClearsNativeResizeInset(t *testing.T) {
	pan := newApp().tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
		ww, wh := WindowSize(scale)
		clearance := float64(nativeResizeInset + 1)
		if x < clearance || y < clearance || float64(ww)-(x+w) < clearance || float64(wh)-(y+h) < clearance {
			t.Fatalf("scale %.2f: frame (%.1f,%.1f %.1fx%.1f) is inside the %dpx native inset of window %dx%d",
				scale, x, y, w, h, nativeResizeInset, ww, wh)
		}
	}
}

func TestMinScaleKeepsFeedSetAndEdgesUsable(t *testing.T) {
	a := newApp()
	pan := a.tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	scale := minScale
	feedMidX := dipX(pan.FX+pan.FW/2, scale)
	feedMidY := dipY(pan.FY+pan.FH/2, scale)
	if got := hitResizeEdge(feedMidX, feedMidY, fx, fy, fw, fh, scale, rectWH{}); got != resizeNone {
		t.Fatalf("FEED mid at 35%% is %s, want none", got)
	}
	hit := feedHitRect(pan, scale)
	if hit.H < minFeedHitH {
		t.Fatalf("FEED hit height %.1f, want at least %.0f", hit.H, minFeedHitH)
	}
	if !hit.contains(feedMidX, feedMidY) {
		t.Fatalf("FEED hit %+v does not contain mid %.1f,%.1f", hit, feedMidX, feedMidY)
	}
	x, y, w, h := windowFrame(fx, fy, fw, fh, scale)
	inward := resizeInwardHit(scale)
	if hit.Y < y+inward-0.01 {
		t.Fatalf("FEED hit top %.1f overlaps the inward resize band ending at %.1f", hit.Y, y+inward)
	}
	if got := hitResizeEdge(x+w/2, y, fx, fy, fw, fh, scale, rectWH{}); got != resizeN {
		t.Fatalf("top edge mid at 35%% is %s, want n", got)
	}
	outward, _, _ := resizeHandlePads(fx, fy, fw, fh, scale)
	if got := hitResizeEdge(x+w/2, y-outward+0.5, fx, fy, fw, fh, scale, rectWH{}); got != resizeN {
		t.Fatalf("outward top pad at 35%% is %s, want n", got)
	}
	if y-outward < float64(nativeResizeInset) {
		t.Fatalf("outward top handle at %.1f enters the %d DIP native inset", y-outward, nativeResizeInset)
	}
	set := a.setButtonRect(scale)
	if set.W < minSetW || set.H < minSetH {
		t.Fatalf("SET hit %.1fx%.1f, want at least %.0fx%.0f", set.W, set.H, minSetW, minSetH)
	}
	if got := hitResizeEdge(set.X+set.W/2, set.Y+set.H/2, fx, fy, fw, fh, scale, set); got != resizeNone {
		t.Fatalf("SET mid at 35%% is %s, want none", got)
	}
	standMidX := dipX(pan.BX+pan.BW/2, scale)
	standMidY := dipY(pan.BY+(pan.BH+pan.LegH)/2, scale)
	if got := hitResizeEdge(standMidX, standMidY, fx, fy, fw, fh, scale, set); got != resizeNone {
		t.Fatalf("stand mid at 35%% is %s, want none", got)
	}
	// A side edge mid, away from SET, remains a resize handle.
	if got := hitResizeEdge(x, y+h/2, fx, fy, fw, fh, scale, set); got != resizeW {
		t.Fatalf("free W edge at 35%% is %s, want w", got)
	}
}
