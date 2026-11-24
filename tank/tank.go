// Package tank is the fish-tank simulation: motion is stepped with a real
// time delta, independent of the window toolkit.
package tank

import (
	"math"
	"math/rand"
)

const (
	MaxFood     = 8
	maxStep     = 1.0 / 20.0 // cap dt so a stall does not fling the fish
	minStep     = 1.0 / 1000.0
	eatRadius   = 22.0
	foodFall    = 70.0
	bubbleRise  = 36.0
	wanderSpeed = 52.0
	chaseSpeed  = 115.0
	margin      = 16.0
	chompLife   = 0.85
	fishW       = 86.0
	fishH       = 22.0
	foodW       = 14.0
	foodH       = 16.0
)

// SpriteRight and SpriteLeft are monospaced character fish.
const (
	SpriteRight = "><(((*>"
	SpriteLeft  = "<*)))><"
	FoodGlyph   = "*"
)

// Fish is the swimmer in the tank.
type Fish struct {
	X, Y     float64
	VX, VY   float64
	Facing   float64 // +1 right, -1 left
	TargetX  float64
	TargetY  float64
	Retarget float64
}

// Food falls after a click.
type Food struct {
	ID   int
	X, Y float64
}

// Bubble drifts upward.
type Bubble struct {
	ID     int
	X, Y   float64
	Speed  float64
	Glyph  string
	Phase  float64
	Wobble float64
}

// Chomp is a short "chomp" or "gulp" label at a bite.
type Chomp struct {
	ID   int
	X, Y float64
	Text string
	Age  float64
	Life float64
}

// Tank holds the simulation state.
type Tank struct {
	W, H    float64
	Fish    Fish
	Foods   []Food
	Bubbles []Bubble
	Chomps  []Chomp
	Time    float64
	nextID  int
	rng     *rand.Rand
	bites   int
}

// New returns a tank of the given size, with one fish and a few bubbles.
func New(w, h float64, seed int64) *Tank {
	t := &Tank{W: w, H: h, rng: rand.New(rand.NewSource(seed))}
	t.Fish = Fish{
		X:      w * 0.35,
		Y:      h * 0.45,
		Facing: 1,
	}
	t.pickWanderTarget()
	for i := 0; i < 7; i++ {
		t.spawnBubble(true)
	}
	return t
}

// Resize updates the world size and keeps entities inside it.
func (t *Tank) Resize(w, h float64) {
	if w < 80 {
		w = 80
	}
	if h < 80 {
		h = 80
	}
	t.W, t.H = w, h
	t.clampFish()
	for i := range t.Foods {
		t.Foods[i].X = clamp(t.Foods[i].X, margin, t.W-margin)
	}
}

// Step advances the simulation by dt seconds (real time). A long stall is
// capped so the next frame does not jump.
func (t *Tank) Step(dt float64) {
	if dt < minStep {
		return
	}
	if dt > maxStep {
		dt = maxStep
	}
	t.Time += dt

	// Food falls and leaves at the bottom.
	alive := t.Foods[:0]
	for _, f := range t.Foods {
		f.Y += foodFall * dt
		if f.Y < t.H-8 {
			alive = append(alive, f)
		}
	}
	clear(t.Foods[len(alive):])
	t.Foods = alive

	t.stepFish(dt)
	t.stepBubbles(dt)
	t.stepChomps(dt)

	// Occasional new bubbles, by the clock, not by frame count.
	if t.rng.Float64() < dt*1.3 && len(t.Bubbles) < 14 {
		t.spawnBubble(false)
	}
}

func (t *Tank) stepFish(dt float64) {
	f := &t.Fish
	tx, ty, chasing := t.nearestFood()
	speed := wanderSpeed
	if chasing {
		speed = chaseSpeed
		f.TargetX, f.TargetY = tx, ty
	} else {
		f.Retarget -= dt
		if f.Retarget <= 0 {
			t.pickWanderTarget()
		}
	}
	dx := f.TargetX - (f.X + fishW/2)
	dy := f.TargetY - (f.Y + fishH/2)
	dist := math.Hypot(dx, dy)
	if dist > 1 {
		f.VX = dx / dist * speed
		f.VY = dy / dist * speed
		f.X += f.VX * dt
		f.Y += f.VY * dt
	}
	if math.Abs(f.VX) > 6 {
		if f.VX > 0 {
			f.Facing = 1
		} else {
			f.Facing = -1
		}
	}
	t.clampFish()

	if chasing && dist < eatRadius {
		t.eatNearest()
	}
}

func (t *Tank) stepBubbles(dt float64) {
	alive := t.Bubbles[:0]
	for _, b := range t.Bubbles {
		b.Phase += dt
		b.Y -= b.Speed * dt
		b.X += math.Sin(b.Phase*1.7+b.Wobble) * 10 * dt
		if b.Y+16 > 0 {
			alive = append(alive, b)
		}
	}
	clear(t.Bubbles[len(alive):])
	t.Bubbles = alive
}

func (t *Tank) stepChomps(dt float64) {
	alive := t.Chomps[:0]
	for _, c := range t.Chomps {
		c.Age += dt
		c.Y -= 28 * dt
		if c.Age < c.Life {
			alive = append(alive, c)
		}
	}
	clear(t.Chomps[len(alive):])
	t.Chomps = alive
}

// DropFood places a piece of food at (x, y) if under the limit.
func (t *Tank) DropFood(x, y float64) bool {
	if len(t.Foods) >= MaxFood {
		return false
	}
	x = clamp(x, margin, t.W-margin)
	y = clamp(y, margin, t.H-margin)
	t.nextID++
	t.Foods = append(t.Foods, Food{ID: t.nextID, X: x, Y: y})
	return true
}

func (t *Tank) nearestFood() (x, y float64, ok bool) {
	if len(t.Foods) == 0 {
		return 0, 0, false
	}
	fx := t.Fish.X + fishW/2
	fy := t.Fish.Y + fishH/2
	best := 0
	bestD := math.MaxFloat64
	for i, f := range t.Foods {
		d := math.Hypot(f.X-fx, f.Y-fy)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return t.Foods[best].X, t.Foods[best].Y, true
}

func (t *Tank) eatNearest() {
	_, _, ok := t.nearestFood()
	if !ok {
		return
	}
	fx := t.Fish.X + fishW/2
	fy := t.Fish.Y + fishH/2
	best := 0
	bestD := math.MaxFloat64
	for i, f := range t.Foods {
		d := math.Hypot(f.X-fx, f.Y-fy)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	food := t.Foods[best]
	t.Foods = append(t.Foods[:best], t.Foods[best+1:]...)
	t.bites++
	text := "chomp"
	if t.bites%2 == 0 {
		text = "gulp"
	}
	mouthX := t.Fish.X + fishW
	if t.Fish.Facing < 0 {
		mouthX = t.Fish.X
	}
	t.nextID++
	t.Chomps = append(t.Chomps, Chomp{
		ID:   t.nextID,
		X:    mouthX,
		Y:    t.Fish.Y - 4,
		Text: text,
		Life: chompLife,
	})
	_ = food
}

func (t *Tank) pickWanderTarget() {
	t.Fish.TargetX = margin + fishW/2 + t.rng.Float64()*(t.W-2*margin-fishW)
	t.Fish.TargetY = margin + fishH/2 + t.rng.Float64()*(t.H-2*margin-fishH)
	t.Fish.Retarget = 1.4 + t.rng.Float64()*2.2
}

func (t *Tank) spawnBubble(anywhere bool) {
	t.nextID++
	y := t.H + t.rng.Float64()*40
	if anywhere {
		y = t.rng.Float64() * t.H
	}
	glyphs := []string{".", "o", "O", "°"}
	t.Bubbles = append(t.Bubbles, Bubble{
		ID:     t.nextID,
		X:      margin + t.rng.Float64()*math.Max(1, t.W-2*margin),
		Y:      y,
		Speed:  bubbleRise * (0.65 + t.rng.Float64()*0.7),
		Glyph:  glyphs[t.rng.Intn(len(glyphs))],
		Phase:  t.rng.Float64() * 6,
		Wobble: t.rng.Float64() * 6,
	})
}

func (t *Tank) clampFish() {
	t.Fish.X = clamp(t.Fish.X, margin, math.Max(margin, t.W-margin-fishW))
	t.Fish.Y = clamp(t.Fish.Y, margin, math.Max(margin, t.H-margin-fishH))
}

// Sprite returns the facing-appropriate fish characters.
func (t *Tank) Sprite() string {
	if t.Fish.Facing < 0 {
		return SpriteLeft
	}
	return SpriteRight
}

// FishSize is the drawing box of the fish, in DIPs.
func FishSize() (w, h float64) { return fishW, fishH }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
