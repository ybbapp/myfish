package tank

import (
	"testing"
	"time"
)

func t0() time.Time { return time.Date(2026, 11, 24, 16, 0, 0, 0, time.UTC) }

func TestShortPressDropsOnceOnRelease(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 40, 10)
	if f.Tick(now.Add(100*time.Millisecond), true) {
		t.Fatal("a press under the threshold must not drop on tick")
	}
	if !f.Up(now.Add(120*time.Millisecond), true) {
		t.Fatal("short press should drop once on release")
	}
	if f.Holding() {
		t.Fatal("release should end the press")
	}
	if f.Up(now.Add(130*time.Millisecond), true) {
		t.Fatal("a second up must not drop again")
	}
}

func TestDownDoesNotDrop(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 40, 10)
	if f.Dropped() != 0 || f.Tick(now, true) {
		t.Fatal("pointer-down must not drop food")
	}
}

func TestHoldThresholdThenRepeat(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 10, 8)
	if f.Tick(now.Add(HoldThreshold-time.Millisecond), true) {
		t.Fatal("one millisecond before the threshold must not drop")
	}
	if !f.Tick(now.Add(HoldThreshold), true) {
		t.Fatal("the first drop should fire at the threshold")
	}
	if f.Tick(now.Add(HoldThreshold+RepeatEvery-time.Millisecond), true) {
		t.Fatal("must not drop again before the interval")
	}
	if !f.Tick(now.Add(HoldThreshold+RepeatEvery), true) {
		t.Fatal("the second drop should fire after the interval")
	}
	if !f.Tick(now.Add(HoldThreshold+2*RepeatEvery), true) {
		t.Fatal("the third drop should fire after another interval")
	}
	if f.Dropped() != 3 {
		t.Fatalf("got %d pulses, want 3", f.Dropped())
	}
}

func TestReleaseAfterHoldDoesNotDoubleDrop(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 10, 8)
	if !f.Tick(now.Add(HoldThreshold), true) {
		t.Fatal("expected the threshold drop")
	}
	if f.Up(now.Add(HoldThreshold+50*time.Millisecond), true) {
		t.Fatal("release after a hold must not drop again")
	}
}

func TestLeaveZoneStopsBeforeAndAfterThreshold(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 10, 8)
	f.Move(10, 8, false)
	if f.Holding() || f.Up(now.Add(50*time.Millisecond), false) {
		t.Fatal("leaving the strip before the threshold should cancel with no drop")
	}

	f.Down(now, 10, 8)
	if !f.Tick(now.Add(HoldThreshold), true) {
		t.Fatal("expected the threshold drop")
	}
	if f.Tick(now.Add(HoldThreshold+RepeatEvery), false) {
		t.Fatal("leaving the strip must stop repeating")
	}
	if f.Holding() {
		t.Fatal("leaving should cancel the press")
	}
}

func TestBlurOrCancelStopsRepeating(t *testing.T) {
	f := NewFeeder()
	now := t0()
	f.Down(now, 10, 8)
	if !f.Tick(now.Add(HoldThreshold), true) {
		t.Fatal("expected the threshold drop")
	}
	f.Cancel()
	if f.Tick(now.Add(HoldThreshold+RepeatEvery), true) {
		t.Fatal("cancel (blur) must stop repeating even if the pointer is still in the strip")
	}
	if f.Up(now.Add(HoldThreshold+RepeatEvery+10*time.Millisecond), true) {
		t.Fatal("up after cancel must not drop")
	}
}

func TestFeedFromAboveHonorsMaxFoodDuringRepeat(t *testing.T) {
	tk := New(420, 520, 2)
	p := tk.Panel
	f := NewFeeder()
	now := t0()
	x, y := p.FX+p.FW/2, p.FY+p.FH/2
	f.Down(now, x, y)
	drops := 0
	for i := 0; i < 20; i++ {
		at := now.Add(HoldThreshold + time.Duration(i)*RepeatEvery)
		if f.Tick(at, true) {
			if tk.FeedFromAbove(f.X, f.Y) {
				drops++
			}
		}
	}
	if drops != MaxFood {
		t.Fatalf("accepted %d drops, want cap %d (foods=%d)", drops, MaxFood, len(tk.Foods))
	}
	if tk.FeedFromAbove(x, y) {
		t.Fatal("FeedFromAbove succeeded over MaxFood")
	}
}

func TestHoverDoesNotUseTheFeeder(t *testing.T) {
	tk := New(420, 520, 3)
	tk.Foods = nil
	f := NewFeeder()
	p := tk.Panel
	tk.SetHover(p.IX+p.IW/2, p.IY+p.IH/3)
	for i := 0; i < 10; i++ {
		tk.Step(0.05)
	}
	if f.Holding() || f.Dropped() != 0 || len(tk.Foods) != 0 {
		t.Fatalf("hover must not feed: holding=%v dropped=%d foods=%d", f.Holding(), f.Dropped(), len(tk.Foods))
	}
}
