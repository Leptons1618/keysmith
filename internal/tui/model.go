package tui

import (
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
	opTest
)

type model struct {
	screen screen
	nav    []screen // back stack for esc

	keys     []core.KeyInfo
	store    *core.Store
	selected string // key under cursor in browser / subject of result
	newKey   string // freshly generated key (highlight target)

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

	width  int
	height int

	svc      core.Service
	lastTest core.HostResult
	success  bool

	busy     bool
	busyMsg  string
	cancelCh chan struct{}
	spinner  spinner.Model

	status string
	errMsg string
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
		return ti
	}

	sp := spinner.New(spinner.WithSpinner(spinner.Line))

	m := model{
		screen:  scrHome,
		store:   core.LoadStore(),
		spinner: sp,
		name:    mk("e.g. id_work_github", false),
		comment: mk("you@laptop", false),
		pass:    mk("passphrase", true),
		confirm: mk("repeat passphrase", true),
		status:  "",
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
	m.menuIdx = 0
	m.errMsg = ""
	switch s {
	case scrBrowser:
		m.loadKeys()
	case scrService, scrInstructions:
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
	if prev == scrBrowser {
		m.loadKeys()
	}
}

func (m *model) loadKeys() {
	m.keys = core.ListKeys()
	if len(m.keys) == 0 {
		m.selected = ""
		return
	}
	found := -1
	for i, k := range m.keys {
		if k.Name == m.selected {
			found = i
			break
		}
	}
	if found < 0 {
		found = 0
	}
	m.menuIdx = found
	m.selected = m.keys[found].Name
}

func (m *model) reloadKeepingCursor() {
	old := m.keys
	m.keys = core.ListKeys()
	if m.menuIdx >= len(m.keys) {
		m.menuIdx = maxInt(0, len(m.keys)-1)
	}
	_ = old
	if len(m.keys) > 0 {
		m.selected = m.keys[m.menuIdx].Name
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
	m.status = s
}

func (m *model) setError(s string) {
	m.errMsg = s
	m.status = ""
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

func timestamp() string {
	return time.Now().Format("15:04:05")
}
