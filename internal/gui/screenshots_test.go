//go:build !tui

package gui

import (
	"flag"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"keysmith/internal/core"
)

// shotsDir is set with -kshots=<dir> to dump every screen to a PNG. It is off
// by default so ordinary test runs stay hermetic.
//
//	go test ./internal/gui -run TestRenderScreenshots -kshots=/tmp/shots
var shotsDir = flag.String("kshots", "", "render all screens to PNGs in this directory")

// TestRenderScreenshots renders every page of the app in both theme variants.
// It is a design-review tool as much as a test: the images are how the layout
// gets checked without a display.
func TestRenderScreenshots(t *testing.T) {
	if *shotsDir == "" {
		t.Skip("set -kshots=<dir> to render screens")
	}
	if err := os.MkdirAll(*shotsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	seedRealisticKeys(t)

	size := fyne.NewSize(1180, 780)
	setVariantOverride(theme.VariantLight)
	defer setVariantOverride(theme.VariantLight)

	for _, variant := range []struct {
		name string
		dark bool
	}{{"light", false}, {"dark", true}} {
		app := fynetest.NewApp()
		app.Settings().SetTheme(&keysmithTheme{Theme: theme.DefaultTheme()})
		win := app.NewWindow("KeySmith")
		win.Resize(size)
		ui := &appUI{win: win, store: core.LoadStore()}
		ui.keys = core.ListKeys()
		ui.selected = "id_ed25519"
		ui.svc = core.Services[0]

		if variant.dark {
			setVariantOverride(theme.VariantDark)
		} else {
			setVariantOverride(theme.VariantLight)
		}

		// A believable state: one verified key, one in progress.
		_ = ui.store.MarkCopied("id_ed25519")
		_ = ui.store.MarkAgentLoaded("id_ed25519")
		_ = ui.store.RecordTest("id_ed25519", true)
		_ = ui.store.MarkCopied("id_github_work")

		scenes := []struct {
			name  string
			setup func()
		}{
			{"01-overview", func() { ui.navigate(scrOverview) }},
			{"02-create-key", func() { ui.navigate(scrNewKey) }},
			{"03-create-key-existing-name", func() {
				ui.navigate(scrNewKey)
				ui.form.name.SetText("id_ed25519")
			}},
			{"04-create-key-rsa-and-pass", func() {
				ui.navigate(scrNewKey)
				ui.form.selectAlgo(1)
				ui.form.pass.SetText("hunter2")
			}},
			{"05-create-key-invalid", func() {
				ui.navigate(scrNewKey)
				ui.form.name.SetText("bad name!")
				ui.formError("Use only letters, numbers, dot (.), underscore (_) or dash (-).")
			}},
			{"06-keys", func() { ui.navigate(scrKeys) }},
			{"07-keys-rsa-selected", func() {
				ui.selected = "id_rsa_work"
				ui.navigate(scrKeys)
			}},
			{"08-agent-unchecked", func() { ui.navigate(scrAgent) }},
			{"09-agent-loaded", func() {
				ui.agentChecked = true
				ui.agent = core.AgentState{
					Reachable: true,
					Keys: []core.AgentKey{
						{Bits: 256, Fingerprint: "SHA256:7Yq1n0K8s3BqWzT4mXpLvC9dF2hJ5rEuA0oIzYwXcVb", Comment: "you@laptop", Kind: "ED25519"},
						{Bits: 4096, Fingerprint: "SHA256:1Ab2Cd3Ef4Gh5Ij6Kl7Mn8Op9Qr0St1Uv2Wx3Yz", Comment: "work@box", Kind: "RSA"},
					},
				}
				ui.navigate(scrAgent)
			}},
			{"10-agent-down", func() {
				ui.agentChecked = true
				ui.agent = core.AgentState{Reachable: false}
				ui.navigate(scrAgent)
			}},
			{"11-service", func() { ui.navigate(scrService) }},
			{"12-instructions", func() { ui.navigate(scrInstructions) }},
			{"13-result-success", func() {
				ui.success = true
				ui.checkedKey = "id_ed25519"
				ui.lastTest = core.HostResult{
					OK:     true,
					Host:   "github.com",
					Output: "OK (port 22)\nHi jai! You've successfully authenticated, but GitHub does not provide shell access.",
				}
				ui.navigate(scrResult)
			}},
			{"14-result-denied", func() {
				ui.success = false
				ui.checkedKey = "id_github_work"
				ui.lastTest = core.HostResult{
					Host:   "github.com",
					Output: "--- port 22 ---\ngit@github.com: Permission denied (publickey).",
				}
				ui.navigate(scrResult)
			}},
			{"15-result-blocked", func() {
				ui.success = false
				ui.checkedKey = "id_rsa_work"
				ui.lastTest = core.HostResult{
					Host: "github.com",
					Output: "--- port 22 ---\nssh: connect to host github.com port 22: Operation timed out\n\n" +
						"--- port 443 (ssh.github.com) ---\nssh: connect to host ssh.github.com port 443: Connection refused",
				}
				ui.navigate(scrResult)
			}},
			{"16-empty-keys", func() {
				// Point at an empty home, then restore it so the scenes after
				// this one still have the seeded keys.
				original := os.Getenv("HOME")
				t.Setenv("HOME", t.TempDir())
				t.Setenv("USERPROFILE", os.Getenv("HOME"))
				ui.navigate(scrKeys)
				ui.win.Content().MinSize()
				t.Setenv("HOME", original)
				t.Setenv("USERPROFILE", original)
			}},
			{"17-status-error", func() {
				ui.navigate(scrKeys)
				ui.setStatus("No clipboard tool found (install xclip, wl-copy, or xsel).", toneDanger)
			}},
		}

		for _, scene := range scenes {
			canvas := win.Canvas()
			win.Resize(size)
			scene.setup()
			if canvas.Content() != nil {
				canvas.Content().Resize(canvas.Size())
			}
			path := filepath.Join(*shotsDir, scene.name+"-"+variant.name+".png")
			f, err := os.Create(path)
			if err != nil {
				t.Fatalf("create %s: %v", path, err)
			}
			if err := png.Encode(f, canvas.Capture()); err != nil {
				t.Fatalf("encode %s: %v", path, err)
			}
			f.Close()
			t.Logf("wrote %s", path)
		}
	}
}

// seedRealisticKeys writes a few plausible key pairs plus one that has a
// deliberately long name, so elision and monospace rendering are exercised.
func seedRealisticKeys(t *testing.T) {
	t.Helper()
	dir := core.SSHDir()
	if err := mkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	pubs := map[string]string{
		"id_ed25519":  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHq0m2XnR8vKk1pQfZ7yL3dJ5sW9aBcDeFgHiJkLmNoP you@laptop",
		"id_rsa_work": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7x2vN8kLmR4tY6uI0oP2aS4dF6gH8jK1lZ3xC5vB7nM9qR2sT4uV6wX8yZ0aB1cD3eF5gH7jK9lZ1xC3vB5nM7qR9s work@box",
		"a_very_long_key_name_for_testing_elision_behaviour": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG5nR7tY2uI4oP6aS8dF0gH2jK4lZ6xC8vB0nM2qR4sT6uV8w long@name",
	}
	for name, pub := range pubs {
		if err := writeFile(filepath.Join(dir, name), "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----\n"); err != nil {
			t.Fatal(err)
		}
		if err := writeFile(filepath.Join(dir, name+".pub"), pub+"\n"); err != nil {
			t.Fatal(err)
		}
	}
}
