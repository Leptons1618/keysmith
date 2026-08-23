package gui

import (
	"fmt"
	"image/color"
	"strconv"
)

// parseHex parses #rrggbb into an RGBA color.
func parseHex(s string) (color.Color, error) {
	if len(s) != 7 || s[0] != '#' {
		return nil, fmt.Errorf("invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return nil, err
	}
	return color.NRGBA{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 255,
	}, nil
}
