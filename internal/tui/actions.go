package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

// copyPubKey copies the key's public half to the system clipboard.
func (m model) copyPubKey(keyName string) (tea.Model, tea.Cmd) {
	if keyName == "" {
		m.setError("No key selected.")
		return m, clearErrLater()
	}
	pub := core.PublicKey(keyName)
	if pub == "" {
		m.setError("Could not load the public key.")
		return m, clearErrLater()
	}
	tool, args := clipboardCmd()
	if tool == "" {
		m.setError("No clipboard tool found (install xclip, wl-copy, or xsel).")
		return m, clearErrLater()
	}
	cmd := exec.Command(tool, args...)
	cmd.Stdin = strings.NewReader(pub)
	if err := cmd.Run(); err != nil {
		m.setError(fmt.Sprintf("Clipboard write failed: %v", err))
		return m, clearErrLater()
	}
	if err := m.store.MarkCopied(keyName); err != nil {
		m.setError("Public key copied, but saving key state failed: " + err.Error())
		return m, clearErrLater()
	}
	m.setStatusTone("Public key copied to clipboard", toneSuccess)
	return m, nil
}

func (m model) copyPubKeySilent() (tea.Model, tea.Cmd) {
	tm, _ := m.copyPubKey(m.subjectKey())
	return tm, nil
}

func clipboardCmd() (string, []string) {
	return clipboardCommand(runtime.GOOS)
}

func clipboardCommand(goos string) (string, []string) {
	if goos == "darwin" {
		return "pbcopy", nil
	}
	if goos == "windows" {
		return "clip.exe", nil
	}
	for _, c := range []struct {
		bin  string
		args []string
	}{
		{"wl-copy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	} {
		if _, err := exec.LookPath(c.bin); err == nil {
			return c.bin, c.args
		}
	}
	return "", nil
}

func (m model) addToAgent(keyName string) (tea.Model, tea.Cmd) {
	if keyName == "" {
		m.setError("No key selected.")
		return m, clearErrLater()
	}
	return m.startOp(opAddAgent, func(ctx context.Context) (core.Result, []core.HostResult, core.AgentState) {
		return core.AddToAgentContext(ctx, keyName), nil, core.AgentState{}
	}, "Adding key to SSH agent...")
}

// deleteSelected removes the selected key pair after an inline confirm:
// first d arms, second d within the timeout deletes.
func (m model) deleteSelected() (tea.Model, tea.Cmd) {
	key := m.selected
	if key == "" {
		m.setError("Select a key first.")
		return m, clearErrLater()
	}
	if !m.confirmDelete || m.confirmDeleteKey != key {
		m.confirmDelete = true
		m.confirmDeleteKey = key
		m.setError(fmt.Sprintf("Press d again to delete '%s'", key))
		return m, clearConfirmLater()
	}
	m.confirmDelete = false
	m.confirmDeleteKey = ""

	if err := core.DeleteKey(key); err != nil {
		m.setError(fmt.Sprintf("Could not delete '%s': %v", key, err))
		return m, clearErrLater()
	}
	if err := m.store.ForgetKey(key); err != nil {
		m.setError(fmt.Sprintf("Deleted '%s', but saving key state failed: %v", key, err))
		return m, clearErrLater()
	}
	if m.selected == key {
		m.selected = ""
	}
	m.reloadKeepingCursor()
	m.setStatusTone(fmt.Sprintf("Deleted '%s'", key), toneSuccess)
	return m, nil
}

func (m model) runTest() (tea.Model, tea.Cmd) {
	key := m.subjectKey()
	if key == "" {
		m.setError("Select a key first.")
		return m, clearErrLater()
	}
	svc := m.svc
	if svc.ID == "" {
		svc, _ = core.ServiceByID("github")
		m.svc = svc
	}
	return m.startOp(opTest, func(ctx context.Context) (core.Result, []core.HostResult, core.AgentState) {
		r := core.TestServiceContext(ctx, svc, key)
		return core.Result{}, []core.HostResult{r}, core.AgentState{}
	}, "Testing connection to "+svc.Name+"...")
}

func (m model) runTestAgain() (tea.Model, tea.Cmd) {
	return m.runTest()
}

func openInBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
