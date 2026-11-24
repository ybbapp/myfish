package main

import (
	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

var (
	colorGlassFill   = ui.RGBA(230, 242, 248, 28)
	colorGlassEdge   = ui.RGBA(210, 226, 234, 210)
	colorGlassSide   = ui.RGBA(228, 238, 244, 150)
	colorGlassTop    = ui.RGBA(245, 250, 252, 170)
	colorGlassHi     = ui.RGBA(255, 255, 255, 150)
	colorStand       = ui.RGB(164, 222, 214)
	colorStandDark   = ui.RGB(132, 196, 188)
	colorStandLight  = ui.RGB(190, 236, 230)
	colorStandShadow = ui.RGBA(70, 120, 116, 50)
	colorFish        = ui.RGB(28, 196, 220)
	colorFishGlow    = ui.RGBA(170, 240, 255, 140)
	colorBubble      = ui.RGB(56, 200, 230)
	colorBubbleGlow  = ui.RGBA(180, 240, 255, 130)
	colorFood        = ui.RGB(255, 176, 28)
	colorFoodEdge    = ui.RGB(230, 110, 16)
	colorPlant       = ui.RGB(46, 230, 118)
	colorPlantGlow   = ui.RGBA(160, 255, 200, 120)
	colorChomp       = ui.RGB(255, 214, 70)
	colorChompEdge   = ui.RGB(40, 50, 40)
	colorHandle      = ui.RGB(248, 255, 252)
	colorHandleMute  = ui.RGB(70, 130, 124)
)

const (
	fishFontSize   = 24
	bubbleFontSize = 18
	foodFontSize   = 20
	plantFontSize  = 22
	chompFontSize  = 16
	handleFontSize = 13
)

func drawChassis(p *ui.Painter, pan tank.Panel) {
	gx, gy, gw, gh, t := float32(pan.GX), float32(pan.GY), float32(pan.GW), float32(pan.GH), float32(pan.Thickness)

	// Stand first, so the plate sits on it.
	bx, by, bw, bh := float32(pan.BX), float32(pan.BY), float32(pan.BW), float32(pan.BH)
	legH, legW := float32(pan.LegH), float32(pan.LegW)
	// Soft contact shadow under the stand.
	p.Fill(ui.Rect{X: bx + 6, Y: by + bh + legH - 4, W: bw - 12, H: 8}, colorStandShadow, 6)

	leg := func(x float32) {
		r := ui.Rect{X: x, Y: by + bh, W: legW, H: legH}
		p.Fill(r, colorStand, 3)
		p.Fill(ui.Rect{X: x + legW - 5, Y: by + bh, W: 5, H: legH}, colorStandDark, 2)
		p.Stroke(r, colorStandDark, 3, 1)
	}
	leg(float32(pan.Leg1X))
	leg(float32(pan.Leg2X))

	plat := ui.Rect{X: bx, Y: by, W: bw, H: bh}
	p.Fill(plat, colorStand, 5)
	p.Fill(ui.Rect{X: bx, Y: by, W: bw, H: 10}, colorStandLight, 5)
	p.Fill(ui.Rect{X: bx + bw - 7, Y: by + 8, W: 7, H: bh - 8}, colorStandDark, 3)
	p.Stroke(plat, colorStandDark, 5, 1.15)
	// Slot where the plate meets the stand.
	slotW := gw + t*0.4
	slotX := gx - (slotW-gw)/2
	p.Fill(ui.Rect{X: slotX, Y: by + 3, W: slotW, H: 4}, colorStandDark, 1)

	// Top thickness of the plate, then the right face, then the front.
	top := &ui.Path{}
	top.MoveTo(gx, gy)
	top.LineTo(gx+t*0.45, gy-t*0.38)
	top.LineTo(gx+gw+t, gy-t*0.38)
	top.LineTo(gx+gw, gy)
	top.Close()
	p.FillPath(top, colorGlassTop)
	p.StrokePath(top, 1, colorGlassEdge)

	side := &ui.Path{}
	side.MoveTo(gx+gw, gy)
	side.LineTo(gx+gw+t, gy-t*0.38)
	side.LineTo(gx+gw+t, gy+gh-t*0.2)
	side.LineTo(gx+gw, gy+gh)
	side.Close()
	p.FillPath(side, colorGlassSide)
	p.StrokePath(side, 1, colorGlassEdge)

	front := ui.Rect{X: gx, Y: gy, W: gw, H: gh}
	p.Fill(front, colorGlassFill, 2)
	p.Stroke(front, colorGlassEdge, 2, 1.35)

	// Sharp inner highlights along the left and top — a thin glass edge,
	// not a thick blur.
	hi := &ui.Path{}
	hi.MoveTo(gx+3, gy+gh-4)
	hi.LineTo(gx+3, gy+3)
	hi.LineTo(gx+gw-4, gy+3)
	p.StrokePath(hi.Cap(ui.ButtCap), 1.5, colorGlassHi)
}

func glowText(c *ui.Context, s, key string, size float32, fill, glow ui.Color, x, y float64) {
	ui.Text(c, s).Key(key).
		Font("monospace").FontSize(size).Bold().
		TextColor(fill).TextStroke(1.15, glow).
		Absolute().Left(float32(x)).Top(float32(y))
}
