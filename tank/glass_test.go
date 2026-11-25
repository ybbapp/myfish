package tank

import (
	"math"
	"testing"
)

func glassDistance(f Fish, x, y float64) float64 {
	cx, cy := f.Center()
	return math.Hypot(cx-x, cy-y)
}

func TestGlassTemperamentsMixedStableAndSeeded(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		a, b := New(420, 520, seed), New(420, 520, seed)
		seen := map[GlassReaction]bool{}
		for i, f := range a.Fishes {
			seen[f.GlassReaction] = true
			if f.GlassReaction != b.Fishes[i].GlassReaction {
				t.Fatal("temperament not reproducible for same seed")
			}
		}
		if !seen[GlassCurious] || !seen[GlassShy] {
			t.Fatalf("seed %d must expose both reactions, got %v", seed, seen)
		}
		before := append([]Fish(nil), a.Fishes...)
		for tick := 0; tick < 120; tick++ {
			a.SetHover(200+float64(tick%10), 220)
			a.Step(1.0 / 60)
			for i, f := range a.Fishes {
				if f.GlassReaction != before[i].GlassReaction {
					t.Fatal("temperament rerolled while cursor moved")
				}
			}
		}
	}
}

func TestGlassApproachVersusRetreatAndFacing(t *testing.T) {
	for _, reaction := range []GlassReaction{GlassCurious, GlassShy} {
		tk := New(420, 520, 7)
		tk.Fishes = tk.Fishes[:1]
		tk.Foods, tk.Bubbles = nil, nil
		f := &tk.Fishes[0]
		f.X, f.Y, f.VX, f.VY = 120, 180, 0, 0
		f.GlassReaction = reaction
		cx, cy := f.Center()
		x, y := cx+85, cy
		before := glassDistance(*f, x, y)
		tk.SetHover(x, y)
		for i := 0; i < 60; i++ {
			tk.stepFish(0, 1.0/60, -1)
		}
		after := glassDistance(*f, x, y)
		if reaction == GlassCurious && (after >= before-8 || f.Facing != 1) {
			t.Fatalf("curious fish should approach and watch: before %v after %v facing %v", before, after, f.Facing)
		}
		if reaction == GlassShy && (after <= before+15 || f.Facing != -1) {
			t.Fatalf("shy fish should flee and face away: before %v after %v facing %v", before, after, f.Facing)
		}
		if len(tk.Foods) != 0 {
			t.Fatal("glass interaction generated food")
		}
	}
}

func TestGlassStableContactDeadBandAndBounds(t *testing.T) {
	for _, reaction := range []GlassReaction{GlassCurious, GlassShy} {
		tk := New(420, 520, 7)
		tk.Fishes = tk.Fishes[:1]
		f := &tk.Fishes[0]
		f.GlassReaction = reaction
		for _, at := range [][2]float64{{34, 68}, {275, 200}, {130, 300}} {
			f.X, f.Y = at[0], at[1]
			cx, cy := f.Center()
			tk.SetHover(cx, cy) // exact-center retreat must not produce NaN
			for i := 0; i < 600; i++ {
				tk.stepFish(0, 1.0/60, -1)
				if math.IsNaN(f.X) || math.IsNaN(f.Y) || f.X < tk.Panel.IX-0.01 || f.X+f.W() > tk.Panel.IX+tk.Panel.IW+0.01 || f.Y < tk.Panel.IY-0.01 || f.Y+f.H() > tk.Panel.FloorY()+0.01 {
					t.Fatalf("invalid/out-of-bounds fish %+v", *f)
				}
			}
			x, y := f.X, f.Y
			for i := 0; i < 60; i++ {
				tk.stepFish(0, 1.0/60, -1)
			}
			if math.Hypot(f.X-x, f.Y-y) > 0.1 {
				t.Fatalf("static glass contact jitters after settling: %v", math.Hypot(f.X-x, f.Y-y))
			}
		}
	}
}

func TestGlassExitSmoothAndLifecyclePriority(t *testing.T) {
	tk := New(420, 520, 7)
	tk.Fishes = tk.Fishes[:1]
	f := &tk.Fishes[0]
	f.GlassReaction = GlassShy
	cx, cy := f.Center()
	tk.SetHover(cx+60, cy)
	for i := 0; i < 20; i++ {
		tk.stepFish(0, 1.0/60, -1)
	}
	vx, vy := f.VX, f.VY
	tk.ClearHover()
	if f.VX != vx || f.VY != vy {
		t.Fatal("leaving glass snapped velocity")
	}
	tk.stepFish(0, 1.0/60, -1)
	if math.Hypot(f.VX-vx, f.VY-vy) > 8 {
		t.Fatal("leaving glass abruptly reversed velocity")
	}
	for _, state := range []LifeState{LifeBirthPause, LifeBirthSwell, LifeDying} {
		f.State = state
		x, y := f.X, f.Y
		tk.SetHover(300, 220)
		tk.stepFish(0, 0.05, -1)
		if f.X != x || f.Y != y {
			t.Fatalf("glass overrides lifecycle state %v", state)
		}
	}
	f.State, f.Emerging = LifeSwim, true
	x, y := f.X, f.Y
	tk.stepFish(0, 0.05, -1)
	if f.X != x || f.Y != y {
		t.Fatal("glass overrides emerging newborn")
	}
}

func TestNewbornGlassReactionInitialized(t *testing.T) {
	tk := New(420, 520, 7)
	// Exercise the real production birth path used by life tests.
	tk.Fishes[0].Size = MaxSize
	if !tk.spawnNewbornFrom(0) {
		t.Fatal("newborn not spawned")
	}
	baby := tk.Fishes[len(tk.Fishes)-1]
	if baby.GlassReaction != initialGlassReaction(int64(tk.Fishes[0].ID), baby.ID) {
		t.Fatal("newborn missing stable temperament initialization")
	}
}

func TestGlassFoodPriorityAndWeakness(t *testing.T) {
	tk := New(420, 520, 7)
	tk.Fishes = tk.Fishes[:1]
	f := &tk.Fishes[0]
	f.GlassReaction = GlassShy
	f.X, f.Y = 120, 150
	cx, cy := f.Center()
	tk.SetHover(cx+50, cy)
	tk.Foods = []Food{{ID: 999, X: cx + 70, Y: cy - 4, Life: FoodLife()}}
	tk.stepFish(0, 0.01, 0)
	if f.VX <= 0 || f.Facing != 1 {
		t.Fatal("glass retreat must not override food chasing")
	}
	f.Size = MaxSize
	f.MatureAge = WeakAfter + 1
	weak := tk.glassTarget(f)
	f.MatureAge = 0
	strong := tk.glassTarget(f)
	if weak >= strong {
		t.Fatalf("weak fish should retreat more slowly: %v >= %v", weak, strong)
	}
}

func TestGlassFrameRateIndependentEasing(t *testing.T) {
	run := func(dt float64) Fish {
		tk := New(420, 520, 7)
		tk.Fishes = tk.Fishes[:1]
		f := &tk.Fishes[0]
		f.X, f.Y, f.VX, f.VY = 100, 160, 0, 0
		f.GlassReaction = GlassShy
		cx, cy := f.Center()
		tk.SetHover(cx+60, cy)
		for i := 0; i < int(math.Round(0.5/dt)); i++ {
			tk.stepFish(0, dt, -1)
		}
		return *f
	}
	a, b := run(1.0/30), run(1.0/120)
	if math.Hypot(a.X-b.X, a.Y-b.Y) > 1 || math.Hypot(a.VX-b.VX, a.VY-b.VY) > 0.1 {
		t.Fatalf("frame-rate-dependent retreat: 30fps %+v 120fps %+v", a, b)
	}
}
