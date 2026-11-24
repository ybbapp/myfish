package tank

import (
	"math"
	"testing"
)

func TestNewScattersSeveralFish(t *testing.T) {
	tk := New(560, 380, 7)
	if len(tk.Fishes) != FishCount {
		t.Fatalf("got %d fish, want %d", len(tk.Fishes), FishCount)
	}
	for i, a := range tk.Fishes {
		for j := i + 1; j < len(tk.Fishes); j++ {
			b := tk.Fishes[j]
			dx := (a.X + fishW/2) - (b.X + fishW/2)
			dy := (a.Y + fishH/2) - (b.Y + fishH/2)
			if math.Hypot(dx, dy) < minFishGap {
				t.Fatalf("fish %d and %d overlap: (%v,%v) and (%v,%v)", i, j, a.X, a.Y, b.X, b.Y)
			}
		}
	}
}

func TestStepIgnoresTinyAndCapsLongDelta(t *testing.T) {
	tk := New(400, 300, 1)
	x, y := tk.Fishes[0].X, tk.Fishes[0].Y
	tk.Step(0)
	if tk.Fishes[0].X != x || tk.Fishes[0].Y != y {
		t.Fatalf("Step(0) moved the fish to (%v,%v)", tk.Fishes[0].X, tk.Fishes[0].Y)
	}
	before := tk.Fishes[0]
	tk.Fishes[0].TargetX = before.X + fishW/2 + 400
	tk.Fishes[0].TargetY = before.Y + fishH/2
	// Park the others so they do not shove this one.
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = 10, 10
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = 10, 10
	}
	tk.Step(2)
	dx := tk.Fishes[0].X - before.X
	max := chaseSpeed*maxStep + 180*maxStep + 0.01 // include a little separation nudge
	if dx < 0 || dx > max {
		t.Fatalf("capped step moved fish by %v, want 0..%v", dx, max)
	}
}

func TestDropFoodLimitAndFall(t *testing.T) {
	tk := New(400, 300, 2)
	for i := 0; i < MaxFood; i++ {
		if !tk.DropFood(40, 20) {
			t.Fatalf("DropFood %d failed before the limit", i+1)
		}
	}
	if tk.DropFood(40, 20) {
		t.Fatal("DropFood succeeded over MaxFood")
	}
	if len(tk.Foods) != MaxFood {
		t.Fatalf("got %d foods, want %d", len(tk.Foods), MaxFood)
	}
	y0 := tk.Foods[0].Y
	tk.Step(0.05)
	if tk.Foods[0].Y <= y0 {
		t.Fatalf("food did not fall: %v -> %v", y0, tk.Foods[0].Y)
	}
	tk.Foods = []Food{{ID: 1, X: 100, Y: tk.H - 1}}
	tk.Step(0.05)
	if len(tk.Foods) != 0 {
		t.Fatalf("food still there at bottom: %+v", tk.Foods)
	}
}

func TestFishChasesAndEatsFood(t *testing.T) {
	tk := New(400, 300, 3)
	tk.Foods = nil
	tk.Fishes[0].X, tk.Fishes[0].Y = 80, 80
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = tk.W-margin-fishW, margin
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = tk.Fishes[i].X+fishW/2, tk.Fishes[i].Y+fishH/2
	}
	if !tk.DropFood(90, 88) {
		t.Fatal("drop")
	}
	start := tk.Fishes[0].X
	for i := 0; i < 80 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if tk.Fishes[0].X <= start && len(tk.Foods) > 0 {
		t.Fatalf("fish did not move toward food: start %v now %v foods %d", start, tk.Fishes[0].X, len(tk.Foods))
	}
	if len(tk.Foods) != 0 {
		t.Fatalf("food was not eaten, fish at (%v,%v)", tk.Fishes[0].X, tk.Fishes[0].Y)
	}
	if len(tk.Chomps) != 1 {
		t.Fatalf("want one chomp label, got %d", len(tk.Chomps))
	}
	if tk.Chomps[0].Text != "chomp" {
		t.Fatalf("first bite text %q, want chomp", tk.Chomps[0].Text)
	}
	if tk.Chomps[0].Life < 1 {
		t.Fatalf("chomp life %v, want at least 1s", tk.Chomps[0].Life)
	}
	tk.DropFood(tk.Fishes[0].X+10, tk.Fishes[0].Y+8)
	for i := 0; i < 80 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	foundGulp := false
	for _, c := range tk.Chomps {
		if c.Text == "gulp" {
			foundGulp = true
		}
	}
	if !foundGulp {
		t.Fatalf("second bite texts: %#v", tk.Chomps)
	}
}

func TestBubblesRiseAndChompsExpire(t *testing.T) {
	tk := New(400, 300, 4)
	tk.Bubbles = []Bubble{{ID: 1, X: 50, Y: 100, Speed: 40, Glyph: "o"}}
	y0 := tk.Bubbles[0].Y
	tk.Step(0.05)
	if tk.Bubbles[0].Y >= y0 {
		t.Fatalf("bubble did not rise: %v -> %v", y0, tk.Bubbles[0].Y)
	}
	tk.Chomps = []Chomp{{ID: 2, X: 10, Y: 10, Text: "chomp", Life: 0.1}}
	tk.Step(0.05)
	tk.Step(0.05)
	if len(tk.Chomps) != 0 {
		t.Fatalf("chomp still alive: %+v", tk.Chomps)
	}
}

func TestChompLifeIsOverOneSecond(t *testing.T) {
	if ChompLife() < 1 {
		t.Fatalf("ChompLife %v, want >= 1", ChompLife())
	}
	tk := New(400, 300, 8)
	tk.Foods = nil
	tk.Fishes = tk.Fishes[:1]
	tk.Fishes[0].X, tk.Fishes[0].Y = 80, 80
	tk.DropFood(tk.Fishes[0].X+fishW/2, tk.Fishes[0].Y+fishH/2)
	for i := 0; i < 40 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if len(tk.Chomps) != 1 {
		t.Fatalf("want a chomp, got %d (foods %d)", len(tk.Chomps), len(tk.Foods))
	}
	// Still visible after one second.
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
	}
	if len(tk.Chomps) == 0 {
		t.Fatal("chomp disappeared before 1s")
	}
}

func TestFishStaysInBounds(t *testing.T) {
	tk := New(200, 160, 5)
	for i := range tk.Fishes {
		tk.Fishes[i].X, tk.Fishes[i].Y = -40, 999
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = 1000, -20
	}
	for i := 0; i < 80; i++ {
		tk.Step(0.05)
		for _, f := range tk.Fishes {
			if f.X < 0 || f.Y < 0 || f.X > tk.W || f.Y > tk.H {
				t.Fatalf("fish out of bounds at step %d: (%v,%v) in %vx%v", i, f.X, f.Y, tk.W, tk.H)
			}
		}
	}
}

func TestSpriteFollowsFacing(t *testing.T) {
	tk := New(200, 160, 6)
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
	tk := New(560, 380, 9)
	tk.Foods = nil
	tk.Fishes[0].X, tk.Fishes[0].Y = 80, 80
	tk.Fishes[1].X, tk.Fishes[1].Y = 80, 200
	tk.Fishes[2].X, tk.Fishes[2].Y = 400, 140
	if !tk.DropFood(280, 160) {
		t.Fatal("drop")
	}
	minAllowed := 70.0
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
		t.Fatalf("food was never eaten")
	}
}
