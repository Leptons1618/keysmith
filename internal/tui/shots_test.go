package tui

import (
	"flag"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"keysmith/internal/core"
)

// flagString declares a string flag for a test binary.

// shotsDir is set with -kshots=<dir> to render every TUI screen to a PNG.
//
//	go test -tags tui ./internal/tui -run TestRenderShots -kshots=/tmp/tui-shots
var shotsDir = flag.String("kshots", "", "render all TUI screens to PNGs in this directory")

// TestRenderShots renders every screen of the terminal UI so the layout can be
// reviewed as an image rather than squinted at in a terminal.
func TestRenderShots(t *testing.T) {
	if *shotsDir == "" {
		t.Skip("set -kshots=<dir> to render screens")
	}
	if err := os.MkdirAll(*shotsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	seedKeys(t)

	// Render in both variants so the adaptive palette is exercised.
	for _, variant := range []string{"light", "dark"} {
		lipgloss.SetColorProfile(terminfoFor(variant))
		lipgloss.SetHasDarkBackground(variant == "dark")

		m := newModel()
		m.width = 100
		m.height = 40
		m.keys = core.ListKeys()
		m.loadKeys()
		m.selected = "id_ed25519"
		m.svc = core.Services[0]
		_ = m.store.MarkCopied("id_ed25519")
		_ = m.store.MarkAgentLoaded("id_ed25519")
		_ = m.store.RecordTest("id_ed25519", true)
		_ = m.store.MarkCopied("id_github_work")

		withStatus := m
		withStatus.status = "Public key copied to clipboard"
		withStatus.statusTone = toneSuccess

		agentLoaded := m
		agentLoaded.screen = scrAgent
		agentLoaded.agentChecked = true
		agentLoaded.agent = core.AgentState{
			Reachable: true,
			Keys: []core.AgentKey{
				{Bits: 256, Fingerprint: "SHA256:7Yq1n0K8s3BqWzT4mXpLvC9dF2hJ5rEuA0oIzYwXcVb", Comment: "you@laptop", Kind: "ED25519"},
				{Bits: 4096, Fingerprint: "SHA256:1Ab2Cd3Ef4Gh5Ij6Kl7Mn8Op9Qr0St1Uv2Wx3Yz", Comment: "work@box", Kind: "RSA"},
			},
		}

		success := m
		success.screen = scrResult
		success.success = true
		success.lastTest = core.HostResult{OK: true, Output: "OK (port 22)\nHi jai! You've successfully authenticated."}

		failure := m
		failure.screen = scrResult
		failure.success = false
		failure.lastTest = core.HostResult{Output: "--- port 22 ---\ngit@github.com: Permission denied (publickey)."}

		form := m
		form.screen = scrForm
		form.formPos = fPass
		form.pass.SetValue("hunter2")

		filtered := m
		filtered.screen = scrBrowser
		filtered.filterQuery = "work"
		filtered.visible = filterKeys(filtered.keys, "work")
		filtered.menuIdx = 0
		filtered.selected = "id_rsa_work"

		help := m
		help.help = true
		help.helpTopic = helpBrowser

		ready := m
		ready.screen = scrKeyReady
		ready.newKey = "id_ed25519"

		browser := m
		browser.screen = scrBrowser
		home2 := m
		home2.screen = scrHome

		agent := m
		agent.screen = scrAgent

		instr := m
		instr.screen = scrInstructions

		svc := m
		svc.screen = scrService

		empty := newModel()
		empty.width, empty.height = 100, 40
		empty.keys = nil
		empty.visible = nil
		empty.screen = scrBrowser

		scenes := []struct {
			name string
			m    model
		}{
			{"01-home", home2},
			{"02-form", form},
			{"03-browser", browser},
			{"04-browser-filtered", filtered},
			{"05-key-ready", ready},
			{"06-service", svc},
			{"07-instructions", instr},
			{"08-agent-unchecked", agent},
			{"09-agent-loaded", agentLoaded},
			{"10-result-success", success},
			{"11-result-denied", failure},
			{"12-help", help},
			{"13-empty-browser", empty},
			{"14-status", withStatus},
		}

		for _, s := range scenes {
			path := filepath.Join(*shotsDir, s.name+"-"+variant+".txt")
			if err := writeScreenDump(path, s.m.View()); err != nil {
				t.Fatalf("%s: %v", s.name, err)
			}
			t.Logf("wrote %s", path)
		}
	}
}

// writeScreenDump writes a screen's rendered output as text with the ANSI
// styling removed, plus a summary of the colours it used. Reviewing the plain
// text is how the layout, alignment and copy get checked; the colour summary
// is how the palette gets checked.
func writeScreenDump(path, text string) error {
	var b strings.Builder
	b.WriteString("+" + strings.Repeat("-", 100) + "+")

	used := map[string]int{}
	for _, span := range sgrPattern.FindAllString(text, -1) {
		if c := colourFromParams(stripParams(span)); c != nil {
			used[hexOf(c)]++
		}
	}

	b.WriteString("\n")
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString("|" + padTo(stripANSI(line), 100) + "|\n")
	}
	b.WriteString("+" + strings.Repeat("-", 100) + "+")

	b.WriteString("\ncolours used (" + itoa(len(used)) + "):\n")
	for hex, n := range used {
		b.WriteString("  " + hex + "  x" + itoa(n) + "\n")
	}
	b.WriteString("\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// sgrPattern matches an SGR escape sequence.
var sgrPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes styling from a rendered line.
func stripANSI(s string) string { return sgrPattern.ReplaceAllString(s, "") }

// stripParams returns the parameter list inside an SGR sequence.
func stripParams(span string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(span, "\x1b["), "m")
	return inner
}

// colourFromParams reads a truecolour SGR parameter list.
func colourFromParams(params string) color.Color {
	fields := strings.Split(params, ";")
	for i, f := range fields {
		if f != "38" && f != "48" {
			continue
		}
		if i+4 < len(fields) && fields[i+1] == "2" {
			if r, g, b, ok := rgbFrom(fields[i+2], fields[i+3], fields[i+4]); ok {
				return color.RGBA{r, g, b, 0xff}
			}
		}
	}
	return nil
}

func rgbFrom(rs, gs, bs string) (uint8, uint8, uint8, bool) {
	r, ok1 := atoi(rs)
	g, ok2 := atoi(gs)
	b, ok3 := atoi(bs)
	if !ok1 || !ok2 || !ok3 || r < 0 || g < 0 || b < 0 || r > 255 || g > 255 || b > 255 {
		return 0, 0, 0, false
	}
	return uint8(r), uint8(g), uint8(b), true
}

func atoi(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return "#" + hex2(r>>8) + hex2(g>>8) + hex2(b>>8)
}

func hex2(v uint32) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[(v>>4)&0xf], digits[v&0xf]})
}

// padTo pads on the right so every line in a dump lines up in the review.
func padTo(s string, w int) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// terminfoFor pins the colour profile so rendered output carries real colour
// codes even though a test binary is not attached to a terminal.
func terminfoFor(string) termenv.Profile { return termenv.TrueColor }

// seedKeys writes a few plausible key pairs into the temporary ~/.ssh.
func seedKeys(t *testing.T) {
	t.Helper()
	dir := core.SSHDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	pubs := map[string]string{
		"id_ed25519":     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHq0m2XnR8vKk1pQfZ7yL3dJ5sW9aBcDeFgHiJkLmNoP you@laptop",
		"id_rsa_work":    "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7x2vN8kLmR4tY6uI0oP2aS4dF6gH8jK1lZ3xC5vB7nM9qR2sT4uV6wX8yZ0aB1cD3eF5gH7jK9lZ1xC3vB5nM7qR9s work@box",
		"id_gh_personal": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG5nR7tY2uI4oP6aS8dF0gH2jK4lZ6xC8vB0nM2qR4sT6uV8w gh@mbp",
	}
	for name, pub := range pubs {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("private\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".pub"), []byte(pub+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
