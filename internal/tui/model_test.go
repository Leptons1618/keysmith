package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

func testModel(t *testing.T) model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	m := newModel()
	m.width = 100
	m.height = 40
	return m
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func writeKeyPair(t *testing.T, name string) {
	t.Helper()
	dir := core.SSHDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".pub"), []byte(name+" public"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHomeQQuits(t *testing.T) {
	m := testModel(t)
	_, cmd := m.handleKey(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q on home did not return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q on home command returned %T, want tea.QuitMsg", cmd())
	}
}

func TestExistingKeyEnterOpensService(t *testing.T) {
	m := testModel(t)
	m.screen = scrBrowser
	m.selected = "work"
	m.keys = []core.KeyInfo{{Name: "work"}, {Name: "personal"}}

	updated, _ := m.handleKey(keyMsg("enter"))
	got := updated.(model)
	if got.screen != scrService {
		t.Fatalf("enter on existing key opened %v, want service screen", got.screen)
	}
	if got.selected != "work" {
		t.Fatalf("enter changed selected key to %q", got.selected)
	}
}

func TestManagementClearsBrowserPick(t *testing.T) {
	m := testModel(t)
	m.menuIdx = 2
	updated, _ := m.homeAction()
	m = updated.(model)
	if !m.browserPick || m.screen != scrBrowser {
		t.Fatal("test connection did not open browser in pick mode")
	}
	m.back()
	if m.screen != scrHome {
		t.Fatalf("esc from browser returned to %v, want home", m.screen)
	}
	m.menuIdx = 1
	updated, _ = m.homeAction()
	m = updated.(model)
	if m.browserPick {
		t.Fatal("management reused stale browser-pick mode")
	}
}

func TestNewKeyDoesNotOverrideLaterSelection(t *testing.T) {
	m := testModel(t)
	m.newKey = "fresh"
	m.selected = "work"
	if got := m.subjectKey(); got != "work" {
		t.Fatalf("subjectKey = %q, want later selection %q", got, "work")
	}
}

func TestDeleteConfirmationIsBoundToSelectedKey(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "work")
	writeKeyPair(t, "personal")
	m.screen = scrBrowser
	m.menuIdx = 0
	m.selected = "work"
	m.keys = []core.KeyInfo{{Name: "work"}, {Name: "personal"}}

	updated, _ := m.deleteSelected()
	m = updated.(model)
	m.menuIdx = 1
	m.selected = "personal"
	updated, _ = m.deleteSelected()
	m = updated.(model)

	for _, name := range []string{"work", "personal"} {
		if _, err := os.Stat(core.PrivateKeyPath(name)); err != nil {
			t.Fatalf("changing selection during confirmation deleted %q: %v", name, err)
		}
	}
	if !strings.Contains(m.errMsg, "personal") {
		t.Fatalf("new confirmation error = %q, want selected key personal", m.errMsg)
	}
}

func TestDeleteErrorSurfaces(t *testing.T) {
	m := testModel(t)
	dir := core.SSHDir()
	if err := os.MkdirAll(filepath.Join(dir, "blocked"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "blocked", "keep"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "blocked.pub"), []byte("public"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.screen = scrBrowser
	m.selected = "blocked"
	m.keys = []core.KeyInfo{{Name: "blocked"}}

	updated, _ := m.deleteSelected()
	m = updated.(model)
	updated, _ = m.deleteSelected()
	m = updated.(model)
	if m.errMsg == "" || strings.Contains(m.errMsg, "Press d again") || !strings.Contains(strings.ToLower(m.errMsg), "delete") {
		t.Fatalf("delete error = %q, want surfaced delete failure", m.errMsg)
	}
}

func TestSuccessfulStatusClearsStaleError(t *testing.T) {
	m := testModel(t)
	m.setError("old clipboard failure")
	m.setStatus("copied successfully")
	if m.errMsg != "" {
		t.Fatalf("successful status left stale error %q", m.errMsg)
	}
	if m.status != "copied successfully" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestShowPassphrasesChangesEchoAndRendering(t *testing.T) {
	m := testModel(t)
	m.screen = scrForm
	m.formPos = fShow
	m.pass.SetValue("correct horse")
	m.confirm.SetValue("correct horse")

	updated, _ := m.handleKey(keyMsg(" "))
	got := updated.(model)
	if got.pass.EchoMode != textinput.EchoNormal || got.confirm.EchoMode != textinput.EchoNormal {
		t.Fatalf("show passphrases left echo modes %v/%v", got.pass.EchoMode, got.confirm.EchoMode)
	}
	if !strings.Contains(got.View(), "correct horse") {
		t.Fatal("shown passphrase was not rendered")
	}
}

func TestNewKeyFormResetsStaleValues(t *testing.T) {
	m := testModel(t)
	m.name.SetValue("stale-name")
	m.comment.SetValue("stale@example")
	m.algoIdx = 2
	m.force = true
	m.showPass = true
	m.pass.SetValue("old-secret")
	m.confirm.SetValue("old-secret")

	m.menuIdx = 0
	updated, _ := m.homeAction()
	got := updated.(model)
	if got.name.Value() != "id_ed25519" {
		t.Fatalf("new form name = %q, want default id_ed25519", got.name.Value())
	}
	if got.comment.Value() != "" {
		t.Fatalf("new form comment = %q, want empty", got.comment.Value())
	}
	if got.algoIdx != 0 {
		t.Fatalf("new form algorithm index = %d, want ed25519", got.algoIdx)
	}
	if got.pass.Value() != "" || got.confirm.Value() != "" || got.force || got.showPass {
		t.Fatal("new form retained stale passphrase or checkbox state")
	}
}

func TestWindowsClipboardCommand(t *testing.T) {
	tool, _ := clipboardCommand("windows")
	if tool != "clip.exe" {
		t.Fatalf("Windows clipboard command = %q, want clip.exe", tool)
	}
}

func TestHomeExposesCheckAgent(t *testing.T) {
	m := testModel(t)
	m.menuIdx = 3
	updated, _ := m.updateHome("4")
	got := updated.(model)
	if got.menuIdx != 3 {
		t.Fatalf("menu index after selecting fourth action = %d, want 3", got.menuIdx)
	}
}

func TestExistingKeyBrowserAdvertisesActions(t *testing.T) {
	m := testModel(t)
	m.screen = scrBrowser
	m.keys = []core.KeyInfo{{Name: "work"}}
	m.selected = "work"
	view := m.viewBrowser()
	for _, hint := range []string{"copy", "add", "delete"} {
		if !strings.Contains(view, hint) {
			t.Errorf("browser view missing %q action: %s", hint, view)
		}
	}
}

func TestStaleAsyncCompletionIsIgnored(t *testing.T) {
	m := testModel(t)
	m.busy = true
	m.opID = 7
	got := m.finishOp(opDoneMsg{opID: 6, kind: opTest, results: []core.HostResult{{OK: true}}})
	if !got.busy || got.success {
		t.Fatalf("stale completion changed active operation: busy=%v success=%v", got.busy, got.success)
	}
}

func TestInstructionsCanMarkServiceUse(t *testing.T) {
	m := testModel(t)
	m.screen = scrInstructions
	m.selected = "work"
	m.svc = core.Services[0]
	updated, _ := m.updateInstructions("u")
	got := updated.(model)
	if !got.store.UsedKeys["work"] {
		t.Fatal("service-use marker was not persisted")
	}
}

func TestHomeClearsGeneratedKeyState(t *testing.T) {
	m := testModel(t)
	m.newKey = "fresh"
	m.browserPick = true
	m.gotoScreen(scrHome)
	if m.newKey != "" || m.browserPick {
		t.Fatalf("home retained transient key state: new=%q pick=%v", m.newKey, m.browserPick)
	}
}
