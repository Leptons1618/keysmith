//go:build !tui

package gui

import (
	"image/color"
	"os"
	"path/filepath"
)

// Small filesystem helpers shared by the screen tests, kept here so the test
// files read as intent rather than as plumbing.

func mkdirAll(dir string) error { return os.MkdirAll(dir, 0o700) }

func writeFile(path, contents string) error {
	return os.WriteFile(filepath.Clean(path), []byte(contents), 0o600)
}

// isNilColor reports whether a theme lookup came back unset.
func isNilColor(c color.Color) bool { return c == nil }

// sameColor compares two colors, tolerating nil.
func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// resolveIn is an identity helper kept for readability in palette assertions.
func resolveIn(c color.Color) color.Color { return c }
