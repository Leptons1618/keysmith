//go:build tui

package main

import "testing"

func TestTUIAdapterRejectsGUI(t *testing.T) {
	if err := runFrontend(guiMode); err == nil {
		t.Fatal("runFrontend(guiMode) error = nil, want opposite-mode error")
	}
}
