package tank

import "math"

// GlassReaction is a lifetime temperament, not a fresh lottery on each tick.
type GlassReaction uint8

const (
	GlassCurious GlassReaction = iota
	GlassShy
)

// A seed selects which temperament leads the starting group; alternating
// assignments guarantee both reactions can be observed in a three-fish tank.
// This does not consume the simulation RNG (food/wander/lifecycle replay stays
// reproducible). Newborns use their parent and their own ID for variation.
func initialGlassReaction(seed int64, ordinal int) GlassReaction {
	x := uint64(seed)
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	return GlassReaction((x + uint64(ordinal)) & 1)
}

// glassTarget computes a restrained approach or retreat in design space.
// The target is clamped with the fish box; no invisible world-space cursor
// tracking or global input hook is needed. Busy/emerging fish never call it.
func (t *Tank) glassTarget(f *Fish) float64 {
	cx, cy := f.Center()
	dx, dy := t.HoverX-cx, t.HoverY-cy
	dist := math.Hypot(dx, dy)
	stand, speed := hoverStand, hoverSpeed
	if f.GlassReaction == GlassShy {
		stand, speed = 150, 44
		if dist < 0.01 {
			// A cursor exactly on the body must still have a stable escape
			// direction (no division by zero or per-frame randomness).
			dx, dy, dist = -f.Facing, 0.35, math.Hypot(f.Facing, 0.35)
		}
		if dist < stand {
			f.TargetX = cx - dx*(stand-dist)/dist
			f.TargetY = cy - dy*(stand-dist)/dist
		} else {
			f.TargetX, f.TargetY = cx, cy
		}
	} else if dist > stand {
		f.TargetX = cx + dx*(dist-stand)/dist
		f.TargetY = cy + dy*(dist-stand)/dist
	} else {
		f.TargetX, f.TargetY = cx, cy
	}
	x, y := t.Panel.ClampInside(f.TargetX-f.W()/2, f.TargetY-f.H()/2, f.W(), f.H())
	f.TargetX, f.TargetY = x+f.W()/2, y+f.H()/2
	return speed * f.SpeedScale()
}
