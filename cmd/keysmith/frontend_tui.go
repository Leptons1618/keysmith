//go:build tui

package main

import (
	"fmt"

	"keysmith/internal/tui"
)

func runFrontend(mode frontendMode) error {
	if mode != tuiMode {
		return fmt.Errorf("this binary only supports the terminal UI; use the default keysmith binary for --gui")
	}
	return tui.Run()
}
