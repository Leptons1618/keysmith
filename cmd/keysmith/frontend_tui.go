//go:build tui

package main

import "keysmith/internal/tui"

// runFrontend serves the terminal UI only: tui-tagged builds are pure Go,
// so they can be cross-compiled without any C toolchain. Frontend flags
// are accepted but ignored.
func runFrontend(bool) error { return tui.Run() }
