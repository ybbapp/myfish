package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
)

// lifecycle writes only process/quit milestones, never pointer positions or
// personal settings. Together with an OS PID/wait check, run-return proves the
// native loop really returned rather than only making the window disappear.
func lifecycle(event, format string, args ...any) {
	log.Printf("myfish-lifecycle pid=%d event=%s "+format, append([]any{os.Getpid(), event}, args...)...)
	dbg("lifecycle pid=%d event=%s "+format, append([]any{os.Getpid(), event}, args...)...)
}

// Closing the tank means exiting on every OS, regardless of tray availability.
// Let the application own the complete close sequence (including any hidden
// settings window). Do not recursively Close settings from a window's OnClose.
func (a *app) onMainClose(e *mygo.CloseEvent) {
	lifecycle("main-close", "quitting=%v tray=%v", a.quitting, a.tray != nil)
	if a.quitting {
		return // App.Quit/native Quit is closing all windows: allow destruction.
	}
	e.PreventDefault() // Defer this one close to the queued application-wide quit.
	a.quit()
}

// Both explicit UI actions and native menu/Cmd+Q/AppleEvent Quit share this
// preparation via OnBeforeQuit. It runs before any settings OnClose handler.
func (a *app) prepareToQuit() {
	a.quitting = true
	a.feeder.Cancel()
	a.dragging = false
	a.finishResize()
	a.persistSettings()
}

func (a *app) onBeforeQuit(e *mygo.QuitEvent) {
	lifecycle("before-quit", "sessionEnding=%v", e.SessionEnding)
	a.prepareToQuit()
}

func (a *app) onQuitCanceled() {
	// MyGo resets its own quitting flag on cancellation. Our mirror must do
	// the same or the next main-close can leave a hidden settings window
	// and no request to end the event loop.
	a.quitting = false
	lifecycle("quit-canceled", "reset application quit state; retry allowed")
}

func (a *app) quit() {
	if a.quitting {
		lifecycle("quit-repeat", "application quit already requested")
		return
	}
	lifecycle("quit-request", "save state, then request application-wide quit")
	a.prepareToQuit()
	a.quitApp()
}
