package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"mygo-desktop-fish/tank"
)

func TestMaxSwellingFishDoesNotEscapeVisualInterior(t *testing.T) {
	for _, scale := range []float64{minScale, 0.55, defaultScale, maxScale} {
		a := newApp()
		a.settings.Scale = scale
		a.tank.Foods, a.tank.Bubbles, a.tank.Chomps = nil, nil, nil
		a.tank.Fishes = a.tank.Fishes[:1]
		f := &a.tank.Fishes[0]
		f.Size, f.Swell, f.Facing = tank.MaxSize, tank.SwellPeak, -1
		f.State = tank.LifeBirthSwell
		p := a.tank.Panel
		f.X, f.Y = p.ClampInside(1000, 1000, f.W(), f.H())
		tt := testView(a)
		r, ok := tt.Find(f.Sprite())
		if !ok {
			t.Fatal("missing swelled adult")
		}
		right, bottom := dipX(p.IX+p.IW, scale), dipY(p.FloorY(), scale)
		if float64(r.X+r.W) > right+0.01 || float64(r.Y+r.H) > bottom+0.01 {
			t.Fatalf("scale %.2f: measured run %v escapes interior right=%.2f bottom=%.2f (physics %.2fx%.2f)", scale, r, right, bottom, f.W()*scale, f.H()*scale)
		}
	}
}

// The bounds here are the real element bounds produced by the platform text
// backend, not an estimate from character count or a re-test of the clamp math.
func TestFishVisualBoundsAcrossLifeFacingResizeAndDPI(t *testing.T) {
	a := newApp()
	a.tank.Foods, a.tank.Bubbles, a.tank.Chomps = nil, nil, nil
	a.tank.Fishes = a.tank.Fishes[:1]
	f := &a.tank.Fishes[0]
	p := a.tank.Panel
	tt := testView(a)
	for _, dpi := range []float32{1, 1.25, 2, 3} {
		tt.SetScale(dpi)
		for _, scale := range []float64{1.25, .75, .35, .55, .35, .75} {
			a.settings.Scale = scale
			w, h := WindowSize(scale)
			tt.SetSize(w, h)
			for _, size := range []float64{tank.NewbornSize, .53, tank.AdultLookSize, tank.StartSize, tank.MaxSize} {
				for _, state := range []tank.LifeState{tank.LifeSwim, tank.LifeBirthSwell, tank.LifeDying} {
					for _, face := range []float64{-1, 1} {
						f.Size, f.State, f.Facing, f.Swell = size, state, face, 1
						if state == tank.LifeBirthSwell {
							f.Swell = tank.SwellPeak
						}
						for _, corner := range [][2]float64{{-1000, -1000}, {1000, -1000}, {-1000, 1000}, {1000, 1000}} {
							f.X, f.Y = p.ClampInside(corner[0], corner[1], f.W(), f.H())
							original := *f
							tt.Frame()
							r, ok := tt.Find(f.Sprite())
							pad := glowStroke(scale)
							left, top := float32(dipX(p.IX, scale)), float32(dipY(p.IY, scale))
							right, bottom := float32(dipX(p.IX+p.IW, scale)), float32(dipY(p.FloorY(), scale))
							if !ok || r.X-pad < left-.01 || r.Y-pad < top-.01 || r.X+r.W+pad > right+.01 || r.Y+r.H+pad > bottom+.01 {
								t.Fatalf("dpi=%v scale=%v size=%v state=%v face=%v sprite=%q bounds=%v pad=%v interior=%v,%v-%v,%v", dpi, scale, size, state, face, f.Sprite(), r, pad, left, top, right, bottom)
							}
							if *f != original {
								t.Fatal("drawing mutated fish lifecycle/physical state")
							}
							font := tankGlyphFont(float32(fishFontSize*f.VisualScale()), scale)
							if r.H > font.Size*1.85+1 {
								t.Fatal("sprite wrapped")
							}
						}
					}
				}
			}
		}
	}
}

func TestMeasuredPlacementDoesNotMoveOrdinaryFishOrShrinkGrowth(t *testing.T) {
	p := tank.Layout(designW, designH)
	for _, scale := range []float64{minScale, defaultScale, maxScale} {
		for _, size := range []float64{.32, .53, 1, 1.2, 1.344} {
			designSize := float32(fishFontSize * size)
			x, y := fishTextPosition(tank.SpriteLeft, designSize, 100, 140, scale, p)
			if math.Abs(x-100) > 1e-5 || math.Abs(y-140) > 1e-5 {
				t.Fatalf("ordinary in-bounds fish moved %v,%v", x, y)
			}
			font := tankGlyphFont(designSize, scale)
			if font.Size != tankFontSize(float64(designSize), scale) {
				t.Fatal("font size changed")
			}
			w, h := ui.ShapeText(tank.SpriteLeft, font, 0).Size()
			if w <= 0 || h <= 0 {
				t.Fatal("empty measured sprite")
			}
		}
	}
}

func TestDocumentedJuvenilePartialStageAndAdultRemainWholeRuns(t *testing.T) {
	for _, c := range []struct {
		size        float64
		right, left string
	}{
		{.32, "o", "o"}, {.4, "o>", "<o"}, {.46, "(o>", "<o)"},
		{.53, "((o>", "<o))"}, {.6, "(((o>", "<o)))"},
		{.66, "<(((o>", "<o)))>"}, {.7, tank.SpriteRight, tank.SpriteLeft},
	} {
		for _, face := range []float64{-1, 1} {
			f := tank.Fish{Size: c.size, Facing: face, Swell: 1}
			want := c.right
			if face < 0 {
				want = c.left
			}
			if f.Sprite() != want {
				t.Fatalf("size=%v face=%v got=%q want=%q", c.size, face, f.Sprite(), want)
			}
			for _, scale := range []float64{minScale, defaultScale, maxScale} {
				font := tankGlyphFont(float32(fishFontSize*f.VisualScale()), scale)
				lines := ui.ShapeText(f.Sprite(), font, 0).Lines()
				if len(lines) != 1 || lines[0].Start != 0 || lines[0].End != len(want) {
					t.Fatalf("not a complete one-line ASCII run: %q %v", want, lines)
				}
			}
		}
	}
}

func BenchmarkFishTextPosition(b *testing.B) {
	p := tank.Layout(designW, designH)
	b.ReportAllocs()
	for b.Loop() {
		fishTextPosition(tank.SpriteLeft, float32(fishFontSize*1.344), p.IX+p.IW-108, p.FloorY()-28, .75, p)
	}
}
