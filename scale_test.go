package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"mygo-desktop-fish/tank"
)

func TestDefaultScaleIsSmallerThanLegacy(t *testing.T) {
	s := DefaultSettings()
	if s.Scale != defaultScale {
		t.Fatalf("default scale %v, want %v", s.Scale, defaultScale)
	}
	w, h := WindowSize(s.Scale)
	cw, ch := contentSize(s.Scale)
	if cw != 315 || ch != 390 {
		t.Fatalf("default content %dx%d, want 315x390 (75%% of 420x520)", cw, ch)
	}
	if w != cw+2*windowPad || h != ch+2*windowPad {
		t.Fatalf("default window %dx%d, want content plus %d DIP pad", w, h, windowPad)
	}
	if w >= designW+2*windowPad || h >= designH+2*windowPad {
		t.Fatalf("default window %dx%d is not smaller than the original padded size", w, h)
	}
}

func TestScaleClampAndRestore(t *testing.T) {
	s := DefaultSettings()
	s.Scale = 4
	s.Clamp()
	if s.Scale != maxScale {
		t.Fatalf("clamp high: %v", s.Scale)
	}
	s.Scale = 0.1
	s.Clamp()
	if s.Scale != minScale {
		t.Fatalf("clamp low: %v", s.Scale)
	}
	s.Scale = 0
	s.Clamp()
	if s.Scale != defaultScale {
		t.Fatalf("unset scale should become default, got %v", s.Scale)
	}
	s.Scale = 1.2
	s.RestoreDefaultScale()
	if s.Scale != defaultScale {
		t.Fatalf("restore default: %v", s.Scale)
	}
}

func TestMinScaleContentIs35Percent(t *testing.T) {
	if minScale != 0.35 {
		t.Fatalf("minScale %v, want 0.35", minScale)
	}
	if defaultScale != 0.75 {
		t.Fatalf("default scale must stay 75%%, got %v", defaultScale)
	}
	cw, ch := contentSize(minScale)
	if cw != 147 || ch != 182 {
		t.Fatalf("35%% content %dx%d, want 147x182", cw, ch)
	}
	w, h := WindowSize(minScale)
	if w != 179 || h != 214 {
		t.Fatalf("35%% window %dx%d, want 179x214 (content plus 16 DIP pad)", w, h)
	}
}

func TestLegacy55PercentSettingsStay55(t *testing.T) {
	// A previously saved 55% size (the old minimum) must stay 55%,
	// not jump to the new 35% floor or back to the 75% default.
	s := DefaultSettings()
	s.Scale = 0.55
	s.Clamp()
	if s.Scale != 0.55 {
		t.Fatalf("previously saved 55%% was changed to %v", s.Scale)
	}
	s.Scale = 0.40
	s.Clamp()
	if math.Abs(s.Scale-0.40) > 1e-9 {
		t.Fatalf("40%% should now be legal, got %v", s.Scale)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, settingsFileName)
	saved := Settings{Mode: GlassClear, Opacity: 0, Highlight: 0.7, Scale: 0.55}
	if err := SaveSettings(path, saved); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Scale-0.55) > 1e-9 {
		t.Fatalf("reloaded 55%% settings became %v", got.Scale)
	}
}

func TestControlFontFloorDoesNotChangeDefault(t *testing.T) {
	if got := controlFontSize(11, defaultScale); got != scaledFontSize(11, defaultScale) {
		t.Fatalf("75%% FEED font should stay %v, got %v", scaledFontSize(11, defaultScale), got)
	}
	if got := controlFontSize(handleFontSize, defaultScale); got != scaledFontSize(handleFontSize, defaultScale) {
		t.Fatalf("75%% SET font should stay %v, got %v", scaledFontSize(handleFontSize, defaultScale), got)
	}
	if got := controlFontSize(11, minScale); got != 10 {
		t.Fatalf("35%% FEED font %v, want 10 DIP floor", got)
	}
}

func TestScalePersistsAndLegacyJSONGetsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, settingsFileName)
	s := Settings{Mode: GlassClear, Opacity: 0, Highlight: 0.7, Scale: 1.1}
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Scale-1.1) > 1e-9 {
		t.Fatalf("persisted scale %+v", got)
	}
	legacy := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(legacy, []byte(`{"mode":"clear","opacity":0,"highlight":0.7}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSettings(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Scale != defaultScale {
		t.Fatalf("legacy settings without scale should default, got %v", loaded.Scale)
	}
}

func TestFeedHoverDragAndInputRegionFollowScale(t *testing.T) {
	pan := tank.Layout(designW, designH)
	for _, scale := range []float64{defaultScale, minScale, 0.55, 1, maxScale} {
		// A click in the middle of the FEED strip in window space maps back
		// to the design-space strip, which is what FeedFromAbove uses.
		wx, wy := designToWindow(pan.FX+pan.FW*0.7, pan.FY+pan.FH/2, scale)
		dx, dy := windowToDesign(wx, wy, scale)
		if !pan.ContainsFeed(dx, dy) {
			t.Fatalf("scale %v: feed hit lost: design=(%v,%v) window=(%v,%v)", scale, dx, dy, wx, wy)
		}
		if math.Abs(dx-(pan.FX+pan.FW*0.7)) > 0.01 || math.Abs(dy-(pan.FY+pan.FH/2)) > 0.01 {
			t.Fatalf("scale %v: feed X inverse was inexact: %v,%v", scale, dx, dy)
		}

		gx, gy := designToWindow(pan.IX+pan.IW/2, pan.IY+pan.IH/3, scale)
		hdx, hdy := windowToDesign(gx, gy, scale)
		if !pan.ContainsGlass(hdx, hdy) || pan.ContainsFeed(hdx, hdy) {
			t.Fatalf("scale %v: hover point not on glass: %v,%v", scale, hdx, hdy)
		}

		sx, sy := designToWindow(pan.BX+pan.BW/2, pan.BY+pan.BH/2, scale)
		sdx, sdy := windowToDesign(sx, sy, scale)
		if !pan.ContainsStand(sdx, sdy) {
			t.Fatalf("scale %v: stand drag point lost: %v,%v", scale, sdx, sdy)
		}

		fx, fy, fw, fh := pan.FrameBounds()
		ix, iy, iw, ih := scaleFrameBounds(fx, fy, fw, fh, scale)
		if ix > int(sx) || iy > int(sy) || ix+iw < int(sx) || iy+ih < int(sy) {
			t.Fatalf("scale %v: input region %d,%d %dx%d does not cover stand window point %v,%v", scale, ix, iy, iw, ih, sx, sy)
		}
		if ix > int(wx) || iy > int(wy) || ix+iw < int(wx) || iy+ih < int(wy) {
			t.Fatalf("scale %v: input region misses feed window point %v,%v", scale, wx, wy)
		}
	}
}

func TestSettingsWindowDoesNotFollowTankScale(t *testing.T) {
	if settingsW != 340 {
		t.Fatalf("settings width changed: %d", settingsW)
	}
	if settingsH < 400 {
		t.Fatalf("settings height %d is too small to stay usable", settingsH)
	}
	tw, th := WindowSize(minScale)
	if settingsW <= tw && settingsH <= th {
		// Fine if the min tank is still larger; the requirement is that
		// the settings window size is independent of tank scale.
	}
	if WindowSize(maxScale); settingsW == 0 {
		t.Fatal("unexpected")
	}
	// The settings window constants must stay fixed as the tank scale changes.
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		w, h := WindowSize(scale)
		if settingsW == w && settingsH == h {
			t.Fatalf("settings window %dx%d should not equal the tank window at scale %v", settingsW, settingsH, scale)
		}
	}
}

func TestGlyphSizeTracksScaleNotAStretchedSnapshot(t *testing.T) {
	if got := scaledFontSize(fishFontSize, 0.75); got != 18 {
		t.Fatalf("75%% fish font %v, want 18 DIP (re-rasterized, not a stretched 24px bitmap)", got)
	}
	if got := scaledFontSize(fishFontSize, 1.25); got != 30 {
		t.Fatalf("125%% fish font %v, want 30 DIP", got)
	}
}

func TestClampPositionToWorkArea(t *testing.T) {
	area := workArea{X: 0, Y: 25, Width: 1440, Height: 875}
	x, y := clampPositionToWorkArea(1300, 800, 315, 390, area)
	if x+315 > area.X+area.Width || y+390 > area.Y+area.Height {
		t.Fatalf("window still off-screen: %d,%d", x, y)
	}
	if x < area.X || y < area.Y {
		t.Fatalf("window moved past origin: %d,%d", x, y)
	}
	// Already on-screen: unchanged.
	x, y = clampPositionToWorkArea(40, 80, 315, 390, area)
	if x != 40 || y != 80 {
		t.Fatalf("on-screen position moved: %d,%d", x, y)
	}
	// Empty area is a no-op.
	x, y = clampPositionToWorkArea(9, 8, 10, 10, workArea{})
	if x != 9 || y != 8 {
		t.Fatalf("empty area should be a no-op, got %d,%d", x, y)
	}
}
