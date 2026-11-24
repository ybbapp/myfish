package tank

import (
	"math"
	"testing"
	"time"
)

func centerToFood(f Fish, food Food) float64 {
	return math.Hypot(food.X-(f.X+fishW/2), food.Y-(f.Y+fishH/2))
}

func TestCornerFoodCenterIsOutsideEatRadius(t *testing.T) {
	// Diagnosis of the pile-in-the-corner bug: eat used the body center
	// of a 108-wide fish against eatRadius 28, so a pellet clamped to
	// the left or right wall could never be reached. Confirmed on the
	// left (user report) and the right (4a02002 demo review, ~20–29s).
	tk := New(420, 520, 1)
	p := tk.Panel
	left, ly := p.ClampInside(p.IX, p.FloorY()-8, 8, 8)
	right, ry := p.ClampInside(p.IX+p.IW, p.FloorY()-8, 8, 8)
	fx, fy := p.ClampInside(p.IX, p.FloorY()-fishH, fishW, fishH)
	dLeft := centerToFood(Fish{X: fx, Y: fy}, Food{X: left, Y: ly})
	fxR, fyR := p.ClampInside(p.IX+p.IW-fishW, p.FloorY()-fishH, fishW, fishH)
	dRight := centerToFood(Fish{X: fxR, Y: fyR}, Food{X: right, Y: ry})
	t.Logf("center-to-wall food: left=%0.1f right=%0.1f eatRadius=%0.1f fishW=%0.0f", dLeft, dRight, eatRadius, fishW)
	if dLeft < eatRadius && dRight < eatRadius {
		t.Fatalf("center already reaches both walls (L=%v R=%v r=%v); diagnosis stale", dLeft, dRight, eatRadius)
	}
}

func TestFishEatFoodAtLeftCenterAndRightBottom(t *testing.T) {
	p0 := Layout(420, 520)
	for _, side := range []struct {
		name string
		x    float64
	}{
		{"left", p0.FX + 4},
		{"center", p0.FX + p0.FW/2},
		{"right", p0.FX + p0.FW - 4},
	} {
		t.Run(side.name, func(t *testing.T) {
			tk := New(420, 520, 3)
			tk.Foods = nil
			tk.Hovering = false
			if !tk.FeedFromAbove(side.x, p0.FY+p0.FH/2) {
				t.Fatal("feed")
			}
			for i := 0; i < 240; i++ {
				tk.Step(0.05)
				assertNoBoxOverlap(t, tk, i)
			}
			if len(tk.Foods) != 0 || tk.bites == 0 {
				t.Fatalf("%s food uneaten at %v floor=%0.1f fishes=%v bites=%d",
					side.name, foods(tk), tk.Panel.FloorY(), positions(tk), tk.bites)
			}
		})
	}
}

func TestRightEdgeLongPressFoodIsEventuallyEaten(t *testing.T) {
	tk := New(420, 520, 9)
	tk.Foods = nil
	p := tk.Panel
	x, y := p.FX+p.FW-6, p.FY+p.FH/2
	f := NewFeeder()
	now := t0()
	f.Down(now, x, y)
	for i := 0; i < 10; i++ {
		at := now.Add(HoldThreshold + time.Duration(i)*RepeatEvery)
		if f.Tick(at, true) {
			tk.FeedFromAbove(f.X, f.Y)
		}
	}
	if f.Up(now.Add(2*time.Second), true) {
		tk.FeedFromAbove(f.X, f.Y)
	}
	n := len(tk.Foods)
	if n < 2 {
		t.Fatalf("expected several pellets from a ~2s hold, got %d", n)
	}
	for i := 0; i < 300; i++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	if len(tk.Foods) != 0 || tk.bites != n {
		t.Fatalf("right-edge long press: ate %d of %d, leftover %v (must be eaten, not TTL-wiped)", tk.bites, n, foods(tk))
	}
}

func TestHoverDoesNotPreventCornerEating(t *testing.T) {
	tk := New(420, 520, 5)
	tk.Foods = nil
	p := tk.Panel
	tk.SetHover(p.IX+p.IW/2, p.IY+p.IH/3)
	if !tk.FeedFromAbove(p.FX+4, p.FY+p.FH/2) {
		t.Fatal("feed")
	}
	for i := 0; i < 240; i++ {
		tk.Step(0.05)
	}
	if len(tk.Foods) != 0 || tk.bites == 0 {
		t.Fatalf("hover left corner food uneaten: %v bites=%d", foods(tk), tk.bites)
	}
}

func TestFishSwimThroughPlantsToEat(t *testing.T) {
	tk := New(420, 520, 6)
	tk.Foods = nil
	p := tk.Panel
	fx, fy := p.IX+p.IW*0.2, p.FloorY()-8
	if !p.ContainsInterior(fx, fy) {
		t.Fatalf("plant/floor band is outside interior: floor=%v planty=%v", p.FloorY(), p.PlantY)
	}
	if !tk.PlaceFood(fx, fy) {
		t.Fatal("place at floor")
	}
	tk.Fishes[0].X, tk.Fishes[0].Y = p.ClampInside(p.IX+20, p.IY+20, fishW, fishH)
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = p.ClampInside(p.IX+p.IW-fishW-4, p.IY+4, fishW, fishH)
	}
	reachedPlant := false
	for i := 0; i < 240; i++ {
		tk.Step(0.05)
		for _, f := range tk.Fishes {
			if f.Y+fishH > p.PlantY {
				reachedPlant = true
			}
		}
		if len(tk.Foods) == 0 {
			break
		}
	}
	if !reachedPlant {
		t.Fatal("no fish entered the plant band; plants must not be a solid floor")
	}
	if len(tk.Foods) != 0 || tk.bites == 0 {
		t.Fatalf("food among the plants was not eaten: %v bites=%d", foods(tk), tk.bites)
	}
}

func TestUneatenFoodFadesAfterALifetimeNotInstantly(t *testing.T) {
	tk := New(420, 520, 8)
	tk.Foods = nil
	tk.Fishes = nil
	p := tk.Panel
	if !tk.FeedFromAbove(p.FX+p.FW/2, p.FY+p.FH/2) {
		t.Fatal("feed")
	}
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
	}
	if len(tk.Foods) != 1 {
		t.Fatal("food vanished almost immediately; TTL must not be a cover-up wipe")
	}
	for i := 0; i < 400; i++ {
		tk.Step(0.05)
		if len(tk.Foods) == 0 {
			if tk.Time < FoodLife()-0.6 {
				t.Fatalf("food faded at t=%v, want about %v", tk.Time, FoodLife())
			}
			return
		}
	}
	t.Fatal("uneaten food never faded")
}

func TestPlantsAreDecorativeNotSolid(t *testing.T) {
	tk := New(420, 520, 2)
	p := tk.Panel
	// Move a fish onto a plant glyph; clamp must allow it.
	x, y := p.ClampInside(tk.Plants[0].X-10, p.PlantY-4, fishW, fishH)
	tk.Fishes[0].X, tk.Fishes[0].Y = x, y
	if tk.Fishes[0].Y+fishH <= p.PlantY {
		// after clamp, if still forced above plants, floor is wrong
		if y+fishH <= p.PlantY {
			t.Fatalf("clamp rejected the plant band: y=%v planty=%v floor=%v", y, p.PlantY, p.FloorY())
		}
	}
}

func positions(tk *Tank) [][2]float64 {
	out := make([][2]float64, len(tk.Fishes))
	for i, f := range tk.Fishes {
		out[i] = [2]float64{f.X, f.Y}
	}
	return out
}

func foods(tk *Tank) [][2]float64 {
	out := make([][2]float64, len(tk.Foods))
	for i, f := range tk.Foods {
		out[i] = [2]float64{f.X, f.Y}
	}
	return out
}
