package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"keysmith/internal/core"
)

// --- help overlay -------------------------------------------------------------

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	m := testModel(t)
	m.screen = scrBrowser

	updated, _ := m.handleKey(keyMsg("?"))
	got := updated.(model)
	if !got.help {
		t.Fatal("? did not open the help overlay")
	}
	if !strings.Contains(got.View(), "GLOBAL") {
		t.Error("help overlay did not render the global bindings")
	}

	closed, _ := got.handleKey(keyMsg("?"))
	if closed.(model).help {
		t.Error("? did not close the help overlay")
	}
}

func TestHelpOverlayDescribesCurrentScreen(t *testing.T) {
	m := testModel(t)
	m.screen = scrBrowser
	m.keys = []core.KeyInfo{{Name: "work"}}
	m.visible = m.keys
	m.selected = "work"

	updated, _ := m.handleKey(keyMsg("?"))
	view := updated.(model).View()

	if !strings.Contains(view, "THIS SCREEN") {
		t.Fatal("help did not offer screen-specific bindings")
	}
	for _, binding := range []string{"filter", "/", "delete", "d d"} {
		if !strings.Contains(view, binding) {
			t.Errorf("key-wall help missing %q: %s", binding, view)
		}
	}
}

func TestHelpOverlaySwallowsOtherKeys(t *testing.T) {
	m := testModel(t)
	m.screen = scrHome
	updated, _ := m.handleKey(keyMsg("?"))
	m = updated.(model)

	// "3" would normally jump to Test a connection, but help is modal.
	after, _ := m.handleKey(keyMsg("3"))
	if after.(model).screen != scrHome {
		t.Error("a key press changed the screen while help was open")
	}
	if !after.(model).help {
		t.Error("a key press closed help unexpectedly")
	}
}

func TestHelpTopicCycles(t *testing.T) {
	if nextHelpTopic(helpGlobal) != helpHome {
		t.Error("help topic did not advance from global")
	}
	if nextHelpTopic(helpResult) != helpGlobal {
		t.Error("help topic did not wrap around")
	}
}

// --- key wall filter -----------------------------------------------------------

func TestFilterNarrowsAndRestoresKeys(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "alpha")
	writeKeyPair(t, "work")
	m.screen = scrBrowser
	m.loadKeys()
	if len(m.visibleKeys()) != 2 {
		t.Fatalf("loaded %d keys, want 2", len(m.visibleKeys()))
	}

	opened, cmd := m.handleKey(keyMsg("/"))
	m = opened.(model)
	if !m.filtering || cmd == nil {
		t.Fatal("/ did not open the filter")
	}

	// Type "alp" and check the list narrows.
	for _, r := range "alp" {
		typed, _ := m.filter.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m.filter = typed
	}
	m.applyFilter(m.filter.Value())
	if got := len(m.visibleKeys()); got != 1 {
		t.Fatalf("filter matched %d keys, want 1", got)
	}

	// Esc clears the filter and restores the full list.
	cleared, _ := m.handleKey(keyMsg("esc"))
	m = cleared.(model)
	if m.filtering {
		t.Error("esc did not close the filter")
	}
	if got := len(m.visibleKeys()); got != 2 {
		t.Errorf("esc left %d keys visible, want 2", got)
	}
}

func TestFilterReportsNoMatches(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "alpha")
	m.screen = scrBrowser
	m.loadKeys()
	m.applyFilter("nothing-matches-this")

	if !strings.Contains(m.viewBrowser(), "No keys match") {
		t.Error("an empty filter result did not say so")
	}
}

func TestFilterKeepsCursorInRange(t *testing.T) {
	m := testModel(t)
	for _, n := range []string{"a", "ab", "abc", "abcd"} {
		writeKeyPair(t, n)
	}
	m.screen = scrBrowser
	m.loadKeys()
	m.menuIdx = 3

	m.applyFilter("ab") // one match, but the cursor was at 3
	m.restoreSelection()
	if m.menuIdx > len(m.visibleKeys())-1 {
		t.Fatalf("cursor %d is past the %d filtered keys", m.menuIdx, len(m.visibleKeys()))
	}
}

func TestBrowserMovementStaysInsideFilteredList(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "alpha")
	writeKeyPair(t, "work")
	m.screen = scrBrowser
	m.loadKeys()
	m.menuIdx = 0

	// With one visible key, "down" must not run off the end.
	m.applyFilter("alpha")
	for i := 0; i < 5; i++ {
		updated, _ := m.handleKey(keyMsg("down"))
		m = updated.(model)
	}
	if m.selected != "alpha" {
		t.Errorf("selection ran off the filtered list: %q", m.selected)
	}
}

// --- agent screen ---------------------------------------------------------------

func TestAgentScreenReachesFromHome(t *testing.T) {
	m := testModel(t)
	m.menuIdx = 3
	updated, _ := m.homeAction()
	m = updated.(model)

	if m.screen != scrAgent {
		t.Fatalf("home action 4 opened %v, want the agent screen", m.screen)
	}
	if !strings.Contains(m.viewAgent(), "Not checked yet") {
		t.Error("agent screen did not start in the unchecked state")
	}
}

func TestAgentScreenRendersLoadedKeys(t *testing.T) {
	m := testModel(t)
	m.screen = scrAgent
	m.agentChecked = true
	m.agent = core.AgentState{
		Reachable: true,
		Keys: []core.AgentKey{
			{Fingerprint: "SHA256:abc123", Comment: "you@laptop", Kind: "ED25519"},
		},
	}

	view := m.viewAgent()
	for _, want := range []string{"AGENT REACHABLE", "1 key loaded", "you@laptop", "SHA256:abc123"} {
		if !strings.Contains(view, want) {
			t.Errorf("agent view missing %q:\n%s", want, view)
		}
	}
}

func TestAgentScreenExplainsAnUnreachableAgent(t *testing.T) {
	m := testModel(t)
	m.screen = scrAgent
	m.agentChecked = true
	m.agent = core.AgentState{Reachable: false}

	view := m.viewAgent()
	if !strings.Contains(view, "AGENT UNREACHABLE") {
		t.Fatal("unreachable agent was not reported")
	}
	if !strings.Contains(view, "ssh-agent") {
		t.Errorf("unreachable agent gave no way to fix it:\n%s", view)
	}
}

func TestAgentCountIsPlainEnglish(t *testing.T) {
	m := testModel(t)
	m.screen = scrAgent
	m.agentChecked = true

	for _, tc := range []struct {
		count int
		want  string
	}{{0, "no keys loaded"}, {1, "1 key loaded"}, {3, "3 keys loaded"}} {
		m.agent = core.AgentState{
			Reachable: true,
			Keys:      make([]core.AgentKey, tc.count),
		}
		if !strings.Contains(m.viewAgent(), tc.want) {
			t.Errorf("%d keys: view does not say %q", tc.count, tc.want)
		}
	}
}

// --- key kind labels --------------------------------------------------------------

func TestKeyKindLabelsAvoidRawJargon(t *testing.T) {
	if got := algoLabels[core.AlgoEd25519]; got != "Ed25519" {
		t.Errorf("ed25519 label = %q", got)
	}
	if got := algoLabels[core.AlgoRSA]; got != "RSA 4096" {
		t.Errorf("rsa label = %q", got)
	}

	// Check the Type row specifically: the Name field legitimately contains
	// "id_ed25519", which would match a naive search.
	view := testModel(t).viewForm()
	var typeRow string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Type") {
			typeRow = line
			break
		}
	}
	if typeRow == "" {
		t.Fatalf("form has no Type row:\n%s", view)
	}
	for _, raw := range []string{"ed25519", "rsa ", "ecdsa"} {
		if strings.Contains(typeRow, raw) {
			t.Errorf("Type row still shows the raw name %q: %q", raw, typeRow)
		}
	}
}

func TestKeyWallShowsAlgorithms(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "work")
	m.screen = scrBrowser
	m.loadKeys()

	view := m.viewBrowser()
	if !strings.Contains(view, "work") {
		t.Errorf("key wall does not list the key:\n%s", view)
	}
	if !strings.Contains(view, "~/.ssh/work") {
		t.Errorf("key wall does not show where the key lives:\n%s", view)
	}
}

// --- workflow badges -------------------------------------------------------------

func TestDetailBoxReportsWorkflowState(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "work")
	_ = m.store.MarkAgentLoaded("work")
	_ = m.store.RecordTest("work", true)

	box := detailBox("work", m.store, 90)
	for _, want := range []string{"IN AGENT", "TESTED OK"} {
		if !strings.Contains(box, want) {
			t.Errorf("detail box missing %q:\n%s", want, box)
		}
	}
}

func TestDetailBoxDistinguishesFailedFromUntested(t *testing.T) {
	m := testModel(t)
	writeKeyPair(t, "work")
	_ = m.store.RecordTest("work", false)

	box := detailBox("work", m.store, 90)
	if !strings.Contains(box, "TEST FAILED") {
		t.Errorf("a failed test did not read as failed:\n%s", box)
	}
	if !strings.Contains(box, "NOT IN AGENT") {
		t.Errorf("an unloaded key did not read as not loaded:\n%s", box)
	}
}

// --- help coverage ----------------------------------------------------------------

func TestEveryScreenHasHelpCopy(t *testing.T) {
	screens := []screen{scrHome, scrForm, scrBrowser, scrService, scrInstructions, scrResult, scrAgent}
	for _, s := range screens {
		m := testModel(t)
		m.screen = s
		topic := m.helpTopicFor()
		rows := helpRows[topic]
		if len(rows) == 0 {
			t.Errorf("screen %v maps to help topic %v, which has no rows", s, topic)
		}
		for _, r := range rows {
			if len(r) != 2 || r[0] == "" || r[1] == "" {
				t.Errorf("screen %v has a malformed help row: %v", s, r)
			}
		}
	}
}
