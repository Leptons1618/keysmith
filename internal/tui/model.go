package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

type screen int

const (
	scrHome screen = iota
	scrForm
	scrBrowser
	scrKeyReady
	scrService
	scrInstructions
	scrResult
	scrAgent
)

// form focus order
const (
	fName = iota
	fAlgo
	fComment
	fPass
	fConfirm
	fShow
	fForce
	fFormCount
)

type opKind int

const (
	opGenerate opKind = iota
	opAddAgent
	opCheckAgent
	opTest
)

// helpTopic identifies which key bindings the help overlay should describe.
type helpTopic int

const (
	helpGlobal helpTopic = iota
	helpHome
	helpForm
	helpBrowser
	helpService
	helpInstructions
	helpResult
)

type model struct {
	screen screen
	nav    []screen // back stack for esc

	keys             []core.KeyInfo
	store            *core.Store
	selected         string // key under cursor in browser / subject of result
	newKey           string // freshly generated key (highlight target)
	confirmDeleteKey string

	algoIdx  int
	name     textinput.Model
	comment  textinput.Model
	pass     textinput.Model
	confirm  textinput.Model
	showPass bool
	force    bool
	formPos  int

	menuIdx int // generic menu cursor (home, browser, service)

	browserPick   bool // browser opened to PICK a key for a test
	confirmDelete bool

	// filter is the incremental key search on the key wall.
	filter      textinput.Model
	filtering   bool
	filterQuery string

	// help is the overlay toggle.
	help      bool
	helpTopic helpTopic

	// visible caches the filtered key list. Reads go through visibleKeys so a
	// caller that set keys without filtering still sees them.
	visible []core.KeyInfo

	// agent holds the last agent inventory.
	agent        core.AgentState
	agentChecked bool

	width  int
	height int

	svc      core.Service
	lastTest core.HostResult
	success  bool

	busy     bool
	busyMsg  string
	cancelCh chan struct{}
	opID     uint64
	spinner  spinner.Model

	status     string
	statusTone tone
	errMsg     string
}

func newModel() model {
	mk := func(placeholder string, hidden bool) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 120
		if hidden {
			ti.EchoMode = textinput.EchoPassword
		}
		ti.Prompt = ""
		ti.Cursor.Style = cursorStyle()
		return ti
	}

	sp := spinner.New(spinner.WithSpinner(spinner.Line))

	filter := textinput.New()
	filter.Placeholder = "filter keys…"
	filter.Prompt = "/ "
	filter.CharLimit = 60
	filter.Cursor.Style = cursorStyle()

	m := model{
		screen:  scrHome,
		store:   core.LoadStore(),
		spinner: sp,
		name:    mk("e.g. id_work_github", false),
		comment: mk("you@laptop", false),
		pass:    mk("passphrase", true),
		confirm: mk("repeat passphrase", true),
		filter:  filter,
	}
	m.name.SetValue("id_ed25519")
	return m
}

// push records the current screen for esc-back.
func (m *model) push(s screen) {
	m.nav = append(m.nav, m.screen)
	if len(m.nav) > 8 {
		m.nav = m.nav[1:]
	}
	m.gotoScreen(s)
}

func (m *model) gotoScreen(s screen) {
	m.screen = s
	if s == scrHome {
		m.browserPick = false
		m.newKey = ""
		m.svc = core.Service{}
		m.lastTest = core.HostResult{}
		m.success = false
	}
}

// back pops the nav stack.
func (m *model) back() {
	if len(m.nav) == 0 {
		m.gotoScreen(scrHome)
		return
	}
	prev := m.nav[len(m.nav)-1]
	m.nav = m.nav[:len(m.nav)-1]
	m.errMsg = ""
	m.screen = prev
	m.menuIdx = 0
	m.filtering = false
	m.filterQuery = ""
	m.filter.SetValue("")
	if prev == scrHome {
		m.browserPick = false
	}
	if prev == scrBrowser {
		m.loadKeys()
	}
}

func (m *model) loadKeys() {
	m.keys = core.ListKeys()
	m.applyFilter("")
	if len(m.visible) == 0 {
		m.selected = ""
		return
	}
	found := -1
	for i, k := range m.visible {
		if k.Name == m.selected {
			found = i
			break
		}
	}
	if found < 0 {
		found = 0
	}
	m.menuIdx = found
	m.selected = m.visible[found].Name
}

// applyFilter recomputes the visible key list from the filter query. An empty
// query shows everything, so search is purely additive.
func (m *model) applyFilter(query string) {
	m.filterQuery = query
	m.visible = filterKeys(m.keys, query)
}

// filterKeys returns the keys whose name contains query, case-insensitively.
// An empty query matches everything.
func filterKeys(keys []core.KeyInfo, query string) []core.KeyInfo {
	if query == "" {
		return keys
	}
	needle := strings.ToLower(query)
	var out []core.KeyInfo
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k.Name), needle) {
			out = append(out, k)
		}
	}
	return out
}

// visibleKeys is the filtered key list the browser renders. It falls back to
// filtering keys directly, so a caller that populated keys without going
// through applyFilter still sees them.
func (m model) visibleKeys() []core.KeyInfo {
	if m.visible != nil {
		return m.visible
	}
	return filterKeys(m.keys, m.filterQuery)
}

func (m *model) reloadKeepingCursor() {
	m.keys = core.ListKeys()
	m.applyFilter(m.filterQuery)
	if m.menuIdx >= len(m.visible) {
		m.menuIdx = maxInt(0, len(m.visible)-1)
	}
	if len(m.visible) > 0 {
		m.selected = m.visible[m.menuIdx].Name
	} else {
		m.selected = ""
	}
}

type clearConfirmMsg struct{}

const confirmTimeout = 5 * time.Second

func clearConfirmLater() tea.Cmd {
	return tea.Tick(confirmTimeout, func(time.Time) tea.Msg {
		return clearConfirmMsg{}
	})
}

func (m *model) setStatus(s string) {
	m.errMsg = ""
	m.status = s
	m.statusTone = toneNeutral
}

// setStatusTone records a status message with an explicit tone, so the status
// bar can colour it the same way the GUI does.
func (m *model) setStatusTone(s string, t tone) {
	m.errMsg = ""
	m.status = s
	m.statusTone = t
}

func (m *model) setError(s string) {
	m.errMsg = s
	m.status = ""
	m.statusTone = toneDanger
}

func elideMiddle(s string, max int) string {
	if max < 5 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	half := (max - 1) / 2
	return string(r[:half]) + "…" + string(r[len(r)-half:])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
