//go:build !tui

// Package gui renders the SSH key manager as a Fyne desktop application.
package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"keysmith/internal/core"
)

// keysmithTheme maps the shared palette onto Fyne's semantic colors. The
// whole interface is themed through this one mapping, so a component never
// needs to know which colors exist, and light/dark stay in lockstep.
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

// Semantic color names owned by KeySmith. They extend Fyne's own names so
// widgets (RichText, labels, entries) can be themed by role rather than by
// raw hex, and so every color still follows the light/dark variant.
const (
	keyInk         fyne.ThemeColorName = "keysmithInk"
	keyInkBody     fyne.ThemeColorName = "keysmithInkBody"
	keyMuted       fyne.ThemeColorName = "keysmithMuted"
	keyAccent      fyne.ThemeColorName = "keysmithAccent"
	keyAccentInk   fyne.ThemeColorName = "keysmithAccentInk"
	keyAccentTint  fyne.ThemeColorName = "keysmithAccentTint"
	keyOnAccent    fyne.ThemeColorName = "keysmithOnAccent"
	keySuccess     fyne.ThemeColorName = "keysmithSuccess"
	keySuccessTint fyne.ThemeColorName = "keysmithSuccessTint"
	keyDanger      fyne.ThemeColorName = "keysmithDanger"
	keyDangerTint  fyne.ThemeColorName = "keysmithDangerTint"
	keyWarning     fyne.ThemeColorName = "keysmithWarning"
	keyWarningTint fyne.ThemeColorName = "keysmithWarningTint"
	keySurface     fyne.ThemeColorName = "keysmithSurface"
	keySurfaceAlt  fyne.ThemeColorName = "keysmithSurfaceAlt"
	keyCanvas      fyne.ThemeColorName = "keysmithCanvas"
	keyBorder      fyne.ThemeColorName = "keysmithBorder"
)

// variantSet is the full semantic palette for one theme variant. Every field
// has a meaning; screens compose these tokens rather than raw hex values.
type variantSet struct {
	canvas, canvasAlt     color.Color // app background, recessed wells
	surface, surfaceAlt   color.Color // cards, inset rows
	input, inputBorder    color.Color
	ink, inkBody, muted   color.Color // text ramp: strong / body / caption
	border, borderStrong  color.Color
	accent, accentHover   color.Color
	accentTint, accentInk color.Color // wash fill, text on the wash
	onAccent              color.Color
	success, successTint  color.Color
	danger, dangerTint    color.Color
	warning, warningTint  color.Color
	disabled              color.Color
	focus                 color.Color
}

var (
	light = variantSet{
		canvas:       mustHex(core.ColorCanvas),
		canvasAlt:    mustHex(core.ColorSurfaceAlt),
		surface:      mustHex(core.ColorSurface),
		surfaceAlt:   mustHex(core.ColorSurfaceAlt),
		input:        mustHex(core.ColorInput),
		inputBorder:  mustHex(core.ColorBorderStrong),
		ink:          mustHex(core.ColorInk),
		inkBody:      mustHex(core.ColorInkHover),
		muted:        mustHex(core.ColorMuted),
		border:       mustHex(core.ColorBorder),
		borderStrong: mustHex(core.ColorBorderStrong),
		accent:       mustHex(core.ColorAccent),
		accentHover:  mustHex(core.ColorAccentHover),
		accentTint:   mustHex(core.ColorAccentTint),
		accentInk:    mustHex(core.ColorAccentInk),
		onAccent:     mustHex(core.ColorOnAccent),
		success:      mustHex(core.ColorSuccess),
		successTint:  mustHex(core.ColorSuccessTint),
		danger:       mustHex(core.ColorDanger),
		dangerTint:   mustHex(core.ColorDangerTint),
		warning:      mustHex(core.ColorWarning),
		warningTint:  mustHex(core.ColorWarningTint),
		disabled:     mustHex(core.ColorDisabled),
		focus:        mustHex(core.ColorFocusBg),
	}
	dark = variantSet{
		canvas:       mustHex(core.ColorDarkCanvas),
		canvasAlt:    mustHex(core.ColorDarkSurfaceAlt),
		surface:      mustHex(core.ColorDarkSurface),
		surfaceAlt:   mustHex(core.ColorDarkSurfaceAlt),
		input:        mustHex(core.ColorDarkInput),
		inputBorder:  mustHex(core.ColorDarkBorderStrong),
		ink:          mustHex(core.ColorDarkInk),
		inkBody:      mustHex(core.ColorDarkInkHover),
		muted:        mustHex(core.ColorDarkMuted),
		border:       mustHex(core.ColorDarkBorder),
		borderStrong: mustHex(core.ColorDarkBorderStrong),
		accent:       mustHex(core.ColorDarkAccent),
		accentHover:  mustHex(core.ColorDarkAccentHover),
		accentTint:   mustHex(core.ColorDarkAccentTint),
		accentInk:    mustHex(core.ColorDarkAccentInk),
		onAccent:     mustHex(core.ColorDarkOnAccent),
		success:      mustHex(core.ColorDarkSuccess),
		successTint:  mustHex(core.ColorDarkSuccessTint),
		danger:       mustHex(core.ColorDarkDanger),
		dangerTint:   mustHex(core.ColorDarkDangerTint),
		warning:      mustHex(core.ColorDarkWarning),
		warningTint:  mustHex(core.ColorDarkWarningTint),
		disabled:     mustHex(core.ColorDarkDisabled),
		focus:        mustHex(core.ColorDarkFocusBg),
	}
)

// set returns the palette for a theme variant. It honours the variant Fyne
// passes in, which is the contract widget renderers rely on.
func set(v fyne.ThemeVariant) *variantSet {
	if v == theme.VariantDark {
		return &dark
	}
	return &light
}

// variantOverride pins the variant used by the app's own resolve helper,
// instead of following the app setting. It exists so the headless screenshot
// renderer can capture both looks from a driver that always reports one fixed
// variant. Nil in normal use.
var variantOverride *fyne.ThemeVariant

// setVariantOverride pins the variant for tests and tooling.
func setVariantOverride(v fyne.ThemeVariant) {
	variantOverride = &v
}

// effectiveVariant is the variant the app's own drawing helpers use.
func effectiveVariant() fyne.ThemeVariant {
	if variantOverride != nil {
		return *variantOverride
	}
	if fyne.CurrentApp() != nil {
		return fyne.CurrentApp().Settings().ThemeVariant()
	}
	return theme.VariantLight
}

func (t *keysmithTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	v := set(variant)
	switch name {
	case theme.ColorNameBackground:
		return v.canvas
	case theme.ColorNameForeground:
		return v.ink
	case theme.ColorNamePrimary:
		return v.accent
	case theme.ColorNameError:
		return v.danger
	case theme.ColorNameForegroundOnError:
		return v.surface
	case theme.ColorNameWarning:
		return v.warning
	case theme.ColorNameForegroundOnWarning:
		return v.ink
	case theme.ColorNameSuccess:
		return v.success
	case theme.ColorNameForegroundOnSuccess:
		return v.surface
	case theme.ColorNameForegroundOnPrimary:
		return v.onAccent
	case theme.ColorNameInputBackground:
		return v.input
	case theme.ColorNameInputBorder:
		return v.inputBorder
	case theme.ColorNamePlaceHolder:
		return v.muted
	case theme.ColorNameButton:
		// Distinct from the card surface, or secondary buttons disappear into
		// the white panel they sit on.
		return v.surfaceAlt
	case theme.ColorNameDisabledButton:
		return v.surfaceAlt
	case theme.ColorNameDisabled:
		return v.disabled
	case theme.ColorNameHyperlink:
		return v.accent
	case theme.ColorNameHeaderBackground:
		return v.surface
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return v.surface
	case theme.ColorNameSelection:
		return v.accentTint
	case theme.ColorNameSeparator:
		return v.border
	case theme.ColorNameScrollBar, theme.ColorNameScrollBarBackground:
		return v.border
	case theme.ColorNameFocus:
		return v.focus
	case theme.ColorNameHover:
		return v.focus
	case theme.ColorNamePressed:
		return v.accentTint
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x0f, G: 0x17, B: 0x2a, A: 0x1a}

	// KeySmith's own semantic names.
	case keyInk:
		return v.ink
	case keyInkBody:
		return v.inkBody
	case keyMuted:
		return v.muted
	case keyAccent:
		return v.accent
	case keyAccentInk:
		return v.accentInk
	case keyAccentTint:
		return v.accentTint
	case keyOnAccent:
		return v.onAccent
	case keySuccess:
		return v.success
	case keySuccessTint:
		return v.successTint
	case keyDanger:
		return v.danger
	case keyDangerTint:
		return v.dangerTint
	case keyWarning:
		return v.warning
	case keyWarningTint:
		return v.warningTint
	case keySurface:
		return v.surface
	case keySurfaceAlt:
		return v.surfaceAlt
	case keyCanvas:
		return v.canvas
	case keyBorder:
		return v.border
	default:
		return t.Theme.Color(name, variant)
	}
}

// Sizes: soft, generously rounded shapes read as a modern product surface
// rather than a machined tool.
func (t *keysmithTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameButtonRadius:
		return 8
	case theme.SizeNameCardRadius:
		return 14
	case theme.SizeNameDialogRadius:
		return 16
	case theme.SizeNamePopupRadius:
		return 12
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	default:
		return t.Theme.Size(name)
	}
}
