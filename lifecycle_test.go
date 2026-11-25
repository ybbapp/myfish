package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// These are close-policy tests, not a substitute for native App.Run returning
// and the OS reaping the exact PID. No native window is created here.
func TestMainCloseAlwaysRequestsApplicationQuit(t *testing.T) {
	for _, tray := range []bool{false, true} {
		for _, settings := range []string{"never-opened", "open", "hidden"} {
			t.Run(fmt.Sprintf("tray-%v/settings-%s", tray, settings), func(t *testing.T) {
				a := newApp()
				if tray {
					a.tray = &mygo.Tray{} // presence only; no native tray calls
				}
				if settings != "never-opened" {
					a.setWin = &mygo.Window{} // marker: quit must not call Close recursively
					a.settingsOpen = settings == "open"
				}
				a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
				a.settings.Scale = minScale
				a.settings.Highlight = 0.73
				requests := 0
				a.quitApp = func() {
					requests++
					if !a.quitting {
						t.Fatal("request reached App.Quit without setting quit intent first")
					}
					// MyGo's native/programmatic Quit order: OnBeforeQuit,
					// then every window OnClose, then OnWillQuit and loop stop.
					a.onBeforeQuit(&mygo.QuitEvent{})
					main, settings := &mygo.CloseEvent{}, &mygo.CloseEvent{}
					a.onMainClose(main)
					a.onSettingsClose(settings)
					if main.DefaultPrevented() || settings.DefaultPrevented() {
						t.Fatal("a window vetoed application-wide quit")
					}
				}
				e := &mygo.CloseEvent{}
				a.onMainClose(e)
				if !e.DefaultPrevented() || requests != 1 {
					t.Fatalf("main close must route once to whole-app quit: prevented=%v requests=%d", e.DefaultPrevented(), requests)
				}
				a.quit() // repeats do not reenter window destruction
				if requests != 1 {
					t.Fatal("reentrant quit request")
				}
				got, err := LoadSettings(a.settingsPath)
				if err != nil || got.Scale != minScale || got.Highlight != 0.73 {
					t.Fatalf("quit did not save settings: %+v, %v", got, err)
				}
			})
		}
	}
}

func TestNativeQuitAllowsBothWindowClosesAndSaves(t *testing.T) {
	// A native Cmd+Q / Dock Quit / menu RoleQuit calls App.prepareQuit
	// directly; it need not call a.quit first. Session ending uses the
	// same OnBeforeQuit preparation without a cancelable window close.
	for _, session := range []bool{false, true} {
		a := newApp()
		a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
		a.settings.Scale = 1.14
		a.onBeforeQuit(&mygo.QuitEvent{SessionEnding: session})
		main, settings := &mygo.CloseEvent{}, &mygo.CloseEvent{}
		a.onMainClose(main)
		a.onSettingsClose(settings)
		if !a.quitting || main.DefaultPrevented() || settings.DefaultPrevented() {
			t.Fatal("native quit got turned into hide/cancel")
		}
		got, err := LoadSettings(a.settingsPath)
		if err != nil || got.Scale != 1.14 {
			t.Fatalf("native quit did not save settings: %+v %v", got, err)
		}
	}
}

func TestCanceledQuitCanBeRetriedAndSettingsStillHide(t *testing.T) {
	a := newApp()
	requests := 0
	a.quitApp = func() { requests++ }
	a.quit()
	a.onQuitCanceled()
	if a.quitting {
		t.Fatal("cancellation left quit flag latched")
	}
	a.settingsOpen = true
	e := &mygo.CloseEvent{}
	a.onSettingsClose(e)
	if !e.DefaultPrevented() || a.quitting || a.settingsOpen {
		t.Fatal("ordinary settings close must hide only, not quit")
	}
	a.onMainClose(&mygo.CloseEvent{})
	if requests != 2 || !a.quitting {
		t.Fatal("second close did not retry application-wide quit")
	}
}

func TestQuitStopsTransientInputAndPersistsResize(t *testing.T) {
	a := newApp()
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	a.feeder.Down(time.Now(), 200, 10)
	a.dragging = true
	a.resizing = resizeE
	a.settings.Scale = 0.63
	a.quitApp = func() {}
	a.quit()
	if a.feeder.Holding() || a.feeder.FeedHovering() || a.dragging || a.resizing != resizeNone {
		t.Fatal("quit left transient interaction active")
	}
	got, err := LoadSettings(a.settingsPath)
	if err != nil || got.Scale != 0.63 {
		t.Fatalf("in-progress resize not saved at quit: %+v %v", got, err)
	}
}
