package tank

import (
	"math"
	"testing"
)

func adultAtMax(tk *Tank, i int) {
	tk.setFishSize(i, MaxSize)
	tk.Fishes[i].MatureAge = MatureAfter
	tk.Fishes[i].State = LifeSwim
	tk.Fishes[i].Emerging = false
	tk.Fishes[i].Swell = 1
}

func stepSeconds(tk *Tank, seconds float64) {
	const dt = 1.0 / 30.0
	left := seconds
	for left > 0 {
		step := dt
		if step > left {
			step = left
		}
		tk.Step(step)
		left -= step
	}
}

func TestEatingGrowsUntilMaxAndThenStops(t *testing.T) {
	tk := New(420, 520, 11)
	tk.Foods = nil
	f := &tk.Fishes[0]
	if !f.CanEat() || f.AtMax() || math.Abs(f.Size-StartSize) > 1e-9 {
		t.Fatalf("starter fish should eat and not be at max, size=%v can=%v", f.Size, f.CanEat())
	}
	start := f.Size
	p := tk.Panel
	// Park the other fish far away so only fish 0 eats.
	for i := 1; i < len(tk.Fishes); i++ {
		tk.Fishes[i].X, tk.Fishes[i].Y = p.ClampInside(p.IX+p.IW-fishW-4, p.IY+4, tk.Fishes[i].W(), tk.Fishes[i].H())
		tk.Fishes[i].TargetX, tk.Fishes[i].TargetY = tk.Fishes[i].Center()
		tk.Fishes[i].Retarget = 30
	}
	eaten := 0
	for n := 0; n < 20 && !tk.Fishes[0].AtMax(); n++ {
		mx, my := tk.Fishes[0].Mouth()
		if !tk.PlaceFood(mx, my) {
			t.Fatal("PlaceFood failed")
		}
		before := len(tk.Foods)
		for k := 0; k < 40 && len(tk.Foods) == before; k++ {
			tk.Step(maxStep)
		}
		if len(tk.Foods) == before {
			t.Fatalf("fish 0 did not eat pellet %d, size=%v pos=%.1f,%.1f mouth=%.1f,%.1f foods=%d",
				n, tk.Fishes[0].Size, tk.Fishes[0].X, tk.Fishes[0].Y, mx, my, len(tk.Foods))
		}
		eaten++
	}
	if eaten < 4 {
		t.Fatalf("expected several bites to reach max, got %d size=%v", eaten, tk.Fishes[0].Size)
	}
	if !tk.Fishes[0].AtMax() || tk.Fishes[0].Size < MaxSize-1e-9 {
		t.Fatalf("size after growth %v, want max %v", tk.Fishes[0].Size, MaxSize)
	}
	if tk.Fishes[0].Size < start {
		t.Fatal("size shrank")
	}
	if tk.Fishes[0].CanEat() {
		t.Fatal("max-size fish must stop eating")
	}

	cx, cy := tk.Fishes[0].Center()
	if !tk.PlaceFood(cx, cy) {
		t.Fatal("PlaceFood failed at max")
	}
	size := tk.Fishes[0].Size
	foods := len(tk.Foods)
	for k := 0; k < 20; k++ {
		tk.Step(maxStep)
	}
	if len(tk.Foods) != foods {
		t.Fatalf("max-size fish ate food, foods %d -> %d", foods, len(tk.Foods))
	}
	if tk.Fishes[0].Size != size {
		t.Fatalf("size changed after cap: %v -> %v", size, tk.Fishes[0].Size)
	}
}

func TestNoBirthBelowMaxSizeOrAtCap(t *testing.T) {
	tk := New(420, 520, 13)
	if tk.StartBirth(0) {
		t.Fatal("birth must not start below max size")
	}
	adultAtMax(tk, 0)
	// Fill to the hard cap with inert extras.
	for len(tk.Fishes) < MaxFish {
		tk.Fishes = append(tk.Fishes, Fish{
			ID: 100 + len(tk.Fishes), Size: StartSize, Swell: 1, SpeedVar: 1,
			X: tk.Panel.IX + 8, Y: tk.Panel.IY + 8,
		})
	}
	if tk.StartBirth(0) {
		t.Fatal("birth must not start at the fish cap")
	}
}

func TestNoDeathBelowMaxSizeOrAtFloor(t *testing.T) {
	tk := New(420, 520, 17)
	if tk.StartDeath(0) {
		t.Fatal("death must not start below max size")
	}
	// Leave a single live max-size fish.
	tk.Fishes = tk.Fishes[:1]
	adultAtMax(tk, 0)
	if tk.liveCount() != 1 {
		t.Fatalf("live=%d", tk.liveCount())
	}
	if tk.StartDeath(0) {
		t.Fatal("death must not start at the population floor")
	}
}

func TestBirthAnimationStagesAndPlacement(t *testing.T) {
	tk := New(420, 520, 19)
	// Two companions stay, so the cap is not the reason to refuse.
	adultAtMax(tk, 0)
	// Put the parent in open water so the slide need not clamp hard.
	p := tk.Panel
	w, h := tk.Fishes[0].W(), tk.Fishes[0].H()
	tk.Fishes[0].X, tk.Fishes[0].Y = p.ClampInside(p.IX+p.IW*0.45-w/2, p.IY+p.IH*0.4-h/2, w, h)
	tk.Fishes[0].Facing = 1
	startX, startY := tk.Fishes[0].X, tk.Fishes[0].Y
	acx, acy := tk.Fishes[0].Center()
	if !tk.StartBirth(0) {
		t.Fatal("expected StartBirth to succeed")
	}
	if tk.Fishes[0].State != LifeBirthPause {
		t.Fatalf("state %v, want pause", tk.Fishes[0].State)
	}

	stepSeconds(tk, BirthPauseDur-0.05)
	if tk.Fishes[0].State != LifeBirthPause {
		t.Fatalf("still in pause, got %v", tk.Fishes[0].State)
	}
	if len(tk.Fishes) != FishCount {
		t.Fatalf("newborn appeared during pause: %d", len(tk.Fishes))
	}
	if math.Hypot(tk.Fishes[0].X-startX, tk.Fishes[0].Y-startY) > 0.5 {
		t.Fatalf("parent moved during pause: %v,%v -> %v,%v", startX, startY, tk.Fishes[0].X, tk.Fishes[0].Y)
	}
	if math.Abs(tk.Fishes[0].Swell-1) > 0.02 {
		t.Fatalf("swell during pause %v", tk.Fishes[0].Swell)
	}

	stepSeconds(tk, 0.08+BirthSwellDur/2)
	if tk.Fishes[0].State != LifeBirthSwell {
		t.Fatalf("state %v, want swell", tk.Fishes[0].State)
	}
	if tk.Fishes[0].Swell <= 1.02 {
		t.Fatalf("body should swell visually, swell=%v", tk.Fishes[0].Swell)
	}
	if math.Abs(tk.Fishes[0].W()-fishW) > 0.01 {
		t.Fatalf("swell changed the collision box to %v", tk.Fishes[0].W())
	}
	if tk.Fishes[0].VisualScale() <= MaxSize+0.01 {
		t.Fatalf("visual scale should include the swell pulse, got %v", tk.Fishes[0].VisualScale())
	}

	// Finish swell and enter the slide; the newborn should exist now.
	stepSeconds(tk, BirthSwellDur)
	if tk.Fishes[0].State != LifeBirthSlide {
		t.Fatalf("state %v, want slide", tk.Fishes[0].State)
	}
	if math.Abs(tk.Fishes[0].Swell-1) > 0.05 {
		t.Fatalf("swell should return to 1, got %v", tk.Fishes[0].Swell)
	}
	if len(tk.Fishes) != FishCount+1 {
		t.Fatalf("got %d fish after swell, want %d", len(tk.Fishes), FishCount+1)
	}
	baby := tk.Fishes[len(tk.Fishes)-1]
	if !baby.Emerging || baby.ParentID != tk.Fishes[0].ID || baby.Sprite() != "" {
		t.Fatalf("newborn should be emerging with no symbol: %+v sprite=%q", baby, baby.Sprite())
	}
	bcx, bcy := baby.Center()
	// Starts inside / at the adult body.
	if math.Hypot(bcx-acx, bcy-acy) > tk.Fishes[0].W()/2 {
		t.Fatalf("newborn start %v,%v not inside adult center %v,%v", bcx, bcy, acx, acy)
	}

	stepSeconds(tk, BirthSlideDur+0.05)
	parent := tk.Fishes[0]
	if parent.State != LifeSwim {
		t.Fatalf("parent should resume swim, state=%v", parent.State)
	}
	var born *Fish
	for i := range tk.Fishes {
		if tk.Fishes[i].ParentID == parent.ID {
			born = &tk.Fishes[i]
			break
		}
	}
	if born == nil {
		t.Fatal("newborn missing after slide")
	}
	if born.Emerging {
		t.Fatal("newborn still emerging after slide")
	}
	if born.Sprite() == "" || born.Sprite() == SpriteRight || born.Sprite() == SpriteLeft {
		t.Fatalf("placed newborn should show a juvenile symbol, got %q", born.Sprite())
	}
	pcx, pcy := parent.Center()
	ncx, ncy := born.Center()
	dist := math.Hypot(ncx-pcx, ncy-pcy)
	if dist < born.W()/2 || dist > parent.W()+born.W()+20 {
		t.Fatalf("newborn end distance %v, parent %v,%v baby %v,%v", dist, pcx, pcy, ncx, ncy)
	}
	// Must stay inside the glass interior.
	clampedX, clampedY := tk.Panel.ClampInside(born.X, born.Y, born.W(), born.H())
	if math.Hypot(clampedX-born.X, clampedY-born.Y) > 0.01 {
		t.Fatalf("newborn outside glass: %v,%v clamped %v,%v", born.X, born.Y, clampedX, clampedY)
	}
	// Slide was along the facing (right) axis: baby ends to the right of the parent.
	if ncx <= pcx {
		t.Fatalf("expected slide to the facing side, parentX=%v babyX=%v", pcx, ncx)
	}
}

func TestBellyUpDeathRemovesFish(t *testing.T) {
	tk := New(420, 520, 23)
	adultAtMax(tk, 0)
	startY := tk.Fishes[0].Y
	if !tk.StartDeath(0) {
		t.Fatal("StartDeath failed")
	}
	if tk.Fishes[0].Sprite() != BellyUpRight && tk.Fishes[0].Sprite() != BellyUpLeft {
		t.Fatalf("dying sprite %q", tk.Fishes[0].Sprite())
	}
	stepSeconds(tk, 0.4)
	if tk.Fishes[0].State != LifeDying {
		t.Fatalf("state %v", tk.Fishes[0].State)
	}
	if tk.Fishes[0].Y >= startY {
		t.Fatalf("belly-up fish should drift up, y %v -> %v", startY, tk.Fishes[0].Y)
	}
	if tk.Fishes[0].Opacity() >= 0.99 {
		t.Fatalf("dying fish should fade, opacity=%v", tk.Fishes[0].Opacity())
	}
	if len(tk.Fishes) != FishCount {
		t.Fatalf("removed too early: %d", len(tk.Fishes))
	}
	stepSeconds(tk, DeathAnimDur)
	if len(tk.Fishes) != FishCount-1 {
		t.Fatalf("got %d fish after death, want %d", len(tk.Fishes), FishCount-1)
	}
}

func TestPopulationStaysAroundThree(t *testing.T) {
	if BirthChance(MaxFish) != 0 || BirthChance(4) != 0 {
		t.Fatal("birth chance must be 0 at or above 4 living")
	}
	if BirthChance(1) <= BirthChance(3) || BirthChance(2) <= BirthChance(3) {
		t.Fatal("birth should be more likely when below the target of 3")
	}
	if DeathChance(1) != 0 {
		t.Fatal("death chance must be 0 at the floor")
	}
	if DeathChance(4) <= DeathChance(3) || DeathChance(3) <= DeathChance(2) {
		t.Fatal("death should be more likely when above the target of 3")
	}

	tk := New(420, 520, 29)
	for i := range tk.Fishes {
		adultAtMax(tk, i)
	}
	minN, maxN := FishCount, FishCount
	for n := 0; n < 800; n++ {
		tk.Step(maxStep)
		live := tk.liveCount()
		if live < minN {
			minN = live
		}
		if live > maxN {
			maxN = live
		}
		if live < MinLiveFish || tk.totalCount() > MaxFish {
			t.Fatalf("population out of bounds: live=%d total=%d", live, tk.totalCount())
		}
	}
	if maxN > MaxFish {
		t.Fatalf("exceeded hard cap, max=%d", maxN)
	}
	// With a balanced lottery, a long run of max-size adults should
	// hover around three and not collapse to one or sit at the cap.
	if maxN < 3 || minN > 3 {
		t.Fatalf("range [%d,%d] never crossed the target of 3", minN, maxN)
	}
	if minN < 1 {
		t.Fatal("tank emptied")
	}
}

func TestJuvenileSymbolProgression(t *testing.T) {
	if lengthSprite(NewbornSize, 1) != "o" {
		t.Fatalf("placed newborn should be %q, got %q", "o", lengthSprite(NewbornSize, 1))
	}
	if lengthSprite(NewbornSize, -1) != "o" {
		t.Fatalf("left newborn %q", lengthSprite(NewbornSize, -1))
	}
	var last string
	var seen []string
	for s := NewbornSize; s < AdultLookSize-0.001; s += 0.04 {
		cur := lengthSprite(s, 1)
		if cur == SpriteRight {
			t.Fatalf("juvenile size %v used the adult sprite", s)
		}
		if cur == "" {
			t.Fatalf("placed juvenile at %v had no symbol", s)
		}
		if cur != last {
			seen = append(seen, cur)
			if last != "" && len(cur) < len(last) {
				t.Fatalf("symbol shrank as length increased: %q -> %q at %v", last, cur, s)
			}
			last = cur
		}
	}
	if lengthSprite(AdultLookSize, 1) != SpriteRight || lengthSprite(MaxSize, 1) != SpriteRight {
		t.Fatalf("adults must keep %q, got %q / %q", SpriteRight, lengthSprite(AdultLookSize, 1), lengthSprite(MaxSize, 1))
	}
	if lengthSprite(AdultLookSize, -1) != SpriteLeft {
		t.Fatalf("left adult %q", lengthSprite(AdultLookSize, -1))
	}
	if len(seen) < 4 {
		t.Fatalf("expected several juvenile stages, got %v", seen)
	}
	// Emerging hides the symbol (hatching feel).
	baby := Fish{Size: NewbornSize, Facing: 1, Emerging: true}
	if baby.Sprite() != "" {
		t.Fatalf("emerging newborn should have no symbol, got %q", baby.Sprite())
	}
}

func TestWeakPhaseSlowsAndFades(t *testing.T) {
	f := Fish{Size: MaxSize, State: LifeSwim, MatureAge: WeakAfter + 0.1, SpeedVar: 1, Swell: 1}
	if !f.Weak() || f.Opacity() >= 0.9 || f.SpeedScale() >= 0.8 {
		t.Fatalf("weak phase not applied: weak=%v opacity=%v speed=%v", f.Weak(), f.Opacity(), f.SpeedScale())
	}
	g := Fish{Size: StartSize, State: LifeSwim, SpeedVar: 1, Swell: 1}
	if g.Weak() || g.Opacity() != 1 || math.Abs(g.SpeedScale()-1) > 1e-9 {
		t.Fatalf("starter should not be weak: %+v opacity=%v speed=%v", g, g.Opacity(), g.SpeedScale())
	}
}
