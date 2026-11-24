package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

func TestViewShowsPlateFishHandleAndFeedHint(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, windowW, windowH)
	if !tt.HasText(dragLabel) {
		t.Errorf("missing drag label, texts %q", tt.Texts())
	}
	if !tt.HasText(feedLabel) {
		t.Errorf("missing feed hint, texts %q", tt.Texts())
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
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.ClickAt(float32(p.IX+p.IW/2), float32(p.IY+p.IH/3))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click on the plate dropped food: %+v", a.tank.Foods)
	}
}

func TestViewClickAbovePlateDropsFoodAtX(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := ui.NewTester(a.view, windowW, windowH)
	x := p.FX + p.FW*0.7
	tt.ClickAt(float32(x), float32(p.FY+p.FH/2))
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
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.ClickAt(float32(p.BX+p.BW/2), float32(p.BY+p.BH/2))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click on the stand dropped food: %+v", a.tank.Foods)
	}
}

func TestViewClickOutsideDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.ClickAt(4, 4)
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click outside the plate dropped food: %+v", a.tank.Foods)
	}
}

func TestViewHoverDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.Move(float32(p.IX+p.IW/2), float32(p.IY+p.IH/3))
	if len(a.tank.Foods) != 0 {
		t.Fatalf("hover dropped food: %+v", a.tank.Foods)
	}
	if !a.tank.Hovering {
		t.Fatal("hover over the plate was not applied")
	}
}

func TestViewHoverDoesNotStealFeedClick(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	p := a.tank.Panel
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.Move(float32(p.IX+p.IW/2), float32(p.IY+p.IH/3))
	if !a.tank.Hovering {
		t.Fatal("hover over the plate was not applied")
	}
	x := p.FX + p.FW*0.6
	tt.ClickAt(float32(x), float32(p.FY+p.FH/2))
	if len(a.tank.Foods) != 1 {
		t.Fatalf("feed click after hover did not drop food: %d", len(a.tank.Foods))
	}
}

func TestViewHoverLeaveResumesWander(t *testing.T) {
	a := newApp()
	p := a.tank.Panel
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.Move(float32(p.IX+p.IW/2), float32(p.IY+p.IH/3))
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
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.Key(0, ui.KeyEscape)
	if a.clickThrough {
		t.Fatal("Escape did not restore interaction")
	}
}

func TestCtrlAltIRestoresAndDoesNotToggle(t *testing.T) {
	a := newApp()
	a.clickThrough = true
	tt := ui.NewTester(a.view, windowW, windowH)
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
