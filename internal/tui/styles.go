// Package tui renders the SSH key manager as a Bubble Tea terminal UI.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"keysmith/internal/core"
)

// Keysmith's-workshop palette, degraded through lipgloss adaptive colors:
// copper on warm paper (light), glowing copper on oiled iron (dark).
var (
	styleCanvas = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: core.ColorInk, Dark: "252"})
	styleTitle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "209"})
	styleSubtitle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorMuted, Dark: "245"})
	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: core.ColorBorderStrong, Dark: "240"})
	styleBoxTitle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorMuted, Dark: "245"})
	styleBadgeOn = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#5f3a12", Dark: "230"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorAccentTint, Dark: "94"})
	styleBadgeWarn = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorWarning, Dark: "179"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorWarningTint, Dark: "236"})
	styleBadgeOff = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorMuted, Dark: "245"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorSurface, Dark: "236"})
	styleStepDone = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "209"})
	styleSuccess = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1e6b52", Dark: "115"}) // verdigris
	styleDanger = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorDanger, Dark: "174"})
	styleStatus = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorMuted, Dark: "245"})
	styleBusy = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "209"})
	styleMono = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorInk, Dark: "252"})
	styleSelectedRow = lipgloss.NewStyle().
				Bold(true)
)

// Rectangular button chrome: ┌ label ┐ with a rule underneath.
func actionButton(label string, focused bool) string {
	text := " " + strings.ToUpper(label) + " "
	if focused {
		return styleBtnFocus.Render(text) + "\n" + styleRuleAccent.Render(strings.Repeat("─", lipgloss.Width(text)))
	}
	return styleBtnIdle.Render(text) + "\n" + styleRule.Render(strings.Repeat("─", lipgloss.Width(text)))
}

func keycap(k string) string {
	return styleKeycap.Render(" " + k + " ")
}

// Button + rule chrome for rectangular actions.
var (
	styleBtnFocus = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#fdf6ee", Dark: "234"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "131"})
	styleBtnIdle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorInkHover, Dark: "250"})
	styleRule = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorBorderStrong, Dark: "240"})
	styleRuleAccent = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "209"})
	styleKeycap = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: core.ColorInk, Dark: "230"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorBorder, Dark: "239"})
	styleTracker = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#fdf6ee", Dark: "234"}).
			Background(lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: "94"})
	styleBadgeSuccess = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#f2faf7", Dark: "235"}).
				Background(lipgloss.AdaptiveColor{Light: "#1e6b52", Dark: "71"})
)

func lipglossAccent() lipgloss.Color {
	return lipgloss.Color(core.ColorAccent)
}

// badge renders an on/off state chip.
func badge(on bool, text string) string {
	if on {
		return styleBadgeOn.Render(text)
	}
	return styleBadgeOff.Render(text)
}
