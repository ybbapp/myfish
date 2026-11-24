package tank

import "testing"

func TestStepIgnoresTinyAndCapsLongDelta(t *testing.T) {
	tk := New(400, 300, 1)
	x, y := tk.Fish.X, tk.Fish.Y
	tk.Step(0)
	if tk.Fish.X != x || tk.Fish.Y != y {
		t.Fatalf("Step(0) moved the fish to (%v,%v)", tk.Fish.X, tk.Fish.Y)
	}
	// A two-second stall must not jump farther than a 50ms cap allows.
	before := tk.Fish
	tk.Fish.TargetX = before.X + fishW/2 + 400
	tk.Fish.TargetY = before.Y + fishH/2
	tk.Step(2)
	dx := tk.Fish.X - before.X
	max := chaseSpeed*maxStep + 0.01
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
	// Food that reaches the bottom is removed.
	tk.Foods = []Food{{ID: 1, X: 100, Y: tk.H - 1}}
	tk.Step(0.05)
	if len(tk.Foods) != 0 {
		t.Fatalf("food still there at bottom: %+v", tk.Foods)
	}
}

func TestFishChasesAndEatsFood(t *testing.T) {
	tk := New(400, 300, 3)
	tk.Foods = nil
	tk.Fish.X, tk.Fish.Y = 80, 80
	if !tk.DropFood(90, 88) {
		t.Fatal("drop")
	}
	start := tk.Fish.X
	for i := 0; i < 40 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if tk.Fish.X <= start && len(tk.Foods) > 0 {
		t.Fatalf("fish did not move toward food: start %v now %v foods %d", start, tk.Fish.X, len(tk.Foods))
	}
	if len(tk.Foods) != 0 {
		t.Fatalf("food was not eaten, fish at (%v,%v) food at (%v,%v)", tk.Fish.X, tk.Fish.Y, tk.Foods[0].X, tk.Foods[0].Y)
	}
	if len(tk.Chomps) != 1 {
		t.Fatalf("want one chomp label, got %d", len(tk.Chomps))
	}
	if tk.Chomps[0].Text != "chomp" {
		t.Fatalf("first bite text %q, want chomp", tk.Chomps[0].Text)
	}
	// Second bite is gulp.
	tk.DropFood(tk.Fish.X+10, tk.Fish.Y+8)
	for i := 0; i < 40 && len(tk.Foods) > 0; i++ {
		tk.Step(0.05)
	}
	if len(tk.Chomps) < 1 {
		t.Fatal("expected a gulp label")
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

func TestFishStaysInBounds(t *testing.T) {
	tk := New(200, 160, 5)
	tk.Fish.X, tk.Fish.Y = -40, 999
	tk.Fish.TargetX, tk.Fish.TargetY = 1000, -20
	for i := 0; i < 80; i++ {
		tk.Step(0.05)
		if tk.Fish.X < 0 || tk.Fish.Y < 0 || tk.Fish.X > tk.W || tk.Fish.Y > tk.H {
			t.Fatalf("fish out of bounds at step %d: (%v,%v) in %vx%v", i, tk.Fish.X, tk.Fish.Y, tk.W, tk.H)
		}
	}
}

func TestSpriteFollowsFacing(t *testing.T) {
	tk := New(200, 160, 6)
	tk.Fish.Facing = 1
	if tk.Sprite() != SpriteRight {
		t.Fatalf("right sprite %q", tk.Sprite())
	}
	tk.Fish.Facing = -1
	if tk.Sprite() != SpriteLeft {
		t.Fatalf("left sprite %q", tk.Sprite())
	}
}
