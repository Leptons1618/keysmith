// Package tui renders the SSH key manager as a Bubble Tea terminal UI.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"keysmith/internal/core"
)

// The TUI shares the GUI's palette: a neutral slate scale with a single indigo
// accent. Colours are declared as lipgloss adaptive colours so the same styles
// work on light and dark terminals.
var (
	colInk      = lipgloss.AdaptiveColor{Light: core.ColorInk, Dark: core.ColorDarkInk}
	colBody     = lipgloss.AdaptiveColor{Light: core.ColorInkHover, Dark: core.ColorDarkInkHover}
	colMuted    = lipgloss.AdaptiveColor{Light: core.ColorMuted, Dark: core.ColorDarkMuted}
	colAccent   = lipgloss.AdaptiveColor{Light: core.ColorAccent, Dark: core.ColorDarkAccent}
	colOnAcc    = lipgloss.AdaptiveColor{Light: core.ColorOnAccent, Dark: "#ffffff"}
	colSuccess  = lipgloss.AdaptiveColor{Light: core.ColorSuccess, Dark: core.ColorDarkSuccess}
	colDanger   = lipgloss.AdaptiveColor{Light: core.ColorDanger, Dark: core.ColorDarkDanger}
	colWarning  = lipgloss.AdaptiveColor{Light: core.ColorWarning, Dark: core.ColorDarkWarning}
	colBorder   = lipgloss.AdaptiveColor{Light: core.ColorBorder, Dark: core.ColorDarkBorder}
	colSurface  = lipgloss.AdaptiveColor{Light: core.ColorSurface, Dark: core.ColorDarkSurface}
	colInset    = lipgloss.AdaptiveColor{Light: core.ColorSurfaceAlt, Dark: core.ColorDarkSurfaceAlt}
	colAccentBg = lipgloss.AdaptiveColor{
		Light: core.ColorAccentTint, Dark: core.ColorDarkAccentTint}
)

// tone mirrors the GUI's semantic tone vocabulary so both frontends describe
// state with the same words and the same colours.
type tone int

const (
	toneNeutral tone = iota
	toneAccent
	toneSuccess
	toneDanger
	toneWarning
)

// pair returns the foreground and background for a tone.
func (t tone) pair() (lipgloss.TerminalColor, lipgloss.TerminalColor) {
	switch t {
	case toneAccent:
		return colAccent, colAccentBg
	case toneSuccess:
		return colSuccess, colSurface
	case toneDanger:
		return colDanger, colSurface
	case toneWarning:
		return colWarning, colSurface
	default:
		return colMuted, colSurface
	}
}

// --- text styles -----------------------------------------------------------

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleHead  = lipgloss.NewStyle().Bold(true).Foreground(colInk)
	styleBody  = lipgloss.NewStyle().Foreground(colBody)
	styleMuted = lipgloss.NewStyle().Foreground(colMuted)
	styleMono  = lipgloss.NewStyle().Foreground(colInk)
	styleLabel = lipgloss.NewStyle().Bold(true).Foreground(colMuted)
)

// --- structural styles -----------------------------------------------------

var (
	styleRule    = lipgloss.NewStyle().Foreground(colBorder)
	styleRuleAcc = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleCard    = lipgloss.NewStyle().Foreground(colInk)
	styleBox     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)
	styleKeycap    = lipgloss.NewStyle().Bold(true).Foreground(colInk).Background(colInset)
	styleBtnFocus  = lipgloss.NewStyle().Bold(true).Foreground(colOnAcc).Background(colAccent)
	styleBtnIdle   = lipgloss.NewStyle().Foreground(colBody)
	styleSelected  = lipgloss.NewStyle().Bold(true).Foreground(colInk)
	styleStatus    = lipgloss.NewStyle().Foreground(colMuted)
	styleStatusErr = lipgloss.NewStyle().Bold(true).Foreground(colDanger)
	styleBusy      = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
)

// statusStyle returns the style for the current status message's tone.
func statusStyle(t tone) lipgloss.Style {
	switch t {
	case toneDanger:
		return lipgloss.NewStyle().Bold(true).Foreground(colDanger)
	case toneSuccess:
		return lipgloss.NewStyle().Foreground(colSuccess)
	case toneAccent:
		return lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	default:
		return styleStatus
	}
}

// --- components ------------------------------------------------------------

// badge is a compact state pill.
func badge(on bool, text string) string {
	return badgeTone(boolTone(on), text)
}

// boolTone maps a done/not-done flag to the shared tone vocabulary.
func boolTone(on bool) tone {
	if on {
		return toneSuccess
	}
	return toneNeutral
}

// badgeTone renders text on a tone's background.
func badgeTone(t tone, text string) string {
	fg, bg := t.pair()
	return lipgloss.NewStyle().Bold(true).Foreground(fg).Background(bg).
		Render(" " + text + " ")
}

// dot is a single status cell in a tone's color. The TUI's equivalent of the
// GUI's status dot.
func dot(t tone) string {
	fg, _ := t.pair()
	return lipgloss.NewStyle().Foreground(fg).Render("●")
}

// emptyDot is the hollow counterpart.
var emptyDot = lipgloss.NewStyle().Foreground(colBorder).Render("○")

// keycap renders a key hint.
func keycap(k string) string { return styleKeycap.Render(" " + k + " ") }

// cursorStyle is the text-input cursor, tinted to match the accent.
func cursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colAccent)
}

// actionButton renders a rectangular action with a rule underneath.
func actionButton(label string, focused bool) string {
	text := " " + strings.ToUpper(label) + " "
	if focused {
		return styleBtnFocus.Render(text) + "\n" + styleRuleAcc.Render(strings.Repeat("─", lipgloss.Width(text)))
	}
	return styleBtnIdle.Render(text) + "\n" + styleRule.Render(strings.Repeat("─", lipgloss.Width(text)))
}

// sectionLabel is the small uppercase label above a group of content.
func sectionLabel(text string) string {
	return styleLabel.Render(strings.ToUpper(text))
}

// hints renders a row of key hints from label/key pairs.
func hints(pairs ...[2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, keycap(p[1])+" "+styleMuted.Render(p[0]))
	}
	return strings.Join(parts, styleMuted.Render("   "))
}

// ruleLine is a horizontal divider trimmed to the content width.
func ruleLine(width int) string {
	if width < 8 {
		width = 8
	}
	if width > 92 {
		width = 92
	}
	return styleRule.Render(strings.Repeat("─", width))
}
