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
	colorStand       = ui.RGB(164, 222, 214)
	colorStandDark   = ui.RGB(132, 196, 188)
	colorStandLight  = ui.RGB(190, 236, 230)
	colorStandShadow = ui.RGBA(70, 120, 116, 0.22)
	colorFish        = ui.RGB(28, 196, 220)
	colorFishGlow    = ui.RGBA(170, 240, 255, 0.45)
	colorBubble      = ui.RGB(56, 200, 230)
	colorBubbleGlow  = ui.RGBA(180, 240, 255, 0.40)
	colorFood        = ui.RGB(255, 176, 28)
	colorFoodEdge    = ui.RGB(230, 110, 16)
	colorPlant       = ui.RGB(46, 230, 118)
	colorPlantGlow   = ui.RGBA(160, 255, 200, 0.40)
	colorChomp       = ui.RGB(255, 214, 70)
	colorChompEdge   = ui.RGB(40, 50, 40)
	colorHandle      = ui.RGB(248, 255, 252)
	colorHandleMute  = ui.RGB(70, 130, 124)
	colorFeedHint    = ui.RGBA(90, 150, 148, 0.45)
	colorFeedHintHot = ui.RGBA(70, 140, 138, 0.70)
	colorFeedLine    = ui.RGBA(190, 220, 228, 0.40)
	colorFeedLineHot = ui.RGBA(210, 234, 240, 0.62)
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
	sw := func(v float64) float32 {
		out := float32(v * clampScale(scale))
		if out < 0.6 {
			return 0.6
		}
		return out
	}
	gx, gy, gw, gh, th := s(pan.GX), s(pan.GY), s(pan.GW), s(pan.GH), s(pan.Thickness)

	bx, by, bw, bh := s(pan.BX), s(pan.BY), s(pan.BW), s(pan.BH)
	legH, legW := s(pan.LegH), s(pan.LegW)
	p.Fill(ui.Rect{X: bx + s(8), Y: by + bh + legH - s(3), W: bw - s(16), H: s(6)}, colorStandShadow, sw(5))

	leg := func(x float32) {
		r := ui.Rect{X: x, Y: by + bh, W: legW, H: legH}
		p.Fill(r, colorStand, sw(2))
		p.Fill(ui.Rect{X: x + legW - s(4), Y: by + bh, W: s(4), H: legH}, colorStandDark, sw(1))
		p.Stroke(r, colorStandDark, sw(2), sw(1))
	}
	leg(s(pan.Leg1X))
	leg(s(pan.Leg2X))

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

	// Subtle hairline just above the plate marks the click-to-feed strip.
	line := colorFeedLine
	if feedHot {
		line = colorFeedLineHot
	}
	p.Stroke(ui.Rect{X: s(pan.FX) + s(8), Y: s(pan.FY+pan.FH) - s(3), W: s(pan.FW) - s(16), H: s(1)}, line, sw(0.5), sw(1))

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
	stroke := 1.15 * clampScale(scale)
	if stroke < 0.7 {
		stroke = 0.7
	}
	ui.Text(c, s).Key(key).
		Font("monospace").FontSize(scaledFontSize(float64(size), scale)).Bold().
		TextColor(fill).TextStroke(float32(stroke), glow).
		Absolute().Left(float32(x * clampScale(scale))).Top(float32(y * clampScale(scale)))
}
