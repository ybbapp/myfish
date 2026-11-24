package main

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

func TestViewShowsFishAndHandle(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, windowW, windowH)
	if !tt.HasText(":::") {
		t.Errorf("missing drag handle, texts %q", tt.Texts())
	}
	if !tt.HasText(tank.SpriteRight) && !tt.HasText(tank.SpriteLeft) {
		t.Errorf("missing fish sprite, texts %q", tt.Texts())
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
