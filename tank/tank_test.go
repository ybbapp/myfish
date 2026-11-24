package tank

import (
	"math"
	"testing"
)

func TestLayoutHasVerticalPlateAndStand(t *testing.T) {
	p := Layout(420, 520)
	if p.GW <= 0 || p.GH <= 0 || p.BW <= 0 || p.BH <= 0 || p.LegH <= 0 {
		t.Fatalf("empty chassis: %+v", p)
	}
	if p.GH < p.GW*0.7 {
		t.Fatalf("plate should be a tall rectangle, gh=%v gw=%v", p.GH, p.GW)
	}
	if p.BY < p.GY+p.GH-0.5 {
		t.Fatalf("stand should sit under the plate")
	}
	cx, cy := p.IX+p.IW/2, p.IY+p.IH/2
	if !p.ContainsInterior(cx, cy) {
		t.Fatal("center of the plate should be interior")
	}
	if p.ContainsInterior(2, 2) {
		t.Fatal("window corner should be outside the plate")
	}
	if !p.ContainsStand(p.BX+p.BW/2, p.BY+p.BH/2) {
		t.Fatal("platform center should be on the stand")
	}
	if !p.ContainsFrame(p.GX+1, p.GY+1) || !p.ContainsFrame(p.Leg1X+2, p.BY+p.BH+2) {
		t.Fatal("frame should include plate and a leg")
	}
}

func TestNewScattersSeveralFishOnThePlate(t *testing.T) {
	tk := New(420, 520, 7)
	if len(tk.Fishes) != FishCount {
		t.Fatalf("got %d fish, want %d", len(tk.Fishes), FishCount)
	}
	if len(tk.Plants) != 2 {
		t.Fatalf("got %d plants, want 2", len(tk.Plants))
	}
	for i, a := range tk.Fishes {
		cx, cy := a.X+fishW/2, a.Y+fishH/2
		if !tk.Panel.ContainsInterior(cx, cy) {
			t.Fatalf("fish %d start outside the plate: (%v,%v)", i, cx, cy)
		}
		for j := i + 1; j < len(tk.Fishes); j++ {
			b := tk.Fishes[j]
			d := math.Hypot((a.X+fishW/2)-(b.X+fishW/2), (a.Y+fishH/2)-(b.Y+fishH/2))
			if d < minFishGap {
				t.Fatalf("fish %d and %d overlap: d=%v", i, j, d)
			}
		}
	}
}

func TestStepIgnoresTinyAndCapsLongDelta(t *testing.T) {
	tk := New(420, 520, 1)
	x, y := tk.Fishes[0].X, tk.Fishes[0].Y
	tk.Step(0)
	if tk.Fishes[0].X != x || tk.Fishes[0].Y != y {
		t.Fatalf("Step(0) moved the fish to (%v,%v)", tk.Fishes[0].X, tk.Fishes[0].Y)
	}
	before := tk.Fishes[0]
	tk.Fishes[0].TargetX = before.X + fishW/2 + 400
	tk.Fishes[0].TargetY = before.Y + fishH/2
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = tk.Panel.ClampInside(tk.Panel.IX, tk.Panel.IY, fishW, fishH)
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = tk.Fishes[i].X, tk.Fishes[i].Y
	}
	tk.Step(2)
	dx := tk.Fishes[0].X - before.X
	max := chaseSpeed*maxStep + 180*maxStep + 0.01
	if dx < 0 || dx > max {
		t.Fatalf("capped step moved fish by %v, want 0..%v", dx, max)
	}
}

func TestDropFoodOnlyInsideThePlate(t *testing.T) {
	tk := New(420, 520, 2)
	if tk.DropFood(2, 2) {
		t.Fatal("DropFood succeeded outside the plate")
	}
	cx, cy := tk.Panel.IX+tk.Panel.IW/2, tk.Panel.IY+tk.Panel.IH/3
	for i := 0; i < MaxFood; i++ {
		if !tk.DropFood(cx, cy) {
			t.Fatalf("DropFood %d failed inside the plate", i+1)
		}
	}
	if tk.DropFood(cx, cy) {
		t.Fatal("DropFood succeeded over MaxFood")
	}
	y0 := tk.Foods[0].Y
	tk.Step(0.05)
	if tk.Foods[0].Y <= y0 {
		t.Fatalf("food did not fall: %v -> %v", y0, tk.Foods[0].Y)
	}
}

func TestFishChasesAndEatsFood(t *testing.T) {
	tk := New(420, 520, 3)
	tk.Foods = nil
	cx, cy := tk.Panel.IX+40, tk.Panel.IY+40
	tk.Fishes[0].X, tk.Fishes[0].Y = tk.Panel.ClampInside(cx, cy, fishW, fishH)
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = tk.Panel.ClampInside(tk.Panel.IX+tk.Panel.IW-fishW-4, tk.Panel.IY+4, fishW, fishH)
		tk.Fishes[i].TargetX = tk.Fishes[i].X + fishW/2
		tk.Fishes[i].TargetY = tk.Fishes[i].Y + fishH/2
	}
	if !tk.DropFood(tk.Fishes[0].X+24, tk.Fishes[0].Y+10) {
		t.Fatal("drop")
	}
	start := tk.Fishes[0].X
	for i := 0; i < 80 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if tk.Fishes[0].X <= start && len(tk.Foods) > 0 {
		t.Fatalf("fish did not move toward food")
	}
	if len(tk.Foods) != 0 {
		t.Fatalf("food was not eaten")
	}
	if len(tk.Chomps) != 1 || tk.Chomps[0].Text != "chomp" {
		t.Fatalf("first bite %#v", tk.Chomps)
	}
	if tk.Chomps[0].Life < 1 {
		t.Fatalf("chomp life %v", tk.Chomps[0].Life)
	}
	tk.DropFood(tk.Fishes[0].X+10, tk.Fishes[0].Y+8)
	for i := 0; i < 80 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	found := false
	for _, c := range tk.Chomps {
		if c.Text == "crunch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("second bite should be crunch, got %#v", tk.Chomps)
	}
}

func TestChompLifeIsOverOneSecond(t *testing.T) {
	if ChompLife() < 1 {
		t.Fatalf("ChompLife %v, want >= 1", ChompLife())
	}
	tk := New(420, 520, 8)
	tk.Foods = nil
	tk.Fishes = tk.Fishes[:1]
	tk.Fishes[0].X, tk.Fishes[0].Y = tk.Panel.ClampInside(tk.Panel.IX+30, tk.Panel.IY+30, fishW, fishH)
	tk.DropFood(tk.Fishes[0].X+fishW/2, tk.Fishes[0].Y+fishH/2)
	for i := 0; i < 40 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if len(tk.Chomps) != 1 {
		t.Fatalf("want a chomp, got %d", len(tk.Chomps))
	}
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
	}
	if len(tk.Chomps) == 0 {
		t.Fatal("chomp disappeared before 1s")
	}
}

func TestBubblesStayOnThePlate(t *testing.T) {
	tk := New(420, 520, 4)
	tk.Bubbles = []Bubble{{ID: 1, X: tk.Panel.IX + 40, Y: tk.Panel.IY + 80, Speed: 40, Glyph: "o"}}
	y0 := tk.Bubbles[0].Y
	tk.Step(0.05)
	if len(tk.Bubbles) == 0 || tk.Bubbles[0].Y >= y0 {
		t.Fatalf("bubble did not rise")
	}
}

func TestFishStayInsideThePlate(t *testing.T) {
	tk := New(420, 520, 5)
	for i := range tk.Fishes {
		tk.Fishes[i].X, tk.Fishes[i].Y = -40, 999
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = 1000, -20
	}
	for i := 0; i < 80; i++ {
		tk.Step(0.05)
		for j, f := range tk.Fishes {
			cx, cy := f.X+fishW/2, f.Y+fishH/2
			if !tk.Panel.ContainsInterior(cx, cy) && (f.X < tk.Panel.IX-1 || f.Y < tk.Panel.IY-1) {
				t.Fatalf("fish %d left the plate at step %d: (%v,%v)", j, i, f.X, f.Y)
			}
			// The sprite box is clamped to the interior.
			if f.X < tk.Panel.IX-0.5 || f.Y < tk.Panel.IY-0.5 {
				t.Fatalf("fish %d origin outside interior: (%v,%v)", j, f.X, f.Y)
			}
		}
	}
}

func TestSpriteFollowsFacing(t *testing.T) {
	tk := New(420, 520, 6)
	tk.Fishes[0].Facing = 1
	if tk.Sprite(0) != SpriteRight {
		t.Fatalf("right sprite %q", tk.Sprite(0))
	}
	tk.Fishes[0].Facing = -1
	if tk.Sprite(0) != SpriteLeft {
		t.Fatalf("left sprite %q", tk.Sprite(0))
	}
}

func TestFishDoNotStackOnTheSameFood(t *testing.T) {
	tk := New(420, 520, 9)
	tk.Foods = nil
	p := tk.Panel
	tk.Fishes[0].X, tk.Fishes[0].Y = p.ClampInside(p.IX+10, p.IY+20, fishW, fishH)
	tk.Fishes[1].X, tk.Fishes[1].Y = p.ClampInside(p.IX+10, p.IY+90, fishW, fishH)
	tk.Fishes[2].X, tk.Fishes[2].Y = p.ClampInside(p.IX+p.IW-fishW-8, p.IY+50, fishW, fishH)
	if !tk.DropFood(p.IX+p.IW/2, p.IY+p.IH/2) {
		t.Fatal("drop")
	}
	minAllowed := 60.0
	for n := 0; n < 80 && len(tk.Foods) > 0; n++ {
		tk.Step(0.05)
		for i := 0; i < len(tk.Fishes); i++ {
			for j := i + 1; j < len(tk.Fishes); j++ {
				a, b := tk.Fishes[i], tk.Fishes[j]
				d := math.Hypot((a.X+fishW/2)-(b.X+fishW/2), (a.Y+fishH/2)-(b.Y+fishH/2))
				if d < minAllowed {
					t.Fatalf("step %d: fish %d and %d stacked (d=%v)", n, i, j, d)
				}
			}
		}
	}
	if len(tk.Foods) != 0 {
		t.Fatal("food was never eaten")
	}
}
