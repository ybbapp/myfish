// Package tank is the fish-tank simulation: motion is stepped with a real
// time delta, independent of the window toolkit.
package tank

import (
	"math"
	"math/rand"
	"sort"
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
	hoverSpeed  = 26.0
	hoverStand  = 56.0
	chompLife   = 1.35
	foodLife    = 16.0
	foodFade    = 2.0
	fishW       = 108.0
	fishH       = 28.0
	fishGap     = 10.0
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
	ID                     int
	X, Y                   float64
	VX, VY                 float64
	Facing                 float64 // +1 right, -1 left
	TargetX                float64
	TargetY                float64
	Retarget               float64
	Size                   float64 // body scale; 1 is the historical 108×28 adult
	Swell                  float64 // visual pulse only, 1 when idle
	MatureAge              float64
	State                  LifeState
	AnimT                  float64
	Emerging               bool
	ParentID               int
	SlideFromX, SlideFromY float64
	SlideToX, SlideToY     float64
	SpeedVar               float64
	Tint                   int
	GlassReaction          GlassReaction // stable per-fish temperament, not re-rolled per frame
	glassRecovery          float64
	lotteryCD              float64
}

// Food falls after a click on the strip above the plate.
// Uneaten pellets fade after FoodLife seconds; this is a per-pellet
// timeout, not a global wipe of the bowl.
type Food struct {
	ID    int
	X, Y  float64
	Glyph string
	Age   float64
	Life  float64
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
	W, H      float64
	Panel     Panel
	Fishes    []Fish
	Foods     []Food
	Bubbles   []Bubble
	Chomps    []Chomp
	Plants    []Plant
	Time      float64
	Hovering  bool
	HoverX    float64
	HoverY    float64
	nextID    int
	rng       *rand.Rand
	bites     int
	foodTrace *FoodTrace
}

// FoodRecord is a test/debug event for one pellet. It is never drawn
// and holds no user or host data — only simulation IDs and positions.
type FoodRecord struct {
	FoodID int     `json:"food_id"`
	Reason string  `json:"reason"` // spawned, consumed, expired
	Time   float64 `json:"time"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	FishID int     `json:"fish_id,omitempty"`
	Age    float64 `json:"age,omitempty"`
	Life   float64 `json:"life,omitempty"`
}

// FoodTrace records pellet fates when EnableFoodTrace is called.
// The default (nil) costs a nil check and does not change the UI.
type FoodTrace struct {
	Events   []FoodRecord `json:"events"`
	Spawned  int          `json:"spawned"`
	Consumed int          `json:"consumed"`
	Expired  int          `json:"expired"`
}

// EnableFoodTrace starts recording pellet spawn/consume/expire events.
// It is for tests and debugging; the window never displays this.
func (t *Tank) EnableFoodTrace() { t.foodTrace = &FoodTrace{} }

// FoodTrace returns the recording, or nil if tracing is off.
func (t *Tank) FoodTrace() *FoodTrace { return t.foodTrace }

func (t *Tank) recordFood(rec FoodRecord) {
	if t.foodTrace == nil {
		return
	}
	t.foodTrace.Events = append(t.foodTrace.Events, rec)
	switch rec.Reason {
	case "spawned":
		t.foodTrace.Spawned++
	case "consumed":
		t.foodTrace.Consumed++
	case "expired":
		t.foodTrace.Expired++
	}
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
			ID:            t.nextID,
			X:             x,
			Y:             y,
			Facing:        s[2],
			Size:          StartSize,
			Swell:         1,
			SpeedVar:      1,
			Tint:          (t.nextID - 1) % 3,
			GlassReaction: initialGlassReaction(seed, len(t.Fishes)),
		})
		t.pickWanderTarget(len(t.Fishes) - 1)
	}
	p := t.Panel
	// Plants are visual decoration only: not a collision body, not a
	// swim floor, and they do not block eating.
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

	floor := t.Panel.FloorY() - 8
	alive := t.Foods[:0]
	for _, f := range t.Foods {
		f.Age += dt
		if f.Life <= 0 {
			f.Life = foodLife
		}
		if f.Age >= f.Life {
			t.recordFood(FoodRecord{
				FoodID: f.ID, Reason: "expired", Time: t.Time,
				X: f.X, Y: f.Y, Age: f.Age, Life: f.Life,
			})
			continue
		}
		f.Y += foodFall * dt
		f.X, f.Y = t.Panel.ClampInside(f.X, f.Y, 8, 8)
		if f.Y >= floor {
			f.Y = floor
		}
		alive = append(alive, f)
	}
	clear(t.Foods[len(alive):])
	t.Foods = alive

	hunters := t.assignHunters()
	for i := range t.Fishes {
		t.stepFish(i, dt, hunters[i])
	}
	t.separateFish(hunters)
	t.stepBubbles(dt)
	t.stepChomps(dt)
	t.stepLife(dt)

	if t.rng.Float64() < dt*1.3 && len(t.Bubbles) < 12 {
		t.spawnBubble(false)
	}
}

func (t *Tank) stepFish(i int, dt float64, foodIdx int) {
	f := &t.Fishes[i]
	if f.busy() || f.Emerging {
		return
	}
	chasing := foodIdx >= 0 && foodIdx < len(t.Foods) && f.CanEat()
	speed := wanderSpeed * f.SpeedScale()
	if chasing {
		speed = chaseSpeed * f.SpeedScale()
		fx, fy := t.Foods[foodIdx].Center()
		f.TargetX, f.TargetY = fx, fy
		if fx < f.X+f.W()/2-4 {
			f.Facing = -1
		} else if fx > f.X+f.W()/2+4 {
			f.Facing = 1
		}
	} else if t.Hovering {
		speed = t.glassTarget(f)
		f.glassRecovery = 0.65
	} else {
		f.Retarget -= dt
		if f.Retarget <= 0 {
			t.pickWanderTarget(i)
		}
	}
	originX, originY := f.Center()
	if chasing {
		originX, originY = f.Mouth()
	}
	dx := f.TargetX - originX
	dy := f.TargetY - originY
	dist := math.Hypot(dx, dy)
	wantVX, wantVY := 0.0, 0.0
	if dist > 1 {
		wantVX, wantVY = dx/dist*speed, dy/dist*speed
	}
	if !chasing && (t.Hovering || f.glassRecovery > 0) {
		// A real-time exponential blend makes entry/exit gentle and stable
		// across frame rates without changing normal food-chasing physics.
		blend := 1 - math.Exp(-dt/0.22)
		f.VX += (wantVX - f.VX) * blend
		f.VY += (wantVY - f.VY) * blend
		if !t.Hovering {
			f.glassRecovery = math.Max(0, f.glassRecovery-dt)
		}
	} else {
		f.VX, f.VY = wantVX, wantVY
	}
	f.X += f.VX * dt
	f.Y += f.VY * dt
	if !chasing {
		if t.Hovering && f.GlassReaction != GlassShy {
			// Curious fish watch the contact even when holding still.
			if t.HoverX < f.X+f.W()/2-8 {
				f.Facing = -1
			} else if t.HoverX > f.X+f.W()/2+8 {
				f.Facing = 1
			}
		} else if math.Abs(f.VX) > 6 {
			f.Facing = math.Copysign(1, f.VX)
		}
	}
	t.clampFish(i)

	if chasing {
		mx, my := f.Mouth()
		cx, cy := t.Foods[foodIdx].Center()
		if math.Hypot(cx-mx, cy-my) < eatRadius {
			t.eatFood(i, foodIdx)
		}
	}
}

func (t *Tank) nearestHover() int {
	best, bestD := 0, math.MaxFloat64
	for i, f := range t.Fishes {
		if !f.Live() || f.Emerging {
			continue
		}
		cx, cy := f.Center()
		d := math.Hypot(cx-t.HoverX, cy-t.HoverY)
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func (t *Tank) assignHunters() []int {
	assigned := make([]int, len(t.Fishes))
	for i := range assigned {
		assigned[i] = -1
	}
	if len(t.Foods) == 0 || len(t.Fishes) == 0 {
		return assigned
	}
	type pair struct {
		fi, fo int
		d      float64
	}
	pairs := make([]pair, 0, len(t.Fishes)*len(t.Foods))
	for i, f := range t.Fishes {
		if !f.CanEat() {
			continue
		}
		mx, my := f.Mouth()
		for j, food := range t.Foods {
			cx, cy := food.Center()
			pairs = append(pairs, pair{i, j, math.Hypot(cx-mx, cy-my)})
		}
	}
	sort.Slice(pairs, func(a, b int) bool { return pairs[a].d < pairs[b].d })
	fishUsed := make([]bool, len(t.Fishes))
	foodUsed := make([]bool, len(t.Foods))
	for _, p := range pairs {
		if fishUsed[p.fi] || foodUsed[p.fo] {
			continue
		}
		fishUsed[p.fi] = true
		foodUsed[p.fo] = true
		assigned[p.fi] = p.fo
	}
	return assigned
}

func (t *Tank) separateFish(hunters []int) {
	hunting := make([]bool, len(t.Fishes))
	for i, fo := range hunters {
		if i < len(hunting) && fo >= 0 {
			hunting[i] = true
		}
	}
	for range 3 {
		for i := 0; i < len(t.Fishes); i++ {
			for j := i + 1; j < len(t.Fishes); j++ {
				a, b := &t.Fishes[i], &t.Fishes[j]
				if a.Emerging || b.Emerging || a.State == LifeDying || b.State == LifeDying || a.busy() && a.State != LifeSwim || b.busy() && b.State != LifeSwim {
					continue
				}
				aL, aT := a.X-fishGap/2, a.Y-fishGap/2
				aR, aB := a.X+a.W()+fishGap/2, a.Y+a.H()+fishGap/2
				bL, bT := b.X-fishGap/2, b.Y-fishGap/2
				bR, bB := b.X+b.W()+fishGap/2, b.Y+b.H()+fishGap/2
				ox := math.Min(aR, bR) - math.Max(aL, bL)
				oy := math.Min(aB, bB) - math.Max(aT, bT)
				if ox <= 0 || oy <= 0 {
					continue
				}
				var dx, dy float64
				if ox < oy {
					if a.X+a.W()/2 <= b.X+b.W()/2 {
						dx = -ox
					} else {
						dx = ox
					}
				} else {
					if a.Y+a.H()/2 <= b.Y+b.H()/2 {
						dy = -oy
					} else {
						dy = oy
					}
				}
				wa, wb := 0.5, 0.5
				if hunting[i] && !hunting[j] {
					wa, wb = 0.2, 0.8
				} else if hunting[j] && !hunting[i] {
					wa, wb = 0.8, 0.2
				}
				a.X += dx * wa
				a.Y += dy * wa
				b.X -= dx * wb
				b.Y -= dy * wb
				t.clampFish(i)
				t.clampFish(j)
			}
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

// FeedFromAbove drops food into the plate from a click on the strip
// above it. The food starts at the top of the interior, at the click's
// horizontal position. Clicks on the plate itself must not call this
// successfully: ContainsFeed rejects them.
func (t *Tank) FeedFromAbove(x, y float64) bool {
	if len(t.Foods) >= MaxFood {
		return false
	}
	if !t.Panel.ContainsFeed(x, y) {
		return false
	}
	return t.placeFood(x-4, t.Panel.IY)
}

// PlaceFood puts food at (x, y) inside the plate. It is for tests and
// the simulator; the desktop click path uses FeedFromAbove.
func (t *Tank) PlaceFood(x, y float64) bool {
	if len(t.Foods) >= MaxFood {
		return false
	}
	if !t.Panel.ContainsInterior(x, y) {
		return false
	}
	return t.placeFood(x-4, y-4)
}

func (t *Tank) placeFood(x, y float64) bool {
	x, y = t.Panel.ClampInside(x, y, 8, 8)
	// A long press on one X would otherwise stack every pellet on the
	// same pixel. Nudge later drops toward the middle of the plate.
	mid := t.Panel.IX + t.Panel.IW/2
	for range 6 {
		stacked := false
		for _, f := range t.Foods {
			if math.Abs(f.X-x) < 7 && math.Abs(f.Y-y) < 16 {
				stacked = true
				break
			}
		}
		if !stacked {
			break
		}
		dir := 1.0
		if x > mid {
			dir = -1
		}
		x, y = t.Panel.ClampInside(x+dir*10, y, 8, 8)
	}
	t.nextID++
	glyph := FoodGlyph
	if t.rng.Intn(3) == 0 {
		glyph = FoodStar
	}
	food := Food{ID: t.nextID, X: x, Y: y, Glyph: glyph, Life: foodLife}
	t.Foods = append(t.Foods, food)
	t.recordFood(FoodRecord{
		FoodID: food.ID, Reason: "spawned", Time: t.Time,
		X: food.X, Y: food.Y, Life: food.Life,
	})
	return true
}

// SetHover records glass contact in design coordinates. Each fish keeps its
// own stable curious/shy reaction; lifecycle animations and eating take priority.
// It never drops food.
func (t *Tank) SetHover(x, y float64) {
	t.Hovering = true
	t.HoverX, t.HoverY = x, y
}

// ClearHover ends the pointer interaction and returns idle fish to wandering.
func (t *Tank) ClearHover() {
	if !t.Hovering {
		return
	}
	t.Hovering = false
	for i := range t.Fishes {
		if !t.Fishes[i].busy() && !t.Fishes[i].Emerging {
			t.pickWanderTarget(i)
		}
	}
}

func (t *Tank) eatFood(i, foodIdx int) {
	if i < 0 || i >= len(t.Fishes) || foodIdx < 0 || foodIdx >= len(t.Foods) {
		return
	}
	f := &t.Fishes[i]
	food := t.Foods[foodIdx]
	t.Foods = append(t.Foods[:foodIdx], t.Foods[foodIdx+1:]...)
	t.recordFood(FoodRecord{
		FoodID: food.ID, Reason: "consumed", Time: t.Time,
		X: food.X, Y: food.Y, FishID: f.ID, Age: food.Age, Life: food.Life,
	})
	t.bites++
	text := biteWords[(t.bites-1)%len(biteWords)]
	mx, my := f.Mouth()
	t.nextID++
	t.Chomps = append(t.Chomps, Chomp{
		ID:   t.nextID,
		X:    mx,
		Y:    my - f.H()/2 - 6,
		Text: text,
		Life: chompLife,
	})
	t.growFromEat(i)
}

func (t *Tank) pickWanderTarget(i int) {
	if i < 0 || i >= len(t.Fishes) {
		return
	}
	f := &t.Fishes[i]
	bestX, bestY, bestScore := 0.0, 0.0, -1.0
	for n := 0; n < 8; n++ {
		x, y := t.Panel.RandomPoint(t.rng.Float64(), t.rng.Float64(), 10)
		fw, fh := f.W(), f.H()
		fx, fy := t.Panel.ClampInside(x-fw/2, y-fh/2, fw, fh)
		score := 1e9
		for j, o := range t.Fishes {
			if j == i {
				continue
			}
			if boxesOverlap(fx, fy, fw, fh, o.X, o.Y, o.W(), o.H(), fishGap) {
				score = -1
				break
			}
			d := boxSeparationSized(fx, fy, fw, fh, o.X, o.Y, o.W(), o.H())
			if d < score {
				score = d
			}
		}
		if score > bestScore {
			bestScore, bestX, bestY = score, x, y
		}
	}
	f.TargetX, f.TargetY = bestX, bestY
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
	w, h := t.Fishes[i].W(), t.Fishes[i].H()
	t.Fishes[i].X, t.Fishes[i].Y = t.Panel.ClampInside(t.Fishes[i].X, t.Fishes[i].Y, w, h)
}

func boxesOverlap(ax, ay, aw, ah, bx, by, bw, bh, gap float64) bool {
	return ax < bx+bw+gap && bx < ax+aw+gap && ay < by+bh+gap && by < ay+ah+gap
}

func fishBoxesOverlap(ax, ay, bx, by, gap float64) bool {
	return boxesOverlap(ax, ay, fishW, fishH, bx, by, fishW, fishH, gap)
}

func boxSeparation(ax, ay, bx, by float64) float64 {
	return boxSeparationSized(ax, ay, fishW, fishH, bx, by, fishW, fishH)
}

func boxSeparationSized(ax, ay, aw, ah, bx, by, bw, bh float64) float64 {
	ox := math.Min(ax+aw, bx+bw) - math.Max(ax, bx)
	oy := math.Min(ay+ah, by+bh) - math.Max(ay, by)
	if ox > 0 && oy > 0 {
		return -math.Min(ox, oy)
	}
	dx, dy := 0.0, 0.0
	if ox <= 0 {
		if ax+aw < bx {
			dx = bx - (ax + aw)
		} else {
			dx = ax - (bx + bw)
		}
	}
	if oy <= 0 {
		if ay+ah < by {
			dy = by - (ay + ah)
		} else {
			dy = ay - (by + bh)
		}
	}
	return math.Hypot(dx, dy)
}

func fishesOverlap(a, b Fish, eps float64) bool {
	return a.X < b.X+b.W()-eps && b.X < a.X+a.W()-eps && a.Y < b.Y+b.H()-eps && b.Y < a.Y+a.H()-eps
}

// Sprite returns the facing-appropriate characters for fish i.
func (t *Tank) Sprite(i int) string {
	if i < 0 || i >= len(t.Fishes) {
		return SpriteRight
	}
	return t.Fishes[i].Sprite()
}

// Mouth is the bite point on the leading glyph, not the body center.
// Eating from the body center left edge food more than eatRadius
// from a clamped 108-wide fish (the corner-pile bug).
func (f Fish) Mouth() (x, y float64) {
	y = f.Y + f.H()/2
	if f.Facing < 0 {
		return f.X, y
	}
	return f.X + f.W(), y
}

// Center is the middle of the food glyph's drawing box.
func (f Food) Center() (x, y float64) { return f.X + 4, f.Y + 4 }

// Fade is 1 until the last foodFade seconds of Life, then eases to 0.
func (f Food) Fade() float64 {
	left := f.Life - f.Age
	if left >= foodFade {
		return 1
	}
	if left <= 0 {
		return 0
	}
	return left / foodFade
}

// FishSize is the drawing box of one fish, in DIPs.
func FishSize() (w, h float64) { return fishW, fishH }

// ChompLife is how long a bite label stays visible, in seconds.
func ChompLife() float64 { return chompLife }

// FoodLife is how long an uneaten pellet stays before fading out.
func FoodLife() float64 { return foodLife }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
