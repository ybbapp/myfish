package tank

import "math"

// Panel is the video-like chassis: a vertical rectangular glass plate
// with a slight thickness, sitting on a light cyan stand with two legs.
// Coordinates are in window DIPs.
type Panel struct {
	W, H float64

	GX, GY, GW, GH float64 // glass front face
	Thickness      float64

	IX, IY, IW, IH float64 // interior, where fish and food stay
	PlantY         float64

	BX, BY, BW, BH float64 // stand platform
	LegH, LegW     float64
	Leg1X, Leg2X   float64
}

// Layout places a vertical plate and stand inside a window of size w×h.
func Layout(w, h float64) Panel {
	if w < 200 {
		w = 200
	}
	if h < 260 {
		h = 260
	}
	p := Panel{W: w, H: h, Thickness: 9}
	marginL := 22.0
	marginR := 16.0 + p.Thickness
	marginT := 26.0
	marginB := 16.0
	legH := math.Min(54, h*0.11)
	baseH := math.Min(64, h*0.125)
	p.GX = marginL
	p.GY = marginT
	p.GW = w - marginL - marginR
	p.GH = h - marginT - marginB - baseH - legH
	if p.GW < 80 {
		p.GW = 80
	}
	if p.GH < 80 {
		p.GH = 80
	}
	inset := 12.0
	p.IX = p.GX + inset
	p.IY = p.GY + inset
	p.IW = p.GW - inset*2
	p.IH = p.GH - inset*2
	p.PlantY = p.IY + p.IH - 26

	p.BW = p.GW + p.Thickness + 14
	p.BH = baseH
	p.BX = (w - p.BW) / 2
	p.BY = p.GY + p.GH
	p.LegH = legH
	p.LegW = p.BW * 0.20
	p.Leg1X = p.BX + p.BW*0.12
	p.Leg2X = p.BX + p.BW - p.BW*0.12 - p.LegW
	return p
}

// ContainsInterior reports whether (x, y) is inside the glass content area.
func (p Panel) ContainsInterior(x, y float64) bool {
	return x >= p.IX && x <= p.IX+p.IW && y >= p.IY && y <= p.IY+p.IH
}

// ContainsGlass reports whether (x, y) is on the plate, including thickness.
func (p Panel) ContainsGlass(x, y float64) bool {
	if x >= p.GX && x <= p.GX+p.GW && y >= p.GY && y <= p.GY+p.GH {
		return true
	}
	// Right-edge thickness, and a short top lip.
	if x >= p.GX+p.GW && x <= p.GX+p.GW+p.Thickness && y >= p.GY-p.Thickness*0.35 && y <= p.GY+p.GH {
		return true
	}
	return false
}

// ContainsStand reports whether (x, y) is on the platform or a leg.
func (p Panel) ContainsStand(x, y float64) bool {
	if x >= p.BX && x <= p.BX+p.BW && y >= p.BY && y <= p.BY+p.BH {
		return true
	}
	ly := p.BY + p.BH
	if y >= ly && y <= ly+p.LegH {
		if x >= p.Leg1X && x <= p.Leg1X+p.LegW {
			return true
		}
		if x >= p.Leg2X && x <= p.Leg2X+p.LegW {
			return true
		}
	}
	return false
}

// ContainsFrame is the visible object: plate plus stand.
func (p Panel) ContainsFrame(x, y float64) bool {
	return p.ContainsGlass(x, y) || p.ContainsStand(x, y)
}

// FrameBounds is a tight rectangle around the plate and stand.
func (p Panel) FrameBounds() (x, y, w, h float64) {
	x = math.Min(p.GX, p.BX)
	y = p.GY - p.Thickness*0.35
	if y < 0 {
		y = 0
	}
	right := math.Max(p.GX+p.GW+p.Thickness, p.BX+p.BW)
	bottom := p.BY + p.BH + p.LegH
	return x, y, right - x, bottom - y
}

// ClampInside keeps a box of size bw×bh inside the interior.
func (p Panel) ClampInside(x, y, bw, bh float64) (float64, float64) {
	loX, loY := p.IX, p.IY
	hiX := math.Max(p.IX, p.IX+p.IW-bw)
	hiY := math.Max(p.IY, p.PlantY-bh)
	if hiX < loX {
		hiX = loX
	}
	if hiY < loY {
		hiY = loY
	}
	return clamp(x, loX, hiX), clamp(y, loY, hiY)
}

// RandomPoint returns a point inside the interior, above the plants.
func (p Panel) RandomPoint(u, v, pad float64) (x, y float64) {
	w := math.Max(1, p.IW-2*pad)
	h := math.Max(1, p.PlantY-p.IY-2*pad)
	return p.IX + pad + u*w, p.IY + pad + v*h
}
