package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"keysmith/internal/core"
)

// Run starts the terminal wizard.
func Run() error {
	m := newModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(m.currentView())
	if m.busy {
		b.WriteString("\n")
		b.WriteString(styleBusy.Render(fmt.Sprintf("  %s %s", m.spinner.View(), m.busyMsg)))
		b.WriteString(styleSubtitle.Render("   (esc cancels)"))
	}
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(styleDanger.Render("  ✗ " + m.errMsg))
	}
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	return b.String()
}

// header draws the workbench brand band: wordmark, tagline, step tracker,
// and a heavy copper rule.
func (m model) header() string {
	step := ""
	switch m.screen {
	case scrForm, scrBrowser:
		step = "1 · THE KEY"
	case scrKeyReady, scrService, scrInstructions:
		step = "2 · THE SERVICE"
	case scrResult:
		step = "3 · PROOF"
	}

	wordmark := styleTitle.Render("KEYSMITH")
	tagline := styleSubtitle.Render("the keysmith's bench")
	tracker := ""
	if step != "" {
		tracker = styleTracker.Render("STEP " + step)
	}
	gap := m.width - lipgloss.Width(wordmark) - lipgloss.Width(tagline) - lipgloss.Width(tracker) - 4
	if gap < 1 {
		gap = 1
	}
	top := fmt.Sprintf(" %s  %s%s%s", wordmark, tagline, strings.Repeat(" ", gap), tracker)

	ruleW := m.width - 2
	if ruleW > 90 {
		ruleW = 90
	}
	if ruleW < 20 {
		ruleW = 20
	}
	rule := styleRuleAccent.Render(strings.Repeat("━", ruleW))

	sub := m.headerSubtitle()
	out := top + "\n" + rule
	if sub != "" {
		out += "\n " + styleSubtitle.Render(sub)
	}
	return out
}

func (m model) headerSubtitle() string {
	switch m.screen {
	case scrHome:
		return "What would you like to do?"
	case scrForm:
		return "Choose the metal and mark your key. Defaults are fine."
	case scrBrowser:
		if m.browserPick {
			return "Pick the key you want to test, then press enter."
		}
		return "Your key wall. Enter opens a key's actions."
	case scrKeyReady:
		return "Fresh from the forge. Copy it, load it, or wire it to a service."
	case scrService:
		return "Which door does this key open?"
	case scrInstructions:
		return "Hand the public half to " + m.svc.Name + ". Your secret half never leaves this machine."
	case scrResult:
		if m.success {
			return "The lock turned."
		}
		return "Not yet. Read the diagnosis below and try again."
	}
	return ""
}

// statusBar: thin rule with the latest message underneath.
func (m model) statusBar() string {
	ruleW := m.width - 2
	if ruleW > 90 {
		ruleW = 90
	}
	if ruleW < 20 {
		ruleW = 20
	}
	line := styleRule.Render(strings.Repeat("─", ruleW))
	msg := m.errMsg
	st := styleDanger
	if msg == "" {
		msg = m.status
		st = styleStatus
	}
	if msg == "" {
		return line
	}
	return line + "\n " + st.Render(elideMiddle(msg, maxInt(30, ruleW-4)))
}

func (m model) currentView() string {
	switch m.screen {
	case scrHome:
		return m.viewHome()
	case scrForm:
		return m.viewForm()
	case scrBrowser:
		return m.viewBrowser()
	case scrKeyReady:
		return m.viewKeyReady()
	case scrService:
		return m.viewService()
	case scrInstructions:
		return m.viewInstructions()
	case scrResult:
		return m.viewResult()
	}
	return ""
}

// --- HOME: three slab buttons ---------------------------------------------

func (m model) viewHome() string {
	labels := []string{
		"Set up a new SSH key",
		"Manage existing keys",
		"Test a connection to a Git service",
	}
	var b strings.Builder
	b.WriteString("\n")
	for i, label := range labels {
		b.WriteString("  ")
		b.WriteString(actionButton(fmt.Sprintf("%d  %s", i+1, label), m.menuIdx == i))
		b.WriteString("\n\n")
	}
	b.WriteString("\n  " + keycap("↑↓") + " move   " + keycap("1-3") + " jump   " + keycap("enter") + " choose   " + keycap("q") + " quit")
	return b.String()
}

// --- FORM -------------------------------------------------------------------

func (m model) viewForm() string {
	algoRow := ""
	for i, a := range algos {
		cell := " " + string(a) + " "
		if i == m.algoIdx {
			algoRow += styleBtnFocus.Render(cell)
		} else if m.formPos == fAlgo {
			algoRow += styleRuleAccent.Render(cell)
		} else {
			algoRow += styleSubtitle.Render(cell)
		}
	}

	echo := "••••••"
	if m.showPass {
		echo = "(shown)"
	}

	focusMark := func(pos int) string {
		if m.formPos == pos {
			return styleStepDone.Render(" ▸ ")
		}
		return "   "
	}
	label := func(s string) string { return styleMono.Render(fmt.Sprintf("%-11s", s)) }
	check := func(on bool, pos int, text string) string {
		box := "[ ]"
		if on {
			box = "[■]"
		}
		line := focusMark(pos) + box + " " + text
		if m.formPos == pos {
			return styleRuleAccent.Render(line)
		}
		return styleSubtitle.Render(line)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(focusMark(fName) + label("Name") + m.name.View())
	b.WriteString("\n")
	b.WriteString(focusMark(fAlgo) + label("Type") + algoRow)
	b.WriteString("\n")
	b.WriteString(focusMark(fComment) + label("Comment") + m.comment.View())
	b.WriteString("\n")
	b.WriteString(focusMark(fPass) + label("Passphrase") + "[" + echo + "]  " + styleSubtitle.Render("(optional)"))
	b.WriteString("\n")
	b.WriteString(focusMark(fConfirm) + label("Confirm") + "[" + echo + "]")
	b.WriteString("\n\n")
	b.WriteString(check(m.showPass, fShow, "show passphrases"))
	b.WriteString("\n")
	b.WriteString(check(m.force, fForce, "overwrite an existing key of this name"))
	b.WriteString("\n\n  ")
	b.WriteString(keycap("tab") + " next field   " + keycap("enter") + " strike the key   " + keycap("esc") + " back")
	return b.String()
}

// --- BROWSER -----------------------------------------------------------------

func (m model) viewBrowser() string {
	if len(m.keys) == 0 {
		return "\n  " + styleSubtitle.Render("No keys on the wall yet.") +
			"\n  " + styleSubtitle.Render("Choose \"Set up a new SSH key\" from home.") +
			"\n\n  " + keycap("esc") + " back"
	}
	var b strings.Builder
	b.WriteString("\n")
	for i, k := range m.keys {
		name := elideMiddle(k.Name, maxInt(20, m.width/2-8))
		line := menuLine(i, m.menuIdx, name)
		if k.Name == m.newKey {
			line += " " + styleBadgeOn.Render(" NEW ")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	k := m.keys[m.menuIdx]
	b.WriteString("\n  " + styleBoxTitle.Render("DETAILS — "+k.Name) + "\n")
	fp := core.Fingerprint(k.Name)
	if fp != "" {
		b.WriteString("  " + styleMono.Render(elideMiddle(fp, maxInt(30, m.width-10))) + "\n")
	}
	b.WriteString("  ")
	b.WriteString(badge(m.store.AgentLoadedKeys[k.Name], " IN AGENT "))
	b.WriteString(" ")
	b.WriteString(badge(m.store.TestedKeysOK[k.Name], " TESTED OK "))
	b.WriteString("\n\n  ")
	if m.browserPick {
		b.WriteString(keycap("enter") + " test this key   " + keycap("esc") + " back")
	} else {
		b.WriteString(keycap("enter") + " connect to a service   " + keycap("d d") + " delete   " + keycap("esc") + " back")
	}
	return b.String()
}

// --- KEY READY ------------------------------------------------------------------

func (m model) viewKeyReady() string {
	name := m.newKey
	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(styleBadgeSuccess.Render(" ✓ KEY CREATED "))
	b.WriteString("\n\n  ")
	b.WriteString(styleTitle.Render(name))
	pub := core.PublicKey(name)
	if pub != "" {
		lines := strings.Split(pub, "\n")
		shown := lines[0]
		if len(shown) > m.width-12 && m.width > 24 {
			shown = shown[:m.width-15] + "…"
		}
		b.WriteString("\n  " + styleMono.Render(shown))
	}
	b.WriteString("\n\n")

	done := func(on bool) string {
		if on {
			return styleBadgeSuccess.Render(" done ")
		}
		return styleBadgeOff.Render(" todo ")
	}
	b.WriteString("  " + done(true) + " Key forged in ~/.ssh\n")
	agentState := m.store.AgentLoadedKeys[name]
	b.WriteString("  " + done(agentState) + " Loaded into the agent (needed only for passphrase keys)\n\n")
	b.WriteString("  " + keycap("c") + " copy public key    ")
	b.WriteString(keycap("a") + " add to agent\n  ")
	b.WriteString(keycap("s") + " set up a Git service now    ")
	b.WriteString(keycap("h") + " home")
	return b.String()
}

// --- SERVICE PICK -------------------------------------------------------------------

func (m model) viewService() string {
	var b strings.Builder
	b.WriteString("\n")
	for i, s := range core.Services {
		b.WriteString("  ")
		b.WriteString(menuLine(m.menuIdx, i, fmt.Sprintf("%d  %s — %s", i+1, s.Name, s.Host)))
		b.WriteString("\n")
	}
	b.WriteString("  ")
	b.WriteString(menuLine(m.menuIdx, len(core.Services), "4  Skip for now"))
	b.WriteString("\n\n  ")
	b.WriteString(keycap("↑↓") + " move   " + keycap("1-4") + " jump   " + keycap("enter") + " choose   " + keycap("esc") + " back")
	return b.String()
}

// --- INSTRUCTIONS ----------------------------------------------------------------------

func (m model) viewInstructions() string {
	boxed := styleBox.Render(wrapBlock(m.svc.InstructionText(), m.width))
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(boxed)
	b.WriteString("\n\n  ")
	b.WriteString(keycap("o") + " open " + m.svc.Name + " page (key copied first)\n  ")
	b.WriteString(keycap("t") + " I added it — test the connection\n  ")
	b.WriteString(keycap("c") + " copy public key again   ")
	b.WriteString(keycap("esc") + " back")
	return b.String()
}

// --- RESULT --------------------------------------------------------------------------------

func (m model) viewResult() string {
	var b strings.Builder
	if m.success {
		b.WriteString("\n  ")
		b.WriteString(styleBadgeSuccess.Render(" ✓ CONNECTED "))
		b.WriteString("\n\n  ")
		b.WriteString(styleTitle.Render("Your key works with " + m.svc.Name + "."))
		detail := firstLineDetail(m.lastTest.Output)
		b.WriteString("\n  " + styleSubtitle.Render(elideMiddle(detail, maxInt(40, m.width-8))))
	} else {
		b.WriteString("\n  ")
		b.WriteString(styleBadgeWarn.Render(" ✗ NOT CONNECTED YET "))
		b.WriteString("\n\n  ")
		b.WriteString(styleTitle.Render("The test against " + m.svc.Name + " failed."))
		b.WriteString("\n  " + styleSubtitle.Render("Most likely cause, and how to fix it:") + "\n")
		for _, dg := range core.Diagnose(m.lastTest.Output) {
			b.WriteString("\n  " + styleDanger.Render("✂ "+dg.Cause) + "\n")
			b.WriteString(styleMono.Render(wrapBlock(dg.Fix, m.width-6)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n  ")
	b.WriteString(keycap("r") + " retry test   " + keycap("i") + " instructions again\n  ")
	if m.success {
		b.WriteString(keycap("n") + " set up another key   ")
	}
	b.WriteString(keycap("h") + " home   " + keycap("q") + " quit")
	return b.String()
}

// --- shared pieces ---------------------------------------------------------------------------

func menuLine(cursor, idx int, label string) string {
	if cursor == idx {
		return styleRuleAccent.Render(" ▸ ") + styleSelectedRow.Foreground(lipgloss.Color(core.ColorInkHover)).Render(label)
	}
	return styleSubtitle.Render("   " + label)
}

func firstLineDetail(out string) string {
	for _, l := range strings.Split(out, "\n") {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "---") {
			return t
		}
	}
	return out
}

// wrapBlock hard-wraps plain text to width w.
func wrapBlock(text string, w int) string {
	if w < 40 {
		w = 40
	}
	if w > 90 {
		w = 90
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		indent := ""
		t := para
		if strings.HasPrefix(t, "  ") {
			indent = "  "
			t = strings.TrimPrefix(t, "  ")
		}
		for len([]rune(t)) > w-len(indent) {
			cut := w - len(indent)
			r := []rune(t)
			for cut > 0 && cut < len(r) && r[cut] != ' ' {
				cut--
			}
			if cut == 0 {
				cut = w - len(indent)
			}
			out = append(out, indent+string(r[:cut]))
			t = strings.TrimLeft(string(r[cut:]), " ")
		}
		out = append(out, indent+t)
	}
	return strings.Join(out, "\n")
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}
