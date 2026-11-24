package main

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

func testView(a *app) *ui.Tester {
	w, h := WindowSize(a.settings.ClampedScale())
	return ui.NewTester(a.view, w, h)
}

func winXY(a *app, x, y float64) (float32, float32) {
	wx, wy := designToWindow(x, y, a.settings.ClampedScale())
	return float32(wx), float32(wy)
}

func TestViewShowsPlateFishHandleAndFeedHint(t *testing.T) {
	a := newApp()
	tt := testView(a)
	if !tt.HasText(dragLabel) {
		t.Errorf("missing drag label, texts %q", tt.Texts())
	}
	if !tt.HasText(feedLabel) {
		t.Errorf("missing feed hint, texts %q", tt.Texts())
	}
	if !tt.HasText(setLabel) {
		t.Errorf("missing SET control, texts %q", tt.Texts())
	}
	if !tt.HasText(tank.PlantLeft) && !tt.HasText(tank.PlantRight) {
		t.Errorf("missing plants, texts %q", tt.Texts())
	}
	n := 0
	for _, s := range tt.Texts() {
		if s == tank.SpriteRight || s == tank.SpriteLeft {
			n++
		}
	}
	if n < tank.FishCount {
		t.Errorf("got %d fish sprites, want %d, texts %q", n, tank.FishCount, tt.Texts())
	}
}

func TestViewClickInPlateDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := testView(a)
	tt.ClickAt(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click on the plate dropped food: %+v", a.tank.Foods)
	}
}

func TestViewClickAbovePlateDropsFoodAtX(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := testView(a)
	x := p.FX + p.FW*0.7
	tt.ClickAt(winXY(a, x, p.FY+p.FH/2))
	if len(a.tank.Foods) != 1 {
		t.Fatalf("click on the feed strip did not drop food, texts %q", tt.Texts())
	}
	if math.Abs(a.tank.Foods[0].X+4-x) > 10 {
		t.Fatalf("food x=%v, click x=%v", a.tank.Foods[0].X, x)
	}
	if a.tank.Foods[0].Y > p.IY+12 {
		t.Fatalf("food should start at the top of the plate, y=%v iy=%v", a.tank.Foods[0].Y, p.IY)
	}
}

func TestViewClickOnStandDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := testView(a)
	tt.ClickAt(winXY(a, p.BX+p.BW/2, p.BY+p.BH/2))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click on the stand dropped food: %+v", a.tank.Foods)
	}
}

func TestViewClickOutsideDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	tt := testView(a)
	tt.ClickAt(4, 4)
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click outside the plate dropped food: %+v", a.tank.Foods)
	}
}

func TestViewHoverDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	a.tank.EnableFoodTrace()
	p := a.tank.Panel
	tt := testView(a)
	tt.Move(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("hover dropped food: %+v", a.tank.Foods)
	}
	if !a.tank.Hovering {
		t.Fatal("hover over the plate was not applied")
	}
	// Glass hover must stay approach-only even after the feed-strip dwell.
	tt.Advance(tank.HoverDwell + tank.HoverInterval)
	tt.Frame()
	if spawned := a.tank.FoodTrace().Spawned; spawned != 0 || len(a.tank.Foods) != 0 {
		t.Fatalf("glass hover dropped food after dwell: spawned=%d foods=%d", spawned, len(a.tank.Foods))
	}
	if !a.tank.Hovering {
		t.Fatal("glass hover was cleared while the pointer stayed on the plate")
	}
}

func TestViewFeedHoverDropsAfterDwell(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	a.tank.EnableFoodTrace()
	p := a.tank.Panel
	x := p.FX + p.FW*0.6
	tt := testView(a)
	tt.Move(winXY(a, x, p.FY+p.FH/2))
	if a.tank.FoodTrace().Spawned != 0 {
		t.Fatal("entering the FEED strip dropped food immediately")
	}
	if a.tank.Hovering {
		t.Fatal("FEED-strip hover should not start glass approach")
	}
	tt.Advance(tank.HoverDwell - time.Millisecond)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 0 {
		t.Fatal("FEED hover dropped food before the dwell")
	}
	tt.Advance(time.Millisecond)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 1 {
		t.Fatalf("FEED hover should drop once after the dwell, spawned=%d", a.tank.FoodTrace().Spawned)
	}
	if a.tank.FoodTrace().Spawned == 1 && len(a.tank.Foods) == 1 && math.Abs(a.tank.Foods[0].X+4-x) > 10 {
		t.Fatalf("hover food x=%v, pointer x=%v", a.tank.Foods[0].X, x)
	}
	tt.Advance(tank.HoverInterval)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 2 {
		t.Fatalf("FEED hover should pulse again after the interval, spawned=%d", a.tank.FoodTrace().Spawned)
	}
}

func TestViewFeedHoverPassThroughDoesNotDrop(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	a.tank.EnableFoodTrace()
	p := a.tank.Panel
	tt := testView(a)
	tt.Move(winXY(a, p.FX+p.FW/2, p.FY+p.FH/2))
	tt.Advance(tank.HoverDwell - 20*time.Millisecond)
	tt.Frame()
	tt.Move(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
	tt.Advance(tank.HoverDwell + tank.HoverInterval)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 0 {
		t.Fatalf("a pass-through of the FEED strip dropped food: spawned=%d", a.tank.FoodTrace().Spawned)
	}
	if !a.tank.Hovering {
		t.Fatal("after leaving FEED for the glass, approach hover should be on")
	}
}

func TestViewFeedHoverStopsOnLeave(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	a.tank.EnableFoodTrace()
	p := a.tank.Panel
	tt := testView(a)
	tt.Move(winXY(a, p.FX+p.FW*0.55, p.FY+p.FH/2))
	tt.Advance(tank.HoverDwell)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 1 {
		t.Fatalf("expected one hover drop before leave, spawned=%d", a.tank.FoodTrace().Spawned)
	}
	tt.Move(4, 4)
	if a.feeder.FeedHovering() {
		t.Fatal("leaving the FEED strip should stop hover feeding immediately")
	}
	tt.Advance(tank.HoverInterval * 3)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 1 {
		t.Fatalf("food kept dropping after leave: spawned=%d", a.tank.FoodTrace().Spawned)
	}
}

func TestViewFeedHoverClickStillBursts(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	a.tank.EnableFoodTrace()
	p := a.tank.Panel
	x := p.FX + p.FW*0.7
	tt := testView(a)
	tt.Move(winXY(a, x, p.FY+p.FH/2))
	tt.Advance(tank.HoverDwell)
	tt.Frame()
	if a.tank.FoodTrace().Spawned != 1 {
		t.Fatalf("expected a hover drop before the click, spawned=%d", a.tank.FoodTrace().Spawned)
	}
	tt.ClickAt(winXY(a, x, p.FY+p.FH/2))
	if a.tank.FoodTrace().Spawned != 2 {
		t.Fatalf("click on FEED after hover should burst one more drop, spawned=%d", a.tank.FoodTrace().Spawned)
	}
}

func TestViewHoverDoesNotStealFeedClick(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := testView(a)
	tt.Move(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
	if !a.tank.Hovering {
		t.Fatal("hover over the plate was not applied")
	}
	x := p.FX + p.FW*0.6
	tt.ClickAt(winXY(a, x, p.FY+p.FH/2))
	if len(a.tank.Foods) != 1 {
		t.Fatalf("feed click after hover did not drop food: %d", len(a.tank.Foods))
	}
}

func TestViewHoverLeaveResumesWander(t *testing.T) {
	a := newApp()
	p := a.tank.Panel
	tt := testView(a)
	tt.Move(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
	if !a.tank.Hovering {
		t.Fatal("hover over the plate was not applied")
	}
	tt.Move(4, 4)
	if a.tank.Hovering {
		t.Fatal("leaving the plate did not resume wander")
	}
	for i, f := range a.tank.Fishes {
		if f.Retarget <= 0 {
			t.Fatalf("fish %d has no wander retarget after leave", i)
		}
	}
}

func TestEscapeRestoresInteraction(t *testing.T) {
	a := newApp()
	a.clickThrough = true
	tt := testView(a)
	tt.Key(0, ui.KeyEscape)
	if a.clickThrough {
		t.Fatal("Escape did not restore interaction")
	}
}

func TestCtrlAltIRestoresAndDoesNotToggle(t *testing.T) {
	a := newApp()
	a.clickThrough = true
	tt := testView(a)
	tt.Key(ui.Ctrl|ui.Alt, ui.KeyI)
	if a.clickThrough {
		t.Fatal("Ctrl+Alt+I did not restore interaction")
	}
	tt.Key(ui.Ctrl|ui.Alt, ui.KeyI)
	if a.clickThrough {
		t.Fatal("a second Ctrl+Alt+I toggled click-through back on")
	}
}

func TestRestoreInteractionIsUnconditional(t *testing.T) {
	a := newApp()
	a.clickThrough = false
	a.restoreInteraction()
	if a.clickThrough {
		t.Fatal("restore turned click-through on")
	}
	a.clickThrough = true
	a.restoreInteraction()
	if a.clickThrough {
		t.Fatal("restore left click-through on")
	}
	a.restoreInteraction()
	if a.clickThrough {
		t.Fatal("a second restore toggled click-through back on")
	}
}

func TestToggleClickThroughIsSeparate(t *testing.T) {
	a := newApp()
	a.clickThrough = false
	a.toggleClickThrough()
	if !a.clickThrough {
		t.Fatal("toggle did not enable click-through")
	}
	a.toggleClickThrough()
	if a.clickThrough {
		t.Fatal("toggle did not restore interaction")
	}
}

func TestViewSetButtonOpensSettings(t *testing.T) {
	a := newApp()
	tt := testView(a)
	if err := tt.Click(setLabel); err != nil {
		t.Fatalf("click SET: %v (texts %q)", err, tt.Texts())
	}
	if !a.settingsOpen {
		r, ok := tt.Find(setLabel)
		t.Fatalf("clicking SET did not open the glass settings (found=%v rect=%+v)", ok, r)
	}
}

func TestSettingsViewListsPresets(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.settingsView, settingsW, settingsH)
	for _, s := range []string{"真透明（无填充）", "浅色底", "深色底", "自动（跟随系统浅深主题）", "恢复默认大小", "关闭"} {
		if !tt.HasText(s) {
			t.Errorf("settings missing %q, texts %q", s, tt.Texts())
		}
	}
}

// Bounded reproduction of the reported settings-exit sequence through the
// production view and controls. This is headless; it cannot clear a native
// window/process crash without desktop logs and another native run.
func TestSettingsReportedAdjustmentSequenceHeadless(t *testing.T) {
	a := newApp()
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	tt := ui.NewTester(a.settingsView, settingsW, settingsH)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	slide := func(key string, fraction float32) {
		t.Helper()
		r, ok := tt.Find(key)
		if !ok {
			t.Fatalf("missing slider %s", key)
		}
		tt.ClickAt(r.X+r.W*fraction, r.Y+r.H/2)
		tt.Frame()
		if a.quitting {
			t.Fatal("settings caused quit")
		}
	}
	click("深色底")
	slide("glass-opacity", 0.59)
	if a.settings.Mode != GlassDark || a.settings.Opacity < 0.5 || a.settings.Opacity > 0.7 {
		t.Fatalf("dark/opacity not adjusted: %+v", a.settings)
	}
	slide("glass-highlight", 0.8)
	if a.settings.Highlight < 0.7 || a.settings.Highlight > 0.9 {
		t.Fatalf("highlight not adjusted: %+v", a.settings)
	}
	click("浅色底")
	click("真透明（无填充）")
	slide("glass-highlight", 0.3)
	if a.settings.Mode != GlassClear || a.settings.Opacity != 0 || a.settings.Highlight < 0.2 || a.settings.Highlight > 0.4 || a.quitting {
		t.Fatalf("unexpected final settings or quit: %+v quitting=%v", a.settings, a.quitting)
	}
	loaded, err := LoadSettings(a.settingsPath)
	if err != nil || loaded != a.settings {
		t.Fatalf("saved settings mismatch: %+v err=%v", loaded, err)
	}
	click("关闭")
	if a.quitting {
		t.Fatal("closing settings requested app quit")
	}
	t.Logf("headless production settings view final=%+v quitting=%v; native lifecycle not exercised", a.settings, a.quitting)
}

func TestViewFeedHoverAndStandAtOtherScales(t *testing.T) {
	for _, scale := range []float64{minScale, 0.55, 1, maxScale} {
		a := newApp()
		a.settings.Scale = scale
		a.settings.Clamp()
		a.tank.Foods = nil
		p := a.tank.Panel
		tt := testView(a)
		tt.Move(winXY(a, p.IX+p.IW/2, p.IY+p.IH/3))
		if !a.tank.Hovering {
			t.Fatalf("scale %v: hover over the plate was not applied", scale)
		}
		if len(a.tank.Foods) != 0 {
			t.Fatalf("scale %v: hover dropped food", scale)
		}
		x := p.FX + p.FW*0.25
		tt.ClickAt(winXY(a, x, p.FY+p.FH/2))
		if len(a.tank.Foods) != 1 {
			t.Fatalf("scale %v: feed click did not drop food", scale)
		}
		if math.Abs(a.tank.Foods[0].X+4-x) > 10 {
			t.Fatalf("scale %v: food x=%v click x=%v", scale, a.tank.Foods[0].X, x)
		}
		a.tank.Foods = nil
		tt.ClickAt(winXY(a, p.BX+p.BW/2, p.BY+p.BH/2))
		if len(a.tank.Foods) != 0 {
			t.Fatalf("scale %v: stand click dropped food", scale)
		}
	}
}

func TestSettingsScaleSliderAndRestore(t *testing.T) {
	a := newApp()
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	tt := ui.NewTester(a.settingsView, settingsW, settingsH)
	r, ok := tt.Find("tank-scale")
	if !ok {
		t.Fatalf("missing tank-scale slider, texts %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W*0.95, r.Y+r.H/2)
	tt.Frame()
	if a.settings.Scale < 1.1 {
		t.Fatalf("scale slider did not increase size: %+v", a.settings)
	}
	if err := tt.Click("恢复默认大小"); err != nil {
		t.Fatal(err)
	}
	if a.settings.Scale != defaultScale {
		t.Fatalf("restore default did not reset scale: %v", a.settings.Scale)
	}
	loaded, err := LoadSettings(a.settingsPath)
	if err != nil || loaded.Scale != defaultScale {
		t.Fatalf("restored scale was not persisted: %+v err=%v", loaded, err)
	}
	if a.quitting {
		t.Fatal("scale controls requested quit")
	}
}

func TestViewEastEdgeHoverAndDragUpdatesScale(t *testing.T) {
	a := newApp()
	start := a.settings.ClampedScale()
	g := visibleGlassRect(start)
	tt := testView(a)
	ex, ey := float32(g.X+g.W), float32(g.Y+g.H/2)
	tt.Move(ex, ey)
	if tt.Cursor() != ui.CursorResizeE {
		t.Fatalf("east edge cursor=%v want CursorResizeE", tt.Cursor())
	}
	tt.Press(ex, ey)
	if a.resizing != resizeE {
		t.Fatalf("press on east edge: resizing=%s", a.resizing)
	}
	sw, _ := WindowSize(start)
	tt.Move(ex+float32(0.20*float64(sw)), ey)
	if a.settings.Scale <= start {
		t.Fatalf("east drag should raise scale, got %v start %v", a.settings.Scale, start)
	}
	if a.settings.Scale > maxScale {
		t.Fatalf("scale %v exceeds max", a.settings.Scale)
	}
	if a.lastScale != a.settings.Scale {
		t.Fatalf("lastScale %v should follow live scale %v so the settings slider can stay in sync", a.lastScale, a.settings.Scale)
	}
	got := a.settings.Scale
	tt.Release(ex+float32(0.20*float64(sw)), ey)
	if a.resizing != resizeNone {
		t.Fatal("resize should stop on release")
	}
	if a.settings.Scale != got {
		t.Fatalf("scale changed after release: %v vs %v", a.settings.Scale, got)
	}
}

func TestViewFeedAndStandStayDistinctFromResize(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	pan := a.tank.Panel
	scale := a.settings.ClampedScale()
	g := visibleGlassRect(scale)
	tt := testView(a)
	tt.ClickAt(float32(g.X+g.W), float32(g.Y+g.H/2))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("clicking the east resize edge dropped food: %+v", a.tank.Foods)
	}
	if a.resizing != resizeNone {
		t.Fatal("click (press+release) left resize active")
	}
	tt.ClickAt(winXY(a, pan.FX+pan.FW*0.7, pan.FY+pan.FH/2))
	if len(a.tank.Foods) != 1 {
		t.Fatalf("FEED mid should still drop food after an edge click, foods=%d", len(a.tank.Foods))
	}
	tt.Move(winXY(a, pan.BX+pan.BW/2, pan.BY+(pan.BH+pan.LegH)/2))
	if tt.Cursor() != ui.CursorMove {
		t.Fatalf("stand interior cursor=%v want CursorMove", tt.Cursor())
	}
}

func TestFinishResizeStopsAndCanPersist(t *testing.T) {
	dir := t.TempDir()
	a := newApp()
	a.settingsPath = filepath.Join(dir, "settings.json")
	a.resizing = resizeE
	a.settings.Scale = 1.1
	a.lastScale = 1.1
	a.finishResize()
	if a.resizing != resizeNone {
		t.Fatal("finishResize should clear the drag")
	}
	got, err := LoadSettings(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Scale-1.1) > 1e-9 {
		t.Fatalf("persisted scale %v want 1.1", got.Scale)
	}
}

func TestFishSpritesStayOneLineAtEachScale(t *testing.T) {
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		t.Run(fmt.Sprintf("scale-%.2f", scale), func(t *testing.T) {
			assertFishSpriteOneLine(t, scale, tank.StartSize)
			assertFishSpriteOneLine(t, scale, tank.MaxSize)
		})
	}
}

func assertFishSpriteOneLine(t *testing.T, scale, size float64) {
	t.Helper()
	a := newApp()
	a.settings.Scale = scale
	p := a.tank.Panel
	if len(a.tank.Fishes) < 2 {
		t.Fatalf("need at least two fish, got %d", len(a.tank.Fishes))
	}
	a.tank.Fishes = a.tank.Fishes[:2]
	a.tank.Foods = nil
	a.tank.Bubbles = nil
	a.tank.Chomps = nil
	place := func(f *tank.Fish, facing, y float64) {
		f.VX, f.VY = 0, 0
		f.Facing = facing
		f.Size = size
		f.Swell = 1
		f.State = tank.LifeSwim
		f.X, f.Y = p.ClampInside(p.IX+p.IW, y, f.W(), f.H())
	}
	place(&a.tank.Fishes[0], -1, p.IY+40)
	place(&a.tank.Fishes[1], 1, p.IY+140)
	tt := testView(a)
	font := float64(tankFontSize(float64(fishFontSize)*size, scale))
	for _, sprite := range []string{tank.SpriteLeft, tank.SpriteRight} {
		r, ok := tt.Find(sprite)
		if !ok {
			t.Fatalf("scale %v size %v: missing %q in %q", scale, size, sprite, tt.Texts())
		}
		t.Logf("scale %v size %v %q box=%.1f,%.1f %.1fx%.1f font=%.2f", scale, size, sprite, r.X, r.Y, r.W, r.H, font)
		if r.H > float32(font)*1.85 {
			t.Errorf("scale %v size %v sprite %q height %.2f is taller than one line (font %.2f); characters likely wrapped", scale, size, sprite, r.H, font)
		}
		if r.W < float32(font)*3 {
			t.Errorf("scale %v size %v sprite %q width %.2f is too narrow for a 7-character fish (font %.2f)", scale, size, sprite, r.W, font)
		}
	}
}
