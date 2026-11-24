// Package tank is the fish-tank simulation: motion is stepped with a real
// time delta, independent of the window toolkit.
package tank

import (
	"math"
	"math/rand"
)

const (
	MaxFood     = 8
	FishCount   = 3
	maxStep     = 1.0 / 20.0 // cap dt so a stall does not fling the fish
	minStep     = 1.0 / 1000.0
	eatRadius   = 32.0
	foodFall    = 70.0
	bubbleRise  = 36.0
	wanderSpeed = 52.0
	chaseSpeed  = 115.0
	margin      = 16.0
	chompLife   = 1.35 // long enough to notice on a screenshot
	fishW       = 120.0
	fishH       = 32.0
	foodW       = 20.0
	foodH       = 22.0
	minFishGap  = 96.0
)

// SpriteRight and SpriteLeft are monospaced character fish.
const (
	SpriteRight = "><(((*>"
	SpriteLeft  = "<*)))><"
	FoodGlyph   = "*"
)

// Fish is one swimmer in the tank.
type Fish struct {
	ID       int
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
	Fishes  []Fish
	Foods   []Food
	Bubbles []Bubble
	Chomps  []Chomp
	Time    float64
	nextID  int
	rng     *rand.Rand
	bites   int
}

// New returns a tank of the given size, with several scattered fish and a
// few bubbles.
func New(w, h float64, seed int64) *Tank {
	t := &Tank{W: w, H: h, rng: rand.New(rand.NewSource(seed))}
	starts := [][3]float64{
		{0.16, 0.26, 1},
		{0.58, 0.50, -1},
		{0.30, 0.74, 1},
	}
	for _, s := range starts {
		t.nextID++
		t.Fishes = append(t.Fishes, Fish{
			ID:     t.nextID,
			X:      clamp(w*s[0], margin, math.Max(margin, w-margin-fishW)),
			Y:      clamp(h*s[1], margin, math.Max(margin, h-margin-fishH)),
			Facing: s[2],
		})
		t.pickWanderTarget(len(t.Fishes) - 1)
	}
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
	for i := range t.Fishes {
		t.clampFish(i)
	}
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

	alive := t.Foods[:0]
	for _, f := range t.Foods {
		f.Y += foodFall * dt
		if f.Y < t.H-8 {
			alive = append(alive, f)
		}
	}
	clear(t.Foods[len(alive):])
	t.Foods = alive

	for i := range t.Fishes {
		t.stepFish(i, dt)
	}
	t.separateFish(dt)
	t.stepBubbles(dt)
	t.stepChomps(dt)

	if t.rng.Float64() < dt*1.3 && len(t.Bubbles) < 14 {
		t.spawnBubble(false)
	}
}

func (t *Tank) stepFish(i int, dt float64) {
	f := &t.Fishes[i]
	tx, ty, chasing := t.nearestFood(i)
	speed := wanderSpeed
	if chasing {
		speed = chaseSpeed
		f.TargetX, f.TargetY = tx, ty
	} else {
		f.Retarget -= dt
		if f.Retarget <= 0 {
			t.pickWanderTarget(i)
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
	t.clampFish(i)

	if chasing && dist < eatRadius {
		t.eatNearest(i)
	}
}

func (t *Tank) separateFish(dt float64) {
	if dt <= 0 {
		return
	}
	nudge := 40 * dt
	for i := 0; i < len(t.Fishes); i++ {
		axi, ayi := t.Fishes[i].X+fishW/2, t.Fishes[i].Y+fishH/2
		for j := i + 1; j < len(t.Fishes); j++ {
			axj, ayj := t.Fishes[j].X+fishW/2, t.Fishes[j].Y+fishH/2
			dx, dy := axi-axj, ayi-ayj
			d := math.Hypot(dx, dy)
			if d >= minFishGap || d < 0.5 {
				continue
			}
			nx, ny := dx/d, dy/d
			push := (minFishGap - d) * nudge / minFishGap
			t.Fishes[i].X += nx * push
			t.Fishes[i].Y += ny * push
			t.Fishes[j].X -= nx * push
			t.Fishes[j].Y -= ny * push
			t.clampFish(i)
			t.clampFish(j)
		}
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

func (t *Tank) nearestFood(i int) (x, y float64, ok bool) {
	if i < 0 || i >= len(t.Fishes) || len(t.Foods) == 0 {
		return 0, 0, false
	}
	f := t.Fishes[i]
	fx := f.X + fishW/2
	fy := f.Y + fishH/2
	best := 0
	bestD := math.MaxFloat64
	for i, food := range t.Foods {
		d := math.Hypot(food.X-fx, food.Y-fy)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return t.Foods[best].X, t.Foods[best].Y, true
}

func (t *Tank) eatNearest(i int) {
	if i < 0 || i >= len(t.Fishes) || len(t.Foods) == 0 {
		return
	}
	f := &t.Fishes[i]
	fx := f.X + fishW/2
	fy := f.Y + fishH/2
	best := 0
	bestD := math.MaxFloat64
	for j, food := range t.Foods {
		d := math.Hypot(food.X-fx, food.Y-fy)
		if d < bestD {
			bestD = d
			best = j
		}
	}
	t.Foods = append(t.Foods[:best], t.Foods[best+1:]...)
	t.bites++
	text := "chomp"
	if t.bites%2 == 0 {
		text = "gulp"
	}
	mouthX := f.X + fishW
	if f.Facing < 0 {
		mouthX = f.X
	}
	t.nextID++
	t.Chomps = append(t.Chomps, Chomp{
		ID:   t.nextID,
		X:    mouthX,
		Y:    f.Y - 6,
		Text: text,
		Life: chompLife,
	})
}

func (t *Tank) pickWanderTarget(i int) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	f := &t.Fishes[i]
	f.TargetX = margin + fishW/2 + t.rng.Float64()*(t.W-2*margin-fishW)
	f.TargetY = margin + fishH/2 + t.rng.Float64()*(t.H-2*margin-fishH)
	f.Retarget = 1.4 + t.rng.Float64()*2.2
}

func (t *Tank) spawnBubble(anywhere bool) {
	t.nextID++
	y := t.H + t.rng.Float64()*40
	if anywhere {
		y = t.rng.Float64() * t.H
	}
	// Skip "." — it vanishes on a light desktop.
	glyphs := []string{"o", "O", "°"}
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

func (t *Tank) clampFish(i int) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	t.Fishes[i].X = clamp(t.Fishes[i].X, margin, math.Max(margin, t.W-margin-fishW))
	t.Fishes[i].Y = clamp(t.Fishes[i].Y, margin, math.Max(margin, t.H-margin-fishH))
}

// Sprite returns the facing-appropriate characters for fish i.
func (t *Tank) Sprite(i int) string {
	if i < 0 || i >= len(t.Fishes) {
		return SpriteRight
	}
	return t.Fishes[i].Sprite()
}

// Sprite returns the facing-appropriate characters for this fish.
func (f Fish) Sprite() string {
	if f.Facing < 0 {
		return SpriteLeft
	}
	return SpriteRight
}

// FishSize is the drawing box of one fish, in DIPs.
func FishSize() (w, h float64) { return fishW, fishH }

// ChompLife is how long a bite label stays visible, in seconds.
func ChompLife() float64 { return chompLife }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
