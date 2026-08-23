package gui

import (
	"context"
	"fmt"
	"image/color"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"keysmith/internal/core"
)

type appUI struct {
	win fyne.Window

	keys     []core.KeyInfo
	store    *core.Store
	selected string

	svc        core.Service
	lastTest   core.HostResult
	success    bool
	busy       bool
	busyDialog dialog.Dialog
	busyCancel chan struct{}

	lastGenResult   core.Result
	lastTestResults []core.HostResult
}

func Run() {
	fyneapp.SetMetadata(fyne.AppMetadata{
		ID:         "io.keysmith.desktop",
		Name:       "KeySmith",
		Version:    "2.0.0",
		Migrations: map[string]bool{"fyneDo": true},
	})
	a := app.New()
	a.Settings().SetTheme(&keysmithTheme{Theme: theme.DefaultTheme()})
	w := a.NewWindow("KeySmith")
	w.Resize(fyne.NewSize(940, 640))
	w.SetMaster()

	ui := &appUI{win: w, store: core.LoadStore()}
	ui.showHome()

	w.ShowAndRun()
}

// --- shared chrome --------------------------------------------------------

// --- screen plumbing ----------------------------------------------------

// showScreen swaps the whole window content to one wizard screen.
func (u *appUI) showScreen(s fyne.CanvasObject) {
	u.win.SetContent(s)
}

// hero is the workbench brand band at the top of every screen: wordmark,
// one-line subtitle, step tracker, and a copper rule.
func hero(title, sub, step string) fyne.CanvasObject {
	mark := widget.NewLabelWithStyle("KEYSMITH", fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true})

	trackerRow := container.NewHBox()
	if step != "" {
		badge := canvas.NewRectangle(mustHex(core.ColorAccentTint))
		badge.CornerRadius = 6
		stepLabel := canvas.NewText(step, mustHex("#5f3a12"))
		stepLabel.TextStyle = fyne.TextStyle{Bold: true}
		chip := container.NewStack(badge, container.NewPadded(stepLabel))
		trackerRow.Add(chip)
	}

	headRow := container.NewBorder(nil, nil, nil, trackerRow,
		container.NewHBox(mark, layout.NewSpacer()))
	rule := canvas.NewRectangle(mustHex(core.ColorAccent))
	rule.SetMinSize(fyne.NewSize(0, 3))

	subLine := widget.NewLabel(sub)
	subLine.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		container.NewPadded(container.NewVBox(headRow, rule)),
		container.NewPadded(subLine),
		layout.NewSpacer(),
	)
}

func footer(hint string) fyne.CanvasObject {
	l := widget.NewLabel(hint)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// banner renders the big state announcement: success = verdigris, failure = ember.
func banner(text string, ok bool) fyne.CanvasObject {
	bg := mustHex(core.ColorDangerTint)
	fg := mustHex(core.ColorDanger)
	if ok {
		bg = colorNRGBA(0xe4, 0xf1, 0xea)
		fg = mustHex("#1e6b52")
	}
	rect := canvas.NewRectangle(bg)
	rect.CornerRadius = 10
	label := canvas.NewText(text, fg)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignCenter
	label.TextSize = 18
	pad := container.NewPadded(label)
	return container.NewStack(rect, pad)
}

func colorNRGBA(r, g, b uint8) color.Color {
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

// nameLines renders an identity line from an elideName pair: bold display
// name, with the untruncated name underneath when it was elided.
func nameLines(disp, full string) fyne.CanvasObject {
	box := container.NewVBox(
		widget.NewLabelWithStyle(disp, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if disp != full {
		whole := widget.NewLabel(full)
		whole.Wrapping = fyne.TextWrapWord
		box.Add(whole)
	}
	return box
}

func accentRule() fyne.CanvasObject {
	r := canvas.NewRectangle(mustHex(core.ColorAccent))
	r.SetMinSize(fyne.NewSize(48, 3))
	return r
}

// choiceCard is one big tappable option with title + description.
func choiceCard(num, title, desc string, onTap func()) fyne.CanvasObject {
	numBadge := canvas.NewRectangle(mustHex(core.ColorAccentTint))
	numBadge.CornerRadius = 8
	numLabel := widget.NewLabelWithStyle(" "+num+" ", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	chip := container.NewStack(numBadge, numLabel)

	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	descLabel := widget.NewLabel(desc)
	descLabel.Wrapping = fyne.TextWrapWord

	row := container.NewBorder(nil, nil, chip, nil,
		container.NewVBox(titleLabel, descLabel))
	btn := widget.NewButton("", onTap)
	layer := container.NewStack(btn, container.NewPadded(row))
	return layer
}

// elideName returns a display-safe name plus the full name for the tooltip.
func elideName(name string) (string, string) {
	if len(name) <= 34 {
		return name, name
	}
	return name[:16] + "…" + name[len(name)-15:], name
}

// --- HOME -------------------------------------------------------------------

func (u *appUI) showHome() {
	cards := container.NewVBox(
		choiceCard("1", "Set up a new SSH key", "Forge a fresh key pair, then wire it to GitHub, GitLab or Bitbucket.", u.showForm),
		choiceCard("2", "Manage existing keys", "Inspect your key wall, copy public keys, add to the agent, delete.", func() { u.showBrowser(false) }),
		choiceCard("3", "Test a connection", "Pick a key and a service; we shake hands and report honestly.", func() { u.showBrowser(true) }),
	)

	content := container.NewVBox(
		hero("Welcome", "What would you like to do?", ""),
		cards,
		footer("Every flow guides you step by step. Nothing leaves your machine."),
	)
	u.showScreen(container.NewPadded(content))
}

// --- FORM ----------------------------------------------------------------------

type algoChoice struct {
	label string
	value core.KeyAlgorithm
	note  string
}

var algoChoices = []algoChoice{
	{"Ed25519", core.AlgoEd25519, "recommended — fast and modern"},
	{"RSA 4096", core.AlgoRSA, "for older systems"},
	{"ECDSA", core.AlgoECDSA, "compatibility choice"},
}

func (u *appUI) showForm() {
	nameEntry := widget.NewEntry()
	nameEntry.SetText("id_ed25519")

	algoGroup := container.NewVBox()
	labels := make([]string, len(algoChoices))
	for i, c := range algoChoices {
		labels[i] = c.label + " — " + c.note
	}
	algoRadio := widget.NewRadioGroup(labels, nil)
	algoRadio.SetSelected(labels[0])
	algoGroup.Add(algoRadio)

	commentEntry := widget.NewEntry()
	passEntry := widget.NewPasswordEntry()
	confirmEntry := widget.NewPasswordEntry()
	forceCheck := widget.NewCheck("Overwrite an existing key of this name", nil)

	generate := widget.NewButton("Strike the key", func() {
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			u.warn("Key name cannot be empty.")
			return
		}
		if !core.ValidKeyName(name) {
			u.warn("Use only letters, numbers, dot (.), underscore (_) or dash (-).")
			return
		}
		if passEntry.Text != confirmEntry.Text {
			u.warn("Passphrase and confirmation do not match.")
			return
		}
		var chosen core.KeyAlgorithm
		for i, l := range labels {
			if l == algoRadio.Selected {
				chosen = algoChoices[i].value
			}
		}
		u.runOp("Forging your key...", func(ctx context.Context) {
			u.lastGenResult = core.GenerateKey(chosen, name, strings.TrimSpace(commentEntry.Text), passEntry.Text, forceCheck.Checked)
		}, func() {
			res := u.lastGenResult
			if res.OK {
				passEntry.SetText("")
				confirmEntry.SetText("")
				u.showKeyReady(name)
			} else {
				u.warn(res.Message)
			}
		})
	})
	generate.Importance = widget.HighImportance

	back := widget.NewButton("Back", u.showHome)
	form := widget.NewForm(
		widget.NewFormItem("Key name in ~/.ssh", nameEntry),
		widget.NewFormItem("Type", algoGroup),
		widget.NewFormItem("Comment (optional)", commentEntry),
		widget.NewFormItem("Passphrase (optional)", passEntry),
		widget.NewFormItem("", confirmEntry),
		widget.NewFormItem("", forceCheck),
	)
	form.OnSubmit = generate.OnTapped
	form.SubmitText = "Strike the key"
	form.CancelText = "Back"
	form.OnCancel = u.showHome

	scroll := container.NewVScroll(container.NewVBox(
		hero("New SSH key", "Unsure? The defaults are good. A passphrase adds safety but asks you to type it when used.", "STEP 1 · THE KEY"),
		form,
		container.NewHBox(back, layout.NewSpacer(), generate),
	))
	u.showScreen(container.NewPadded(scroll))
}

// --- KEY READY ---------------------------------------------------------------------

func (u *appUI) showKeyReady(newName string) {
	u.selected = newName
	u.keys = core.ListKeys()

	pub := core.PublicKey(newName)
	pubBox := widget.NewMultiLineEntry()
	pubBox.SetText(pub)
	pubBox.Wrapping = fyne.TextWrapBreak
	pubBox.Disable()

	list := widget.NewList(
		func() int { return len(u.keys) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			lbl := o.(*widget.Label)
			disp, _ := elideName(u.keys[i].Name)
			if u.keys[i].Name == newName {
				lbl.Text = "➜ " + disp + "   (just forged)"
			} else {
				lbl.Text = "   " + disp
			}
			lbl.TextStyle = fyne.TextStyle{Bold: u.keys[i].Name == newName}
			lbl.Refresh()
		},
	)
	list.Select(len(u.keys) - 1)

	copyBtn := widget.NewButton("Copy public key", func() { u.copyPubKey(newName) })
	copyBtn.Importance = widget.HighImportance
	agentBtn := widget.NewButton("Add to agent", func() { u.addToAgentFlow(newName) })
	nextBtn := widget.NewButton("Continue: connect to a Git service", func() { u.showServicePick() })
	nextBtn.Importance = widget.HighImportance
	homeBtn := widget.NewButton("Home", u.showHome)

	doneChip := func(done bool, text string) fyne.CanvasObject {
		bg := mustHex(core.ColorBorder)
		fgc := mustHex(core.ColorInkHover)
		if done {
			bg = colorNRGBA(0xe4, 0xf1, 0xea)
			fgc = mustHex("#1e6b52")
		}
		r := canvas.NewRectangle(bg)
		r.CornerRadius = 8
		l := canvas.NewText(" "+text+" ", fgc)
		l.TextStyle = fyne.TextStyle{Bold: true}
		l.Alignment = fyne.TextAlignCenter
		return container.NewStack(r, container.NewPadded(l))
	}

	card := widget.NewCard("", "",
		container.NewVBox(
			banner("✓ Key forged", true),
			nameLines(elideName(newName)),
			widget.NewLabel("Public key (the .pub file is the shareable half):"),
			pubBox,
			container.NewGridWithColumns(2, copyBtn, agentBtn),
			container.NewHBox(doneChip(true, "forged"), doneChip(u.store.AgentLoadedKeys[newName], "in agent")),
		))

	content := container.NewBorder(
		hero("Step 2 of 3 · Connect it to a service", "", "STEP 2 · THE SERVICE"),
		container.NewVBox(
			container.NewHBox(homeBtn, layout.NewSpacer(), nextBtn),
			footer("“Add to agent” matters only if your key has a passphrase."),
		),
		nil, nil,
		container.NewHSplit(card, container.NewVScroll(list)),
	)
	u.showScreen(content)
}

// --- BROWSER --------------------------------------------------------------------------

func (u *appUI) showBrowser(pickMode bool) {
	u.keys = core.ListKeys()
	if len(u.keys) == 0 {
		u.info("No keys yet", "No SSH keys found. Choose “Set up a new SSH key” on the home screen first.")
		u.showHome()
		return
	}
	u.selected = u.keys[len(u.keys)-1].Name

	var detail *widget.Label
	var list *widget.List

	selectKey := func(i widget.ListItemID) {
		if detail == nil || i < 0 || i >= len(u.keys) {
			return
		}
		u.selected = u.keys[i].Name
		fp := core.Fingerprint(u.selected)
		if fp == "" {
			fp = "(fingerprint unavailable)"
		}
		marks := ""
		if u.store.AgentLoadedKeys[u.selected] {
			marks += "\n● loaded in agent"
		} else {
			marks += "\n○ not in agent"
		}
		if v, ok := u.store.TestedKeysOK[u.selected]; ok {
			if v {
				marks += "\n● tested OK against a service"
			} else {
				marks += "\n● last test failed"
			}
		}
		detail.Text = fmt.Sprintf("%s\n%s%s", u.selected, fp, marks)
		detail.Refresh()
	}

	list = widget.NewList(
		func() int { return len(u.keys) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			disp, _ := elideName(u.keys[i].Name)
			o.(*widget.Label).SetText(disp)
		},
	)
	list.OnSelected = selectKey
	list.Select(len(u.keys) - 1)

	copyBtn := widget.NewButton("Copy public key", func() { u.copyPubKey(u.selected) })
	agentBtn := widget.NewButton("Add to agent", func() { u.addToAgentFlow(u.selected) })
	delBtn := widget.NewButton("Delete…", func() { u.deleteFlow(u.selected) })
	delBtn.Importance = widget.DangerImportance

	var nextBtn *widget.Button
	if pickMode {
		nextBtn = widget.NewButton("Continue: test against a service", func() { u.showServicePick() })
		nextBtn.Importance = widget.HighImportance
	}

	backBtn := widget.NewButton("Back", u.showHome)

	detail = widget.NewLabel("")
	detail.Wrapping = fyne.TextWrapWord

	right := container.NewVBox(detail, layout.NewSpacer(),
		container.NewGridWithColumns(2, copyBtn, agentBtn), delBtn)
	if pickMode && nextBtn != nil {
		right.Objects = append(right.Objects, nextBtn)
	}

	stepTxt := "Your key wall. Select a key for details and actions."
	if pickMode {
		stepTxt = "Pick the key to test, then continue."
	}
	content := container.NewBorder(
		hero("Manage keys", stepTxt, "THE KEY WALL"),
		container.NewHBox(backBtn),
		nil, nil,
		container.NewHSplit(container.NewVScroll(list), container.NewPadded(right)),
	)
	u.showScreen(content)
}

// --- SERVICE PICK -------------------------------------------------------------------------

func (u *appUI) showServicePick() {
	box := container.NewVBox()
	for n, s := range core.Services {
		svc := s
		num := fmt.Sprint(n + 1)
		box.Add(choiceCard(num, s.Name,
			"Hand this key to "+s.Host+" — guided steps and a live connection test.",
			func() { u.svc = svc; u.showInstructions() }))
	}
	box.Add(choiceCard("–", "Skip for now", "Return home; you can run this wizard any time.", u.showHome))

	content := container.NewVBox(
		hero("Connect to a Git service", "Which door should this key open? You can run the wizard again for others.", "STEP 2 · THE SERVICE"),
		box,
	)
	u.showScreen(container.NewPadded(content))
}

// --- INSTRUCTIONS -----------------------------------------------------------------------------

func (u *appUI) showInstructions() {
	body := widget.NewLabel(strings.Join(u.svc.Steps, "\n"))
	body.Wrapping = fyne.TextWrapWord

	openBtn := widget.NewButton("Open "+u.svc.Name+" page in browser", func() {
		u.copyPubKeySilent()
		page, err := url.Parse(u.svc.KeysURL)
		if err == nil {
			_ = fyne.CurrentApp().OpenURL(page)
		}
	})
	openBtn.Importance = widget.HighImportance

	testBtn := widget.NewButton("I added it — test now", func() { u.runTestFlow() })
	backBtn := widget.NewButton("Back", u.showServicePick)

	content := container.NewVBox(
		hero("Add the key to "+u.svc.Name, "Your public key lands on your clipboard before the page opens.", "STEP 2 · THE SERVICE"),
		widget.NewCard("", "", body),
		container.NewVBox(openBtn, testBtn, backBtn),
	)
	u.showScreen(container.NewPadded(container.NewVScroll(content)))
}

// --- RESULT ---------------------------------------------------------------------------------------

func (u *appUI) showResult() {
	keyDisp, _ := elideName(u.subjectKey())
	var body fyne.CanvasObject
	if u.success {
		out := firstLineOfDetail(u.lastTest.Output)
		lb := widget.NewLabel(out)
		lb.Wrapping = fyne.TextWrapWord
		body = lb
	} else {
		items := container.NewVBox()
		for _, dg := range core.Diagnose(u.lastTest.Output) {
			cause := widget.NewLabelWithStyle("Most likely: "+dg.Cause, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			cause.Wrapping = fyne.TextWrapWord
			fix := widget.NewLabel(dg.Fix)
			fix.Wrapping = fyne.TextWrapWord
			items.Add(cause)
			items.Add(fix)
		}
		body = items
	}

	retryBtn := widget.NewButton("Retry test", func() { u.runTestFlow() })
	instrBtn := widget.NewButton("Instructions again", u.showInstructions)
	anotherBtn := widget.NewButton("Set up another key", func() { u.showForm() })
	homeBtn := widget.NewButton("Home", u.showHome)

	row := container.NewHBox(retryBtn, instrBtn, anotherBtn, homeBtn)

	head := ""
	if u.success {
		head = fmt.Sprintf("Your key works with %s.", u.svc.Name)
	} else {
		head = fmt.Sprintf("The test against %s failed.", u.svc.Name)
	}

	content := container.NewVBox(
		hero(head, "Key: "+keyDisp, "STEP 3 · PROOF"),
		banner(map[bool]string{true: "✓ Connected", false: "✗ Not connected yet"}[u.success], u.success),
		widget.NewCard("", "", body),
		row,
	)
	u.showScreen(container.NewPadded(container.NewVScroll(content)))
}

// --- flows ---------------------------------------------------------------------------------------------

func (u *appUI) subjectKey() string {
	return u.selected
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

func (u *appUI) copyPubKey(keyName string) {
	if keyName == "" {
		u.warn("No key selected.")
		return
	}
	pub := core.PublicKey(keyName)
	if pub == "" {
		u.warn("Could not load the public key.")
		return
	}
	u.win.Clipboard().SetContent(pub)
	u.store.CopiedKeys[keyName] = true
	_ = u.store.Save()
	u.statusFlash("Public key copied to clipboard")
}

func (u *appUI) copyPubKeySilent() {
	key := u.selected
	if key == "" {
		return
	}
	if pub := core.PublicKey(key); pub != "" {
		u.win.Clipboard().SetContent(pub)
		u.store.CopiedKeys[key] = true
		_ = u.store.Save()
	}
}

func (u *appUI) addToAgentFlow(keyName string) {
	if keyName == "" {
		u.warn("No key selected.")
		return
	}
	u.runOp("Adding key to agent...", func(ctx context.Context) {
		u.lastGenResult = core.AddToAgent(keyName)
	}, func() {
		res := u.lastGenResult
		if res.OK {
			u.store.AgentLoadedKeys[keyName] = true
			_ = u.store.Save()
			u.statusFlash(res.Message)
		} else {
			u.warn(res.Message)
		}
	})
}

func (u *appUI) deleteFlow(keyName string) {
	dialog.NewConfirm("Delete key",
		fmt.Sprintf("Delete '%s' (both files) from ~/.ssh? This cannot be undone.", keyName),
		func(ok bool) {
			if !ok {
				return
			}
			_ = core.DeleteKey(keyName)
			delete(u.store.UsedKeys, keyName)
			delete(u.store.CopiedKeys, keyName)
			delete(u.store.TestedKeysOK, keyName)
			delete(u.store.AgentLoadedKeys, keyName)
			_ = u.store.Save()
			u.statusFlash(fmt.Sprintf("Deleted '%s'", keyName))
			u.showHome()
		}, u.win).Show()
}

func (u *appUI) runTestFlow() {
	key := u.subjectKey()
	if key == "" {
		u.keyWarning()
		return
	}
	svc := u.svc
	if svc.ID == "" {
		svc, _ = core.ServiceByID("github")
	}
	u.runOp("Testing connection to "+svc.Name+"...", func(ctx context.Context) {
		r := core.TestService(svc, key)
		u.lastTestResults = []core.HostResult{r}
	}, func() {
		u.lastTest = u.lastTestResults[0]
		u.success = u.lastTest.OK
		u.store.TestedKeysOK[key] = u.success
		_ = u.store.Save()
		fyne.Do(func() { u.showResult() })
	})
}

func (u *appUI) keyWarning() {
	u.warn("Select a key first.")
}

func (u *appUI) statusFlash(msg string) {
	note := widget.NewLabel(msg)
	popup := widget.NewPopUp(note, u.win.Canvas())
	popup.Show()
	go func() {
		time.Sleep(2 * time.Second)
		fyne.Do(popup.Hide)
	}()
}

func (u *appUI) warn(msg string) {
	dialog.NewError(fmt.Errorf("%s", msg), u.win)
}

func (u *appUI) info(title, msg string) {
	dialog.NewInformation(title, msg, u.win)
}

// runOp runs fn off-thread with busy lockout + cancel, then onDone on the UI
// goroutine via fyne.Do.
func (u *appUI) runOp(busyMsg string, fn func(ctx context.Context), onDone func()) {
	if u.busy {
		return
	}
	u.busy = true
	u.busyCancel = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	progress := widget.NewProgressBarInfinite()
	cancelBtn := widget.NewButton("Cancel operation", func() { close(u.busyCancel) })
	d := dialog.NewCustom("Working…", "", container.NewVBox(widget.NewLabel(busyMsg), progress, cancelBtn), u.win)
	u.busyDialog = d
	d.Show()

	go func() {
		go func() {
			select {
			case <-u.busyCancel:
				cancel()
			case <-ctx.Done():
			}
		}()
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
