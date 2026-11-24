package main

import "testing"

func TestDebugInputOffByDefault(t *testing.T) {
	if debugInputEnabled() {
		t.Fatal("debug input must stay off unless MYFISH_DEBUG_INPUT is set")
	}
	if debugInput {
		t.Fatal("debug logger must not start in tests")
	}
	// dbg is a no-op when disabled and must not panic.
	dbg("should not write %s", "anything")
}
