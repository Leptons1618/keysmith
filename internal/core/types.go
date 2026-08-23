// Package core holds all SSH key management business logic.
// It shells out to the OpenSSH tools (ssh-keygen, ssh-add, ssh) exactly like
// the original Python implementation and must never import any UI package.
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

// AgentState is the tri-state SSH agent status.
type AgentState string

const (
	AgentOn      AgentState = "on"
	AgentOff     AgentState = "off"
	AgentUnknown AgentState = "unknown"
)

// Keysmith's-workshop palette shared by both frontends.
// Light: warm paper over the workbench. Dark: oiled iron at night.
// Copper is the working metal; verdigris (oxidized copper) marks success.
const (
	// Light variant
	ColorCanvas       = "#f3efe8" // workbench paper, warm
	ColorSurface      = "#faf7f2" // raised card
	ColorInput        = "#fffdf9" // entry fields
	ColorInk          = "#2b2118" // dark walnut ink
	ColorInkHover     = "#443627"
	ColorMuted        = "#6d6055" // worn oak
	ColorBorder       = "#ddd3c4"
	ColorBorderStrong = "#c2b49e"
	ColorAccent       = "#b05d21" // copper
	ColorAccentTint   = "#f7e3d0" // polished copper tint
	ColorDanger       = "#a03430" // ember red
	ColorDangerTint   = "#fbe3dd"
	ColorWarning      = "#8a6410" // brass
	ColorWarningTint  = "#f6ecd0"
	ColorDisabled     = "#857b70" // report fix: perceivable disabled
	ColorFocusBg      = "#eee2cf"

	// Dark variant ("oiled iron")
	ColorDarkCanvas       = "#1d1a17" // oiled iron
	ColorDarkSurface      = "#282420" // planished panel
	ColorDarkInput        = "#211e1b"
	ColorDarkInk          = "#ece4da" // candlelight text
	ColorDarkMuted        = "#a89a89" // aged brass text
	ColorDarkBorder       = "#3a342e"
	ColorDarkBorderStrong = "#55493d"
	ColorDarkAccent       = "#e0813f" // glowing copper
	ColorDarkAccentTint   = "#4a2f18" // copper ember bed
	ColorDarkSuccess      = "#62b39a" // verdigris glow
	ColorDarkDanger       = "#e07a6a" // ember on iron
	ColorDarkWarning      = "#d9a94f" // lit brass
	ColorDarkDisabled     = "#7d7266"
	ColorDarkFocusBg      = "#37302a"
)
