package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

type opDoneMsg struct {
	opID    uint64
	kind    opKind
	res     core.Result
	results []core.HostResult // opTest and opCheckAgent carry agent details
	agent   core.AgentState
}

type clearErrMsg struct{}

const errTimeout = 6 * time.Second

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
		m.confirmDeleteKey = ""
		if strings.HasPrefix(m.errMsg, "Press d again") {
			m.errMsg = ""
		}
		return m, nil
	case opDoneMsg:
		if m.busy && msg.opID == m.opID {
			return m.finishOp(msg), nil
		}
		return m, nil
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
			m.setStatusTone("Cancelled", toneWarning)
		}
		return m, nil
	}

	// The help overlay swallows everything except the keys that close it.
	if m.help {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "?", "esc", "q", "enter":
			m.help = false
		case "t":
			m.helpTopic = nextHelpTopic(m.helpTopic)
		}
		return m, nil
	}

	// The key wall's filter owns the keyboard while it is open.
	if m.filtering {
		return m.updateFilter(msg, key)
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "?":
		m.help = true
		m.helpTopic = m.helpTopicFor()
		return m, nil
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
	case scrAgent:
		return m.updateAgent(key)
	}
	return m, nil
}

// nextHelpTopic cycles through the help sections.
func nextHelpTopic(t helpTopic) helpTopic {
	order := []helpTopic{helpGlobal, helpHome, helpForm, helpBrowser, helpService, helpInstructions, helpResult}
	for i, o := range order {
		if o == t {
			return order[(i+1)%len(order)]
		}
	}
	return helpGlobal
}

// updateFilter drives the incremental key search on the key wall.
func (m model) updateFilter(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.filtering = false
		m.applyFilter("")
		m.filter.SetValue("")
		m.restoreSelection()
		return m, nil
	case "enter":
		m.filtering = false
		m.restoreSelection()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	query := m.filter.Value()
	if query != m.filterQuery {
		m.applyFilter(query)
		m.clampCursor()
	}
	return m, cmd
}

// restoreSelection keeps the highlighted key when the filter is dismissed.
func (m *model) restoreSelection() {
	visible := m.visibleKeys()
	m.clampCursor()
	if len(visible) > 0 {
		m.selected = visible[m.menuIdx].Name
	}
}

// clampCursor keeps the cursor inside the visible list.
func (m *model) clampCursor() {
	n := len(m.visibleKeys())
	if m.menuIdx >= n {
		m.menuIdx = maxInt(0, n-1)
	}
	if m.menuIdx < 0 {
		m.menuIdx = 0
	}
}

// --- HOME ------------------------------------------------------------

func (m model) updateHome(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
		}
	case "down", "j":
		if m.menuIdx < 3 {
			m.menuIdx++
		}
	case "1", "2", "3", "4":
		m.menuIdx = int(key[0] - '1')
	}
	if key == "1" || key == "2" || key == "3" || key == "4" || key == "enter" {
		return m.homeAction()
	}
	return m, nil
}

func (m model) homeAction() (tea.Model, tea.Cmd) {
	switch m.menuIdx {
	case 0:
		m.resetForm()
		m.push(scrForm)
		m.focusFirstFormField()
	case 1:
		m.browserPick = false
		m.loadKeys()
		m.push(scrBrowser)
		if len(m.keys) == 0 {
			m.setError("No keys yet. Choose \"Set up a new SSH key\" first.")
		}
	case 2:
		m.browserPick = true
		m.loadKeys()
		m.push(scrBrowser)
		if len(m.keys) == 0 {
			m.setError("No keys yet. Choose \"Set up a new SSH key\" first.")
		}
	case 3:
		m.push(scrAgent)
		return m.checkAgent()
	}
	return m, nil
}

func (m *model) resetForm() {
	m.name.SetValue("id_ed25519")
	m.comment.SetValue("")
	m.pass.SetValue("")
	m.confirm.SetValue("")
	m.pass.EchoMode = textinput.EchoPassword
	m.confirm.EchoMode = textinput.EchoPassword
	m.algoIdx = 0
	m.showPass = false
	m.force = false
}

// --- FORM -------------------------------------------------------------

var algos = []core.KeyAlgorithm{core.AlgoEd25519, core.AlgoRSA, core.AlgoECDSA}

// algoLabels are the display names for the algorithm selector. The raw
// ssh-keygen names read as jargon on a picker.
var algoLabels = map[core.KeyAlgorithm]string{
	core.AlgoEd25519: "Ed25519",
	core.AlgoRSA:     "RSA 4096",
	core.AlgoECDSA:   "ECDSA",
}

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

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	name := msg.String()

	if name == "enter" {
		return m.submitGenerate()
	}

	switch name {
	case "tab":
		m.formPos = (m.formPos + 1) % fFormCount
		m.syncFormFocus()
		return m, nil
	case "shift+tab":
		m.formPos = (m.formPos + fFormCount - 1) % fFormCount
		m.syncFormFocus()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.formPos {
	case fAlgo:
		switch name {
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
		if name == " " {
			m.showPass = !m.showPass
			if m.showPass {
				m.pass.EchoMode = textinput.EchoNormal
				m.confirm.EchoMode = textinput.EchoNormal
			} else {
				m.pass.EchoMode = textinput.EchoPassword
				m.confirm.EchoMode = textinput.EchoPassword
			}
		}
	case fForce:
		if name == " " {
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

	return m.startOp(opGenerate, func(ctx context.Context) (core.Result, []core.HostResult, core.AgentState) {
		return core.GenerateKeyContext(ctx, algo, name, comment, pass, force), nil, core.AgentState{}
	}, "Generating your key...")
}

// --- BROWSER ----------------------------------------------------------

func (m model) updateBrowser(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "/":
		m.filtering = true
		m.filter.SetValue(m.filterQuery)
		m.filter.CursorEnd()
		m.filter.Focus()
		return m, textinput.Blink
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
			m.syncBrowserSelection()
		}
	case "down", "j":
		if m.menuIdx < len(m.visibleKeys())-1 {
			m.menuIdx++
			m.syncBrowserSelection()
		}
	case "enter":
		m.filtering = false
		m.push(scrService)
		return m, nil
	case "c":
		return m.copyPubKey(m.selected)
	case "a":
		return m.addToAgent(m.selected)
	case "d":
		return m.deleteSelected()
	}
	return m, nil
}

// syncBrowserSelection keeps the subject key in step with the cursor.
func (m *model) syncBrowserSelection() {
	visible := m.visibleKeys()
	if len(visible) == 0 {
		m.selected = ""
		return
	}
	m.clampCursor()
	m.selected = visible[m.menuIdx].Name
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
		if m.menuIdx < n { // n is the index of "Skip for now"
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
			m.setStatusTone("Setup skipped.", toneNeutral)
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
		return m.copyPubKeySilent()
	case "t":
		return m.runTest()
	case "c":
		return m.copyPubKey(m.subjectKey())
	case "u":
		if err := m.store.MarkUsed(m.subjectKey()); err != nil {
			m.setError("Could not save service workflow state: " + err.Error())
			return m, clearErrLater()
		}
		m.setStatusTone("Marked key as added to "+m.svc.Name+".", toneSuccess)
		return m, nil
	}
	return m, nil
}

// --- AGENT ---------------------------------------------------------------

func (m model) updateAgent(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "c":
		return m.checkAgent()
	case "h":
		m.gotoScreen(scrHome)
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
		m.resetForm()
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

// subjectKey is the key the current flow operates on. A later browser
// selection always supersedes the generated-key highlight.
func (m model) subjectKey() string {
	return m.selected
}

// --- async ops ------------------------------------------------------------

// startOp runs fn off the UI goroutine, showing a spinner and accepting a
// cancel. onDone is delivered back on the UI goroutine through the model.
func (m model) startOp(
	kind opKind,
	fn func(ctx context.Context) (core.Result, []core.HostResult, core.AgentState),
	busyMsg string,
) (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	ch := make(chan struct{})
	m.busy = true
	m.busyMsg = busyMsg
	m.cancelCh = ch
	m.opID++
	opID := m.opID
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		type outcome struct {
			res     core.Result
			results []core.HostResult
			agent   core.AgentState
		}
		done := make(chan outcome, 1)
		go func() {
			res, results, agent := fn(ctx)
			done <- outcome{res, results, agent}
		}()
		select {
		case <-ch:
			return
		case out := <-done:
			completions <- opDoneMsg{opID: opID, kind: kind, res: out.res, results: out.results, agent: out.agent}
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

// checkAgent probes the agent and records both its reachability and its
// contents, so the agent screen can show real data.
func (m model) checkAgent() (tea.Model, tea.Cmd) {
	return m.startOp(opCheckAgent, func(ctx context.Context) (core.Result, []core.HostResult, core.AgentState) {
		return core.CheckAgentContext(ctx), nil, core.AgentInfo(ctx)
	}, "Checking the SSH agent...")
}

func (m model) finishOp(msg opDoneMsg) model {
	if !m.busy || msg.opID != m.opID {
		return m
	}
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
			m.setStatusTone("Your key is ready.", toneSuccess)
		} else {
			m.setError(msg.res.Message)
		}
	case opAddAgent:
		if !msg.res.OK {
			m.setError(msg.res.Message)
			break
		}
		key := m.subjectKey()
		if err := m.store.MarkAgentLoaded(key); err != nil {
			m.setError("Key added to agent, but saving key state failed: " + err.Error())
			break
		}
		m.agentChecked = false
		m.setStatusTone(msg.res.Message, toneSuccess)
	case opCheckAgent:
		m.agentChecked = true
		m.agent = msg.agent
		if msg.res.OK {
			m.setStatusTone(msg.res.Message, toneSuccess)
		} else {
			m.setStatusTone(msg.res.Message, toneWarning)
		}
		if m.screen != scrAgent {
			m.gotoScreen(scrAgent)
		}
	case opTest:
		if len(msg.results) == 0 {
			m.setError("Connection test returned no result.")
			return m
		}
		m.lastTest = msg.results[0]
		m.success = msg.results[0].OK
		key := m.subjectKey()
		if err := m.store.RecordTest(key, m.success); err != nil {
			m.setError("Test finished, but saving key state failed: " + err.Error())
		}
		m.gotoScreen(scrResult)
		if m.success {
			m.setStatusTone("Connected to "+m.svc.Name+"!", toneSuccess)
		} else {
			m.setStatusTone("Not connected to "+m.svc.Name+". See the diagnosis.", toneWarning)
		}
	}
	return m
}

func clearErrLater() tea.Cmd {
	return tea.Tick(errTimeout, func(time.Time) tea.Msg {
		return clearErrMsg{}
	})
}
