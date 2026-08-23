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
		m.setError("No clipboard tool found (install xclip or wl-copy).")
		return m, clearErrLater()
	}
	cmd := exec.Command(tool, args...)
	cmd.Stdin = strings.NewReader(pub)
	if err := cmd.Run(); err != nil {
		m.setError(fmt.Sprintf("Clipboard write failed: %v", err))
		return m, clearErrLater()
	}
	m.store.CopiedKeys[keyName] = true
	_ = m.store.Save()
	m.setStatus("Public key copied to clipboard")
	return m, nil
}

func (m model) copyPubKeySilent() (tea.Model, tea.Cmd) {
	tm, _ := m.copyPubKey(m.subjectKey())
	return tm, nil
}

func clipboardCmd() (string, []string) {
	if runtime.GOOS == "darwin" {
		return "pbcopy", nil
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
	return m.startOp(opAddAgent, func(ctx context.Context) (core.Result, []core.HostResult) {
		_ = ctx
		return core.AddToAgent(keyName), nil
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
	if !m.confirmDelete {
		m.confirmDelete = true
		m.setError(fmt.Sprintf("Press d again to delete '%s'", key))
		return m, clearConfirmLater()
	}
	m.confirmDelete = false

	_ = core.DeleteKey(key)
	delete(m.store.UsedKeys, key)
	delete(m.store.CopiedKeys, key)
	delete(m.store.TestedKeysOK, key)
	delete(m.store.AgentLoadedKeys, key)
	_ = m.store.Save()
	if m.selected == key {
		m.selected = ""
	}
	m.reloadKeepingCursor()
	m.setStatus(fmt.Sprintf("Deleted '%s'", key))
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
	}
	return m.startOp(opTest, func(ctx context.Context) (core.Result, []core.HostResult) {
		_ = ctx
		r := core.TestService(svc, key)
		return core.Result{}, []core.HostResult{r}
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
