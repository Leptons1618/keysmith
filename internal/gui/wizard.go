//go:build !tui

package gui

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"keysmith/internal/core"
)

// appUI holds the state every screen reads and writes. Navigation is a page
// switch inside the shell rather than a one-way wizard, so users can leave a
// flow at any point and come back to it.
type appUI struct {
	win    fyne.Window
	shell  *shell
	screen screen

	keys     []core.KeyInfo
	store    *core.Store
	selected string

	svc      core.Service
	lastTest core.HostResult
	success  bool

	busy       bool
	busyDialog dialog.Dialog
	busyCancel chan struct{}

	// status is the persistent message in the bottom strip.
	status     string
	statusTone tone

	// form holds live widgets for the create-key page so the summary panel can
	// update as the user types without rebuilding (and unfocusing) the form.
	form *newKeyForm

	lastGenResult   core.Result
	lastTestResults []core.HostResult
	checkedKey      string

	agent        core.AgentState
	agentChecked bool
}

// keyForTest resolves which key a test should use: a retried result keeps its
// original subject even if the selection has since moved.
func keyForTest(selected, checked string) string {
	if checked != "" {
		return checked
	}
	return selected
}

func deleteAndForget(keyName string, deleteKey func(string) error, forget func(string) error) error {
	if err := deleteKey(keyName); err != nil {
		return fmt.Errorf("could not delete key %q: %w", keyName, err)
	}
	if err := forget(keyName); err != nil {
		return fmt.Errorf("key %q was deleted, but its workflow state could not be saved: %w", keyName, err)
	}
	return nil
}

// workflowState is the slice of the store the screens record transitions
// against, so those transitions can be tested without touching disk.
type workflowState interface {
	MarkCopied(string) error
	MarkAgentLoaded(string) error
	RecordTest(string, bool) error
	MarkUsed(string) error
	ForgetKey(string) error
}

func recordCopy(state workflowState, keyName string) error {
	if err := state.MarkCopied(keyName); err != nil {
		return fmt.Errorf("could not save copied-key state: %w", err)
	}
	return nil
}

func recordAgentLoad(state workflowState, keyName string) error {
	if err := state.MarkAgentLoaded(keyName); err != nil {
		return fmt.Errorf("could not save agent state: %w", err)
	}
	return nil
}

func recordTest(state workflowState, keyName string, ok bool) error {
	if err := state.RecordTest(keyName, ok); err != nil {
		return fmt.Errorf("could not save test result: %w", err)
	}
	return nil
}

func recordServiceUse(state workflowState, keyName string) error {
	if err := state.MarkUsed(keyName); err != nil {
		return fmt.Errorf("could not save service state: %w", err)
	}
	return nil
}

func forgetDeletedKey(state workflowState, keyName string) error {
	if err := state.ForgetKey(keyName); err != nil {
		return fmt.Errorf("could not save deleted-key workflow state: %w", err)
	}
	return nil
}

func forgetDeletedKeyStore(state workflowState, keyName string) error {
	return state.ForgetKey(keyName)
}

func checkAgent(ctx context.Context, operation func(context.Context) core.Result) core.Result {
	return operation(ctx)
}

func keyIndex(keys []core.KeyInfo, name string) int {
	for i := range keys {
		if keys[i].Name == name {
			return i
		}
	}
	return -1
}

// Run starts the desktop GUI and returns after its application loop ends.
func Run(version string) error {
	app.SetMetadata(fyne.AppMetadata{
		ID:         "io.keysmith.desktop",
		Name:       "KeySmith",
		Version:    version,
		Migrations: map[string]bool{"fyneDo": true},
	})
	a := app.NewWithID("io.keysmith.desktop")
	a.Settings().SetTheme(&keysmithTheme{Theme: theme.DefaultTheme()})

	w := a.NewWindow("KeySmith")
	w.Resize(fyne.NewSize(1180, 780))
	w.SetMaster()

	ui := &appUI{win: w, store: core.LoadStore()}
	ui.navigate(scrOverview)

	// Rebuild the current page when the user flips light/dark, so raw canvas
	// text and the semantic palette both follow the switch.
	a.Settings().AddListener(func(fyne.Settings) {
		fyne.Do(func() { ui.rerender() })
	})

	w.ShowAndRun()
	return nil
}

// rerender redraws the current page against the current theme.
func (u *appUI) rerender() {
	current := u.screen
	u.navigate(current)
}

// --- shared helpers ---------------------------------------------------------

// subjectKey is the key the current flow operates on.
func (u *appUI) subjectKey() string {
	return keyForTest(u.selected, u.checkedKey)
}

func firstLineOfDetail(out string) string {
	for _, l := range strings.Split(out, "\n") {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "---") {
			return t
		}
	}
	return out
}

// elideName returns a display-safe name plus the full name, so long key names
// stay readable without losing information.
func elideName(name string) (string, string) {
	if len(name) <= 34 {
		return name, name
	}
	return name[:16] + "…" + name[len(name)-15:], name
}

// keyTone maps a key's workflow state to a tone, so the key wall and the
// detail panel agree on what "done" looks like.
func (u *appUI) keyTone(name string) tone {
	switch {
	case u.store.TestedKeysOK[name]:
		return toneSuccess
	case u.store.CopiedKeys[name] || u.store.UsedKeys[name] || u.store.AgentLoadedKeys[name]:
		return toneAccent
	default:
		return toneNeutral
	}
}

// keyStatusLine is the one-line summary of how far a key has progressed.
func (u *appUI) keyStatusLine(name string) string {
	switch {
	case u.store.TestedKeysOK[name]:
		return "Verified against a Git service"
	case u.store.UsedKeys[name]:
		return "Added to a service, not yet verified"
	case u.store.AgentLoadedKeys[name]:
		return "Loaded in the agent"
	case u.store.CopiedKeys[name]:
		return "Public key copied"
	default:
		return "Not connected to anything yet"
	}
}

// keyDir is the directory shown in the UI.
func keyDir() string { return core.SSHDir() }

// keyPathLabel is the shortened, human-facing location of a key. The absolute
// path is rarely what someone needs, and it wraps badly in a narrow column.
func keyPathLabel(name string) string {
	if name == "" {
		return "~/.ssh"
	}
	return "~/.ssh/" + name
}

// --- flows ------------------------------------------------------------------

// warn reports a failure in the status strip instead of throwing a modal
// dialog over the page, which keeps the user oriented.
func (u *appUI) warn(msg string) {
	u.setStatus(msg, toneDanger)
}

func (u *appUI) notice(msg string) {
	u.setStatus(msg, toneSuccess)
}

func (u *appUI) copyPubKey(keyName string) {
	if keyName == "" {
		u.warn("No key selected.")
		return
	}
	pub := core.PublicKey(keyName)
	if pub == "" {
		u.warn("Could not read the public key for " + keyName + ".")
		return
	}
	u.win.Clipboard().SetContent(pub)
	if err := recordCopy(u.store, keyName); err != nil {
		u.warn(err.Error())
		return
	}
	u.setStatus("Public key for "+keyName+" copied to your clipboard.", toneSuccess)
}

// copyPubKeySilent puts the key on the clipboard without narrating it, used
// when the next action immediately consumes it.
func (u *appUI) copyPubKeySilent() {
	key := u.subjectKey()
	if key == "" {
		return
	}
	pub := core.PublicKey(key)
	if pub == "" {
		return
	}
	u.win.Clipboard().SetContent(pub)
	if err := recordCopy(u.store, key); err != nil {
		u.warn(err.Error())
		return
	}
	u.setStatus("Public key for "+key+" copied to your clipboard.", toneSuccess)
}

func (u *appUI) addToAgentFlow(keyName string) {
	if keyName == "" {
		u.warn("No key selected.")
		return
	}
	u.runOp("Loading "+keyName+" into the SSH agent…", func(ctx context.Context) {
		u.lastGenResult = core.AddToAgentContext(ctx, keyName)
	}, func() {
		res := u.lastGenResult
		if !res.OK {
			u.warn(res.Message)
			return
		}
		if err := recordAgentLoad(u.store, keyName); err != nil {
			u.warn(err.Error())
			return
		}
		u.agentChecked = false
		u.setStatus(res.Message, toneSuccess)
		u.rerender()
	})
}

// checkAgentFlow probes the agent and records what it holds, so the agent page
// and the rail footer reflect reality instead of a guess.
func (u *appUI) checkAgentFlow() {
	u.runOp("Checking the SSH agent…", func(ctx context.Context) {
		u.lastGenResult = checkAgent(ctx, core.CheckAgentContext)
		u.agent = core.AgentInfo(ctx)
	}, func() {
		u.agentChecked = true
		res := u.lastGenResult
		if res.OK {
			u.setStatus(res.Message, toneSuccess)
		} else {
			u.setStatus(res.Message, toneWarning)
		}
		if u.screen != scrAgent {
			u.navigate(scrAgent)
		} else {
			u.rerender()
		}
	})
}

func (u *appUI) deleteFlow(keyName string) {
	dialog.NewConfirm("Delete key",
		fmt.Sprintf("Delete %q and its .pub file from %s?\n\nThis cannot be undone.", keyName, keyDir()),
		func(ok bool) {
			if !ok {
				return
			}
			if err := deleteAndForget(keyName, core.DeleteKey, func(name string) error {
				return forgetDeletedKeyStore(u.store, name)
			}); err != nil {
				u.warn(err.Error())
				return
			}
			if u.selected == keyName {
				u.selected = ""
			}
			u.checkedKey = ""
			u.setStatus("Deleted "+keyName+".", toneSuccess)
			u.rerender()
		}, u.win).Show()
}

// startConnectionTestFlow goes straight to service selection for the currently
// highlighted key, which is the fastest path to an answer.
func (u *appUI) startConnectionTestFlow() {
	keys := core.ListKeys()
	if len(keys) == 0 {
		u.navigate(scrNewKey)
		u.setStatus("Create a key first, then come back to test it.", toneWarning)
		return
	}
	u.keys = keys
	if keyIndex(keys, u.selected) < 0 {
		u.selected = keys[len(keys)-1].Name
	}
	u.checkedKey = ""
	u.navigate(scrService)
}

// runTestFlow runs the connection test off the UI goroutine and shows the
// result page when it finishes.
func (u *appUI) runTestFlow() {
	key := u.subjectKey()
	if key == "" {
		u.warn("Select a key first.")
		return
	}
	svc := u.svc
	if svc.ID == "" {
		svc, _ = core.ServiceByID("github")
		u.svc = svc
	}
	u.checkedKey = key
	u.runOp("Testing your connection to "+svc.Name+"…", func(ctx context.Context) {
		u.lastTestResults = []core.HostResult{core.TestServiceContext(ctx, svc, key)}
	}, func() {
		if len(u.lastTestResults) == 0 {
			u.warn("The connection test returned no result.")
			return
		}
		u.lastTest = u.lastTestResults[0]
		u.success = u.lastTest.OK
		if err := recordTest(u.store, key, u.success); err != nil {
			u.warn(err.Error())
			return
		}
		if u.success {
			if err := recordServiceUse(u.store, key); err != nil {
				u.warn(err.Error())
				return
			}
			u.setStatus(key+" is verified against "+svc.Name+".", toneSuccess)
		} else {
			u.setStatus("The test against "+svc.Name+" did not succeed. See the diagnosis.", toneWarning)
		}
		u.navigate(scrResult)
	})
}

// runOp runs fn off the UI goroutine with a busy lockout and a cancel button,
// then calls onDone back on the UI goroutine.
func (u *appUI) runOp(busyMsg string, fn func(ctx context.Context), onDone func()) {
	if u.busy {
		return
	}
	u.busy = true
	u.busyCancel = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	progress := widget.NewProgressBarInfinite()

	cancelBtn := quiet("Cancel", func() { close(u.busyCancel) })
	d := dialog.NewCustom("Working", busyMsg,
		inset(gapLG, 0, gapLG, gapLG,
			vStack(gapMD, progress,
				container.NewHBox(layout.NewSpacer(), cancelBtn))),
		u.win)
	d.Resize(fyne.NewSize(360, 160))
	u.busyDialog = d
	d.Show()

	go func() {
		fn(ctx)
		cancel()
		fyne.Do(func() {
			u.busy = false
			u.busyCancel = nil
			d.Hide()
			if onDone != nil {
				onDone()
			}
		})
	}()
}

// openServicePage copies the key and opens the host's keys page.
func (u *appUI) openServicePage() {
	u.copyPubKeySilent()
	page, err := url.Parse(u.svc.KeysURL)
	if err != nil {
		u.warn("Could not open the browser.")
		return
	}
	if err := fyne.CurrentApp().OpenURL(page); err != nil {
		u.warn("Could not open the browser.")
		return
	}
	u.setStatus("Opened "+u.svc.Name+". Paste the key from your clipboard, then test.", toneSuccess)
}

// noteCard is a tinted aside used for contextual guidance.
func noteCard(t tone, heading, explanation string) fyne.CanvasObject {
	return tintedCard(t, vStack(gapXS, label(heading), body(explanation)))
}

// gradientBand paints a soft two-stop wash behind hero content.
func gradientBand(start, end fyne.ThemeColorName, content fyne.CanvasObject, pad float32) fyne.CanvasObject {
	g := canvas.NewLinearGradient(resolve(start), resolve(end), 0)
	return container.NewStack(g, inset(pad, pad, pad, pad, content))
}
