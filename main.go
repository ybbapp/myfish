// Desktop Fish is a small MyGo widget: a vertical transparent glass plate
// on a light cyan stand, with monospaced fish, bubbles and plants on the
// plate. It is a software prototype based on still frames from a reference
// video (the video itself was not watched in this implementation).
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

const (
	dragLabel = "DRAG"
	feedLabel = "FEED"
)

type app struct {
	win              *mygo.Window
	tray             *mygo.Tray
	tank             *tank.Tank
	last             time.Time
	pressX           float32
	pressY           float32
	ptrX             float32
	ptrY             float32
	ptrSeen          bool
	dragging         bool
	dragLocalX       float32
	dragLocalY       float32
	alwaysOnTop      bool
	clickThrough     bool
	quitting         bool
	shortcutsOn      bool
	feeder           *tank.Feeder
	settings         Settings
	settingsPath     string
	setWin           *mygo.Window
	systemDark       bool
	systemDarkOK     bool
	lastMode         GlassMode
	lastOpacity      float64
	lastHighlight    float64
	lastScale        float64
	settingsOpen     bool
	resizing         resizeEdge
	resizeStartScale float64
	resizeStartX     int
	resizeStartY     int
	resizeStartW     int
	resizeStartH     int
	resizeStartSX    int
	resizeStartSY    int
}

func newApp() *app {
	s := DefaultSettings()
	return &app{
		tank:          tank.New(designW, designH, 7),
		alwaysOnTop:   true,
		feeder:        tank.NewFeeder(),
		settings:      s,
		lastMode:      s.Mode,
		lastOpacity:   s.Opacity,
		lastHighlight: s.Highlight,
		lastScale:     s.Scale,
	}
}

func (a *app) view(c *ui.Context) {
	now := c.Now()
	scale := a.settings.ClampedScale()
	if a.last.IsZero() {
		a.last = now
	}
	dt := now.Sub(a.last).Seconds()
	a.last = now
	a.tank.Step(dt)
	c.AnimationFrame()

	if c.Shortcut(ui.Cmd, ui.KeyQ) || c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyQ) {
		a.quit()
	}
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyI) {
		a.restoreInteraction()
	}
	if c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyT) {
		a.toggleClickThrough()
	}
	if c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyG) {
		a.openSettings()
	}

	c.Root().Background(ui.Transparent)

	pan := a.tank.Panel
	fx, fy, fw, fh := pan.FrameBounds()
	setRect := a.setButtonRect(scale)
	hoverEdge := resizeNone
	if a.resizing != resizeNone {
		hoverEdge = a.resizing
	} else if a.ptrSeen {
		hoverEdge = hitResizeEdge(float64(a.ptrX), float64(a.ptrY), fx, fy, fw, fh, scale, setRect)
	}
	feedHot := false
	stage := ui.Box(c).Fill()
	stage.HitTest(func(x, y float32, r ui.Rect) bool {
		dx, dy := windowToDesign(float64(x), float64(y), scale)
		if pan.ContainsFrame(dx, dy) {
			return true
		}
		return hitResizeEdge(float64(x), float64(y), fx, fy, fw, fh, scale, setRect) != resizeNone
	})
	stage.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown {
			a.pressX, a.pressY = ev.X, ev.Y
		}
		if ev.Kind == ui.InputPointerMove || ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp {
			a.ptrX, a.ptrY = ev.X, ev.Y
			a.ptrSeen = true
		}
		// Hover tracking must not consume the event or it would swallow clicks.
		return false
	})
	ptrDX, ptrDY := windowToDesign(float64(a.ptrX), float64(a.ptrY), scale)
	if a.ptrSeen && stage.Hovered() && pan.ContainsGlass(ptrDX, ptrDY) {
		a.tank.SetHover(ptrDX, ptrDY)
	} else if a.tank.Hovering && !(stage.Hovered() && pan.ContainsGlass(ptrDX, ptrDY)) {
		a.tank.ClearHover()
	}
	if a.ptrSeen && stage.Hovered() && pan.ContainsFeed(ptrDX, ptrDY) && hoverEdge == resizeNone {
		feedHot = true
	}
	style := a.settings.Resolve(a.systemDark)
	stage.Draw(func(p *ui.Painter, r ui.Rect) {
		drawChassis(p, pan, feedHot, style, scale)
		drawResizeHint(p, fx, fy, fw, fh, scale, hoverEdge)
	})

	for i, pl := range a.tank.Plants {
		glowText(c, pl.Text, fmt.Sprintf("plant-%d", i), plantFontSize, colorPlant, colorPlantGlow, pl.X, pl.Y, scale)
	}
	for _, b := range a.tank.Bubbles {
		glowText(c, b.Glyph, fmt.Sprintf("bubble-%d", b.ID), bubbleFontSize, colorBubble, colorBubbleGlow, b.X, b.Y, scale)
	}
	for _, f := range a.tank.Foods {
		fade := f.Fade()
		glowText(c, f.Glyph, fmt.Sprintf("food-%d", f.ID), foodFontSize, withAlpha(colorFood, fade), withAlpha(colorFoodEdge, fade), f.X, f.Y, scale)
	}
	for _, fish := range a.tank.Fishes {
		glowText(c, fish.Sprite(), fmt.Sprintf("fish-%d", fish.ID), fishFontSize, colorFish, colorFishGlow, fish.X, fish.Y, scale)
	}
	for _, ch := range a.tank.Chomps {
		fade := 1 - ch.Age/ch.Life
		if fade < 0 {
			fade = 0
		}
		if fade > 0.35 {
			fade = 0.65 + 0.35*fade
		}
		ui.Text(c, ch.Text).Key(fmt.Sprintf("chomp-%d", ch.ID)).
			Font("monospace").FontSize(scaledFontSize(chompFontSize, scale)).Bold().
			TextColor(colorChomp).TextStroke(scaledFontSize(1.1, scale), colorChompEdge).
			Opacity(float32(fade)).
			Absolute().Left(float32(dipX(ch.X, scale))).Top(float32(dipY(ch.Y, scale)))
	}

	hint := colorFeedHint
	if feedHot {
		hint = colorFeedHintHot
	}
	feedHit := feedHitRect(pan, scale)
	feed := ui.Box(c).Key("feed-hit").
		Absolute().Left(float32(feedHit.X)).Top(float32(feedHit.Y)).
		Size(float32(feedHit.W), float32(feedHit.H)).
		Cursor(ui.CursorPointer).
		HitTest(func(x, y float32, r ui.Rect) bool {
			// HitTest x,y are in the same space as r (window DIPs), not
			// 0-based local coordinates. The expanded FEED box already
			// starts below the top resize band.
			return hitResizeEdge(float64(x), float64(y), fx, fy, fw, fh, scale, setRect) == resizeNone
		}).
		HandleInput(func(ev ui.InputEvent) bool {
			if a.clickThrough {
				return false
			}
			// InputEvent X/Y are local to this (possibly expanded) box.
			// Map to design space and clamp into the visual strip so
			// feeding still uses a top-of-tank coordinate.
			wx := float64(ev.X) + feedHit.X
			wy := float64(ev.Y) + feedHit.Y
			if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp || ev.Kind == ui.InputPointerMove {
				a.debugPointer("feed", ev, float32(wx), float32(wy), fmt.Sprintf("foods=%d holding=%v", len(a.tank.Foods), a.feeder.Holding()))
			}
			x, y := windowToDesign(wx, wy, scale)
			if x < pan.FX {
				x = pan.FX
			}
			if x > pan.FX+pan.FW {
				x = pan.FX + pan.FW
			}
			if y < pan.FY {
				y = pan.FY
			}
			if y > pan.FY+pan.FH {
				y = pan.FY + pan.FH
			}
			in := true
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button != 0 {
					return false
				}
				a.pressX, a.pressY = float32(wx), float32(wy)
				a.feeder.Down(now, x, y)
				return true
			case ui.InputPointerMove:
				a.feeder.Move(x, y, in)
				return a.feeder.Holding()
			case ui.InputPointerUp:
				if a.feeder.Up(now, in) {
					ok := a.tank.FeedFromAbove(a.feeder.X, a.feeder.Y)
					dbg("feed drop from click ok=%v foods=%d x=%.1f y=%.1f", ok, len(a.tank.Foods), a.feeder.X, a.feeder.Y)
				} else {
					dbg("feed up did not drop in=%v foods=%d canceled_or_hold", in, len(a.tank.Foods))
				}
				return true
			default:
				return false
			}
		})
	_ = feed
	if a.feeder.Holding() {
		in := pan.ContainsFeed(a.feeder.X, a.feeder.Y)
		if a.feeder.Tick(now, in) {
			a.tank.FeedFromAbove(a.feeder.X, a.feeder.Y)
		}
	}
	ui.Text(c, feedLabel).Key("feed-hint").
		Font("monospace").FontSize(controlFontSize(11, scale)).Bold().
		TextColor(hint).
		Absolute().Left(float32(dipX(pan.FX+pan.FW/2-16, scale))).Top(float32(dipY(pan.FY+pan.FH*0.28, scale)))

	// The stand is the drag area. Official DragWindow and SetPosition
	// were both observed not to move this frameless window on Linux;
	// the path is kept but not claimed to work.
	stand := ui.Box(c).Key("drag").
		Absolute().Left(float32(dipX(pan.BX, scale))).Top(float32(dipY(pan.BY, scale))).
		Size(float32(pan.BW*scale), float32((pan.BH+pan.LegH)*scale)).
		Cursor(ui.CursorMove).
		HitTest(func(x, y float32, r ui.Rect) bool {
			// Leave the SET control a click target of its own.
			// HitTest x,y are window DIPs (same space as r.X/r.Y).
			if setRect.contains(float64(x), float64(y)) {
				return false
			}
			// Edge/corner bands are the resize handles, not a move drag.
			if hitResizeEdge(float64(x), float64(y), fx, fy, fw, fh, scale, setRect) != resizeNone {
				return false
			}
			return true
		}).
		HandleInput(a.onDragInput)
	if stand.RightClicked() {
		a.showGlassMenu()
	}
	ui.Text(c, dragLabel).Key("drag-label").
		Font("monospace").FontSize(controlFontSize(handleFontSize, scale)).Bold().
		TextColor(colorHandleMute).
		Absolute().Left(float32(dipX(pan.BX+pan.BW/2-18, scale))).Top(float32(dipY(pan.BY+pan.BH*0.35, scale)))

	setW, setH := setRect.W, setRect.H
	setX, setY := setRect.X, setRect.Y
	rx, ry, rw, rh := resizeInputBounds(fx, fy, fw, fh, scale)
	resize := ui.Box(c).Key("resize-hit").Label("resize-hit").
		Absolute().Left(float32(rx)).Top(float32(ry)).
		Size(float32(rw), float32(rh)).
		Cursor(cursorForResize(hoverEdge)).
		HitTest(func(x, y float32, r ui.Rect) bool {
			if a.resizing != resizeNone {
				// Keep receiving moves if capture is lost after the
				// pointer leaves the 8 DIP handle.
				debugHitChange("resize", fmt.Sprintf("capturing edge=%s at %.1f,%.1f", a.resizing, x, y))
				return true
			}
			// x,y are window DIPs (same space as r.X/r.Y), not local.
			edge := hitResizeEdge(float64(x), float64(y), fx, fy, fw, fh, scale, setRect)
			debugHitChange("resize", fmt.Sprintf("edge=%s at %.1f,%.1f scale=%.3f box=%.0f,%.0f %.0fx%.0f", edge, x, y, scale, r.X, r.Y, r.W, r.H))
			return edge != resizeNone
		}).
		HandleInput(a.onResizeInput)
	_ = resize
	set := ui.Box(c).Key("glass-set").
		Absolute().Left(float32(setX)).Top(float32(setY)).
		Size(float32(setW), float32(setH)).
		Cursor(ui.CursorPointer).
		HitTest(func(x, y float32, r ui.Rect) bool { return true })
	setLabelEl := ui.Text(c, setLabel).Key("set-label").
		Font("monospace").FontSize(controlFontSize(handleFontSize, scale)).Bold().
		TextColor(colorHandleMute).
		Absolute().Left(float32(setX + max(4*scale, 4))).Top(float32(setY + max(1*scale, 2)))
	if set.Clicked() || setLabelEl.Clicked() {
		a.openSettings()
	}
}

func (a *app) onDragInput(ev ui.InputEvent) bool {
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp || (ev.Kind == ui.InputPointerMove && a.dragging) {
		a.debugPointer("stand", ev, ev.X, ev.Y, fmt.Sprintf("dragging=%v", a.dragging))
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			dbg("stand down ignored button=%d", ev.Button)
			return false
		}
		a.dragging = true
		a.dragLocalX, a.dragLocalY = ev.X, ev.Y
		return true
	case ui.InputPointerMove:
		if !a.dragging {
			return false
		}
		if a.win != nil {
			x, y := a.win.Position()
			a.win.SetPosition(x+int(ev.X-a.dragLocalX), y+int(ev.Y-a.dragLocalY))
		}
		return true
	case ui.InputPointerUp:
		a.dragging = false
		return true
	default:
		return a.dragging
	}
}

func (a *app) setButtonRect(scale float64) rectWH {
	pan := a.tank.Panel
	s := clampScale(scale)
	setW, setH := 36*s, 18*s
	if setW < minSetW {
		setW = minSetW
	}
	if setH < minSetH {
		setH = minSetH
	}
	r := rectWH{
		X: dipX(pan.BX+pan.BW, s) - setW,
		Y: dipY(pan.BY, s) + 6*s,
		W: setW,
		H: setH,
	}
	ww, wh := WindowSize(s)
	if r.X < 0 {
		r.X = 0
	}
	if r.Y < 0 {
		r.Y = 0
	}
	if r.X+r.W > float64(ww) {
		r.X = float64(ww) - r.W
	}
	if r.Y+r.H > float64(wh) {
		r.Y = float64(wh) - r.H
	}
	return r
}

func cursorForResize(edge resizeEdge) ui.Cursor {
	switch edge {
	case resizeE:
		return ui.CursorResizeE
	case resizeW:
		return ui.CursorResizeW
	case resizeN:
		return ui.CursorResizeN
	case resizeS:
		return ui.CursorResizeS
	case resizeNW, resizeSE:
		return ui.CursorResizeNWSE
	case resizeNE, resizeSW:
		return ui.CursorResizeNESW
	default:
		return ui.CursorDefault
	}
}

func (a *app) pointerScreen(winX, winY float32) (int, int) {
	if a.win == nil {
		return int(winX), int(winY)
	}
	x, y := a.win.Position()
	return x + int(winX), y + int(winY)
}

func (a *app) onResizeInput(ev ui.InputEvent) bool {
	if a.clickThrough {
		return false
	}
	scale := a.settings.ClampedScale()
	fx, fy, fw, fh := a.tank.Panel.FrameBounds()
	rx, ry, _, _ := resizeInputBounds(fx, fy, fw, fh, scale)
	winX := ev.X + float32(rx)
	winY := ev.Y + float32(ry)
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerMove || ev.Kind == ui.InputPointerUp {
		a.ptrX, a.ptrY = winX, winY
		a.ptrSeen = true
	}
	edgeNow := hitResizeEdge(float64(winX), float64(winY), fx, fy, fw, fh, scale, a.setButtonRect(scale))
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp || ev.Kind == ui.InputPointerMove {
		a.debugPointer("resize", ev, winX, winY, fmt.Sprintf("hit=%s resizing=%s start=%dx%d scale=%.4f", edgeNow, a.resizing, a.resizeStartW, a.resizeStartH, scale))
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			dbg("resize down ignored button=%d hit=%s", ev.Button, edgeNow)
			return false
		}
		edge := edgeNow
		if edge == resizeNone {
			dbg("resize down missed edge at win=%.1f,%.1f", winX, winY)
			return false
		}
		a.dragging = false
		a.feeder.Cancel()
		a.resizing = edge
		a.resizeStartScale = scale
		if a.win != nil {
			a.resizeStartX, a.resizeStartY = a.win.Position()
			a.resizeStartW, a.resizeStartH = a.win.Size()
		} else {
			a.resizeStartW, a.resizeStartH = WindowSize(scale)
		}
		a.resizeStartSX, a.resizeStartSY = a.pointerScreen(winX, winY)
		dbg("resize down accepted edge=%s startWin=%dx%d@%d,%d pointerScreen=%d,%d",
			edge, a.resizeStartW, a.resizeStartH, a.resizeStartX, a.resizeStartY, a.resizeStartSX, a.resizeStartSY)
		return true
	case ui.InputPointerMove:
		if a.resizing == resizeNone {
			return false
		}
		sx, sy := a.pointerScreen(winX, winY)
		dx := float64(sx - a.resizeStartSX)
		dy := float64(sy - a.resizeStartSY)
		a.applyAnchoredScale(scaleFromResizeDrag(a.resizing, a.resizeStartScale, a.resizeStartW, a.resizeStartH, dx, dy))
		return true
	case ui.InputPointerUp:
		if a.resizing == resizeNone {
			dbg("resize up with no active drag canceled=%v", ev.Canceled)
			return false
		}
		dbg("resize up edge=%s canceled=%v", a.resizing, ev.Canceled)
		a.finishResize()
		return true
	default:
		return a.resizing != resizeNone
	}
}

func (a *app) applyAnchoredScale(newScale float64) {
	a.settings.Scale = clampScale(newScale)
	a.lastScale = a.settings.Scale
	if a.win == nil {
		return
	}
	w, h := WindowSize(a.settings.Scale)
	x, y := positionAfterResize(a.resizing, a.resizeStartX, a.resizeStartY, a.resizeStartW, a.resizeStartH, w, h)
	a.win.SetSize(w, h)
	a.win.SetPosition(x, y)
	a.debugAfterSize("resize-drag", w, h, x, y)
	a.keepWindowOnScreen()
	a.applyHitRegion()
}

func (a *app) finishResize() {
	if a.resizing == resizeNone {
		return
	}
	edge := a.resizing
	a.resizing = resizeNone
	dbg("finishResize edge=%s scale=%.4f", edge, a.settings.Scale)
	a.applyTankScale()
	a.persistSettings()
}

func (a *app) ready() {
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayIcon(),
		IconIsTemplate: false,
		ToolTip:        "小鱼缸",
	})
	if err != nil {
		log.Println("tray:", err)
		a.tray = nil
		dbg("NewTray failed: %v", err)
	} else {
		dbg("NewTray succeeded tray!=nil")
	}

	if dir, err := mygo.App.Path(mygo.PathUserData); err != nil {
		log.Println("user data path:", err)
	} else {
		a.settingsPath = settingsPath(dir)
		if s, err := LoadSettings(a.settingsPath); err != nil {
			log.Println("load settings:", err)
		} else {
			a.settings = s
			a.lastMode, a.lastOpacity, a.lastHighlight, a.lastScale = s.Mode, s.Opacity, s.Highlight, s.Scale
		}
	}
	tw, th := WindowSize(a.settings.ClampedScale())
	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:             "小鱼缸",
		Width:             tw,
		Height:            th,
		Frameless:         true,
		Transparent:       true,
		AlwaysOnTop:       a.alwaysOnTop,
		SkipTaskbar:       true,
		DisableResize:     true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableFullScreen: true,
		DisableShadow:     true,
		Content:           ui.View(a.view),
	})
	a.debugWindowState("main window created")
	a.win.OnClose(func(e *mygo.CloseEvent) {
		dbg("main OnClose enter quitting=%v tray=%v goos=%s setWin=%s", a.quitting, a.tray != nil, runtime.GOOS, debugSetWinState(a))
		a.debugWindowState("main OnClose")
		if !a.quitting && a.tray != nil && runtime.GOOS != "linux" {
			e.PreventDefault()
			a.win.Hide()
			dbg("main OnClose hid window (non-linux tray), preventDefault")
			return
		}
		// On Linux a tray can be created even when the panel shows no
		// icon. Closing the tank must quit; otherwise Alt+F4 hides or
		// destroys the window and leaves the process (and the launching
		// terminal) running with nothing to click.
		if !a.quitting {
			dbg("main OnClose calling quit()")
			a.quit()
			dbg("main OnClose quit() returned prevented=%v", e.DefaultPrevented())
		} else {
			dbg("main OnClose skipped quit; already quitting prevented=%v", e.DefaultPrevented())
		}
	})
	a.win.OnClosed(func() {
		dbg("main OnClosed destroyed=%v", a.win != nil && a.win.IsDestroyed())
		a.debugWindowState("main OnClosed")
	})
	if dark, ok := tryThemeIsDark(); ok {
		a.systemDark, a.systemDarkOK = dark, true
	}
	mygo.Theme.OnUpdated(func() {
		if dark, ok := tryThemeIsDark(); ok {
			a.systemDark, a.systemDarkOK = dark, true
		}
	})
	a.win.OnBlur(func() {
		dbg("main OnBlur resizing=%s dragging=%v", a.resizing, a.dragging)
		a.feeder.Cancel()
		a.dragging = false
		a.finishResize()
	})
	a.applyHitRegion()
	a.syncTrayMenu()
	if a.tray != nil {
		mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
	}
	a.bindGlobalShortcuts()
}

func tryThemeIsDark() (dark bool, ok bool) {
	return mygo.Theme.IsDark(), true
}

func (a *app) applyHitRegion() {
	if a.win == nil || a.clickThrough {
		return
	}
	x, y, w, h := a.tank.Panel.FrameBounds()
	sx, sy, sw, sh := resizeInputBounds(x, y, w, h, a.settings.ClampedScale())
	a.win.SetInputRegion([]mygo.Rectangle{{
		X: sx, Y: sy, Width: sw, Height: sh,
	}})
}

func (a *app) applyTankScale() {
	a.settings.Clamp()
	a.lastScale = a.settings.Scale
	if a.win == nil {
		return
	}
	w, h := WindowSize(a.settings.Scale)
	x, y := a.win.Position()
	a.win.SetSize(w, h)
	a.debugAfterSize("applyTankScale", w, h, x, y)
	a.keepWindowOnScreen()
	a.applyHitRegion()
}

func (a *app) keepWindowOnScreen() {
	if a.win == nil {
		return
	}
	x, y := a.win.Position()
	w, h := a.win.Size()
	area := workArea{}
	d := mygo.Screen.DisplayMatching(mygo.Rectangle{X: x, Y: y, Width: w, Height: h})
	if d.WorkArea.Width > 0 && d.WorkArea.Height > 0 {
		area = workArea{X: d.WorkArea.X, Y: d.WorkArea.Y, Width: d.WorkArea.Width, Height: d.WorkArea.Height}
	} else {
		d = mygo.Screen.PrimaryDisplay()
		area = workArea{X: d.WorkArea.X, Y: d.WorkArea.Y, Width: d.WorkArea.Width, Height: d.WorkArea.Height}
	}
	nx, ny := clampPositionToWorkArea(x, y, w, h, area)
	if nx != x || ny != y {
		a.win.SetPosition(nx, ny)
	}
}

func (a *app) bindGlobalShortcuts() {
	type binding struct {
		keys, desc string
		fn         func()
	}
	for _, b := range []binding{
		{"CmdOrCtrl+Alt+Q", "退出小鱼缸", a.quit},
		{"CmdOrCtrl+Alt+I", "恢复交互", a.restoreInteraction},
		{"CmdOrCtrl+Alt+T", "切换鼠标穿透", a.toggleClickThrough},
		{settingsAccel, "玻璃设置", a.openSettings},
	} {
		err := mygo.GlobalShortcut.RegisterWith(b.keys, mygo.GlobalShortcutOptions{Description: b.desc}, b.fn)
		if err != nil {
			log.Printf("global shortcut %s: %v", b.keys, err)
			continue
		}
		a.shortcutsOn = true
	}
	if a.shortcutsOn {
		mygo.App.OnWillQuit(func(*mygo.QuitEvent) { mygo.GlobalShortcut.UnregisterAll() })
	}
}

func (a *app) syncTrayMenu() {
	if a.tray == nil {
		return
	}
	through := "开启鼠标穿透"
	if a.clickThrough {
		through = "恢复交互（关闭穿透）"
	}
	top := "关闭置顶"
	if !a.alwaysOnTop {
		top = "开启置顶"
	}
	a.tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示鱼缸", Accelerator: "CmdOrCtrl+Alt+I", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.restoreInteraction()
		}},
		{Label: through, Accelerator: "CmdOrCtrl+Alt+T", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.toggleClickThrough()
		}},
		{Label: top, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.alwaysOnTop = !a.alwaysOnTop
			a.win.SetAlwaysOnTop(a.alwaysOnTop)
			a.syncTrayMenu()
		}},
		{Label: "玻璃设置…", Accelerator: settingsAccel, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettings()
		}},
		mygo.Separator(),
		{Role: mygo.RoleQuit, Accelerator: "CmdOrCtrl+Q"},
	}))
}

func (a *app) setClickThrough(on bool) {
	a.clickThrough = on
	if a.win != nil {
		a.win.SetIgnoreMouseEvents(on)
		if !on {
			a.applyHitRegion()
		}
	}
	a.syncTrayMenu()
}

func (a *app) restoreInteraction() {
	a.setClickThrough(false)
	if a.win != nil {
		a.win.Show()
	}
}

func (a *app) toggleClickThrough() {
	if a.clickThrough {
		a.restoreInteraction()
		return
	}
	a.setClickThrough(true)
}

func (a *app) quit() {
	dbg("quit() enter setWin=%s", debugSetWinState(a))
	a.debugWindowState("quit enter")
	a.quitting = true
	if a.setWin != nil {
		dbg("quit() closing settings window destroyed=%v", a.setWin.IsDestroyed())
		a.setWin.Close()
		dbg("quit() settings Close() returned setWin=%s", debugSetWinState(a))
	}
	dbg("quit() calling App.Quit()")
	mygo.App.Quit()
	dbg("quit() App.Quit() returned")
}

func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	stand := color.NRGBA{R: 164, G: 222, B: 214, A: 255}
	glass := color.NRGBA{R: 200, G: 230, B: 240, A: 200}
	ink := color.NRGBA{R: 30, G: 180, B: 210, A: 255}
	draw.Draw(img, image.Rect(8, 4, 24, 22), image.NewUniform(glass), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(6, 20, 26, 26), image.NewUniform(stand), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(8, 26, 13, 31), image.NewUniform(stand), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(19, 26, 24, 31), image.NewUniform(stand), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(11, 11, 21, 15), image.NewUniform(ink), image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	initDebugInput()
	a := newApp()
	mygo.App.SetName("小鱼缸")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		dbg("App.OnBeforeQuit sessionEnding=%v alreadyAppQuittingFlag=%v", e.SessionEnding, a.quitting)
		a.quitting = true
	})
	mygo.App.OnWillQuit(func(e *mygo.QuitEvent) {
		dbg("App.OnWillQuit sessionEnding=%v preventable, currently prevented=%v setWin=%s", e.SessionEnding, e.DefaultPrevented(), debugSetWinState(a))
	})
	mygo.App.OnQuitCanceled(func() {
		dbg("App.OnQuitCanceled — quit sequence was canceled")
		a.debugWindowState("OnQuitCanceled")
	})
	mygo.App.WhenReady(a.ready)
	dbg("App.Run starting")
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	dbg("App.Run returned")
}
