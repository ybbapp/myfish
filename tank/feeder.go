package tank

import "time"

// Hold-to-feed timing: a short press drops once on release; holding past
// the threshold starts a rate-limited repeat. Nothing is dropped on
// pointer-down, so a later click or up cannot double-fire with it.
//
// Hover-to-feed: entering the FEED strip arms a short dwell so a
// pass-through does not drop food; after the dwell, pulses continue at
// HoverInterval until the pointer leaves. A press takes priority over
// hover pulses. Glass hover is not a feeder input.
const (
	HoldThreshold = 300 * time.Millisecond
	RepeatEvery   = 200 * time.Millisecond
	HoverDwell    = 200 * time.Millisecond
	HoverInterval = 300 * time.Millisecond
)

// Feeder turns pointer-down/move/up, hover, and a clock into feed pulses.
// All times are supplied by the caller so tests can be deterministic.
type Feeder struct {
	Threshold  time.Duration
	Interval   time.Duration
	Dwell      time.Duration
	HoverEvery time.Duration

	holding bool
	started time.Time
	last    time.Time
	dropped int
	X, Y    float64

	hovering     bool
	hoverStarted time.Time
	hoverLast    time.Time
	hoverDropped int
}

// NewFeeder returns a feeder with the default threshold, interval, and hover dwell.
func NewFeeder() *Feeder {
	return &Feeder{
		Threshold:  HoldThreshold,
		Interval:   RepeatEvery,
		Dwell:      HoverDwell,
		HoverEvery: HoverInterval,
	}
}

// Holding reports whether a press is still active.
func (f *Feeder) Holding() bool { return f.holding }

// FeedHovering reports whether the pointer is in a hover-feed dwell or pulse.
func (f *Feeder) FeedHovering() bool { return f.hovering }

// Dropped reports how many pulses the current press has already emitted.
func (f *Feeder) Dropped() int { return f.dropped }

// HoverDropped reports how many pulses the current feed-strip hover has emitted.
func (f *Feeder) HoverDropped() int { return f.hoverDropped }

// Down starts a press. It never itself requests a drop, and it pauses hover feeding
// so a click burst cannot double-fire with a hover pulse.
func (f *Feeder) Down(now time.Time, x, y float64) {
	f.stopHover()
	f.holding = true
	f.started = now
	f.last = time.Time{}
	f.dropped = 0
	f.X, f.Y = x, y
}

// Move updates the drop position, or cancels if the pointer left the strip.
func (f *Feeder) Move(x, y float64, inZone bool) {
	if !f.holding {
		return
	}
	if !inZone {
		f.Cancel()
		return
	}
	f.X, f.Y = x, y
}

// Tick requests a drop once the hold has crossed the threshold, then every
// Interval. The pointer must still be in the feed strip.
func (f *Feeder) Tick(now time.Time, inZone bool) bool {
	if !f.holding {
		return false
	}
	if !inZone {
		f.Cancel()
		return false
	}
	if now.Sub(f.started) < f.Threshold {
		return false
	}
	if f.dropped == 0 {
		f.dropped = 1
		f.last = now
		return true
	}
	if now.Sub(f.last) >= f.Interval {
		f.dropped++
		f.last = now
		return true
	}
	return false
}

// Up ends the press. A short press that has not already pulsed drops once
// if the pointer is still in the strip. A long press that already pulsed
// does not drop again.
func (f *Feeder) Up(now time.Time, inZone bool) bool {
	if !f.holding {
		return false
	}
	drop := false
	if inZone && f.dropped == 0 && now.Sub(f.started) < f.Threshold {
		f.dropped = 1
		drop = true
	}
	f.Cancel()
	return drop
}

// Hover updates hover-to-feed tracking. inFeed must be true only when the
// pointer is in the FEED strip, not on the glass. Leaving stops immediately.
// A press in progress ignores hover so click/hold remains the burst path.
func (f *Feeder) Hover(now time.Time, x, y float64, inFeed bool) {
	if f.holding || !inFeed {
		f.stopHover()
		return
	}
	f.X, f.Y = x, y
	if !f.hovering {
		f.hovering = true
		f.hoverStarted = now
		f.hoverLast = time.Time{}
		f.hoverDropped = 0
	}
}

// HoverTick requests a drop after the dwell, then every HoverEvery.
// The pointer must still be in the feed strip, and no press may be active.
func (f *Feeder) HoverTick(now time.Time, inFeed bool) bool {
	if f.holding {
		return false
	}
	if !f.hovering || !inFeed {
		f.stopHover()
		return false
	}
	dwell := f.Dwell
	if dwell == 0 {
		dwell = HoverDwell
	}
	if now.Sub(f.hoverStarted) < dwell {
		return false
	}
	every := f.HoverEvery
	if every == 0 {
		every = HoverInterval
	}
	if f.hoverDropped == 0 {
		f.hoverDropped = 1
		f.hoverLast = now
		return true
	}
	if now.Sub(f.hoverLast) >= every {
		f.hoverDropped++
		f.hoverLast = now
		return true
	}
	return false
}

// Cancel stops the press and any hover-feed without dropping, used for
// blur, resize, or leaving the strip.
func (f *Feeder) Cancel() {
	f.holding = false
	f.stopHover()
}

func (f *Feeder) stopHover() {
	f.hovering = false
}
