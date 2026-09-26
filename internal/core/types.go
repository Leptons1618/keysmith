// Package core owns all SSH key management business logic.
// It shells out to the OpenSSH tools and must never import a UI package.
package core

import "errors"

// KeyAlgorithm is a supported ssh-keygen key type.
type KeyAlgorithm string

const (
	AlgoEd25519 KeyAlgorithm = "ed25519"
	AlgoRSA     KeyAlgorithm = "rsa"
	AlgoECDSA   KeyAlgorithm = "ecdsa"
)

// KeyAlgorithms lists the algorithms in UI display order.
var KeyAlgorithms = []KeyAlgorithm{AlgoEd25519, AlgoRSA, AlgoECDSA}

// KeyInfo describes one key pair found in ~/.ssh.
type KeyInfo struct {
	Name    string
	Path    string
	PubPath string
}

// Result is the outcome of a core operation. Message is always
// human-readable and suitable for the activity log and status line.
type Result struct {
	OK      bool
	Message string
}

// ErrInvalidKeyName is returned when a key name fails validation.
var ErrInvalidKeyName = errors.New("use only letters, numbers, dot (.), underscore (_) or dash (-)")

// HostResult is the outcome of one SSH authentication attempt against a host.
type HostResult struct {
	Host   string
	OK     bool
	Output string
}

// KeySmith palette shared by both frontends.
//
// The system is a neutral slate scale with a single indigo accent, so the
// interface reads as a calm developer tool: colour is reserved for state
// (success / danger / warning) and the primary action, never decoration.
//
// Light: near-white app canvas, white cards, slate text.
// Dark:  deep navy canvas, raised slate panels, high-contrast text.
const (
	// Light variant
	ColorCanvas       = "#f5f6f8" // app background
	ColorSurface      = "#ffffff" // cards, panels
	ColorSurfaceAlt   = "#f0f2f6" // inset / track / hover wash
	ColorInput        = "#ffffff" // entry fields
	ColorInk          = "#0f172a" // headings, primary text
	ColorInkHover     = "#334155" // body text
	ColorMuted        = "#64748b" // captions, hints, metadata
	ColorBorder       = "#e6e9ef" // hairline dividers
	ColorBorderStrong = "#d3d9e3" // inputs, emphasised borders
	ColorAccent       = "#4f46e5" // indigo 600 - the one accent
	ColorAccentHover  = "#4338ca" // indigo 700 - pressed primary
	ColorAccentTint   = "#eef2ff" // indigo 50 - selected / accent wash
	ColorAccentInk    = "#3730a3" // indigo 800 - text on tint
	ColorOnAccent     = "#ffffff" // text on a filled accent surface
	ColorSuccess      = "#047857" // emerald 700
	ColorSuccessTint  = "#ecfdf5"
	ColorDanger       = "#dc2626" // red 600
	ColorDangerTint   = "#fef2f2"
	ColorWarning      = "#b45309" // amber 700
	ColorWarningTint  = "#fffbeb"
	ColorDisabled     = "#9aa3b2"
	ColorFocusBg      = "#e8ecf3"

	// Dark variant
	ColorDarkCanvas       = "#0b0f1a" // app background
	ColorDarkSurface      = "#141a27" // cards, panels
	ColorDarkSurfaceAlt   = "#1c2333" // inset / track / hover wash
	ColorDarkInput        = "#0e1420"
	ColorDarkInk          = "#eef2f8" // headings, primary text
	ColorDarkInkHover     = "#c2cbdb" // body text
	ColorDarkMuted        = "#8b95a9" // captions, hints, metadata
	ColorDarkBorder       = "#232b3c"
	ColorDarkBorderStrong = "#37415a"
	ColorDarkAccent       = "#818cf8" // indigo 400 - readable on dark
	ColorDarkAccentHover  = "#a5b4fc"
	ColorDarkAccentTint   = "#1e2545" // selected / accent wash
	ColorDarkAccentInk    = "#c7d2fe" // text on tint
	ColorDarkOnAccent     = "#0b0f1a" // text on a filled accent surface
	ColorDarkSuccess      = "#34d399"
	ColorDarkSuccessTint  = "#0c2a22"
	ColorDarkDanger       = "#f87171"
	ColorDarkDangerTint   = "#2c1418"
	ColorDarkWarning      = "#fbbf24"
	ColorDarkWarningTint  = "#2a2010"
	ColorDarkDisabled     = "#5b6577"
	ColorDarkFocusBg      = "#1b2334"
)
