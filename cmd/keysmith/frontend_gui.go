//go:build !tui

package main

import (
	"fmt"

	"keysmith/internal/gui"
)

func runFrontend(mode frontendMode) error {
	if mode != guiMode {
		return fmt.Errorf("this binary only supports the desktop GUI; use a TUI-tagged keysmith binary for --tui")
	}
	return gui.Run(version)
}
