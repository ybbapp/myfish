package tank

import "math"

// Lifecycle numbers are fixed so unit tests can step the animations
// without a GUI. Visual swell never changes the window size.
const (
	// NewbornSize is the body scale of a fish as it leaves the adult.
	NewbornSize = 0.32
	// AdultLookSize is the length at which the current adult sprite is used.
	AdultLookSize = 0.70
	// StartSize is the length of the three fish that open the tank.
	// It matches the historical 108×28 drawing box (Size 1) so existing
	// reach and wander tests keep the same geometry, and it is still
	// below MaxSize so those fish can grow by eating.
	StartSize = 1.0
	// MaxSize is the length at which a fish stops eating.
	MaxSize = 1.20
	// JuvenileGrowth is added to Size per eat while the fish is still
	// filling out to the starting adult length.
	JuvenileGrowth = 0.08
	// AdultGrowth is added to Size per eat after StartSize. Ten bites
	// take a starter adult to MaxSize, so a single 8-pellet feeding
	// does not immediately stop a fish from chasing.
	AdultGrowth = 0.02

	// MaxFish is a hard cap on simultaneous fish, including dying ones.
	MaxFish = 5
	// MinLiveFish is the floor: the last live fish is not started on a death.
	MinLiveFish = 1

	BirthPauseDur = 0.45
	BirthSwellDur = 0.40
	BirthSlideDur = 0.70
	SwellPeak     = 1.12
	DeathAnimDur  = 2.40
	WeakAfter     = 4.0
	MatureAfter   = 6.0
	LotteryEvery  = 1.0

	birthClearance = 8.0
)

// LifeState is the short, testable animation or health phase of one fish.
type LifeState int

const (
	LifeSwim LifeState = iota
	LifeBirthPause
	LifeBirthSwell
	LifeBirthSlide
	LifeDying
)

// W and H are the collision and clamp box. Juveniles shrink with Size.
// Adults at or above StartSize keep the historical 108×28 box so
// existing swim and eat tests stay geometrically stable; further growth
// is visual only (see VisualScale). Swell is never included.
func (f Fish) W() float64 { return fishW * f.boxScale() }
func (f Fish) H() float64 { return fishH * f.boxScale() }

func (f Fish) bodyScale() float64 {
	if f.Size <= 0 {
		return StartSize
	}
	return f.Size
}

func (f Fish) boxScale() float64 {
	s := f.bodyScale()
	if s >= StartSize {
		return StartSize
	}
	return s
}

// Center is the middle of the current body box.
func (f Fish) Center() (x, y float64) {
	return f.X + f.W()/2, f.Y + f.H()/2
}

// VisualScale is the drawing scale, including a short birth swell pulse.
func (f Fish) VisualScale() float64 {
	s := f.bodyScale()
	if f.Swell > 0 {
		s *= f.Swell
	}
	return s
}

// Opacity fades a weak or belly-up fish. Ordinary swimmers stay fully visible.
func (f Fish) Opacity() float64 {
	if f.State == LifeDying {
		fade := 1 - 0.7*(f.AnimT/DeathAnimDur)
		if fade < 0.25 {
			fade = 0.25
		}
		return fade
	}
	if f.Weak() {
		return 0.62
	}
	return 1
}

// Weak is the optional pre-death phase: max-size fish that have lived
// long enough at the cap swim slower and draw faded, until they birth or die.
func (f Fish) Weak() bool {
	return f.AtMax() && f.State == LifeSwim && f.MatureAge >= WeakAfter
}

// AtMax reports whether this fish has reached the eating cap.
func (f Fish) AtMax() bool {
	return f.bodyScale() >= MaxSize-1e-9
}

// CanEat reports whether the fish will still chase and consume food.
func (f Fish) CanEat() bool {
	return f.State == LifeSwim && !f.Emerging && !f.AtMax()
}

// Live reports a fish that still counts toward the population floor.
// Dying fish remain in the slice until the belly-up animation ends,
// but they no longer count as living.
func (f Fish) Live() bool {
	return f.State != LifeDying
}

func (f Fish) busy() bool {
	return f.State == LifeBirthPause || f.State == LifeBirthSwell || f.State == LifeBirthSlide || f.State == LifeDying || f.Emerging
}

func (t *Tank) liveCount() int {
	n := 0
	for _, f := range t.Fishes {
		if f.Live() {
			n++
		}
	}
	return n
}

func (t *Tank) totalCount() int { return len(t.Fishes) }

func (t *Tank) birthing() bool {
	for _, f := range t.Fishes {
		if f.State == LifeBirthPause || f.State == LifeBirthSwell || f.State == LifeBirthSlide {
			return true
		}
	}
	return false
}

// BirthChance is the per-lottery probability that a mature max-size fish
// starts a birth, given the current number of living fish.
func BirthChance(live int) float64 {
	switch {
	case live >= MaxFish:
		return 0
	case live <= 2:
		return 0.55
	case live == 3:
		return 0.12
	default:
		return 0
	}
}

// DeathChance is the per-lottery probability that a mature max-size fish
// starts a belly-up death, given the current number of living fish.
func DeathChance(live int) float64 {
	switch {
	case live <= MinLiveFish:
		return 0
	case live == 2:
		return 0.08
	case live == 3:
		return 0.20
	default:
		return 0.50
	}
}

func (t *Tank) setFishSize(i int, size float64) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	f := &t.Fishes[i]
	oldW, oldH := f.W(), f.H()
	cx, cy := f.Center()
	f.Size = clamp(size, NewbornSize, MaxSize)
	w, h := f.W(), f.H()
	if w == oldW && h == oldH {
		return
	}
	// Grow or shrink around the body center, then keep the box in the glass.
	f.X, f.Y = t.Panel.ClampInside(cx-w/2, cy-h/2, w, h)
}

func (t *Tank) growFromEat(i int) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	f := &t.Fishes[i]
	if !f.CanEat() {
		return
	}
	was := f.Size
	if was <= 0 {
		was = StartSize
	}
	delta := AdultGrowth
	if was < StartSize-1e-9 {
		delta = JuvenileGrowth
	}
	t.setFishSize(i, was+delta)
	if t.Fishes[i].AtMax() && was < MaxSize-1e-9 {
		t.Fishes[i].MatureAge = 0
		t.Fishes[i].lotteryCD = 0
	}
}

// StartBirth begins the pause–swell–slide sequence on a max-size fish.
// It returns false if the fish is not ready or the tank is at the cap.
func (t *Tank) StartBirth(i int) bool {
	if i < 0 || i >= len(t.Fishes) {
		return false
	}
	f := &t.Fishes[i]
	if !f.AtMax() || f.busy() || t.birthing() {
		return false
	}
	if t.liveCount() >= MaxFish || t.totalCount() >= MaxFish {
		return false
	}
	f.State = LifeBirthPause
	f.AnimT = 0
	f.Swell = 1
	f.VX, f.VY = 0, 0
	return true
}

// StartDeath begins the belly-up float on a max-size fish.
// It returns false if the fish is not ready or it would empty the tank.
func (t *Tank) StartDeath(i int) bool {
	if i < 0 || i >= len(t.Fishes) {
		return false
	}
	f := &t.Fishes[i]
	if !f.AtMax() || f.busy() {
		return false
	}
	if t.liveCount() <= MinLiveFish {
		return false
	}
	f.State = LifeDying
	f.AnimT = 0
	f.Swell = 1
	f.VX, f.VY = 0, 0
	return true
}

func (t *Tank) stepLife(dt float64) {
	for i := range t.Fishes {
		t.stepLifeFish(i, dt)
	}
	t.rollLifeEvents(dt)
	t.reapDead()
}

func (t *Tank) stepLifeFish(i int, dt float64) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	f := &t.Fishes[i]
	if f.Size <= 0 {
		f.Size = StartSize
	}
	if f.Swell <= 0 {
		f.Swell = 1
	}
	if f.SpeedVar <= 0 {
		f.SpeedVar = 1
	}

	if f.Emerging {
		t.stepEmerging(i, dt)
		return
	}

	switch f.State {
	case LifeBirthPause:
		f.VX, f.VY = 0, 0
		f.Swell = 1
		f.AnimT += dt
		if f.AnimT >= BirthPauseDur {
			f.State = LifeBirthSwell
			f.AnimT = 0
		}
	case LifeBirthSwell:
		f.VX, f.VY = 0, 0
		f.AnimT += dt
		u := f.AnimT / BirthSwellDur
		if u > 1 {
			u = 1
		}
		// One visual pulse: 1 → SwellPeak → 1, not a window resize.
		f.Swell = 1 + (SwellPeak-1)*math.Sin(math.Pi*u)
		if f.AnimT >= BirthSwellDur {
			f.Swell = 1
			if t.spawnNewbornFrom(i) {
				f.State = LifeBirthSlide
			} else {
				f.State = LifeSwim
			}
			f.AnimT = 0
			f.MatureAge = 0
			f.lotteryCD = 0
		}
	case LifeBirthSlide:
		f.VX, f.VY = 0, 0
		f.Swell = 1
		f.AnimT += dt
		if f.AnimT >= BirthSlideDur {
			f.State = LifeSwim
			f.AnimT = 0
			t.pickWanderTarget(i)
		}
	case LifeDying:
		t.stepDying(i, dt)
	default:
		if f.AtMax() {
			f.MatureAge += dt
		}
	}
}

func (t *Tank) spawnNewbornFrom(parent int) bool {
	if parent < 0 || parent >= len(t.Fishes) {
		return false
	}
	if t.totalCount() >= MaxFish {
		return false
	}
	adult := t.Fishes[parent]
	fromX, fromY, toX, toY := birthPath(t.Panel, adult)
	t.nextID++
	tint := (adult.Tint + 1) % 3
	speed := 0.92 + t.rng.Float64()*0.16
	baby := Fish{
		ID:         t.nextID,
		X:          fromX,
		Y:          fromY,
		Facing:     adult.Facing,
		Size:       NewbornSize,
		Swell:      1,
		State:      LifeSwim,
		Emerging:   true,
		ParentID:   adult.ID,
		SlideFromX: fromX,
		SlideFromY: fromY,
		SlideToX:   toX,
		SlideToY:   toY,
		SpeedVar:   speed,
		Tint:       tint,
	}
	t.Fishes = append(t.Fishes, baby)
	t.clampFish(len(t.Fishes) - 1)
	// Re-apply the intended start after clamp, in case the box was nudged,
	// then recompute a clamped end point.
	i := len(t.Fishes) - 1
	t.Fishes[i].X, t.Fishes[i].Y = fromX, fromY
	t.Fishes[i].SlideFromX, t.Fishes[i].SlideFromY = fromX, fromY
	endX, endY := t.Panel.ClampInside(toX, toY, baby.W(), baby.H())
	t.Fishes[i].SlideToX, t.Fishes[i].SlideToY = endX, endY
	t.clampFish(i)
	return true
}

// birthPath returns the newborn top-left at the adult's center, and a
// side exit just outside the adult along its facing axis, already
// intended to sit inside the glass (the caller still clamps).
func birthPath(p Panel, adult Fish) (fromX, fromY, toX, toY float64) {
	cx, cy := adult.Center()
	bw, bh := fishW*NewbornSize, fishH*NewbornSize
	fromX, fromY = cx-bw/2, cy-bh/2
	face := adult.Facing
	if face == 0 {
		face = 1
	}
	// Slide out of the flank along the facing (side) axis, ending nearby.
	span := adult.W()/2 + bw/2 + birthClearance
	toCX := cx + face*span
	toCY := cy
	toX, toY = toCX-bw/2, toCY-bh/2
	fromX, fromY = p.ClampInside(fromX, fromY, bw, bh)
	toX, toY = p.ClampInside(toX, toY, bw, bh)
	return fromX, fromY, toX, toY
}

func (t *Tank) stepEmerging(i int, dt float64) {
	f := &t.Fishes[i]
	f.AnimT += dt
	u := f.AnimT / BirthSlideDur
	if u > 1 {
		u = 1
	}
	// Smoothstep so the exit eases out of the body.
	s := u * u * (3 - 2*u)
	f.X = f.SlideFromX + (f.SlideToX-f.SlideFromX)*s
	f.Y = f.SlideFromY + (f.SlideToY-f.SlideFromY)*s
	f.VX, f.VY = 0, 0
	t.clampFish(i)
	if f.AnimT >= BirthSlideDur {
		f.Emerging = false
		f.AnimT = 0
		f.X, f.Y = f.SlideToX, f.SlideToY
		t.clampFish(i)
		t.pickWanderTarget(i)
	}
}

func (t *Tank) stepDying(i int, dt float64) {
	f := &t.Fishes[i]
	f.AnimT += dt
	f.Swell = 1
	// Belly-up: drift toward the surface with a slight wobble.
	f.Y -= 22 * dt
	f.X += math.Sin(f.AnimT*3.2) * 10 * dt
	f.VX, f.VY = 0, -22
	t.clampFish(i)
}

func (t *Tank) rollLifeEvents(dt float64) {
	live := t.liveCount()
	bc := BirthChance(live)
	dc := DeathChance(live)
	for i := range t.Fishes {
		f := &t.Fishes[i]
		if !f.AtMax() || f.busy() {
			continue
		}
		if f.MatureAge < MatureAfter {
			continue
		}
		f.lotteryCD += dt
		if f.lotteryCD < LotteryEvery {
			continue
		}
		f.lotteryCD = 0
		// Refresh chances if an earlier fish in this pass already changed counts.
		live = t.liveCount()
		bc = BirthChance(live)
		dc = DeathChance(live)
		roll := t.rng.Float64()
		if bc > 0 && roll < bc {
			if t.StartBirth(i) {
				continue
			}
		}
		// A second independent roll keeps birth and death from always
		// competing on the same number, while still remaining deterministic
		// under a fixed seed.
		if dc > 0 && t.rng.Float64() < dc {
			t.StartDeath(i)
		}
	}
}

func (t *Tank) reapDead() {
	alive := t.Fishes[:0]
	for _, f := range t.Fishes {
		if f.State == LifeDying && f.AnimT >= DeathAnimDur {
			continue
		}
		alive = append(alive, f)
	}
	if len(alive) < len(t.Fishes) {
		clear(t.Fishes[len(alive):])
		t.Fishes = alive
	}
}

// Sprite returns the facing-appropriate characters for this fish.
// Newborns have no symbol while emerging (hatching). The symbol then
// appears and lengthens with Size. Adults at or above AdultLookSize
// keep the historical ><(((o> / <o)))>< look. A dying fish is belly-up.
func (f Fish) Sprite() string {
	if f.State == LifeDying {
		if f.Facing < 0 {
			return BellyUpLeft
		}
		return BellyUpRight
	}
	if f.Emerging {
		return ""
	}
	return lengthSprite(f.Size, f.Facing)
}

// Belly-up approximations of the adult glyphs, used only during death.
const (
	BellyUpRight = ">-)))o<"
	BellyUpLeft  = ">o(((-<"
)

func lengthSprite(size, facing float64) string {
	if size >= AdultLookSize-1e-9 {
		if facing < 0 {
			return SpriteLeft
		}
		return SpriteRight
	}
	t := (size - NewbornSize) / (AdultLookSize - NewbornSize)
	if t < 0 {
		t = 0
	}
	// At placement size (NewbornSize) the first "o" is already visible;
	// only the in-body slide (Emerging) has no symbol.
	var stages []string
	if facing < 0 {
		stages = []string{"o", "<o", "<o)", "<o))", "<o)))", "<o)))>"}
	} else {
		stages = []string{"o", "o>", "(o>", "((o>", "(((o>", "<(((o>"}
	}
	// Map (0, 1) onto the six juvenile stages, then the adult sprite.
	idx := int(t * float64(len(stages)))
	if idx >= len(stages) {
		idx = len(stages) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return stages[idx]
}

// SpeedScale is wander/chase speed after per-fish variation and weakness.
func (f Fish) SpeedScale() float64 {
	s := f.SpeedVar
	if s <= 0 {
		s = 1
	}
	if f.Weak() {
		s *= 0.45
	}
	return s
}
