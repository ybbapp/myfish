package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// Optional event diagnostics, off by default. Set MYFISH_DEBUG_INPUT=1
// (true/yes also work). Logs pointer, hit-test, resize, SetSize, and
// close/quit lifecycle to a file and stderr. It does not record typed
// text, credentials, or other user identity.
const debugInputEnv = "MYFISH_DEBUG_INPUT"
const debugInputLogEnv = "MYFISH_DEBUG_INPUT_LOG"

var (
	debugInput     bool
	debugLog       *log.Logger
	debugLogPath   string
	debugMu        sync.Mutex
	debugLastMove  time.Time
	debugLastHit   string
	debugLastHover string
)

func debugInputEnabled() bool {
	v := strings.TrimSpace(os.Getenv(debugInputEnv))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func initDebugInput() {
	if !debugInputEnabled() {
		return
	}
	path := strings.TrimSpace(os.Getenv(debugInputLogEnv))
	if path == "" {
		path = filepath.Join(os.TempDir(), "myfish-input-debug.log")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("debug input: create log dir: %v", err)
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("debug input: open log: %v", err)
		return
	}
	debugInput = true
	debugLogPath = path
	debugLog = log.New(io.MultiWriter(f, os.Stderr), "myfish-input ", log.LstdFlags|log.Lmicroseconds)
	dbg("start pid=%d path=%s goos=%s log=%s", os.Getpid(), os.Args[0], runtime.GOOS, path)
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if strings.HasPrefix(s.Key, "vcs.") {
				dbg("build %s=%s", s.Key, s.Value)
			}
		}
	}
	dbg("debug logging on; pointer coords, buttons, hit-tests, resize, SetSize and close/quit only")
}

func dbg(format string, args ...any) {
	if !debugInput || debugLog == nil {
		return
	}
	debugLog.Printf(format, args...)
}

func inputKindName(k ui.InputKind) string {
	switch k {
	case ui.InputPointerDown:
		return "down"
	case ui.InputPointerUp:
		return "up"
	case ui.InputPointerMove:
		return "move"
	case ui.InputScroll:
		return "scroll"
	case ui.InputKeyDown:
		return "keydown"
	case ui.InputKeyUp:
		return "keyup"
	default:
		return fmt.Sprintf("kind%d", int(k))
	}
}

func (a *app) debugWindowState(tag string) {
	if !debugInput {
		return
	}
	if a.win == nil {
		dbg("%s win=nil scale=%.4f resizing=%s dragging=%v quitting=%v setWin=%s",
			tag, a.settings.Scale, a.resizing, a.dragging, a.quitting, debugSetWinState(a))
		return
	}
	var w, h, x, y int
	visible, destroyed, resizable := false, a.win.IsDestroyed(), false
	if !destroyed {
		w, h = a.win.Size()
		x, y = a.win.Position()
		visible = a.win.IsVisible()
		resizable = a.win.IsResizable()
	}
	dbg("%s size=%dx%d pos=%d,%d visible=%v destroyed=%v resizable=%v scale=%.4f lastScale=%.4f resizing=%s dragging=%v feederHolding=%v quitting=%v setWin=%s",
		tag, w, h, x, y, visible, destroyed, resizable, a.settings.Scale, a.lastScale, a.resizing, a.dragging, a.feeder.Holding(), a.quitting, debugSetWinState(a))
}

func debugSetWinState(a *app) string {
	if !debugInput {
		return "debug-disabled"
	}
	if a.setWin == nil {
		return "nil"
	}
	destroyed := a.setWin.IsDestroyed()
	visible := false
	if !destroyed {
		visible = a.setWin.IsVisible()
	}
	return fmt.Sprintf("open destroyed=%v visible=%v settingsOpen=%v", destroyed, visible, a.settingsOpen)
}

func (a *app) debugPointer(src string, ev ui.InputEvent, winX, winY float32, extra string) {
	if !debugInput {
		return
	}
	// Hover moves are frequent; keep them when the hit changes, while a
	// button is held, or while resize/stand drag is active, else at most
	// four times a second.
	if ev.Kind == ui.InputPointerMove && ev.Button < 0 && a.resizing == resizeNone && !a.dragging && !a.feeder.Holding() {
		debugMu.Lock()
		changed := extra != debugLastHover
		ready := time.Since(debugLastMove) >= 250*time.Millisecond
		if !changed && !ready {
			debugMu.Unlock()
			return
		}
		if changed {
			debugLastHover = extra
		}
		debugLastMove = time.Now()
		debugMu.Unlock()
	}
	dbg("%s %s button=%d clicks=%d canceled=%v mods=%v local=%.1f,%.1f win=%.1f,%.1f %s",
		src, inputKindName(ev.Kind), ev.Button, ev.Clicks, ev.Canceled, ev.Mods, ev.X, ev.Y, winX, winY, extra)
}

func debugHitChange(src, desc string) {
	if !debugInput {
		return
	}
	debugMu.Lock()
	defer debugMu.Unlock()
	if desc == debugLastHit {
		return
	}
	debugLastHit = desc
	dbg("%s hit %s", src, desc)
}

func (a *app) debugAfterSize(tag string, wantW, wantH, wantX, wantY int) {
	if !debugInput {
		return
	}
	if a.win == nil {
		dbg("%s SetSize skipped win=nil want=%dx%d pos=%d,%d scale=%.4f", tag, wantW, wantH, wantX, wantY, a.settings.Scale)
		return
	}
	gotW, gotH := a.win.Size()
	gotX, gotY := a.win.Position()
	dbg("%s SetSize want=%dx%d got=%dx%d pos want=%d,%d got=%d,%d scale=%.4f resizing=%s visible=%v destroyed=%v",
		tag, wantW, wantH, gotW, gotH, wantX, wantY, gotX, gotY, a.settings.Scale, a.resizing, a.win.IsVisible(), a.win.IsDestroyed())
}
