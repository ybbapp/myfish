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

func TestFeedZoneHoverVersusGlassHover(t *testing.T) {
	now := t0()
	tk := New(420, 520, 3)
	tk.Foods = nil
	p := tk.Panel
	feedX, feedY := p.FX+p.FW*0.6, p.FY+p.FH/2
	glassX, glassY := p.IX+p.IW/2, p.IY+p.IH/3

	t.Run("glass hover does not feed", func(t *testing.T) {
		f := NewFeeder()
		tk.Foods = nil
		tk.SetHover(glassX, glassY)
		f.Hover(now, glassX, glassY, false)
		for i := 0; i < 10; i++ {
			at := now.Add(time.Duration(i+1) * HoverDwell)
			if f.HoverTick(at, false) {
				t.Fatal("glass hover must not request a feed pulse")
			}
			tk.Step(0.05)
		}
		if f.Holding() || f.Dropped() != 0 || f.HoverDropped() != 0 || f.FeedHovering() || len(tk.Foods) != 0 {
			t.Fatalf("glass hover must not feed: holding=%v dropped=%d hoverDropped=%d hovering=%v foods=%d",
				f.Holding(), f.Dropped(), f.HoverDropped(), f.FeedHovering(), len(tk.Foods))
		}
		if !tk.Hovering {
			t.Fatal("glass hover should still be approach-only")
		}
		tk.ClearHover()
	})

	t.Run("pass-through before dwell does not drop", func(t *testing.T) {
		f := NewFeeder()
		tk.Foods = nil
		f.Hover(now, feedX, feedY, true)
		if f.HoverTick(now.Add(HoverDwell-time.Millisecond), true) {
			t.Fatal("a pass-through under the dwell must not drop")
		}
		f.Hover(now.Add(HoverDwell-time.Millisecond), glassX, glassY, false)
		if f.HoverTick(now.Add(HoverDwell+HoverInterval), false) || f.FeedHovering() || len(tk.Foods) != 0 {
			t.Fatalf("leaving before the dwell must stop with no food: hovering=%v foods=%d", f.FeedHovering(), len(tk.Foods))
		}
	})

	t.Run("feed hover drops after dwell at pointer X", func(t *testing.T) {
		f := NewFeeder()
		tk.Foods = nil
		f.Hover(now, feedX, feedY, true)
		if f.HoverTick(now.Add(HoverDwell-time.Millisecond), true) {
			t.Fatal("must not drop before the dwell")
		}
		if !f.HoverTick(now.Add(HoverDwell), true) {
			t.Fatal("the first hover pulse should fire at the dwell")
		}
		if !tk.FeedFromAbove(f.X, f.Y) {
			t.Fatal("FeedFromAbove rejected a FEED-zone hover pulse")
		}
		if len(tk.Foods) != 1 {
			t.Fatalf("got %d foods after the first hover pulse", len(tk.Foods))
		}
		if abs(tk.Foods[0].X+4-feedX) > 10 {
			t.Fatalf("food x=%v, pointer x=%v", tk.Foods[0].X, feedX)
		}
		if f.HoverTick(now.Add(HoverDwell+HoverInterval-time.Millisecond), true) {
			t.Fatal("must not drop again before the hover interval")
		}
		if !f.HoverTick(now.Add(HoverDwell+HoverInterval), true) {
			t.Fatal("the second hover pulse should fire after the interval")
		}
		if f.HoverDropped() != 2 {
			t.Fatalf("got %d hover pulses, want 2", f.HoverDropped())
		}
	})

	t.Run("leaving the feed strip stops pulses", func(t *testing.T) {
		f := NewFeeder()
		f.Hover(now, feedX, feedY, true)
		if !f.HoverTick(now.Add(HoverDwell), true) {
			t.Fatal("expected the dwell pulse")
		}
		f.Hover(now.Add(HoverDwell+10*time.Millisecond), glassX, glassY, false)
		if f.FeedHovering() {
			t.Fatal("leaving the strip should stop hover feeding immediately")
		}
		if f.HoverTick(now.Add(HoverDwell+HoverInterval), false) {
			t.Fatal("a pulse must not fire after leaving the strip")
		}
	})

	t.Run("re-entering restarts the dwell", func(t *testing.T) {
		f := NewFeeder()
		f.Hover(now, feedX, feedY, true)
		if !f.HoverTick(now.Add(HoverDwell), true) {
			t.Fatal("expected the first dwell pulse")
		}
		f.Hover(now.Add(HoverDwell+10*time.Millisecond), glassX, glassY, false)
		reenter := now.Add(HoverDwell + 20*time.Millisecond)
		f.Hover(reenter, feedX, feedY, true)
		if f.HoverTick(reenter.Add(HoverDwell-time.Millisecond), true) {
			t.Fatal("re-entering must restart the dwell")
		}
		if !f.HoverTick(reenter.Add(HoverDwell), true) {
			t.Fatal("the restarted dwell should pulse once it elapses")
		}
		if f.HoverDropped() != 1 {
			t.Fatalf("a new hover should reset the pulse count, got %d", f.HoverDropped())
		}
	})

	t.Run("click still bursts and pre-empts hover", func(t *testing.T) {
		f := NewFeeder()
		tk.Foods = nil
		f.Hover(now, feedX, feedY, true)
		if !f.HoverTick(now.Add(HoverDwell), true) {
			t.Fatal("expected a hover pulse before the click")
		}
		if !tk.FeedFromAbove(f.X, f.Y) {
			t.Fatal("hover pulse was rejected")
		}
		clickAt := now.Add(HoverDwell + 20*time.Millisecond)
		f.Down(clickAt, feedX, feedY)
		if f.FeedHovering() {
			t.Fatal("a press should pause hover feeding")
		}
		if f.HoverTick(clickAt.Add(HoverInterval), true) {
			t.Fatal("hover must not pulse while a press is active")
		}
		if !f.Up(clickAt.Add(50*time.Millisecond), true) {
			t.Fatal("short press should still drop once on release")
		}
		if !tk.FeedFromAbove(f.X, f.Y) {
			t.Fatal("click burst was rejected")
		}
		if len(tk.Foods) != 2 {
			t.Fatalf("want hover+click foods, got %d", len(tk.Foods))
		}
		if f.Dropped() != 1 {
			t.Fatalf("short press dropped %d, want 1", f.Dropped())
		}
	})
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
