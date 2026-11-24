package tank

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fateReport is the machine-readable evidence written when
// FOOD_TRACE_OUT is set. It uses the same Tank.Step / Feeder path as
// the desktop app, not a separate mock of eating.
type fateReport struct {
	FoodLife  float64        `json:"food_life_seconds"`
	Scenarios []scenarioFate `json:"scenarios"`
}

type scenarioFate struct {
	Name         string       `json:"name"`
	Seed         int64        `json:"seed"`
	Spawned      int          `json:"spawned"`
	Consumed     int          `json:"consumed"`
	Expired      int          `json:"expired"`
	FoodsLeft    int          `json:"foods_left"`
	HuntersAfter []int        `json:"hunters_after"`
	HoldingAfter bool         `json:"holding_after"`
	Events       []FoodRecord `json:"events"`
	Passed       bool         `json:"passed"`
	Note         string       `json:"note,omitempty"`
}

func TestFoodFateScenarios(t *testing.T) {
	report := fateReport{FoodLife: FoodLife()}
	report.Scenarios = append(report.Scenarios,
		runWallFeed(t, "left-wall", 3, Layout(420, 520).FX+4),
		runWallFeed(t, "right-wall", 11, Layout(420, 520).FX+Layout(420, 520).FW-4),
		runPlantFloorFeed(t, "plant-floor", 6),
		runLongPressRelease(t, "right-long-press-2s-release", 9),
		runLongPressSettleThenEat(t, "right-long-press-settle-then-eat", 9),
	)
	for _, s := range report.Scenarios {
		assertIndividualFates(t, s)
		if !s.Passed {
			t.Errorf("%s failed: spawned=%d consumed=%d expired=%d left=%d hunters=%v note=%s",
				s.Name, s.Spawned, s.Consumed, s.Expired, s.FoodsLeft, s.HuntersAfter, s.Note)
		}
	}
	if out := os.Getenv("FOOD_TRACE_OUT"); out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(out, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runWallFeed(t *testing.T, name string, seed int64, clickX float64) scenarioFate {
	t.Helper()
	tk := New(420, 520, seed)
	tk.Foods = nil
	tk.Hovering = false
	tk.EnableFoodTrace()
	p := tk.Panel
	if !tk.FeedFromAbove(clickX, p.FY+p.FH/2) {
		return scenarioFate{Name: name, Seed: seed, Note: "FeedFromAbove rejected", Passed: false}
	}
	if !letFoodSettle(t, tk) {
		return scenarioFate{Name: name, Seed: seed, Note: "food never reached the floor", Passed: false}
	}
	for i := 0; i < 240; i++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
		if len(tk.Foods) == 0 {
			break
		}
	}
	s := finishScenario(t, name, seed, tk, "wall drop must settle then be eaten before TTL; no leftover lock")
	if s.Passed {
		for _, ev := range s.Events {
			if ev.Reason == "consumed" && ev.Y < p.FloorY()-20 {
				s.Passed = false
				s.Note = "consumed above the floor; wanted a settled wall pellet"
				break
			}
		}
	}
	return s
}

func runPlantFloorFeed(t *testing.T, name string, seed int64) scenarioFate {
	t.Helper()
	tk := New(420, 520, seed)
	tk.Foods = nil
	tk.EnableFoodTrace()
	p := tk.Panel
	fx, fy := p.IX+p.IW*0.2, p.FloorY()-8
	if !tk.PlaceFood(fx, fy) {
		return scenarioFate{Name: name, Seed: seed, Note: "PlaceFood at plant floor rejected", Passed: false}
	}
	entered := false
	for i := 0; i < 240; i++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
		for _, f := range tk.Fishes {
			if f.Y+fishH > p.PlantY {
				entered = true
			}
		}
	}
	s := finishScenario(t, name, seed, tk, "food among plants must be eaten before TTL")
	if !entered {
		s.Passed = false
		s.Note = "no fish entered the plant band"
	}
	return s
}

func runLongPressSettleThenEat(t *testing.T, name string, seed int64) scenarioFate {
	t.Helper()
	tk := New(420, 520, seed)
	tk.Foods = nil
	tk.EnableFoodTrace()
	p := tk.Panel
	x, y := p.FX+p.FW-6, p.FY+p.FH/2
	// Fish are restored only after every pellet has reached the floor,
	// matching the reported "long press, release, wait to settle" bug.
	saved := append([]Fish(nil), tk.Fishes...)
	tk.Fishes = nil
	f := NewFeeder()
	now := t0()
	f.Down(now, x, y)
	for i := 0; i <= 40; i++ {
		at := now.Add(time.Duration(i) * 50 * time.Millisecond)
		if f.Tick(at, true) {
			tk.FeedFromAbove(f.X, f.Y)
		}
		tk.Step(0.05)
	}
	if f.Up(now.Add(2*time.Second), true) {
		tk.FeedFromAbove(f.X, f.Y)
	}
	assertReleasedFeeder(t, f, now.Add(2*time.Second))
	if f.Holding() {
		tk.Fishes = saved
		return scenarioFate{Name: name, Seed: seed, HoldingAfter: true, Note: "feeder still holding after Up", Passed: false}
	}
	if !letFoodSettle(t, tk) { // Fishes already nil; helper saves empty then restores empty
		tk.Fishes = saved
		s := snapshotScenario(name, seed, tk)
		s.HoldingAfter = false
		s.Note = "held pellets never reached the floor after release"
		s.Passed = false
		return s
	}
	tk.Fishes = saved
	for i := 0; i < 400; i++ {
		stop := false
		for _, food := range tk.Foods {
			if food.Age+0.05 >= food.Life {
				stop = true
				break
			}
		}
		if stop || len(tk.Foods) == 0 {
			break
		}
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	s := finishScenario(t, name, seed, tk, "all settled right-edge pellets from a 2s hold must be eaten before TTL")
	s.HoldingAfter = f.Holding()
	if s.Passed {
		for _, ev := range s.Events {
			if ev.Reason == "consumed" && ev.Y < p.FloorY()-20 {
				s.Passed = false
				s.Note = "a hold pellet was eaten before it settled"
				break
			}
		}
	}
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
		for _, idx := range tk.assignHunters() {
			if idx >= 0 {
				s.Passed = false
				s.Note = "hunter still locked after settled food was gone"
				s.HuntersAfter = tk.assignHunters()
				return s
			}
		}
	}
	s.HuntersAfter = tk.assignHunters()
	return s
}

func runLongPressRelease(t *testing.T, name string, seed int64) scenarioFate {
	t.Helper()
	tk := New(420, 520, seed)
	tk.Foods = nil
	tk.EnableFoodTrace()
	p := tk.Panel
	x, y := p.FX+p.FW-6, p.FY+p.FH/2
	f := NewFeeder()
	now := t0()
	f.Down(now, x, y)
	// Interleave feeder time with Tank.Step so pellets fall during the
	// hold, as they do in the running app.
	for i := 0; i <= 40; i++ { // 0..2.0s at 50ms
		at := now.Add(time.Duration(i) * 50 * time.Millisecond)
		if f.Tick(at, true) {
			tk.FeedFromAbove(f.X, f.Y)
		}
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	if f.Up(now.Add(2*time.Second), true) {
		tk.FeedFromAbove(f.X, f.Y)
	}
	assertReleasedFeeder(t, f, now.Add(2*time.Second))
	if f.Holding() {
		return scenarioFate{Name: name, Seed: seed, HoldingAfter: true, Note: "feeder still holding after Up", Passed: false}
	}
	if len(tk.Foods) > 0 {
		if !letFoodSettle(t, tk) {
			s := snapshotScenario(name, seed, tk)
			s.Note = "held pellets never reached the floor after release"
			s.Passed = false
			s.HoldingAfter = f.Holding()
			return s
		}
	}
	for i := 0; i < 400; i++ {
		stop := false
		for _, food := range tk.Foods {
			if food.Age+0.05 >= food.Life {
				stop = true
				break
			}
		}
		if stop || len(tk.Foods) == 0 {
			break
		}
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	s := finishScenario(t, name, seed, tk, "every pellet from a 2s hold must be eaten, not expired; feeder released")
	s.HoldingAfter = f.Holding()
	// After food is gone, hunters must stay clear — no permanent lock.
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
		h := tk.assignHunters()
		for _, idx := range h {
			if idx >= 0 {
				s.Passed = false
				s.Note = "hunter still locked after food was gone"
				s.HuntersAfter = append([]int(nil), h...)
				return s
			}
		}
	}
	s.HuntersAfter = tk.assignHunters()
	return s
}

// letFoodSettle advances falling pellets to the floor without fish
// (so mid-air chases cannot hide a corner-at-rest bug), then restores
// the same fish. Motion still uses Tank.Step.
func letFoodSettle(t *testing.T, tk *Tank) bool {
	t.Helper()
	saved := append([]Fish(nil), tk.Fishes...)
	tk.Fishes = nil
	ok := false
	floor := tk.Panel.FloorY() - 8
	for i := 0; i < 200; i++ { // 10s of fall, well under 16s TTL
		if len(tk.Foods) == 0 {
			break
		}
		allDown := true
		for _, f := range tk.Foods {
			if f.Y < floor-0.5 {
				allDown = false
				break
			}
		}
		if allDown {
			ok = true
			break
		}
		tk.Step(0.05)
	}
	tk.Fishes = saved
	return ok && len(tk.Foods) > 0
}

func snapshotScenario(name string, seed int64, tk *Tank) scenarioFate {
	tr := tk.FoodTrace()
	s := scenarioFate{
		Name:         name,
		Seed:         seed,
		FoodsLeft:    len(tk.Foods),
		HuntersAfter: append([]int(nil), tk.assignHunters()...),
	}
	if tr != nil {
		s.Spawned, s.Consumed, s.Expired = tr.Spawned, tr.Consumed, tr.Expired
		s.Events = append([]FoodRecord(nil), tr.Events...)
	}
	return s
}

func finishScenario(t *testing.T, name string, seed int64, tk *Tank, expect string) scenarioFate {
	t.Helper()
	tr := tk.FoodTrace()
	s := scenarioFate{
		Name:         name,
		Seed:         seed,
		FoodsLeft:    len(tk.Foods),
		HuntersAfter: append([]int(nil), tk.assignHunters()...),
		Note:         expect,
	}
	if tr != nil {
		s.Spawned, s.Consumed, s.Expired = tr.Spawned, tr.Consumed, tr.Expired
		s.Events = append([]FoodRecord(nil), tr.Events...)
	}
	if s.Spawned == 0 {
		s.Passed = false
		s.Note = "no food spawned"
		return s
	}
	if s.Consumed+s.Expired != s.Spawned || s.FoodsLeft != 0 {
		s.Passed = false
		s.Note = "not every pellet was accounted for as consumed or expired"
		return s
	}
	if s.Consumed == 0 || s.Expired != 0 {
		s.Passed = false
		s.Note = "need at least one eat and zero expires; food must not be cleared only by TTL"
		return s
	}
	if s.Consumed != s.Spawned {
		s.Passed = false
		s.Note = "some pellets expired instead of being eaten"
		return s
	}
	for _, ev := range s.Events {
		switch ev.Reason {
		case "consumed":
			if ev.Time >= FoodLife() || ev.Age >= ev.Life {
				s.Passed = false
				s.Note = "a consume was recorded at or after the TTL"
				return s
			}
			if ev.FishID == 0 {
				s.Passed = false
				s.Note = "consumed event missing fish_id"
				return s
			}
		case "expired":
			s.Passed = false
			s.Note = "a pellet expired; wanted eats only in these scenes"
			return s
		}
	}
	for _, h := range s.HuntersAfter {
		if h >= 0 && s.FoodsLeft == 0 {
			s.Passed = false
			s.Note = "hunter assigned after no food remains"
			return s
		}
	}
	s.Passed = true
	s.Note = expect
	return s
}

func TestHoverClusterClearsWhenPointerLeaves(t *testing.T) {
	// A brief vertical gather on a low hover is allowed; leaving must
	// not leave a permanent lock or unending pile.
	tk := New(420, 520, 5)
	tk.Foods = nil
	p := tk.Panel
	tk.SetHover(p.IX+20, p.FloorY()-20)
	for i := 0; i < 40; i++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	tk.ClearHover()
	for i := 0; i < 80; i++ {
		tk.Step(0.05)
		assertNoBoxOverlap(t, tk, i)
	}
	if tk.Hovering {
		t.Fatal("hover still set after ClearHover")
	}
	for _, idx := range tk.assignHunters() {
		if idx >= 0 {
			t.Fatalf("hunter locked after hover ended: %v", tk.assignHunters())
		}
	}
}

func TestFoodTraceOffByDefaultAndSilent(t *testing.T) {
	tk := New(420, 520, 1)
	if tk.FoodTrace() != nil {
		t.Fatal("FoodTrace must be nil until EnableFoodTrace")
	}
	tk.Foods = nil
	if !tk.FeedFromAbove(tk.Panel.FX+tk.Panel.FW/2, tk.Panel.FY+tk.Panel.FH/2) {
		t.Fatal("feed")
	}
	tk.Step(0.05)
	if tk.FoodTrace() != nil {
		t.Fatal("feeding must not start a trace on its own")
	}
}

// Check every ID, not just aggregate counters, so duplicate eats or missing
// spawn records cannot masquerade as successful cleanup.
func assertIndividualFates(t *testing.T, s scenarioFate) {
	t.Helper()
	spawned := map[int]FoodRecord{}
	consumed := map[int]bool{}
	for _, ev := range s.Events {
		switch ev.Reason {
		case "spawned":
			if _, exists := spawned[ev.FoodID]; exists {
				t.Errorf("%s duplicate spawn %d", s.Name, ev.FoodID)
			}
			spawned[ev.FoodID] = ev
		case "consumed":
			spawn, exists := spawned[ev.FoodID]
			if !exists || consumed[ev.FoodID] {
				t.Errorf("%s missing spawn or duplicate consume %d", s.Name, ev.FoodID)
			}
			if ev.Time < spawn.Time || ev.Age >= FoodLife() || ev.FishID < 1 || ev.FishID > FishCount {
				t.Errorf("%s invalid consumption %+v", s.Name, ev)
			}
			consumed[ev.FoodID] = true
		default:
			t.Errorf("%s unexpected fate %+v", s.Name, ev)
		}
	}
	if len(spawned) != s.Spawned || len(consumed) != s.Consumed || len(spawned) != len(consumed) {
		t.Errorf("%s per-ID accounting mismatch: spawns=%d consumed=%d", s.Name, len(spawned), len(consumed))
	}
}

func assertReleasedFeeder(t *testing.T, f *Feeder, released time.Time) {
	t.Helper()
	for i := 1; i <= 100; i++ {
		if f.Tick(released.Add(time.Duration(i)*50*time.Millisecond), true) || f.Holding() {
			t.Fatal("feeder emitted after release or kept holding")
		}
	}
}

func TestFoodTraceRecordsIndividualExpiry(t *testing.T) {
	tk := New(420, 520, 8)
	tk.Foods = nil
	tk.Fishes = nil
	tk.EnableFoodTrace()
	p := tk.Panel
	tk.FeedFromAbove(p.FX+p.FW/2, p.FY+p.FH/2)
	for i := 0; i < 20; i++ {
		tk.Step(0.05)
	}
	tk.FeedFromAbove(p.FX+p.FW/2, p.FY+p.FH/2)
	for i := 0; i < 301; i++ {
		tk.Step(0.05)
	}
	tr := tk.FoodTrace()
	if tr.Spawned != 2 || tr.Expired != 1 || tr.Consumed != 0 || len(tk.Foods) != 1 {
		t.Fatalf("per-pellet expiry must leave younger food: %+v foods=%v", tr, tk.Foods)
	}
	ev := tr.Events[len(tr.Events)-1]
	if ev.Reason != "expired" || ev.FoodID != tr.Events[0].FoodID || ev.Age < FoodLife() || ev.FishID != 0 {
		t.Fatalf("invalid expiry record: %+v", ev)
	}
}
