package main

import (
	"math"
	"testing"
)

func TestHitResizeEdgesAndCorners(t *testing.T) {
	scale := defaultScale
	pan := productionPanel()
	fx, fy, fw, fh := pan.FrameBounds()
	g := visibleGlassRect(scale)
	none := rectWH{}
	cases := []struct {
		name   string
		wx, wy float64
		want   resizeEdge
	}{
		{"glass inside", g.X + g.W/2, g.Y + g.H/2, resizeNone},
		{"glass n mid", g.X + g.W/2, g.Y, resizeN},
		{"glass s mid", g.X + g.W/2, g.Y + g.H, resizeS},
		{"glass w mid", g.X, g.Y + g.H/2, resizeW},
		{"glass e mid", g.X + g.W, g.Y + g.H/2, resizeE},
		{"glass front right (visible face)", dipX(pan.GX+pan.GW, scale), g.Y + g.H/2, resizeE},
		{"glass nw", g.X, g.Y, resizeNW},
		{"glass ne", g.X + g.W, g.Y, resizeNE},
		{"glass sw", g.X, g.Y + g.H, resizeSW},
		{"glass se", g.X + g.W, g.Y + g.H, resizeSE},
		{"outside pad", g.X + g.W/2, g.Y - resizeEdgeHit - 1, resizeNone},
		{"on glass top lip", g.X + g.W/2, g.Y + 0.5, resizeN},
		{"pad above FEED", dipX(productionPanel().FX+productionPanel().FW/2, scale), dipY(productionPanel().FY, scale) - resizeEdgeHit + 0.5, resizeN},
	}
	for _, tc := range cases {
		if got := hitResizeEdge(tc.wx, tc.wy, fx, fy, fw, fh, scale, none); got != tc.want {
			t.Fatalf("%s: got %s want %s at %.1f,%.1f glass=%+v", tc.name, got, tc.want, tc.wx, tc.wy, g)
		}
	}
}

func TestHitResizeExcludesSetButton(t *testing.T) {
	scale := minScale
	a := newApp()
	pan := a.tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	g := visibleGlassRect(scale)
	set := a.setButtonRect(scale)
	cx, cy := set.X+set.W/2, set.Y+set.H/2
	if got := hitResizeEdge(cx, cy, fx, fy, fw, fh, scale, set); got != resizeNone {
		t.Fatalf("SET should win over resize, got %s", got)
	}
	// Mid-right glass edge, away from SET, stays a resize handle.
	if got := hitResizeEdge(g.X+g.W, g.Y+g.H/2, fx, fy, fw, fh, scale, set); got != resizeE {
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
	if !hit.contains(feedMidX, feedMidY) {
		t.Fatalf("FEED hit %+v does not contain mid %.1f,%.1f", hit, feedMidX, feedMidY)
	}
	if hit.H < pan.FH*scale-0.1 {
		t.Fatalf("FEED hit height %.1f is smaller than the visual strip %.1f", hit.H, pan.FH*scale)
	}
	g := visibleGlassRect(scale)
	if got := hitResizeEdge(g.X+g.W/2, g.Y, fx, fy, fw, fh, scale, rectWH{}); got != resizeN {
		t.Fatalf("glass top mid at 35%% is %s, want n", got)
	}
	outward, _, _ := resizeHandlePads(fx, fy, fw, fh, scale)
	feedTop := dipY(pan.FY, scale)
	if got := hitResizeEdge(feedMidX, feedTop-outward+0.5, fx, fy, fw, fh, scale, rectWH{}); got != resizeN {
		t.Fatalf("pad above FEED at 35%% is %s, want n", got)
	}
	if feedTop-outward < float64(nativeResizeInset) {
		t.Fatalf("outward top handle at %.1f enters the %d DIP native inset", feedTop-outward, nativeResizeInset)
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
	if got := hitResizeEdge(g.X, g.Y+g.H/2, fx, fy, fw, fh, scale, set); got != resizeW {
		t.Fatalf("free W edge at 35%% is %s, want w", got)
	}
}

func TestLoggedVisibleGlassEdgeResizes(t *testing.T) {
	// Exact window-space points from myfish-input-6f74c44-gui.log at 75%:
	// the tester grabbed the visible glass right edge and SE corner,
	// hitResizeEdge returned none because the handle used the stand-wide
	// bounding box (east band started at ~319, glass face is at ~312).
	scale := defaultScale
	pan := productionPanel()
	fx, fy, fw, fh := pan.FrameBounds()
	none := rectWH{}
	if got := hitResizeEdge(312, 195, fx, fy, fw, fh, scale, none); got != resizeE {
		t.Fatalf("logged visible right edge 312,195 is %s, want e (glass front right=%.2f)", got, dipX(pan.GX+pan.GW, scale))
	}
	if got := hitResizeEdge(312, 331, fx, fy, fw, fh, scale, none); got != resizeSE && got != resizeE && got != resizeS {
		t.Fatalf("logged visible SE 312,331 is %s, want se (or e/s)", got)
	}
	// Prefer the true corner classification.
	if got := hitResizeEdge(312, 331, fx, fy, fw, fh, scale, none); got != resizeSE {
		t.Fatalf("logged visible SE 312,331 is %s, want se", got)
	}
}

func TestVisibleGlassEdgesAtMinScale(t *testing.T) {
	scale := minScale
	pan := productionPanel()
	fx, fy, fw, fh := pan.FrameBounds()
	g := visibleGlassRect(scale)
	none := rectWH{}
	if got := hitResizeEdge(g.X+g.W, g.Y+g.H/2, fx, fy, fw, fh, scale, none); got != resizeE {
		t.Fatalf("35%% glass right is %s, want e at x=%.1f", got, g.X+g.W)
	}
	front := dipX(pan.GX+pan.GW, scale)
	if got := hitResizeEdge(front, g.Y+g.H/2, fx, fy, fw, fh, scale, none); got != resizeE {
		t.Fatalf("35%% glass front right %.1f is %s, want e", front, got)
	}
	if got := hitResizeEdge(g.X+g.W, g.Y+g.H, fx, fy, fw, fh, scale, none); got != resizeSE {
		t.Fatalf("35%% glass SE is %s, want se", got)
	}
	// Glass interior must not resize.
	if got := hitResizeEdge(g.X+g.W/2, g.Y+g.H/2, fx, fy, fw, fh, scale, none); got != resizeNone {
		t.Fatalf("35%% glass mid is %s, want none", got)
	}
}

func TestFeedMidStillFeedsNotResize(t *testing.T) {
	pan := productionPanel()
	fx, fy, fw, fh := pan.FrameBounds()
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		wx, wy := dipX(pan.FX+pan.FW/2, scale), dipY(pan.FY+pan.FH/2, scale)
		if got := hitResizeEdge(wx, wy, fx, fy, fw, fh, scale, rectWH{}); got != resizeNone {
			t.Fatalf("FEED mid at %.2f is %s, want none", scale, got)
		}
		hit := feedHitRect(pan, scale)
		if !hit.contains(wx, wy) {
			t.Fatalf("FEED mid at %.2f not in hit box %+v", scale, hit)
		}
		if hit.H < pan.FH*scale-0.1 {
			t.Fatalf("FEED hit height %.1f at %.2f is smaller than the visual strip", hit.H, scale)
		}
	}
}
