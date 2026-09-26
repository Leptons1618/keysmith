//go:build !tui

package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"keysmith/internal/core"
)

// newTestUI builds an appUI against Fyne's headless driver and a temporary
// HOME, so screen construction can be exercised without a display.
func newTestUI(t *testing.T) *appUI {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	app := fynetest.NewApp()
	app.Settings().SetTheme(&keysmithTheme{Theme: theme.DefaultTheme()})
	win := app.NewWindow("test")
	win.Resize(fyne.NewSize(1180, 780))

	return &appUI{win: win, store: core.LoadStore()}
}

// writeKeyPair creates a fake key pair in the temporary ~/.ssh.
func writeKeyPair(t *testing.T, name string) {
	t.Helper()
	dir := core.SSHDir()
	if err := mkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir+"/"+name, "private"); err != nil {
		t.Fatal(err)
	}
	pub := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI" + name + " " + name + "@test"
	if err := writeFile(dir+"/"+name+".pub", pub); err != nil {
		t.Fatal(err)
	}
}

// TestEveryScreenRenders is the smoke test for the redesign: each page must
// build its object tree, attach it to the window, and lay out without
// panicking. A screen that throws here would blank the app in production.
func TestEveryScreenRenders(t *testing.T) {
	ui := newTestUI(t)
	writeKeyPair(t, "id_ed25519")
	writeKeyPair(t, "id_rsa_work")
	ui.store.MarkCopied("id_ed25519")
	ui.store.MarkAgentLoaded("id_ed25519")
	ui.keys = core.ListKeys()
	ui.selected = "id_ed25519"
	ui.svc = core.Services[0]

	cases := []struct {
		name   string
		screen screen
	}{
		{"overview", scrOverview},
		{"new key", scrNewKey},
		{"keys", scrKeys},
		{"agent", scrAgent},
		{"service", scrService},
		{"instructions", scrInstructions},
		{"result success", scrResult},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Both result states must render; they are different trees.
			if tc.name == "result success" {
				ui.success = true
				ui.lastTest = core.HostResult{OK: true, Output: "OK (port 22)\nHi user! You've successfully authenticated"}
			}
			ui.navigate(tc.screen)
			content := ui.shell.content
			if content == nil {
				t.Fatal("screen built no content")
			}
			if ui.shell.current.title == "" {
				t.Error("screen has no header title")
			}
			// Force a layout pass: this is where a bad MinSize or a nil child
			// in a container would surface.
			ui.win.Content().Resize(fyne.NewSize(1180, 780))
			ui.win.Content().MinSize()
		})
	}
}

// TestFailedResultRendersDiagnosis checks the failure tree, which is the one
// people see most often and the one with the most conditional structure.
func TestFailedResultRendersDiagnosis(t *testing.T) {
	ui := newTestUI(t)
	writeKeyPair(t, "work")
	ui.store.RecordTest("work", false)
	ui.selected = "work"
	ui.svc = core.Services[0]
	ui.success = false
	ui.lastTest = core.HostResult{
		Host:   "github.com",
		Output: "--- port 22 ---\ngit@github.com: Permission denied (publickey).",
	}

	ui.navigate(scrResult)
	if ui.shell.content == nil {
		t.Fatal("failed result built no content")
	}
	ui.win.Content().MinSize()
}

// TestEmptyStatesRender makes sure the no-keys paths are handled: a new user
// should never see a blank pane.
func TestEmptyStatesRender(t *testing.T) {
	ui := newTestUI(t)
	ui.navigate(scrKeys)
	if ui.shell.content == nil {
		t.Fatal("empty key wall built no content")
	}
	ui.win.Content().MinSize()

	ui.navigate(scrAgent)
	ui.win.Content().MinSize()
}

// TestNavigationReachesEveryPage guards against a page that can never be
// opened, which would leave a dead sidebar entry.
func TestNavigationReachesEveryPage(t *testing.T) {
	ui := newTestUI(t)
	writeKeyPair(t, "work")
	ui.selected = "work"
	ui.svc = core.Services[0]

	for _, item := range navItems {
		ui.navigate(item.screen)
		if ui.screen != item.screen {
			t.Fatalf("navigating to %q left screen %v", item.label, ui.screen)
		}
		if ui.shell == nil || ui.shell.content == nil {
			t.Fatalf("page %q rendered no content", item.label)
		}
	}
}

// TestThemeCoversEverySemanticColor checks the palette resolves in both
// variants. A missing case would fall through to Fyne defaults and produce a
// mismatched surface.
func TestThemeCoversEverySemanticColor(t *testing.T) {
	app := fynetest.NewApp()
	th := &keysmithTheme{Theme: theme.DefaultTheme()}
	app.Settings().SetTheme(th)

	names := []fyne.ThemeColorName{
		keyInk, keyInkBody, keyMuted, keyAccent, keyAccentInk, keyAccentTint,
		keyOnAccent, keySuccess, keySuccessTint, keyDanger, keyDangerTint,
		keyWarning, keyWarningTint, keySurface, keySurfaceAlt, keyCanvas, keyBorder,
	}
	fyneNames := []fyne.ThemeColorName{
		theme.ColorNameBackground, theme.ColorNameForeground, theme.ColorNamePrimary,
		theme.ColorNameError, theme.ColorNameWarning, theme.ColorNameSuccess,
		theme.ColorNameInputBackground, theme.ColorNameButton, theme.ColorNameDisabled,
		theme.ColorNameDisabledButton, theme.ColorNameSeparator, theme.ColorNameSelection,
		theme.ColorNameFocus, theme.ColorNameHover, theme.ColorNamePressed,
		theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground,
		theme.ColorNameShadow, theme.ColorNamePlaceHolder, theme.ColorNameInputBorder,
		theme.ColorNameScrollBar, theme.ColorNameScrollBarBackground,
		theme.ColorNameHyperlink, theme.ColorNameHeaderBackground,
		theme.ColorNameForegroundOnError, theme.ColorNameForegroundOnPrimary,
		theme.ColorNameForegroundOnSuccess, theme.ColorNameForegroundOnWarning,
	}

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		for _, n := range names {
			if isNilColor(th.Color(n, variant)) {
				t.Errorf("%v: semantic color %q unresolved", variant, n)
			}
		}
		for _, n := range fyneNames {
			if isNilColor(th.Color(n, variant)) {
				t.Errorf("%v: fyne color %q unresolved", variant, n)
			}
		}
	}
}

// TestLightAndDarkPalettesDiffer guards against a variant that was copied by
// accident, which would make dark mode render as light.
func TestLightAndDarkPalettesDiffer(t *testing.T) {
	if sameColor(set(theme.VariantLight).canvas, set(theme.VariantDark).canvas) {
		t.Error("canvas color is identical in both variants")
	}
	if sameColor(set(theme.VariantLight).ink, set(theme.VariantDark).ink) {
		t.Error("ink color is identical in both variants")
	}
	if sameColor(set(theme.VariantLight).accent, set(theme.VariantDark).accent) {
		t.Error("accent color is identical in both variants")
	}
}

// TestTonePairsAreDistinct checks the status vocabulary is actually a
// vocabulary: each tone must map to its own background.
func TestTonePairsAreDistinct(t *testing.T) {
	fynetest.NewApp()
	seen := map[string]tone{}
	for _, tn := range []tone{toneNeutral, toneAccent, toneSuccess, toneDanger, toneWarning} {
		fg, bg := tn.pair()
		if fg == "" || bg == "" {
			t.Fatalf("tone %v has an empty color name", tn)
		}
		if fg == bg {
			t.Errorf("tone %v uses the same color for text and background", tn)
		}
		key := string(fg) + "|" + string(bg)
		if other, dup := seen[key]; dup {
			t.Errorf("tone %v has the same colors as tone %v", tn, other)
		}
		seen[key] = tn
	}
}

// TestTapSurfaceFiresAndHighlights covers the interaction primitive the whole
// app is built on: clicking a card must run its action, and selecting must
// switch it to the accent wash.
func TestTapSurfaceFiresAndHighlights(t *testing.T) {
	fynetest.NewApp()
	fired := 0
	s := newTapSurface(caption("body"), func() { fired++ })
	s.Tapped(&fyne.PointEvent{})

	if fired != 1 {
		t.Fatalf("tap fired %d times, want 1", fired)
	}
	if !s.withSelected(true).selected {
		t.Error("withSelected did not mark the surface selected")
	}
	if s.border != keyAccent {
		t.Errorf("selected surface border = %q, want %q", s.border, keyAccent)
	}
}

// TestKeyWallSelectionDrivesDetail checks that clicking a row on the key wall
// changes the subject the detail panel acts on.
func TestKeyWallSelectionDrivesDetail(t *testing.T) {
	ui := newTestUI(t)
	writeKeyPair(t, "alpha")
	writeKeyPair(t, "beta")
	ui.navigate(scrKeys)

	ui.selected = "beta"
	detail := ui.keyDetailObjects()
	if len(detail) == 0 {
		t.Fatal("detail panel built no objects")
	}
	if ui.subjectKey() != "beta" {
		t.Errorf("subject key = %q, want beta", ui.subjectKey())
	}
}

// TestKeyStatusToneReflectsProgress checks the key wall's state colors track
// the workflow markers, which is the whole point of the status dot.
func TestKeyStatusToneReflectsProgress(t *testing.T) {
	ui := newTestUI(t)
	if got := ui.keyTone("k"); got != toneNeutral {
		t.Errorf("untouched key tone = %v, want neutral", got)
	}
	ui.store.MarkCopied("k")
	if got := ui.keyTone("k"); got != toneAccent {
		t.Errorf("copied key tone = %v, want accent", got)
	}
	ui.store.RecordTest("k", true)
	if got := ui.keyTone("k"); got != toneSuccess {
		t.Errorf("verified key tone = %v, want success", got)
	}
	if !strings.Contains(ui.keyStatusLine("k"), "Verified") {
		t.Errorf("status line = %q, want it to mention verification", ui.keyStatusLine("k"))
	}
}

// TestStatusBarReflectsTone checks the persistent feedback strip picks an icon
// to match the message, so success and failure do not look alike.
func TestStatusBarReflectsTone(t *testing.T) {
	ui := newTestUI(t)
	ui.setStatus("all good", toneSuccess)
	if ui.statusBar() == nil {
		t.Error("status bar returned nothing for a set message")
	}
	ui.setStatus("it broke", toneDanger)
	if ui.statusTone != toneDanger {
		t.Error("status tone was not recorded")
	}
	ui.clearStatus()
	if ui.status != "" {
		t.Error("clearStatus left a message behind")
	}
}

// TestWorkflowCounts covers the rail's progress summary.
func TestWorkflowCounts(t *testing.T) {
	store := core.LoadStore()
	store.RecordTest("a", true)
	store.RecordTest("b", false)
	keys := []core.KeyInfo{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	verified, pending := workflowCounts(store, keys)
	if verified != 1 || pending != 2 {
		t.Fatalf("counts = %d verified/%d pending, want 1/2", verified, pending)
	}
	if verifiedLabel(1) != "1 verified" {
		t.Errorf("singular label = %q", verifiedLabel(1))
	}
	if pendingLabel(0) != "0 pending" {
		t.Errorf("plural label = %q", pendingLabel(0))
	}
}
