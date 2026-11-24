package main

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestSettingsCloseHidesButQuitDoesNotPrevent(t *testing.T) {
	a := newApp()
	user := &mygo.CloseEvent{}
	a.onSettingsClose(user)
	if !user.DefaultPrevented() {
		t.Fatal("closing settings alone must prevent default and hide, not destroy")
	}

	quit := &mygo.CloseEvent{}
	a.quitting = true
	a.onSettingsClose(quit)
	if quit.DefaultPrevented() {
		t.Fatal("settings OnClose must not PreventDefault during app quit (6f74c44 OnQuitCanceled)")
	}
}
