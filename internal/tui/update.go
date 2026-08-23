package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

type opDoneMsg struct {
	kind    opKind
	res     core.Result
	results []core.HostResult // opTest carries exactly one
}

type clearErrMsg struct{}

const errTimeout = 4 * time.Second

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case clearErrMsg:
		m.errMsg = ""
		return m, nil

	case clearConfirmMsg:
		m.confirmDelete = false
		if strings.HasPrefix(m.errMsg, "Press d again") {
			m.errMsg = ""
		}
		return m, nil

	case opDoneMsg:
		return m.finishOp(msg), nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Busy: only cancel/quit accepted.
	if m.busy {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q":
			m.cancelOp()
			m.busy = false
			m.setStatus("Cancelled")
		}
		return m, nil
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.screen == scrHome {
			return m, tea.Quit
		}
		m.back()
		return m, nil
	}

	switch m.screen {
	case scrHome:
		return m.updateHome(key)
	case scrForm:
		return m.updateForm(msg)
	case scrBrowser:
		return m.updateBrowser(key)
	case scrKeyReady:
		return m.updateKeyReady(key)
	case scrService:
		return m.updateService(key)
	case scrInstructions:
		return m.updateInstructions(key)
	case scrResult:
		return m.updateResult(key)
	}
	return m, nil
}

// --- HOME ------------------------------------------------------------

func (m model) updateHome(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
		}
	case "down", "j":
		if m.menuIdx < 2 {
			m.menuIdx++
		}
	case "1", "2", "3":
		m.menuIdx = int(key[0] - '1')
	}
	if key == "1" || key == "2" || key == "3" || key == "enter" {
		return m.homeAction()
	}
	return m, nil
}

func (m model) homeAction() (tea.Model, tea.Cmd) {
	switch m.menuIdx {
	case 0:
		m.push(scrForm)
		m.focusFirstFormField()
	case 1:
		m.push(scrBrowser)
		if len(m.keys) == 0 {
			m.setError("No keys yet. Choose \"Set up a new SSH key\" first.")
		}
	case 2:
		m.browserPick = true
		m.push(scrBrowser)
	}
	return m, nil
}

// --- FORM -------------------------------------------------------------

var algos = []core.KeyAlgorithm{core.AlgoEd25519, core.AlgoRSA, core.AlgoECDSA}

func (m *model) focusFirstFormField() {
	m.formPos = fName
	m.blurAllInputs()
	m.name.Focus()
	m.name.CursorEnd()
}

func (m *model) blurAllInputs() {
	m.name.Blur()
	m.comment.Blur()
	m.pass.Blur()
	m.confirm.Blur()
}

func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)

	if isKey && key.String() == "enter" {
		return m.submitGenerate()
	}

	if isKey {
		switch key.String() {
		case "tab":
			m.formPos = (m.formPos + 1) % fFormCount
			m.syncFormFocus()
			return m, nil
		case "shift+tab":
			m.formPos = (m.formPos + fFormCount - 1) % fFormCount
			m.syncFormFocus()
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.formPos {
	case fAlgo:
		if !isKey {
			break
		}
		switch key.String() {
		case "left", "h":
			m.algoIdx = (m.algoIdx + len(algos) - 1) % len(algos)
		case "right", "l":
			m.algoIdx = (m.algoIdx + 1) % len(algos)
		default:
			m.setError("Use ← or → to choose the algorithm")
			cmd = clearErrLater()
		}
	case fName:
		m.name, cmd = m.name.Update(msg)
	case fComment:
		m.comment, cmd = m.comment.Update(msg)
	case fPass:
		m.pass, cmd = m.pass.Update(msg)
	case fConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
	case fShow:
		if isKey && key.String() == " " {
			m.showPass = !m.showPass
		}
	case fForce:
		if isKey && key.String() == " " {
			m.force = !m.force
		}
	}
	return m, cmd
}

func (m *model) syncFormFocus() {
	m.blurAllInputs()
	m.errMsg = ""
	switch m.formPos {
	case fName:
		m.name.Focus()
	case fComment:
		m.comment.Focus()
	case fPass:
		m.pass.Focus()
	case fConfirm:
		m.confirm.Focus()
	}
}

func (m model) submitGenerate() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.name.Value())
	if name == "" {
		m.setError("Key name cannot be empty.")
		return m, clearErrLater()
	}
	if !core.ValidKeyName(name) {
		m.setError("Use only letters, numbers, dot (.), underscore (_) or dash (-).")
		return m, clearErrLater()
	}
	pass := m.pass.Value()
	if pass != m.confirm.Value() {
		m.setError("Passphrase and confirmation do not match.")
		return m, clearErrLater()
	}

	algo := algos[m.algoIdx]
	comment := strings.TrimSpace(m.comment.Value())
	force := m.force

	return m.startOp(opGenerate, func(ctx context.Context) (core.Result, []core.HostResult) {
		_ = ctx
		return core.GenerateKey(algo, name, comment, pass, force), nil
	}, "Generating your key...")
}

// --- BROWSER ----------------------------------------------------------

func (m model) updateBrowser(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
			if len(m.keys) > 0 {
				m.selected = m.keys[m.menuIdx].Name
			}
		}
	case "down", "j":
		if m.menuIdx < len(m.keys)-1 {
			m.menuIdx++
			if len(m.keys) > 0 {
				m.selected = m.keys[m.menuIdx].Name
			}
		}
	case "enter":
		if m.browserPick {
			m.browserPick = false
			m.push(scrService)
			return m, nil
		}
		return m.openActionsMenu()
	case "d":
		return m.deleteSelected()
	}
	return m, nil
}

// openActionsMenu shows the action list for the selected key inline by
// switching menu semantics: reuse service-style numbered choice via a
// dedicated small overlay rendered inside the browser view.
func (m model) openActionsMenu() (tea.Model, tea.Cmd) {
	// Actions are handled directly by keys in the browser footer.
	return m, nil
}

// --- KEY READY ---------------------------------------------------------

func (m model) updateKeyReady(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "c":
		return m.copyPubKey(m.newKey)
	case "a":
		return m.addToAgent(m.newKey)
	case "s":
		m.selected = m.newKey
		m.push(scrService)
	case "h":
		m.gotoScreen(scrHome)
	}
	return m, nil
}

// --- SERVICE ------------------------------------------------------------

func (m model) updateService(key string) (tea.Model, tea.Cmd) {
	n := len(core.Services)
	switch key {
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
		}
	case "down", "j":
		if m.menuIdx < n { // n = last index is Skip
			m.menuIdx++
		}
	}
	pick := -1
	switch key {
	case "enter":
		pick = m.menuIdx
	case "1", "2", "3", "4":
		pick = int(key[0] - '1')
	}
	if pick >= 0 {
		if pick >= n {
			m.gotoScreen(scrHome)
			m.setStatus("Setup skipped.")
			return m, nil
		}
		m.svc = core.Services[pick]
		m.push(scrInstructions)
	}
	return m, nil
}

// --- INSTRUCTIONS -------------------------------------------------------

func (m model) updateInstructions(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "o":
		openInBrowser(m.svc.KeysURL)
		m.setStatus("Opened " + m.svc.Name + " in your browser. The public key was copied to your clipboard.")
		return m.copyPubKeySilent()
	case "t":
		return m.runTest()
	case "c":
		return m.copyPubKey(m.subjectKey())
	}
	return m, nil
}

// --- RESULT --------------------------------------------------------------

func (m model) updateResult(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "r":
		return m.runTestAgain()
	case "i":
		m.push(scrInstructions)
	case "n":
		m.newKey = ""
		m.push(scrForm)
		m.focusFirstFormField()
	case "h", "q":
		m.gotoScreen(scrHome)
		if key == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// subjectKey is the key the current flow operates on.
func (m model) subjectKey() string {
	if m.newKey != "" {
		return m.newKey
	}
	return m.selected
}

// --- async ops ------------------------------------------------------------

func (m model) startOp(kind opKind, fn func(ctx context.Context) (core.Result, []core.HostResult), busyMsg string) (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	ch := make(chan struct{})
	m.busy = true
	m.busyMsg = busyMsg
	m.cancelCh = ch
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		type outcome struct {
			res     core.Result
			results []core.HostResult
		}
		done := make(chan outcome, 1)
		go func() {
			res, results := fn(ctx)
			done <- outcome{res, results}
		}()
		select {
		case <-ch:
			return
		case out := <-done:
			completions <- opDoneMsg{kind: kind, res: out.res, results: out.results}
		}
	}()
	return m, tea.Batch(m.spinner.Tick, awaitCompletion())
}

func (m *model) cancelOp() {
	if m.cancelCh != nil {
		close(m.cancelCh)
		m.cancelCh = nil
	}
}

var completions = make(chan opDoneMsg)

func awaitCompletion() tea.Cmd {
	return func() tea.Msg {
		return <-completions
	}
}

func (m model) finishOp(msg opDoneMsg) model {
	m.busy = false
	m.cancelCh = nil
	switch msg.kind {
	case opGenerate:
		name := strings.TrimSpace(m.name.Value())
		if msg.res.OK {
			m.pass.SetValue("")
			m.confirm.SetValue("")
			m.force = false
			m.newKey = name
			m.selected = name
			m.loadKeys()
			m.gotoScreen(scrKeyReady)
			m.setStatus("Your key is ready.")
		} else {
			m.setError(msg.res.Message)
		}
	case opAddAgent:
		if msg.res.OK {
			key := m.subjectKey()
			m.store.AgentLoadedKeys[key] = true
			_ = m.store.Save()
			m.setStatus(msg.res.Message)
		} else {
			m.setError(msg.res.Message)
		}
	case opTest:
		m.lastTest = msg.results[0]
		m.success = msg.results[0].OK
		key := m.subjectKey()
		m.store.TestedKeysOK[key] = m.success
		_ = m.store.Save()
		if m.success {
			m.gotoScreen(scrResult)
			m.setStatus("Connected to " + m.svc.Name + "!")
		} else {
			m.gotoScreen(scrResult)
		}
	}
	return m
}

func clearErrLater() tea.Cmd {
	return tea.Tick(errTimeout, func(time.Time) tea.Msg {
		return clearErrMsg{}
	})
}
