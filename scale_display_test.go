package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"mygo-desktop-fish/tank"
)

// Exercise layout invalidation in an existing view, not just a fresh window at
// one scale. This is Linux's headless text backend; it cannot accept a report
// from an unknown user platform or prove that native glyph rasterization works.
func TestDisplayDynamicResizeAndDPIKeepsWholeSprite(t *testing.T) {
	for _, dpi := range []float32{1, 1.25, 1.5, 2, 3} {
		for _, body := range []float64{tank.NewbornSize, 0.5, tank.AdultLookSize, tank.StartSize, tank.MaxSize} {
			t.Run(fmt.Sprintf("dpi-%.2f/body-%.2f", dpi, body), func(t *testing.T) {
				a := newApp()
				a.tank.Foods, a.tank.Bubbles, a.tank.Chomps = nil, nil, nil
				a.tank.Fishes = a.tank.Fishes[:1]
				f := &a.tank.Fishes[0]
				f.Size, f.Swell, f.Facing, f.X, f.Y = body, 1, -1, 100, 140
				// Fresh and reused Testers have different clock origins;
				// do not advance one shared fish while comparing layout.
				view := func(c *ui.Context) { a.last = c.Now(); a.view(c) }
				w, h := WindowSize(a.settings.Scale)
				tt := ui.NewTester(view, w, h)
				tt.SetScale(dpi)
				for _, scale := range []float64{1.25, 0.75, 0.55, 0.35, 0.55, 0.35, 0.75, 1.25} {
					a.settings.Scale = scale
					w, h := WindowSize(scale)
					tt.SetSize(w, h)
					tt.Frame()
					sprite := f.Sprite()
					r, ok := tt.Find(sprite)
					if !ok {
						t.Fatalf("scale %v missing whole sprite %q", scale, sprite)
					}
					font := tankFontSize(fishFontSize*f.VisualScale(), scale)
					if r.H > font*1.85+1 || r.W <= 0 {
						t.Fatalf("scale %v wrapped/empty sprite %q bounds=%v font=%v", scale, sprite, r, font)
					}
					// Compare a newly shaped run with the reused view after
					// resize/DPI updates, including both left and right faces.
					for _, face := range []float64{-1, 1} {
						f.Facing = face
						tt.Frame()
						r, ok = tt.Find(f.Sprite())
						fresh := ui.NewTester(view, w, h)
						fresh.SetScale(dpi)
						want, found := fresh.Find(f.Sprite())
						if !ok || !found || math.Abs(float64(r.W-want.W)) > 1.1 || math.Abs(float64(r.H-want.H)) > 1.1 {
							t.Fatalf("stale layout scale=%v face=%v reused=%v fresh=%v", scale, face, r, want)
						}
					}
				}
			})
		}
	}
}

func TestDisplayGrowthVersusCollisionContainment(t *testing.T) {
	// Record the known visual/collision mismatch for investigation. Do not
	// claim that a single small sprite in a screenshot is necessarily a bug:
	// juvenile Sprite() intentionally progresses from o to partial strings.
	for _, scale := range []float64{0.35, 0.55, 0.75, 1.25} {
		a := newApp()
		a.settings.Scale = scale
		a.tank.Fishes = a.tank.Fishes[:1]
		f := &a.tank.Fishes[0]
		f.Size, f.Swell, f.Facing = tank.MaxSize, tank.SwellPeak, -1
		f.State = tank.LifeBirthSwell
		f.X, f.Y = a.tank.Panel.ClampInside(1000, 1000, f.W(), f.H())
		tt := testView(a)
		r, ok := tt.Find(f.Sprite())
		if !ok {
			t.Fatal("missing max/swell sprite")
		}
		right := dipX(a.tank.Panel.IX+a.tank.Panel.IW, scale)
		if float64(r.X+r.W) > right+0.01 {
			t.Fatal("grown/swelled run escaped visual containment")
		}
		t.Logf("scale=%.2f sprite=%q font=%.2f measured=%v physics=%.2fx%.2f visual-right-overrun=%.2f DIP (contained render; user-platform issue remains OPEN)", scale, f.Sprite(), tankFontSize(fishFontSize*f.VisualScale(), scale), r, f.W()*scale, f.H()*scale, float64(r.X+r.W)-right)
	}
}
