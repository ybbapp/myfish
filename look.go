package main

import (
	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

// MyGo ui.RGBA takes alpha in 0–1 (values >= 1 become fully opaque).
// An earlier pass passed 0–255, so the plate fill rendered as a solid
// pale plaque and hid the desktop. The default remains a clear front
// face; the user can add a light or dark tint from the glass settings.
var (
	colorStand        = ui.RGB(164, 222, 214)
	colorStandDark    = ui.RGB(132, 196, 188)
	colorStandLight   = ui.RGB(190, 236, 230)
	colorStandShadow  = ui.RGBA(70, 120, 116, 0.22)
	colorFish         = ui.RGB(28, 196, 220)
	colorFishGlow     = ui.RGBA(170, 240, 255, 0.45)
	colorFishAlt      = ui.RGB(36, 210, 180)
	colorFishAltGlow  = ui.RGBA(170, 255, 230, 0.45)
	colorFishWarm     = ui.RGB(40, 180, 230)
	colorFishWarmGlow = ui.RGBA(180, 230, 255, 0.45)
	colorBubble       = ui.RGB(56, 200, 230)
	colorBubbleGlow   = ui.RGBA(180, 240, 255, 0.40)
	colorFood         = ui.RGB(255, 176, 28)
	colorFoodEdge     = ui.RGB(230, 110, 16)
	colorPlant        = ui.RGB(46, 230, 118)
	colorPlantGlow    = ui.RGBA(160, 255, 200, 0.40)
	colorChomp        = ui.RGB(255, 214, 70)
	colorChompEdge    = ui.RGB(40, 50, 40)
	colorHandle       = ui.RGB(248, 255, 252)
	colorHandleMute   = ui.RGB(70, 130, 124)
	colorFeedHint     = ui.RGBA(90, 150, 148, 0.45)
	colorFeedHintHot  = ui.RGBA(70, 140, 138, 0.70)
	colorFeedLine     = ui.RGBA(190, 220, 228, 0.40)
	colorFeedLineHot  = ui.RGBA(210, 234, 240, 0.62)
	colorFeedChip     = ui.RGBA(164, 222, 214, 0.82)
	colorFeedChipHot  = ui.RGBA(164, 222, 214, 0.95)
	colorResizeHint   = ui.RGBA(210, 245, 255, 0.75)
)

const (
	fishFontSize   = 24
	bubbleFontSize = 18
	foodFontSize   = 20
	plantFontSize  = 22
	chompFontSize  = 16
	handleFontSize = 13
)

func drawChassis(p *ui.Painter, pan tank.Panel, feedHot bool, g GlassStyle, scale float64) {
	s := func(v float64) float32 { return float32(v * clampScale(scale)) }
	ox, oy := float32(windowPad), float32(windowPad)
	sw := func(v float64) float32 {
		out := float32(v * clampScale(scale))
		if out < 0.6 {
			return 0.6
		}
		return out
	}
	gx, gy, gw, gh, th := s(pan.GX)+ox, s(pan.GY)+oy, s(pan.GW), s(pan.GH), s(pan.Thickness)

	bx, by, bw, bh := s(pan.BX)+ox, s(pan.BY)+oy, s(pan.BW), s(pan.BH)
	legH, legW := s(pan.LegH), s(pan.LegW)
	p.Fill(ui.Rect{X: bx + s(8), Y: by + bh + legH - s(3), W: bw - s(16), H: s(6)}, colorStandShadow, sw(5))

	leg := func(x float32) {
		r := ui.Rect{X: x, Y: by + bh, W: legW, H: legH}
		p.Fill(r, colorStand, sw(2))
		p.Fill(ui.Rect{X: x + legW - s(4), Y: by + bh, W: s(4), H: legH}, colorStandDark, sw(1))
		p.Stroke(r, colorStandDark, sw(2), sw(1))
	}
	leg(s(pan.Leg1X) + ox)
	leg(s(pan.Leg2X) + ox)

	plat := ui.Rect{X: bx, Y: by, W: bw, H: bh}
	p.Fill(plat, colorStand, sw(4))
	hiH := bh * 0.28
	if hiH < s(5) {
		hiH = s(5)
	}
	p.Fill(ui.Rect{X: bx, Y: by, W: bw, H: hiH}, colorStandLight, sw(4))
	p.Fill(ui.Rect{X: bx + bw - s(5), Y: by + hiH*0.6, W: s(5), H: bh - hiH*0.6}, colorStandDark, sw(2))
	p.Stroke(plat, colorStandDark, sw(4), sw(1))
	slotW := gw + th*0.4
	slotX := gx - (slotW-gw)/2
	p.Fill(ui.Rect{X: slotX, Y: by + s(2), W: slotW, H: s(3)}, colorStandDark, sw(1))

	// A visible FEED chip in the strip above the plate. The previous
	// hairline was easy to miss, especially at 35%; testers clicked the
	// glass instead, and no food was dropped.
	line := colorFeedLine
	chip := colorFeedChip
	if feedHot {
		line = colorFeedLineHot
		chip = colorFeedChipHot
	}
	p.Stroke(ui.Rect{X: s(pan.FX) + ox + s(8), Y: s(pan.FY+pan.FH) + oy - s(3), W: s(pan.FW) - s(16), H: s(1)}, line, sw(0.5), sw(1))
	chipW, chipH := s(76), s(22)
	if chipW < 52 {
		chipW = 52
	}
	if chipH < 16 {
		chipH = 16
	}
	cx := s(pan.FX) + ox + s(pan.FW)/2
	cy := s(pan.FY) + oy + s(pan.FH)/2
	p.Fill(ui.Rect{X: cx - chipW/2, Y: cy - chipH/2, W: chipW, H: chipH}, chip, sw(3))
	p.Stroke(ui.Rect{X: cx - chipW/2, Y: cy - chipH/2, W: chipW, H: chipH}, colorStandDark, sw(3), sw(1))

	top := &ui.Path{}
	top.MoveTo(gx, gy)
	top.LineTo(gx+th*0.45, gy-th*0.38)
	top.LineTo(gx+gw+th, gy-th*0.38)
	top.LineTo(gx+gw, gy)
	top.Close()
	p.FillPath(top, g.Top)
	p.StrokePath(top, sw(1), g.Edge)

	side := &ui.Path{}
	side.MoveTo(gx+gw, gy)
	side.LineTo(gx+gw+th, gy-th*0.38)
	side.LineTo(gx+gw+th, gy+gh-th*0.2)
	side.LineTo(gx+gw, gy+gh)
	side.Close()
	p.FillPath(side, g.Side)
	p.StrokePath(side, sw(1), g.Edge)

	// FillA 0 keeps the plate truly clear so the desktop shows through.
	front := ui.Rect{X: gx, Y: gy, W: gw, H: gh}
	if g.FillA > 0.001 {
		p.Fill(front, g.Fill, sw(2))
	}
	p.Stroke(front, g.Edge, sw(2), sw(1.25))

	hi := &ui.Path{}
	hi.MoveTo(gx+s(2.5), gy+gh-s(4))
	hi.LineTo(gx+s(2.5), gy+s(2.5))
	hi.LineTo(gx+gw-s(4), gy+s(2.5))
	p.StrokePath(hi.Cap(ui.ButtCap), sw(1.35), g.Hi)
}

func fishColors(tint int) (fill, glow ui.Color) {
	switch tint % 3 {
	case 1:
		return colorFishAlt, colorFishAltGlow
	case 2:
		return colorFishWarm, colorFishWarmGlow
	default:
		return colorFish, colorFishGlow
	}
}

func withAlpha(c ui.Color, a float64) ui.Color {
	if a >= 1 {
		return c
	}
	if a <= 0 {
		c.A = 0
		return c
	}
	c.A = uint8(float64(c.A)*a + 0.5)
	return c
}

func glowText(c *ui.Context, s, key string, size float32, fill, glow ui.Color, x, y, scale float64) {
	el := ui.Text(c, s).Key(key).
		Font("monospace").FontSize(tankFontSize(float64(size), scale)).
		TextColor(fill).TextStroke(glowStroke(scale), glow).
		PassThrough().NoWrap().
		Absolute().Left(float32(dipX(x, scale))).Top(float32(dipY(y, scale)))
	if tankTextBold(float64(size), scale) {
		el.Bold()
	}
}

func drawResizeHint(p *ui.Painter, fx, fy, fw, fh, scale float64, edge resizeEdge) {
	if edge == resizeNone {
		return
	}
	_ = fx
	_ = fy
	_ = fw
	_ = fh
	g := visibleGlassRect(scale)
	st := visibleStandRect(scale)
	thick := float32(2)
	if thick < 1.2 {
		thick = 1.2
	}
	stroke := func(r rectWH, horiz bool, atEnd bool) {
		if horiz {
			y := r.Y
			if atEnd {
				y = r.Y + r.H
			}
			p.Stroke(ui.Rect{X: float32(r.X), Y: float32(y), W: float32(r.W), H: 0}, colorResizeHint, 0, thick)
			return
		}
		x := r.X
		if atEnd {
			x = r.X + r.W
		}
		p.Stroke(ui.Rect{X: float32(x), Y: float32(r.Y), W: 0, H: float32(r.H)}, colorResizeHint, 0, thick)
	}
	if edge.hasN() {
		stroke(g, true, false)
	}
	if edge.hasS() {
		stroke(g, true, true)
		stroke(st, true, true)
	}
	if edge.hasW() {
		stroke(g, false, false)
		stroke(st, false, false)
	}
	if edge.hasE() {
		stroke(g, false, true)
		stroke(st, false, true)
	}
}
