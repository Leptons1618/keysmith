//go:build !tui

package main

import "testing"

// TestGUIAdapterRejectsTUI guards the desktop binary's frontend switch. It must
// live in a !tui file: under -tags tui the same call would fall through to
// tui.Run() and launch the interactive terminal UI from inside a test.
func TestGUIAdapterRejectsTUI(t *testing.T) {
	if err := runFrontend(tuiMode); err == nil {
		t.Fatal("runFrontend(tuiMode) error = nil, want opposite-mode error")
	}
}
