package tank

import "time"

// Hold-to-feed timing: a short press drops once on release; holding past
// the threshold starts a rate-limited repeat. Nothing is dropped on
// pointer-down, so a later click or up cannot double-fire with it.
const (
	HoldThreshold = 300 * time.Millisecond
	RepeatEvery   = 200 * time.Millisecond
)

// Feeder turns pointer-down/move/up and a clock into feed pulses.
// All times are supplied by the caller so tests can be deterministic.
type Feeder struct {
	Threshold time.Duration
	Interval  time.Duration

	holding bool
	started time.Time
	last    time.Time
	dropped int
	X, Y    float64
}

// NewFeeder returns a feeder with the default threshold and interval.
func NewFeeder() *Feeder {
	return &Feeder{Threshold: HoldThreshold, Interval: RepeatEvery}
}

// Holding reports whether a press is still active.
func (f *Feeder) Holding() bool { return f.holding }

// Dropped reports how many pulses the current press has already emitted.
func (f *Feeder) Dropped() int { return f.dropped }

// Down starts a press. It never itself requests a drop.
func (f *Feeder) Down(now time.Time, x, y float64) {
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

// Cancel stops the press without dropping, used for blur or leaving the strip.
func (f *Feeder) Cancel() {
	f.holding = false
}
