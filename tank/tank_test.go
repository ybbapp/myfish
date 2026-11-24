package tank

import (
	"math"
	"testing"
)

func TestLayoutHasVerticalPlateStandAndFeedStrip(t *testing.T) {
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
	if p.BH > 42 {
		t.Fatalf("stand platform should be a thinner slab, bh=%v", p.BH)
	}
	if p.FY+p.FH > p.GY-p.Thickness*0.2 {
		t.Fatalf("feed strip should sit above the plate: fy+fh=%v gy=%v", p.FY+p.FH, p.GY)
	}
	if !p.ContainsFeed(p.FX+p.FW/2, p.FY+p.FH/2) {
		t.Fatal("feed-strip center should be in the feed zone")
	}
	cx, cy := p.IX+p.IW/2, p.IY+p.IH/2
	if !p.ContainsInterior(cx, cy) {
		t.Fatal("center of the plate should be interior")
	}
	if p.ContainsFeed(cx, cy) {
		t.Fatal("plate interior must not be a feed click")
	}
	if p.ContainsInterior(2, 2) {
		t.Fatal("window corner should be outside the plate")
	}
	if !p.ContainsStand(p.BX+p.BW/2, p.BY+p.BH/2) {
		t.Fatal("platform center should be on the stand")
	}
	if !p.ContainsFrame(p.FX+2, p.FY+2) || !p.ContainsFrame(p.GX+1, p.GY+1) || !p.ContainsFrame(p.Leg1X+2, p.BY+p.BH+2) {
		t.Fatal("frame should include the feed strip, plate and a leg")
	}
	x, y, w, h := p.FrameBounds()
	if y > p.FY+0.5 || y+h < p.BY+p.BH+p.LegH-0.5 || x+w < p.FX+p.FW-0.5 {
		t.Fatalf("frame bounds miss the chassis: %v,%v %vx%v", x, y, w, h)
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
	assertNoBoxOverlap(t, tk, -1)
	for i, a := range tk.Fishes {
		cx, cy := a.X+fishW/2, a.Y+fishH/2
		if !tk.Panel.ContainsInterior(cx, cy) {
			t.Fatalf("fish %d start outside the plate: (%v,%v)", i, cx, cy)
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
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = tk.Fishes[i].X+fishW/2, tk.Fishes[i].Y+fishH/2
	}
	tk.Step(2)
	dx := tk.Fishes[0].X - before.X
	max := wanderSpeed*maxStep + fishW + 0.01
	if dx < 0 || dx > max {
		t.Fatalf("capped step moved fish by %v, want 0..%v", dx, max)
	}
}

func TestFeedFromAboveOnly(t *testing.T) {
	tk := New(420, 520, 2)
	p := tk.Panel
	if tk.FeedFromAbove(2, 2) {
		t.Fatal("FeedFromAbove succeeded outside the strip")
	}
	if tk.FeedFromAbove(p.IX+p.IW/2, p.IY+p.IH/3) {
		t.Fatal("a click on the plate itself dropped food")
	}
	if tk.FeedFromAbove(p.BX+p.BW/2, p.BY+p.BH/2) {
		t.Fatal("a click on the stand dropped food")
	}
	clickX := p.FX + p.FW*0.7
	if !tk.FeedFromAbove(clickX, p.FY+p.FH/2) {
		t.Fatal("FeedFromAbove failed on the strip")
	}
	if len(tk.Foods) != 1 {
		t.Fatalf("got %d foods", len(tk.Foods))
	}
	if math.Abs(tk.Foods[0].X+4-clickX) > 10 {
		t.Fatalf("food x=%v, click x=%v", tk.Foods[0].X, clickX)
	}
	if tk.Foods[0].Y > p.IY+12 {
		t.Fatalf("food should start at the top of the plate, y=%v iy=%v", tk.Foods[0].Y, p.IY)
	}
	y0 := tk.Foods[0].Y
	tk.Step(0.05)
	if tk.Foods[0].Y <= y0 {
		t.Fatalf("food did not fall: %v -> %v", y0, tk.Foods[0].Y)
	}
	for i := 0; i < MaxFood-1; i++ {
		if !tk.FeedFromAbove(p.FX+10, p.FY+p.FH/2) {
			t.Fatalf("FeedFromAbove %d failed", i+2)
		}
	}
	if tk.FeedFromAbove(clickX, p.FY+p.FH/2) {
		t.Fatal("FeedFromAbove succeeded over MaxFood")
	}
}

func TestPlaceFoodStaysInsideThePlate(t *testing.T) {
	tk := New(420, 520, 2)
	if tk.PlaceFood(2, 2) {
		t.Fatal("PlaceFood succeeded outside the plate")
	}
	cx, cy := tk.Panel.IX+tk.Panel.IW/2, tk.Panel.IY+tk.Panel.IH/3
	if !tk.PlaceFood(cx, cy) {
		t.Fatal("PlaceFood failed inside the plate")
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
	if !tk.PlaceFood(tk.Fishes[0].X+24, tk.Fishes[0].Y+10) {
		t.Fatal("place")
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
	tk.PlaceFood(tk.Fishes[0].X+10, tk.Fishes[0].Y+8)
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
	tk.PlaceFood(tk.Fishes[0].X+fishW/2, tk.Fishes[0].Y+fishH/2)
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

func TestFishBoxesDoNotOverlapWhileChasing(t *testing.T) {
	tk := New(420, 520, 9)
	tk.Foods = nil
	p := tk.Panel
	tk.Fishes[0].X, tk.Fishes[0].Y = p.ClampInside(p.IX+8, p.IY+30, fishW, fishH)
	tk.Fishes[1].X, tk.Fishes[1].Y = p.ClampInside(p.IX+20, p.IY+70, fishW, fishH)
	tk.Fishes[2].X, tk.Fishes[2].Y = p.ClampInside(p.IX+p.IW-fishW-8, p.IY+50, fishW, fishH)
	if !tk.PlaceFood(p.IX+p.IW/2, p.IY+p.IH/2) {
		t.Fatal("place")
	}
	for n := 0; n < 80 && len(tk.Foods) > 0; n++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, n)
	}
	if len(tk.Foods) != 0 {
		t.Fatal("food was never eaten")
	}
}

func TestOnlyNearestFishHuntsAFood(t *testing.T) {
	tk := New(420, 520, 11)
	tk.Foods = nil
	p := tk.Panel
	tk.Fishes[0].X, tk.Fishes[0].Y = p.ClampInside(p.IX+16, p.IY+40, fishW, fishH)
	tk.Fishes[1].X, tk.Fishes[1].Y = p.ClampInside(p.IX+16, p.PlantY-fishH-20, fishW, fishH)
	tk.Fishes[2].X, tk.Fishes[2].Y = p.ClampInside(p.IX+p.IW-fishW-8, p.PlantY-fishH-20, fishW, fishH)
	for i := range tk.Fishes {
		tk.Fishes[i].TargetX = tk.Fishes[i].X + fishW/2
		tk.Fishes[i].TargetY = tk.Fishes[i].Y + fishH/2
		tk.Fishes[i].Retarget = 4
	}
	foodX := tk.Fishes[0].X + fishW/2 + 36
	foodY := tk.Fishes[0].Y + fishH/2
	if !tk.PlaceFood(foodX, foodY) {
		t.Fatal("place")
	}
	start1 := [2]float64{tk.Fishes[1].X, tk.Fishes[1].Y}
	start2 := [2]float64{tk.Fishes[2].X, tk.Fishes[2].Y}
	hunters := tk.assignHunters()
	if hunters[0] != 0 {
		t.Fatalf("nearest fish should hunt, assignment=%v", hunters)
	}
	if hunters[1] != -1 || hunters[2] != -1 {
		t.Fatalf("other fish should not hunt the same food, assignment=%v", hunters)
	}
	for n := 0; n < 10; n++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, n)
	}
	d1 := math.Hypot(tk.Fishes[1].X-start1[0], tk.Fishes[1].Y-start1[1])
	d2 := math.Hypot(tk.Fishes[2].X-start2[0], tk.Fishes[2].Y-start2[1])
	// Wander is allowed, but they must not close on the food the way a hunter would.
	toFood1 := math.Hypot(start1[0]+fishW/2-foodX, start1[1]+fishH/2-foodY)
	toFood2 := math.Hypot(start2[0]+fishW/2-foodX, start2[1]+fishH/2-foodY)
	now1 := math.Hypot(tk.Fishes[1].X+fishW/2-foodX, tk.Fishes[1].Y+fishH/2-foodY)
	now2 := math.Hypot(tk.Fishes[2].X+fishW/2-foodX, tk.Fishes[2].Y+fishH/2-foodY)
	if now1 < toFood1-30 && d1 > 40 {
		t.Fatalf("fish 1 closed on the food like a hunter: start d=%v now=%v", toFood1, now1)
	}
	if now2 < toFood2-30 && d2 > 40 {
		t.Fatalf("fish 2 closed on the food like a hunter: start d=%v now=%v", toFood2, now2)
	}
}

func TestHoverDoesNotAddFoodAndIsGentle(t *testing.T) {
	tk := New(420, 520, 12)
	tk.Foods = nil
	p := tk.Panel
	hx, hy := p.IX+p.IW*0.7, p.IY+p.IH*0.4
	before := append([]Fish(nil), tk.Fishes...)
	tk.SetHover(hx, hy)
	if len(tk.Foods) != 0 {
		t.Fatal("SetHover dropped food")
	}
	for n := 0; n < 30; n++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, n)
		if len(tk.Foods) != 0 {
			t.Fatalf("hover added food at step %d", n)
		}
		for i, f := range tk.Fishes {
			dx := math.Abs(f.X - before[i].X)
			dy := math.Abs(f.Y - before[i].Y)
			max := hoverSpeed*0.05*(float64(n)+1) + fishW
			if n < 4 && (dx > hoverSpeed*0.05*4+8 || dy > hoverSpeed*0.05*4+8) {
				t.Fatalf("fish %d jumped on hover: d=(%v,%v)", i, f.X-before[i].X, f.Y-before[i].Y)
			}
			_ = max
		}
	}
	// The nearest fish should have eased somewhat toward the pointer, not teleported onto it.
	i := tk.nearestHover()
	cx, cy := tk.Fishes[i].X+fishW/2, tk.Fishes[i].Y+fishH/2
	if math.Hypot(cx-hx, cy-hy) < 8 {
		t.Fatalf("nearest fish sat on the cursor at (%v,%v)", cx, cy)
	}
}

func TestClearHoverResumesWander(t *testing.T) {
	tk := New(420, 520, 13)
	p := tk.Panel
	tk.SetHover(p.IX+p.IW/2, p.IY+p.IH/3)
	tk.Step(0.05)
	if !tk.Hovering {
		t.Fatal("expected hover")
	}
	hoverTarget := [2]float64{tk.Fishes[tk.nearestHover()].TargetX, tk.Fishes[tk.nearestHover()].TargetY}
	tk.ClearHover()
	if tk.Hovering {
		t.Fatal("ClearHover left hovering on")
	}
	// After leave, fish should have fresh wander targets, not stay locked on the pointer.
	locked := 0
	for _, f := range tk.Fishes {
		if math.Hypot(f.TargetX-hoverTarget[0], f.TargetY-hoverTarget[1]) < 4 && f.Retarget <= 0.01 {
			locked++
		}
	}
	if locked == len(tk.Fishes) {
		t.Fatal("all fish stayed locked on the hover target after leave")
	}
	for i := range tk.Fishes {
		if tk.Fishes[i].Retarget <= 0 {
			t.Fatalf("fish %d has no wander retarget after leave", i)
		}
	}
}

func assertNoBoxOverlap(t *testing.T, tk *Tank, step int) {
	t.Helper()
	for i := 0; i < len(tk.Fishes); i++ {
		for j := i + 1; j < len(tk.Fishes); j++ {
			a, b := tk.Fishes[i], tk.Fishes[j]
			// A newborn is spawned inside the adult and slides out; that
			// overlap is the birth animation, not a swim collision.
			if a.Emerging || b.Emerging || a.State == LifeDying || b.State == LifeDying {
				continue
			}
			if fishesOverlap(a, b, 0.5) {
				t.Fatalf("step %d: fish %d (%.1f,%.1f %gx%g) overlaps fish %d (%.1f,%.1f)",
					step, i, a.X, a.Y, a.W(), a.H(), j, b.X, b.Y)
			}
		}
	}
}
