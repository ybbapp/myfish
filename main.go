// Desktop Fish is a small transparent MyGo window: a monospaced cyan fish,
// bubbles, and click-to-drop yellow food. It is a software prototype, not a
// frame-accurate remake of the reference video (that video was not reviewed
// in this implementation).
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mygo-desktop-fish/tank"
)

const (
	windowW = 560
	windowH = 380
)

var (
	colorFish   = ui.RGB(32, 204, 214)
	colorBubble = ui.RGBA(90, 210, 230, 200)
	colorFood   = ui.RGB(255, 212, 40)
	colorChomp  = ui.RGB(255, 232, 140)
	colorHandle = ui.RGBA(32, 204, 214, 140)
)

type app struct {
	win          *mygo.Window
	tray         *mygo.Tray
	tank         *tank.Tank
	last         time.Time
	pressX       float32
	pressY       float32
	alwaysOnTop  bool
	clickThrough bool
	quitting     bool
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
	// Keep swimming while the window is shown; official motion docs say
	// compute from the clock and ask for the next display frame.
	a.tank.Step(dt)
	c.AnimationFrame()

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

	// A small drag strip, separate from the feeding surface.
	ui.Text(c, ":::").Key("drag").
		Font("monospace").FontSize(14).TextColor(colorHandle).
		Absolute().Top(6).Left(10).
		DragWindow()

	for _, b := range a.tank.Bubbles {
		ui.Text(c, b.Glyph).Key(fmt.Sprintf("bubble-%d", b.ID)).
			Font("monospace").FontSize(13).TextColor(colorBubble).
			Absolute().Left(float32(b.X)).Top(float32(b.Y))
	}
	for _, f := range a.tank.Foods {
		ui.Text(c, tank.FoodGlyph).Key(fmt.Sprintf("food-%d", f.ID)).
			Font("monospace").FontSize(18).Bold().TextColor(colorFood).
			Absolute().Left(float32(f.X - 4)).Top(float32(f.Y - 8))
	}
	fish := a.tank.Fish
	ui.Text(c, a.tank.Sprite()).Key("fish").
		Font("monospace").FontSize(18).TextColor(colorFish).
		Absolute().Left(float32(fish.X)).Top(float32(fish.Y))
	for _, ch := range a.tank.Chomps {
		fade := 1 - ch.Age/ch.Life
		if fade < 0 {
			fade = 0
		}
		ui.Text(c, ch.Text).Key(fmt.Sprintf("chomp-%d", ch.ID)).
			Font("monospace").FontSize(13).TextColor(colorChomp).
			Opacity(float32(fade)).
			Absolute().Left(float32(ch.X)).Top(float32(ch.Y))
	}
}

func (a *app) ready() {
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayIcon(),
		IconIsTemplate: true,
		ToolTip:        "小鱼缸",
	})
	if err != nil {
		log.Println("tray:", err)
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
		if !a.quitting {
			e.PreventDefault()
			a.win.Hide()
		}
	})
	a.syncTrayMenu()
	if a.tray != nil {
		mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
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
		{Label: "显示鱼缸", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.setClickThrough(false)
			a.win.Show()
		}},
		{Label: through, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.setClickThrough(!a.clickThrough)
			if !a.clickThrough {
				a.win.Show()
			}
		}},
		{Label: top, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.alwaysOnTop = !a.alwaysOnTop
			a.win.SetAlwaysOnTop(a.alwaysOnTop)
			a.syncTrayMenu()
		}},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}))
}

func (a *app) setClickThrough(on bool) {
	a.clickThrough = on
	if a.win != nil {
		a.win.SetIgnoreMouseEvents(on)
	}
	a.syncTrayMenu()
}

func inDragHandle(x, y float32) bool {
	// Keep the top-left ":::" strip from also dropping food.
	return x < 56 && y < 28
}

func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{A: 255} // template: black on transparent
	// A tiny fish silhouette.
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
