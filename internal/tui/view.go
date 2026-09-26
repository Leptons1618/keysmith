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

// contentWidth is the width the body is laid out to. The header, body and
// status bar all share it so the rules line up.
func (m model) contentWidth() int {
	w := m.width - 2
	if w > 92 {
		w = 92
	}
	if w < 40 {
		w = 40
	}
	return w
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	if m.help {
		return m.helpOverlay()
	}

	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")
	b.WriteString(m.currentView())
	if m.busy {
		b.WriteString("\n")
		b.WriteString(styleBusy.Render("  "+m.spinner.View()+" "+m.busyMsg) +
			styleMuted.Render("   (esc cancels)"))
	}
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle(toneDanger).Render("  ✗ " + m.errMsg))
	}
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	return b.String()
}

// header is the brand band: wordmark, setup step, a rule, and a one-line
// summary of what this screen is for.
func (m model) header() string {
	step := m.stepLabel()

	wordmark := styleTitle.Render("KEYSMITH")
	tagline := styleMuted.Render("ssh key workbench")

	left := " " + wordmark + styleMuted.Render("  ·  ") + tagline
	out := left
	if step != "" {
		gap := m.contentWidth() - lipgloss.Width(left) - lipgloss.Width(step) - 1
		if gap < 1 {
			gap = 1
		}
		out = left + strings.Repeat(" ", gap) + step
	}
	out += "\n" + ruleLine(m.contentWidth())
	if sub := m.headerSubtitle(); sub != "" {
		out += "\n " + styleMuted.Render(elideMiddle(sub, m.contentWidth()-2))
	}
	return out
}

// stepLabel renders the setup progress as a compact tracker. Stages already
// done are ticked, so the position in the flow is always visible.
func (m model) stepLabel() string {
	current := m.stepNumber()
	if current == 0 {
		return ""
	}
	names := [3]string{"key", "service", "verify"}
	var parts []string
	for i, name := range names {
		switch {
		case i+1 < current:
			parts = append(parts, lipgloss.NewStyle().Foreground(colSuccess).Render("✓ "+name))
		case i+1 == current:
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(colAccent).Render("● "+name))
		default:
			parts = append(parts, styleMuted.Render("○ "+name))
		}
	}
	return styleMuted.Render("[") + strings.Join(parts, styleMuted.Render(" · ")) + styleMuted.Render("]")
}

// stepNumber maps the current screen to its place in the setup flow, or 0 when
// the screen is outside the flow.
func (m model) stepNumber() int {
	switch m.screen {
	case scrForm:
		return 1
	case scrKeyReady, scrService, scrInstructions:
		return 2
	case scrResult:
		return 3
	}
	return 0
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
	case scrAgent:
		return "Whether this session can reach the SSH agent, and what it holds."
	}
	return ""
}

// statusBar is a thin rule with the latest message underneath, coloured by
// tone so failures are impossible to miss.
func (m model) statusBar() string {
	line := ruleLine(m.contentWidth())
	msg := m.errMsg
	st := styleStatus
	if msg == "" {
		msg = m.status
		st = statusStyle(m.statusTone)
	} else {
		st = styleStatusErr
	}
	if msg == "" {
		return line
	}
	return line + "\n " + st.Render(elideMiddle(msg, maxInt(30, m.contentWidth()-2)))
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
	case scrAgent:
		return m.viewAgent()
	}
	return ""
}

// --- HOME -------------------------------------------------------------------

func (m model) viewHome() string {
	descriptions := [4]string{
		"Forge a key pair, then wire it to a Git host.",
		"Fingerprints, copy, agent load, delete.",
		"Pick a key and a service; we shake hands.",
		"Can this session reach your agent?",
	}
	labels := [4]string{
		"Set up a new SSH key",
		"Manage existing keys",
		"Test a connection",
		"Check SSH agent",
	}

	var b strings.Builder
	b.WriteString("\n")
	for i, label := range labels {
		active := m.menuIdx == i
		marker := styleMuted.Render("   ")
		if active {
			marker = styleRuleAcc.Render(" ▸ ")
		}
		number := styleMuted.Render(fmt.Sprintf("%d", i+1))
		if active {
			number = styleRuleAcc.Render(fmt.Sprintf("%d", i+1))
		}
		head := marker + styleHead.Render(number) + "  " + styleHead.Render(label)
		if !active {
			head = marker + number + "  " + styleHead.Render(label)
		}
		b.WriteString(head + "\n")
		b.WriteString("     " + styleMuted.Render(descriptions[i]) + "\n\n")
	}
	b.WriteString(hints(
		[2]string{"move", "↑↓"},
		[2]string{"jump", "1-4"},
		[2]string{"choose", "enter"},
		[2]string{"help", "?"},
		[2]string{"quit", "q"},
	))
	return b.String()
}

// --- FORM -------------------------------------------------------------------

func (m model) viewForm() string {
	algoRow := ""
	for i, a := range algos {
		cell := " " + algoLabels[a] + " "
		switch {
		case i == m.algoIdx:
			algoRow += styleBtnFocus.Render(cell)
		case m.formPos == fAlgo:
			algoRow += styleRuleAcc.Render(cell)
		default:
			algoRow += styleMuted.Render(cell)
		}
	}

	passEcho := m.pass.View()
	confirmEcho := m.confirm.View()
	if !m.showPass {
		passEcho = strings.Repeat("•", maxInt(6, minInt(24, lipgloss.Width(m.pass.Value()))))
		confirmEcho = strings.Repeat("•", maxInt(6, minInt(24, lipgloss.Width(m.confirm.Value()))))
	}

	focusMark := func(pos int) string {
		if m.formPos == pos {
			return styleRuleAcc.Render(" ▸ ")
		}
		return "   "
	}
	label := func(s string) string { return styleMuted.Render(fmt.Sprintf("%-12s", s)) }
	check := func(on bool, pos int, text string) string {
		box := "[ ]"
		if on {
			box = "[■]"
		}
		line := focusMark(pos) + box + " " + text
		if m.formPos == pos {
			return styleRuleAcc.Render(line)
		}
		return styleBody.Render(line)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(focusMark(fName) + label("Name") + m.name.View() + "\n")
	b.WriteString(focusMark(fAlgo) + label("Type") + algoRow + "\n")
	b.WriteString(focusMark(fComment) + label("Comment") + m.comment.View() + "\n")
	b.WriteString(focusMark(fPass) + label("Passphrase") + "[" + passEcho + "]  " + styleMuted.Render("(optional)") + "\n")
	b.WriteString(focusMark(fConfirm) + label("Confirm") + "[" + confirmEcho + "]" + "\n\n")
	b.WriteString(check(m.showPass, fShow, "show passphrases") + "\n")
	b.WriteString(check(m.force, fForce, "overwrite an existing key of this name") + "\n\n  ")
	b.WriteString(hints(
		[2]string{"next field", "tab"},
		[2]string{"strike the key", "enter"},
		[2]string{"back", "esc"},
	))
	return b.String()
}

// --- BROWSER -----------------------------------------------------------------

func (m model) viewBrowser() string {
	visible := m.visibleKeys()
	nameColumn := maxInt(20, m.contentWidth()/2-12)
	if len(visible) == 0 {
		if m.filterQuery != "" {
			return "\n  " + styleMuted.Render("No keys match "+m.filterQuery+".") + "\n\n  " +
				hints([2]string{"clear filter", "esc"})
		}
		return "\n  " + styleMuted.Render("No keys on the wall yet.") + "\n  " +
			styleMuted.Render(`Choose "Set up a new SSH key" from home.`) + "\n\n  " +
			hints([2]string{"back", "esc"})
	}

	var b strings.Builder
	b.WriteString("\n")

	// The filter line doubles as a status line when filtering is idle, so the
	// key wall does not grow a permanent empty input.
	if m.filtering {
		b.WriteString("  " + m.filter.View() + "\n\n")
	} else if m.filterQuery != "" {
		b.WriteString("  " + styleMuted.Render("filter: ") + styleBody.Render(m.filterQuery) +
			styleMuted.Render("  ("+itoa(len(visible))+"/"+itoa(len(m.keys))+")") + "\n\n")
	}

	for i, k := range visible {
		line := menuLine(i, m.menuIdx, padRight(elideMiddle(k.Name, nameColumn), nameColumn)) +
			styleMuted.Render(keyKindLabel(k.Name))
		if k.Name == m.newKey {
			line += badgeTone(toneAccent, "NEW")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n  " + detailBox(visible[minInt(m.menuIdx, len(visible)-1)].Name, m.store, m.contentWidth()))

	b.WriteString("\n  ")
	if m.browserPick {
		b.WriteString(hints([2]string{"test this key", "enter"}, [2]string{"back", "esc"}))
	} else {
		b.WriteString(hints(
			[2]string{"connect", "enter"},
			[2]string{"copy", "c"},
			[2]string{"add to agent", "a"},
			[2]string{"delete", "d d"},
			[2]string{"filter", "/"},
			[2]string{"back", "esc"},
		))
	}
	return b.String()
}

// detailBox renders the key wall's detail panel: identity first, then the
// workflow state, each on its own line so the panel reads the same way in any
// terminal.
func detailBox(name string, store *core.Store, width int) string {
	// The rendered box adds a border and padding on top of the content width,
	// and the caller indents it, so leave room for all three.
	inner := maxInt(28, width-12)

	fp := core.ShortFingerprint(core.Fingerprint(name))
	if fp == "" {
		fp = "fingerprint unavailable"
	}

	var badges []string
	if store.AgentLoadedKeys[name] {
		badges = append(badges, badgeTone(toneSuccess, "IN AGENT"))
	} else {
		badges = append(badges, badgeTone(toneNeutral, "NOT IN AGENT"))
	}
	if store.TestedKeysOK[name] {
		badges = append(badges, badgeTone(toneSuccess, "TESTED OK"))
	} else if v, seen := store.TestedKeysOK[name]; seen && !v {
		badges = append(badges, badgeTone(toneDanger, "TEST FAILED"))
	} else {
		badges = append(badges, badgeTone(toneNeutral, "UNTESTED"))
	}

	body := strings.Join([]string{
		styleHead.Render(name),
		styleMuted.Render(keyPathLabel(name) + "  ·  " + keyKindLabel(name)),
		styleMono.Render(elideMiddle(fp, inner-2)),
		"",
		strings.Join(badges, "  "),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1).
		Width(inner).
		Render(body)
}

// padRight pads on the right so a column of labels lines up.
func padRight(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// indentBlock wraps text to a width and indents every line, so body copy under
// a heading reads as belonging to it instead of running back to column zero.
func indentBlock(text string, w int, indent string) string {
	wrapped := wrapBlock(text, w)
	lines := strings.Split(wrapped, "\n")
	for i, l := range lines {
		lines[i] = indent + l
	}
	return strings.Join(lines, "\n") + "\n"
}

// keyKindLabel is the algorithm shown under a key, or a dash when it cannot be
// determined.
func keyKindLabel(name string) string {
	if kind, ok := core.KeyKind(name); ok {
		return kind
	}
	return "—"
}

// keyPathLabel is the shortened location of a key.
func keyPathLabel(name string) string { return "~/.ssh/" + name }

// --- KEY READY ------------------------------------------------------------------

func (m model) viewKeyReady() string {
	name := m.newKey
	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(badgeTone(toneSuccess, "✓ KEY CREATED"))
	b.WriteString("\n\n  ")
	b.WriteString(styleHead.Render(name))
	b.WriteString("  " + styleMuted.Render(keyKindLabel(name)))
	if pub := core.PublicKey(name); pub != "" {
		b.WriteString("\n  " + styleMono.Render(elideMiddle(firstLine(pub), maxInt(30, m.contentWidth()-6))))
	}
	b.WriteString("\n\n")

	done := func(on bool) string {
		if on {
			return dot(toneSuccess) + " "
		}
		return emptyDot + " "
	}
	b.WriteString("  " + done(true) + "Key forged in ~/.ssh\n")
	b.WriteString("  " + done(m.store.AgentLoadedKeys[name]) + "Loaded into the agent (needed only for passphrase keys)\n")
	b.WriteString("  " + done(m.store.CopiedKeys[name]) + "Public key copied to the clipboard\n\n")
	b.WriteString("  " + hints(
		[2]string{"copy public key", "c"},
		[2]string{"add to agent", "a"},
	))
	b.WriteString("\n  " + hints(
		[2]string{"set up a Git service", "s"},
		[2]string{"home", "h"},
	))
	return b.String()
}

// --- SERVICE PICK -------------------------------------------------------------------

func (m model) viewService() string {
	var b strings.Builder
	b.WriteString("\n")
	for i, s := range core.Services {
		marker := styleMuted.Render("   ")
		if m.menuIdx == i {
			marker = styleRuleAcc.Render(" ▸ ")
		}
		name := fmt.Sprintf("%d  %s", i+1, s.Name)
		host := styleMuted.Render(" — git@" + s.Host)
		if m.menuIdx == i {
			b.WriteString(marker + styleHead.Render(name) + host + "\n")
			if s.AltHost != "" {
				b.WriteString("      " + styleMuted.Render("port 443 fallback: "+s.AltHost) + "\n")
			}
		} else {
			b.WriteString(marker + styleHead.Render(name) + host + "\n")
		}
	}
	b.WriteString(menuLine(m.menuIdx, len(core.Services), "4  Skip for now") + "\n\n  ")
	b.WriteString(hints(
		[2]string{"move", "↑↓"},
		[2]string{"jump", "1-4"},
		[2]string{"choose", "enter"},
		[2]string{"back", "esc"},
	))
	return b.String()
}

// --- INSTRUCTIONS ----------------------------------------------------------------------

func (m model) viewInstructions() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(styleBox.Width(m.contentWidth() - 4).Render(
		wrapBlock(m.svc.InstructionText(), m.contentWidth()-6)))
	b.WriteString("\n\n  ")
	b.WriteString(hints(
		[2]string{"open " + m.svc.Name + " page (key copied first)", "o"},
	))
	b.WriteString("\n  " + hints(
		[2]string{"test the connection", "t"},
		[2]string{"copy public key", "c"},
		[2]string{"mark as added", "u"},
		[2]string{"back", "esc"},
	))
	return b.String()
}

// --- AGENT --------------------------------------------------------------------------------

// viewAgent reports agent reachability and, when it answers, what it holds.
func (m model) viewAgent() string {
	var b strings.Builder
	b.WriteString("\n  " + sectionLabel("Status") + "\n\n  ")

	if !m.agentChecked {
		b.WriteString(styleMuted.Render("Not checked yet. Press c to probe the agent.") + "\n")
	} else if m.agent.Reachable {
		loaded := "no keys loaded"
		switch n := len(m.agent.Keys); {
		case n == 1:
			loaded = "1 key loaded"
		case n > 1:
			loaded = itoa(n) + " keys loaded"
		}
		b.WriteString(badgeTone(toneSuccess, "AGENT REACHABLE") + "  " +
			styleMuted.Render(loaded) + "\n\n")
		b.WriteString("  " + styleMuted.Render(
			"ssh-add answered, so this session can use keys held by the agent.") + "\n")

		if len(m.agent.Keys) > 0 {
			b.WriteString("\n  " + sectionLabel("Loaded keys") + "\n")
			for _, k := range m.agent.Keys {
				comment := k.Comment
				if comment == "" {
					comment = "no comment"
				}
				kind := k.Kind
				if kind == "" {
					kind = "KEY"
				}
				b.WriteString("  " + styleMono.Render(elideMiddle(k.Fingerprint, 44)) + "\n")
				b.WriteString("    " + styleMuted.Render(comment) + "  " + badgeTone(toneNeutral, kind) + "\n")
			}
		}
	} else {
		b.WriteString(badgeTone(toneWarning, "AGENT UNREACHABLE") + "\n\n")
		b.WriteString(indentBlock(
			"This session cannot talk to an SSH agent. Run eval $(ssh-agent) in your terminal, "+
				"or start your desktop key manager, then check again. Testing a key still works "+
				"without an agent — the agent only matters when your key has a passphrase.",
			m.contentWidth()-4, "  "))
	}

	b.WriteString("\n  " + hints([2]string{"check now", "c"}, [2]string{"back", "esc"}))
	return b.String()
}

// --- RESULT --------------------------------------------------------------------------------

func (m model) viewResult() string {
	var b strings.Builder
	if m.success {
		b.WriteString("\n  " + badgeTone(toneSuccess, "✓ CONNECTED") + "\n\n  ")
		b.WriteString(styleHead.Render("Your key works with "+m.svc.Name+".") + "\n")
		b.WriteString("  " + styleMuted.Render(elideMiddle(firstLineDetail(m.lastTest.Output), maxInt(40, m.contentWidth()-8))) + "\n")
	} else {
		b.WriteString("\n  " + badgeTone(toneDanger, "✗ NOT CONNECTED") + "\n\n  ")
		b.WriteString(styleHead.Render("The test against "+m.svc.Name+" failed.") + "\n")
		b.WriteString("  " + styleMuted.Render("Most likely cause, and how to fix it:") + "\n")
		for _, dg := range core.Diagnose(m.lastTest.Output) {
			b.WriteString("\n  " + styleTitle.Render(elideMiddle(dg.Cause, m.contentWidth()-4)) + "\n")
			b.WriteString(indentBlock(dg.Fix, m.contentWidth()-4, "  "))
		}
	}
	b.WriteString("\n  " + hints(
		[2]string{"retry test", "r"},
		[2]string{"instructions again", "i"},
		[2]string{"home", "h"},
	))
	if m.success {
		b.WriteString("\n  " + hints([2]string{"set up another key", "n"}))
	}
	return b.String()
}

// --- HELP -----------------------------------------------------------------------------------

// helpRows are the bindings for each screen. Keeping them next to the views
// means a new binding is documented where it is implemented.
var helpRows = map[helpTopic][][2]string{
	helpGlobal: {
		{"move between screens", "esc"},
		{"show this help", "?"},
		{"quit", "q"},
		{"cancel a running operation", "esc"},
	},
	helpHome: {
		{"move", "↑ ↓ / j k"},
		{"jump to an action", "1 - 4"},
		{"choose", "enter"},
		{"quit", "q"},
	},
	helpForm: {
		{"next / previous field", "tab / shift+tab"},
		{"edit the current field", "type"},
		{"toggle a checkbox", "space"},
		{"choose an algorithm", "← → / h l"},
		{"create the key", "enter"},
		{"back", "esc"},
	},
	helpBrowser: {
		{"move", "↑ ↓ / j k"},
		{"connect this key to a service", "enter"},
		{"copy the public key", "c"},
		{"load into the agent", "a"},
		{"delete (press twice)", "d d"},
		{"filter keys", "/"},
		{"back", "esc"},
	},
	helpService: {
		{"move", "↑ ↓ / j k"},
		{"jump to a service", "1 - 4"},
		{"choose", "enter"},
		{"back", "esc"},
	},
	helpInstructions: {
		{"open the service's key page", "o"},
		{"test the connection", "t"},
		{"copy the public key", "c"},
		{"mark the key as added", "u"},
		{"back", "esc"},
	},
	helpResult: {
		{"retry the test", "r"},
		{"see the instructions again", "i"},
		{"set up another key", "n"},
		{"home", "h"},
		{"quit", "q"},
	},
}

// helpTopicFor picks the binding list that matches the current screen.
func (m model) helpTopicFor() helpTopic {
	switch m.screen {
	case scrHome:
		return helpHome
	case scrForm:
		return helpForm
	case scrBrowser:
		return helpBrowser
	case scrService:
		return helpService
	case scrInstructions:
		return helpInstructions
	case scrResult:
		return helpResult
	}
	return helpGlobal
}

// helpOverlay is a modal cheat sheet. It is a real screen rather than a footer
// so the whole terminal can be dedicated to it on small windows.
func (m model) helpOverlay() string {
	topic := m.helpTopic
	rows := helpRows[topic]
	other := map[helpTopic]string{
		helpHome:         "form",
		helpForm:         "keys",
		helpBrowser:      "service",
		helpService:      "instructions",
		helpInstructions: "result",
		helpResult:       "global",
	}[topic]

	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("KEYSMITH") + styleMuted.Render("  ·  shortcuts") + "\n")
	b.WriteString("  " + ruleLine(m.contentWidth()-2) + "\n\n")

	b.WriteString("  " + styleLabel.Render("  GLOBAL") + "\n")
	for _, r := range helpRows[helpGlobal] {
		b.WriteString("  " + styleMuted.Render(fmt.Sprintf("%-32s", r[0])) + styleMono.Render(r[1]) + "\n")
	}

	if topic != helpGlobal {
		b.WriteString("\n  " + styleLabel.Render("  THIS SCREEN") + "\n")
		for _, r := range rows {
			b.WriteString("  " + styleMuted.Render(fmt.Sprintf("%-32s", r[0])) + styleMono.Render(r[1]) + "\n")
		}
		b.WriteString("\n  " + styleMuted.Render("Press ") + keycap("t") + styleMuted.Render(" for "+other+" shortcuts."))
	}

	b.WriteString("\n\n  " + hints([2]string{"close", "?"}, [2]string{"close", "esc"}, [2]string{"close", "q"}))
	return b.String()
}

// --- shared pieces ---------------------------------------------------------------------------

func menuLine(cursor, idx int, label string) string {
	if cursor == idx {
		return styleRuleAcc.Render(" ▸ ") + styleSelected.Render(label)
	}
	return "   " + styleBody.Render(label)
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

// vStack joins blocks vertically with a blank line between them.
func vStack(gap int, blocks ...string) string {
	sep := strings.Repeat("\n", gap+1)
	return strings.Join(blocks, sep)
}

// itoa keeps int-to-string call sites readable inside string concatenation.
func itoa(i int) string { return fmt.Sprint(i) }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
