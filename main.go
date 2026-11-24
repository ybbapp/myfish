// Desktop Fish is a small transparent MyGo window: monospaced mid-dark
// blue character fish, bubbles, and click-to-drop yellow food. It is a
// software prototype, not a frame-accurate remake of the reference video
// (that video was not reviewed in this implementation).
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
	windowW = 560
	windowH = 380

	fishFontSize   = 26
	bubbleFontSize = 18
	foodFontSize   = 26
	chompFontSize  = 18
	handleFontSize = 16

	dragLeft = 8
	dragTop  = 8
	dragW    = 80
	dragH    = 30

	// ASCII only: a CJK label plus a heavy stroke was hard to read
	// (halo / ghosting) on the Linux screenshot.
	dragLabel = "DRAG"
)

var (
	colorFish       = ui.RGB(16, 82, 180)
	colorFishStroke = ui.RGB(255, 255, 255)
	colorBubble     = ui.RGB(36, 118, 186)
	colorBubStroke  = ui.RGBA(255, 255, 255, 200)
	colorFood       = ui.RGB(255, 196, 16)
	colorFoodEdge   = ui.RGB(220, 104, 8)
	colorChomp      = ui.RGB(255, 210, 40)
	colorChompEdge  = ui.RGB(20, 40, 80)
	colorHandle     = ui.RGB(240, 248, 255)
	colorHandleBg   = ui.RGBA(16, 56, 120, 210)
)

type app struct {
	win          *mygo.Window
	tray         *mygo.Tray
	tank         *tank.Tank
	last         time.Time
	pressX       float32
	pressY       float32
	dragging     bool
	dragLocalX   float32
	dragLocalY   float32
	alwaysOnTop  bool
	clickThrough bool
	quitting     bool
	shortcutsOn  bool
}

func newApp() *app {
	return &app{
		tank:        tank.New(windowW, windowH, 7),
		alwaysOnTop: true,
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
	// Restore is unconditional: these keys never toggle click-through on.
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyI) {
		a.restoreInteraction()
	}
	if c.Shortcut(ui.Ctrl|ui.Alt, ui.KeyT) {
		a.toggleClickThrough()
	}

	c.Root().Background(ui.Transparent)

	stage := ui.Box(c).Fill()
	stage.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown {
			a.pressX, a.pressY = ev.X, ev.Y
		}
		return false
	})
	if stage.Clicked() && !a.clickThrough && !inDragHandle(a.pressX, a.pressY) {
		a.tank.DropFood(float64(a.pressX), float64(a.pressY))
	}

	for _, b := range a.tank.Bubbles {
		ui.Text(c, b.Glyph).Key(fmt.Sprintf("bubble-%d", b.ID)).
			Font("monospace").FontSize(bubbleFontSize).
			TextColor(colorBubble).TextStroke(1.25, colorBubStroke).
			Absolute().Left(float32(b.X)).Top(float32(b.Y))
	}
	for _, f := range a.tank.Foods {
		ui.Text(c, tank.FoodGlyph).Key(fmt.Sprintf("food-%d", f.ID)).
			Font("monospace").FontSize(foodFontSize).Bold().
			TextColor(colorFood).TextStroke(2, colorFoodEdge).
			Absolute().Left(float32(f.X - 8)).Top(float32(f.Y - 12))
	}
	for _, fish := range a.tank.Fishes {
		ui.Text(c, fish.Sprite()).Key(fmt.Sprintf("fish-%d", fish.ID)).
			Font("monospace").FontSize(fishFontSize).Bold().
			TextColor(colorFish).TextStroke(2, colorFishStroke).
			Absolute().Left(float32(fish.X)).Top(float32(fish.Y))
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
			TextColor(colorChomp).TextStroke(1.5, colorChompEdge).
			Opacity(float32(fade)).
			Absolute().Left(float32(ch.X)).Top(float32(ch.Y))
	}

	// Official DragWindow() on Linux calls gtk_window_begin_move_drag
	// only after an 8px move, for undecorated windows. A second Linux
	// run with a visible handle still did not move the window, so this
	// handle instead tracks the pointer and uses Window.SetPosition
	// (works on this X11 session; Wayland compositors may ignore it).
	ui.Box(c).Key("drag").
		Absolute().Top(dragTop).Left(dragLeft).Size(dragW, dragH).
		Radius(8).Background(colorHandleBg).
		Cursor(ui.CursorMove).
		HandleInput(a.onDragInput).
		Children(func() {
			ui.Text(c, dragLabel).Key("drag-label").
				Font("monospace").FontSize(handleFontSize).Bold().
				TextColor(colorHandle)
		})
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
		IconIsTemplate: false, // a black template vanishes on a dark panel
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
		// Hide to the tray only when a tray is likely visible. On this
		// Linux desktop NewTray can succeed with no panel plugin and no
		// StatusNotifierWatcher, so hiding would lose the window.
		if !a.quitting && a.tray != nil && runtime.GOOS != "linux" {
			e.PreventDefault()
			a.win.Hide()
		}
	})
	a.syncTrayMenu()
	if a.tray != nil {
		mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
	}
	a.bindGlobalShortcuts()
}

func (a *app) bindGlobalShortcuts() {
	type binding struct {
		keys string
		desc string
		fn   func()
	}
	for _, b := range []binding{
		{"CmdOrCtrl+Alt+Q", "退出小鱼缸", a.quit},
		{"CmdOrCtrl+Alt+I", "恢复交互", a.restoreInteraction},
		{"CmdOrCtrl+Alt+T", "切换鼠标穿透", a.toggleClickThrough},
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
		mygo.Separator(),
		{Role: mygo.RoleQuit, Accelerator: "CmdOrCtrl+Q"},
	}))
}

func (a *app) setClickThrough(on bool) {
	a.clickThrough = on
	if a.win != nil {
		a.win.SetIgnoreMouseEvents(on)
	}
	a.syncTrayMenu()
}

// restoreInteraction always turns click-through off. It must not toggle.
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
	mygo.App.Quit()
}

func inDragHandle(x, y float32) bool {
	return x >= dragLeft && x < dragLeft+dragW && y >= dragTop && y < dragTop+dragH
}

func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{R: 30, G: 120, B: 220, A: 255}
	for _, r := range []image.Rectangle{
		image.Rect(4, 14, 22, 19),
		image.Rect(22, 12, 26, 21),
		image.Rect(26, 13, 30, 16),
		image.Rect(26, 17, 30, 20),
		image.Rect(8, 12, 18, 14),
		image.Rect(8, 19, 18, 21),
		image.Rect(18, 15, 20, 17),
	} {
		draw.Draw(img, r, image.NewUniform(ink), image.Point{}, draw.Src)
	}
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
