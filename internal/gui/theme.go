// Package gui renders the SSH key manager as a Fyne desktop application.
package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"keysmith/internal/core"
)

// keysmithTheme maps the keysmith's-workshop palette onto Fyne's semantic
// colors, respecting the light/dark variant so both faces of the workshop
// render correctly.
type keysmithTheme struct {
	fyne.Theme
}

func mustHex(s string) color.Color {
	c, err := parseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

type variantSet struct {
	canvas, surface, input, ink, muted       color.Color
	border, borderStrong, accent, accentTint color.Color
	danger, dangerTint, warning, warningTint color.Color
	disabled, focusBg, success               color.Color
}

var (
	light = variantSet{
		canvas:       mustHex(core.ColorCanvas),
		surface:      mustHex(core.ColorSurface),
		input:        mustHex(core.ColorInput),
		ink:          mustHex(core.ColorInk),
		muted:        mustHex(core.ColorMuted),
		border:       mustHex(core.ColorBorder),
		borderStrong: mustHex(core.ColorBorderStrong),
		accent:       mustHex(core.ColorAccent),
		accentTint:   mustHex(core.ColorAccentTint),
		danger:       mustHex(core.ColorDanger),
		dangerTint:   mustHex(core.ColorDangerTint),
		warning:      mustHex(core.ColorWarning),
		warningTint:  mustHex(core.ColorWarningTint),
		disabled:     mustHex(core.ColorDisabled), // report fix: perceivable disabled text
		focusBg:      mustHex(core.ColorFocusBg),
		success:      mustHex("#1e6b52"),
	}
	dark = variantSet{
		canvas:       mustHex(core.ColorDarkCanvas),
		surface:      mustHex(core.ColorDarkSurface),
		input:        mustHex(core.ColorDarkInput),
		ink:          mustHex(core.ColorDarkInk),
		muted:        mustHex(core.ColorDarkMuted),
		border:       mustHex(core.ColorDarkBorder),
		borderStrong: mustHex(core.ColorDarkBorderStrong),
		accent:       mustHex(core.ColorDarkAccent),
		accentTint:   mustHex(core.ColorDarkAccentTint),
		danger:       mustHex(core.ColorDarkDanger),
		dangerTint:   mustHex(core.ColorDangerTint),
		warning:      mustHex(core.ColorDarkWarning),
		warningTint:  mustHex(core.ColorWarningTint),
		disabled:     mustHex(core.ColorDarkDisabled),
		focusBg:      mustHex(core.ColorDarkFocusBg),
		success:      mustHex(core.ColorDarkSuccess),
	}
)

func set(v fyne.ThemeVariant) *variantSet {
	if v == theme.VariantDark {
		return &dark
	}
	return &light
}

func (t *keysmithTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	v := set(variant)
	switch name {
	case theme.ColorNameBackground:
		return v.canvas
	case theme.ColorNameForeground:
		return v.ink
	case theme.ColorNamePrimary:
		return v.accent // copper: every primary action is "work the metal"
	case theme.ColorNameError:
		return v.danger
	case theme.ColorNameWarning:
		return v.warning
	case theme.ColorNameSuccess:
		return v.success
	case theme.ColorNameInputBackground:
		return v.input
	case theme.ColorNameButton:
		return v.surface
	case theme.ColorNameDisabledButton:
		return v.border
	case theme.ColorNameDisabled:
		return v.disabled
	case theme.ColorNameSeparator, theme.ColorNameInputBorder:
		return v.borderStrong
	case theme.ColorNameSelection:
		return v.ink
	case theme.ColorNameFocus:
		return v.accentTint
	case theme.ColorNameHover:
		return v.focusBg
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return v.surface
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x2b, G: 0x21, B: 0x18, A: 0x38}
	default:
		return t.Theme.Color(name, variant)
	}
}

// Sizes: slightly rounder corners and taller inputs read as forged, not machined.
func (t *keysmithTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 6
	case theme.SizeNamePadding:
		return 6
	default:
		return t.Theme.Size(name)
	}
}
