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
	maxStep     = 1.0 / 20.0
	minStep     = 1.0 / 1000.0
	eatRadius   = 28.0
	foodFall    = 70.0
	bubbleRise  = 36.0
	wanderSpeed = 52.0
	chaseSpeed  = 115.0
	chompLife   = 1.35
	fishW       = 108.0
	fishH       = 28.0
	minFishGap  = 80.0
)

// Character sprites follow the reference stills, not a round bowl.
const (
	SpriteRight = "><(((o>"
	SpriteLeft  = "<o)))><"
	FoodGlyph   = "o"
	FoodStar    = "*"
	PlantLeft   = "{~"
	PlantRight  = "~}"
)

var biteWords = []string{"chomp", "crunch", "gulp"}

// Fish is one swimmer on the plate.
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
	ID    int
	X, Y  float64
	Glyph string
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

// Chomp is a short bite label.
type Chomp struct {
	ID   int
	X, Y float64
	Text string
	Age  float64
	Life float64
}

// Plant is a static green character tuft at the bottom of the plate.
type Plant struct {
	X, Y float64
	Text string
}

// Tank holds the simulation state.
type Tank struct {
	W, H    float64
	Panel   Panel
	Fishes  []Fish
	Foods   []Food
	Bubbles []Bubble
	Chomps  []Chomp
	Plants  []Plant
	Time    float64
	nextID  int
	rng     *rand.Rand
	bites   int
}

// New returns a tank of the given window size, with several fish, plants,
// and a few bubbles already on the plate.
func New(w, h float64, seed int64) *Tank {
	t := &Tank{rng: rand.New(rand.NewSource(seed))}
	t.applySize(w, h)
	starts := [][3]float64{
		{0.18, 0.28, 1},
		{0.62, 0.48, -1},
		{0.36, 0.66, 1},
	}
	for _, s := range starts {
		t.nextID++
		x, y := t.Panel.RandomPoint(s[0], s[1], 8)
		x, y = t.Panel.ClampInside(x-fishW/2, y-fishH/2, fishW, fishH)
		t.Fishes = append(t.Fishes, Fish{
			ID:     t.nextID,
			X:      x,
			Y:      y,
			Facing: s[2],
		})
		t.pickWanderTarget(len(t.Fishes) - 1)
	}
	p := t.Panel
	t.Plants = []Plant{
		{X: p.IX + p.IW*0.18, Y: p.PlantY - 4, Text: PlantLeft},
		{X: p.IX + p.IW*0.72, Y: p.PlantY - 4, Text: PlantRight},
	}
	for i := 0; i < 6; i++ {
		t.spawnBubble(true)
	}
	return t
}

func (t *Tank) applySize(w, h float64) {
	t.W, t.H = w, h
	t.Panel = Layout(w, h)
}

// Resize updates the window size and keeps entities on the plate.
func (t *Tank) Resize(w, h float64) {
	t.applySize(w, h)
	for i := range t.Fishes {
		t.clampFish(i)
	}
	for i := range t.Foods {
		t.Foods[i].X, t.Foods[i].Y = t.Panel.ClampInside(t.Foods[i].X, t.Foods[i].Y, 8, 8)
	}
	p := t.Panel
	if len(t.Plants) >= 2 {
		t.Plants[0].X, t.Plants[0].Y = p.IX+p.IW*0.18, p.PlantY-4
		t.Plants[1].X, t.Plants[1].Y = p.IX+p.IW*0.72, p.PlantY-4
	}
}

// Step advances the simulation by dt seconds (real time).
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
		f.X, f.Y = t.Panel.ClampInside(f.X, f.Y, 8, 8)
		if f.Y >= t.Panel.PlantY-6 {
			f.Y = t.Panel.PlantY - 6
		}
		alive = append(alive, f)
	}
	clear(t.Foods[len(alive):])
	t.Foods = alive

	for i := range t.Fishes {
		t.stepFish(i, dt)
	}
	t.separateFish(dt)
	t.stepBubbles(dt)
	t.stepChomps(dt)

	if t.rng.Float64() < dt*1.3 && len(t.Bubbles) < 12 {
		t.spawnBubble(false)
	}
}

func (t *Tank) stepFish(i int, dt float64) {
	f := &t.Fishes[i]
	foodX, foodY, chasing := t.nearestFood(i)
	speed := wanderSpeed
	if chasing {
		speed = chaseSpeed
		f.TargetX, f.TargetY = t.spreadAim(i, foodX, foodY)
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

	if chasing {
		mouthX := f.X + fishW/2
		mouthY := f.Y + fishH/2
		if math.Hypot(foodX-mouthX, foodY-mouthY) < eatRadius {
			t.eatNearest(i)
		}
	}
}

func (t *Tank) spreadAim(i int, foodX, foodY float64) (x, y float64) {
	closest := i
	best := math.MaxFloat64
	var peers []int
	for j, f := range t.Fishes {
		fx, fy, ok := t.nearestFood(j)
		if !ok || math.Hypot(fx-foodX, fy-foodY) > 1 {
			continue
		}
		peers = append(peers, j)
		d := math.Hypot(f.X+fishW/2-foodX, f.Y+fishH/2-foodY)
		if d < best {
			best, closest = d, j
		}
	}
	if i == closest || len(peers) <= 1 {
		return foodX, foodY
	}
	rank := 0
	for _, p := range peers {
		if p == closest {
			continue
		}
		if p == i {
			break
		}
		rank++
	}
	n := len(peers) - 1
	if n < 1 {
		n = 1
	}
	ang := math.Pi*0.55 + 2*math.Pi*float64(rank)/float64(n)
	r := 40.0
	return foodX + math.Cos(ang)*r, foodY + math.Sin(ang)*r
}

func (t *Tank) separateFish(dt float64) {
	if dt <= 0 {
		return
	}
	nudge := 180 * dt
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
	top := t.Panel.IY + 6
	for _, b := range t.Bubbles {
		b.Phase += dt
		b.Y -= b.Speed * dt
		b.X += math.Sin(b.Phase*1.7+b.Wobble) * 8 * dt
		if b.X < t.Panel.IX+4 {
			b.X = t.Panel.IX + 4
		}
		if b.X > t.Panel.IX+t.Panel.IW-12 {
			b.X = t.Panel.IX + t.Panel.IW - 12
		}
		if b.Y > top {
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

// DropFood places food at (x, y) if the point is on the plate interior.
func (t *Tank) DropFood(x, y float64) bool {
	if len(t.Foods) >= MaxFood {
		return false
	}
	if !t.Panel.ContainsInterior(x, y) {
		return false
	}
	x, y = t.Panel.ClampInside(x-4, y-4, 8, 8)
	t.nextID++
	glyph := FoodGlyph
	if t.rng.Intn(3) == 0 {
		glyph = FoodStar
	}
	t.Foods = append(t.Foods, Food{ID: t.nextID, X: x, Y: y, Glyph: glyph})
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
	text := biteWords[(t.bites-1)%len(biteWords)]
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
	x, y := t.Panel.RandomPoint(t.rng.Float64(), t.rng.Float64(), 10)
	f.TargetX, f.TargetY = x, y
	f.Retarget = 1.4 + t.rng.Float64()*2.2
}

func (t *Tank) spawnBubble(anywhere bool) {
	t.nextID++
	var x, y float64
	if anywhere {
		x, y = t.Panel.RandomPoint(t.rng.Float64(), t.rng.Float64(), 8)
	} else {
		x, y = t.Panel.RandomPoint(0.15+t.rng.Float64()*0.7, 0.88, 8)
	}
	glyphs := []string{"o", "O"}
	t.Bubbles = append(t.Bubbles, Bubble{
		ID:     t.nextID,
		X:      x,
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
	t.Fishes[i].X, t.Fishes[i].Y = t.Panel.ClampInside(t.Fishes[i].X, t.Fishes[i].Y, fishW, fishH)
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
