package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	settingsW     = 340
	settingsH     = 340
	setLabel      = "SET"
	settingsAccel = "CmdOrCtrl+Alt+G"
)

func (a *app) settingsView(c *ui.Context) {
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Cmd, ui.KeyW) {
		a.closeSettings()
	}
	th := c.Theme()
	root := ui.Column(c).Fill().Padding(16).Gap(12)
	root.Background(th.Background)
	root.Children(func() {
		ui.Text(c, "玻璃预设").FontSize(13).Bold()
		ui.RadioGroup(c, func() {
			ui.Radio(c, &a.settings.Mode, GlassClear, "真透明（无填充）")
			ui.Radio(c, &a.settings.Mode, GlassLight, "浅色底")
			ui.Radio(c, &a.settings.Mode, GlassDark, "深色底")
			ui.Radio(c, &a.settings.Mode, GlassAuto, "自动（跟随系统浅深主题）")
		})
		if a.settings.Mode != a.lastMode {
			a.settings.SetMode(a.settings.Mode)
			a.lastMode = a.settings.Mode
			a.persistSettings()
		}

		ui.Text(c, fmt.Sprintf("玻璃透明度  %d%%（0 为无填充）", int(a.settings.Opacity*100+0.5))).FontSize(13)
		ui.Slider(c, &a.settings.Opacity, 0, 1).Width(300)
		if a.settings.Opacity != a.lastOpacity {
			a.settings.Clamp()
			a.lastOpacity = a.settings.Opacity
			a.persistSettings()
		}

		ui.Text(c, fmt.Sprintf("边缘高光  %d%%", int(a.settings.Highlight*100+0.5))).FontSize(13)
		ui.Slider(c, &a.settings.Highlight, 0, 1).Width(300)
		if a.settings.Highlight != a.lastHighlight {
			a.settings.Clamp()
			a.lastHighlight = a.settings.Highlight
			a.persistSettings()
		}

		note := "自动只跟随系统的浅色或深色外观（mygo.Theme.IsDark），不是壁纸取色，也不会截屏。"
		if a.settings.Mode == GlassAuto && !a.systemDarkOK {
			note = "系统浅深主题此刻读不到，自动暂按浅色桌面（深色玻璃）处理。仍可用上面的手动预设。"
		}
		ui.Text(c, note).FontSize(11).TextColor(th.TextMuted)
		if ui.Button(c, "关闭").Clicked() {
			a.closeSettings()
		}
	})
}

func (a *app) openSettings() {
	a.lastMode = a.settings.Mode
	a.lastOpacity = a.settings.Opacity
	a.lastHighlight = a.settings.Highlight
	a.settingsOpen = true
	if a.win == nil {
		return
	}
	if a.setWin != nil {
		a.setWin.Show()
		a.setWin.Focus()
		return
	}
	a.setWin = mygo.NewWindow(mygo.WindowOptions{
		Title:             "玻璃设置",
		Width:             settingsW,
		Height:            settingsH,
		DisableResize:     true,
		DisableMaximize:   true,
		DisableFullScreen: true,
		AlwaysOnTop:       true,
		Content:           ui.View(a.settingsView),
	})
	a.setWin.OnClose(func(e *mygo.CloseEvent) {
		e.PreventDefault()
		a.closeSettings()
	})
}

func (a *app) closeSettings() {
	if a.setWin != nil {
		a.setWin.Hide()
	}
}

func (a *app) showGlassMenu() {
	if a.win == nil {
		return
	}
	item := func(label string, mode GlassMode) *mygo.MenuItem {
		return &mygo.MenuItem{
			Label:   label,
			Type:    mygo.MenuItemRadio,
			Checked: a.settings.Mode == mode,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.settings.SetMode(mode)
				a.persistSettings()
			},
		}
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "玻璃设置…", Accelerator: settingsAccel, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettings()
		}},
		mygo.Separator(),
		item("真透明", GlassClear),
		item("浅色底", GlassLight),
		item("深色底", GlassDark),
		item("自动（系统浅深）", GlassAuto),
	})
	menu.Popup(a.win)
}

func (a *app) persistSettings() {
	if a.settingsPath == "" {
		return
	}
	if err := SaveSettings(a.settingsPath, a.settings); err != nil {
		logSettings("save settings:", err)
	}
}

func logSettings(msg string, err error) {
	if err != nil {
		log.Println(msg, err)
	}
}
