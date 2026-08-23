//go:build !tui

package main

import "keysmith/internal/gui"

// runFrontend serves the desktop GUI: default builds link Fyne and need a
// native (non-cross) toolchain. The tui flag is accepted but ignored.
func runFrontend(bool) error { gui.Run(); return nil }
