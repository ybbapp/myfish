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
	windowW   = 420
	windowH   = 520
	dragLabel = "DRAG"
	feedLabel = "FEED"
)

type app struct {
	win           *mygo.Window
	tray          *mygo.Tray
	tank          *tank.Tank
	last          time.Time
	pressX        float32
	pressY        float32
	ptrX          float32
	ptrY          float32
	ptrSeen       bool
	dragging      bool
	dragLocalX    float32
	dragLocalY    float32
	alwaysOnTop   bool
	clickThrough  bool
	quitting      bool
	shortcutsOn   bool
	feeder        *tank.Feeder
	settings      Settings
	settingsPath  string
	setWin        *mygo.Window
	systemDark    bool
	systemDarkOK  bool
	lastMode      GlassMode
	lastOpacity   float64
	lastHighlight float64
	settingsOpen  bool
}

func newApp() *app {
	s := DefaultSettings()
	return &app{
		tank:          tank.New(windowW, windowH, 7),
		alwaysOnTop:   true,
		feeder:        tank.NewFeeder(),
		settings:      s,
		lastMode:      s.Mode,
		lastOpacity:   s.Opacity,
		lastHighlight: s.Highlight,
	}
}

func (a *app) view(c *ui.Context) {
	now := c.Now()
	w, h := c.Size()
	a.tank.Resize(float64(w), float64(h))
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
	feedHot := false
	stage := ui.Box(c).Fill()
	stage.HitTest(func(x, y float32, r ui.Rect) bool {
		return pan.ContainsFrame(float64(x), float64(y))
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
	if a.ptrSeen && stage.Hovered() && pan.ContainsGlass(float64(a.ptrX), float64(a.ptrY)) {
		a.tank.SetHover(float64(a.ptrX), float64(a.ptrY))
	} else if a.tank.Hovering && !(stage.Hovered() && pan.ContainsGlass(float64(a.ptrX), float64(a.ptrY))) {
		a.tank.ClearHover()
	}
	if a.ptrSeen && stage.Hovered() && pan.ContainsFeed(float64(a.ptrX), float64(a.ptrY)) {
		feedHot = true
	}
	style := a.settings.Resolve(a.systemDark)
	stage.Draw(func(p *ui.Painter, r ui.Rect) {
		drawChassis(p, pan, feedHot, style)
	})

	for i, pl := range a.tank.Plants {
		glowText(c, pl.Text, fmt.Sprintf("plant-%d", i), plantFontSize, colorPlant, colorPlantGlow, pl.X, pl.Y)
	}
	for _, b := range a.tank.Bubbles {
		glowText(c, b.Glyph, fmt.Sprintf("bubble-%d", b.ID), bubbleFontSize, colorBubble, colorBubbleGlow, b.X, b.Y)
	}
	for _, f := range a.tank.Foods {
		fade := f.Fade()
		glowText(c, f.Glyph, fmt.Sprintf("food-%d", f.ID), foodFontSize, withAlpha(colorFood, fade), withAlpha(colorFoodEdge, fade), f.X, f.Y)
	}
	for _, fish := range a.tank.Fishes {
		glowText(c, fish.Sprite(), fmt.Sprintf("fish-%d", fish.ID), fishFontSize, colorFish, colorFishGlow, fish.X, fish.Y)
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
			Font("monospace").FontSize(chompFontSize).Bold().
			TextColor(colorChomp).TextStroke(1.1, colorChompEdge).
			Opacity(float32(fade)).
			Absolute().Left(float32(ch.X)).Top(float32(ch.Y))
	}

	hint := colorFeedHint
	if feedHot {
		hint = colorFeedHintHot
	}
	feed := ui.Box(c).Key("feed-hit").
		Absolute().Left(float32(pan.FX)).Top(float32(pan.FY)).
		Size(float32(pan.FW), float32(pan.FH)).
		Cursor(ui.CursorPointer).
		HandleInput(func(ev ui.InputEvent) bool {
			if a.clickThrough {
				return false
			}
			// InputEvent X/Y are local to this box.
			x := float64(ev.X + float32(pan.FX))
			y := float64(ev.Y + float32(pan.FY))
			in := pan.ContainsFeed(x, y)
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button != 0 {
					return false
				}
				a.pressX, a.pressY = ev.X+float32(pan.FX), ev.Y+float32(pan.FY)
				a.feeder.Down(now, x, y)
				return true
			case ui.InputPointerMove:
				a.feeder.Move(x, y, in)
				return a.feeder.Holding()
			case ui.InputPointerUp:
				if a.feeder.Up(now, in) {
					a.tank.FeedFromAbove(a.feeder.X, a.feeder.Y)
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
		Font("monospace").FontSize(11).Bold().
		TextColor(hint).
		Absolute().Left(float32(pan.FX + pan.FW/2 - 16)).Top(float32(pan.FY + pan.FH*0.28))

	// The stand is the drag area. Official DragWindow and SetPosition
	// were both observed not to move this frameless window on Linux;
	// the path is kept but not claimed to work.
	stand := ui.Box(c).Key("drag").
		Absolute().Left(float32(pan.BX)).Top(float32(pan.BY)).
		Size(float32(pan.BW), float32(pan.BH+pan.LegH)).
		Cursor(ui.CursorMove).
		HitTest(func(x, y float32, r ui.Rect) bool {
			// Leave the SET control a click target of its own.
			if x >= r.W-40 && y <= 24 {
				return false
			}
			return true
		}).
		HandleInput(a.onDragInput)
	if stand.RightClicked() {
		a.showGlassMenu()
	}
	ui.Text(c, dragLabel).Key("drag-label").
		Font("monospace").FontSize(handleFontSize).Bold().
		TextColor(colorHandleMute).
		Absolute().Left(float32(pan.BX + pan.BW/2 - 18)).Top(float32(pan.BY + pan.BH*0.35))

	setX := pan.BX + pan.BW - 40
	setY := pan.BY + 6
	set := ui.Box(c).Key("glass-set").
		Absolute().Left(float32(setX)).Top(float32(setY)).
		Size(36, 18).
		Cursor(ui.CursorPointer).
		HitTest(func(x, y float32, r ui.Rect) bool { return true })
	setLabelEl := ui.Text(c, setLabel).Key("set-label").
		Font("monospace").FontSize(handleFontSize).Bold().
		TextColor(colorHandleMute).
		Absolute().Left(float32(setX + 4)).Top(float32(setY + 1))
	if set.Clicked() || setLabelEl.Clicked() {
		a.openSettings()
	}
}

func (a *app) onDragInput(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
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
	}

	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:             "小鱼缸",
		Width:             windowW,
		Height:            windowH,
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
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if !a.quitting && a.tray != nil && runtime.GOOS != "linux" {
			e.PreventDefault()
			a.win.Hide()
		}
	})
	if dir, err := mygo.App.Path(mygo.PathUserData); err != nil {
		log.Println("user data path:", err)
	} else {
		a.settingsPath = settingsPath(dir)
		if s, err := LoadSettings(a.settingsPath); err != nil {
			log.Println("load settings:", err)
		} else {
			a.settings = s
			a.lastMode, a.lastOpacity, a.lastHighlight = s.Mode, s.Opacity, s.Highlight
		}
	}
	if dark, ok := tryThemeIsDark(); ok {
		a.systemDark, a.systemDarkOK = dark, true
	}
	mygo.Theme.OnUpdated(func() {
		if dark, ok := tryThemeIsDark(); ok {
			a.systemDark, a.systemDarkOK = dark, true
		}
	})
	a.win.OnBlur(func() { a.feeder.Cancel() })
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
	a.win.SetInputRegion([]mygo.Rectangle{{
		X: int(x), Y: int(y), Width: int(w + 0.5), Height: int(h + 0.5),
	}})
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
	a.quitting = true
	if a.setWin != nil {
		a.setWin.Close()
	}
	mygo.App.Quit()
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
	a := newApp()
	mygo.App.SetName("小鱼缸")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	mygo.App.WhenReady(a.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
