package main

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

func TestViewShowsFishAndHandle(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, windowW, windowH)
	if !tt.HasText(dragLabel) {
		t.Errorf("missing drag handle %q, texts %q", dragLabel, tt.Texts())
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

func TestViewClickDropsFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.ClickAt(280, 200)
	if len(a.tank.Foods) == 0 {
		t.Fatalf("click did not drop food, texts %q", tt.Texts())
	}
	if !tt.HasText(tank.FoodGlyph) {
		t.Errorf("missing food glyph, texts %q", tt.Texts())
	}
}

func TestViewClickOnHandleDoesNotDropFood(t *testing.T) {
	a := newApp()
	a.tank.Foods = nil
	tt := ui.NewTester(a.view, windowW, windowH)
	tt.ClickAt(dragLeft+dragW/2, dragTop+dragH/2)
	if len(a.tank.Foods) != 0 {
		t.Fatalf("click on drag handle dropped food: %+v", a.tank.Foods)
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
