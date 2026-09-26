package gui

import (
	"context"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"keysmith/internal/core"
)

// ---------------------------------------------------------------------------
// Overview
// ---------------------------------------------------------------------------

// overviewContent is the landing page: one clear primary action, honest counts
// about the current state, and a grid of the other things you can do.
func (u *appUI) overviewContent() fyne.CanvasObject {
	keys := core.ListKeys()
	u.keys = keys
	if keyIndex(keys, u.selected) < 0 && len(keys) > 0 {
		u.selected = keys[0].Name
	}

	return vStack(gapXL,
		u.overviewHero(len(keys)),
		u.overviewStats(keys),
		sectionLabel("Do something"),
		u.overviewActions(),
		caption("KeySmith only ever touches ~/.ssh and shells out to OpenSSH. Your private keys never leave this machine."),
	)
}

// overviewHero is the top band: a gradient panel with the single most likely
// next step, framed by whatever is already been set up.
func (u *appUI) overviewHero(keyCount int) fyne.CanvasObject {
	var heading, explanation string
	var cta *widget.Button

	switch {
	case keyCount == 0:
		heading = "Let's get you a working SSH key"
		explanation = "Generate a key, add it to a Git host, and verify it. Three steps, about two minutes."
		cta = primary("Create your first key", func() { u.navigate(scrNewKey) })
	case !u.anyKeyVerified():
		heading = "Finish wiring up your keys"
		explanation = "You have keys in ~/.ssh, but none are verified against a Git service yet."
		cta = primary("Test a connection", u.startConnectionTestFlow)
	default:
		heading = "Your keys are working"
		explanation = "Everything is set up. Create another key for a different service, or check the agent."
		cta = primary("Create another key", func() { u.navigate(scrNewKey) })
	}

	headline := canvas.NewText(heading, resolve(keyInk))
	headline.TextStyle = fyne.TextStyle{Bold: true}
	headline.TextSize = 24

	return gradientBand(keyAccentTint, keySurface,
		container.NewBorder(
			inset(0, 0, 0, 0, vStack(gapMD, headline, body(explanation))),
			inset(gapLG, 0, 0, 0, pinned(cta)),
			nil, nil,
			layout.NewSpacer(),
		),
		gapXL)
}

func (u *appUI) anyKeyVerified() bool {
	for _, ok := range u.store.TestedKeysOK {
		if ok {
			return true
		}
	}
	return false
}

// overviewStats is a row of three metrics: keys on disk, keys verified, agent
// reachability. These are the numbers people actually want to know.
func (u *appUI) overviewStats(keys []core.KeyInfo) fyne.CanvasObject {
	verified, pending := workflowCounts(u.store, keys)

	agentValue, agentTone := "—", toneNeutral
	agentName := "SSH agent not checked"
	if u.agentChecked {
		if u.agent.Reachable {
			agentValue, agentTone = "Up", toneSuccess
			agentName = "SSH agent, " + strconv.Itoa(len(u.agent.Keys)) + " loaded"
		} else {
			agentValue, agentTone = "Down", toneWarning
			agentName = "SSH agent unreachable"
		}
	}

	tiles := []fyne.CanvasObject{
		metric(strconv.Itoa(len(keys)), "Keys in ~/.ssh", toneAccent),
		metric(strconv.Itoa(verified), "Verified", toneSuccess),
		metric(strconv.Itoa(pending), "Not yet connected", toneWarning),
		metric(agentValue, agentName, agentTone),
	}
	return card(container.NewGridWithColumns(len(tiles), tiles...))
}

// overviewActions is the 2x2 grid of task cards.
func (u *appUI) overviewActions() fyne.CanvasObject {
	cards := []fyne.CanvasObject{
		newTapSurface(iconBadge(theme.ContentAddIcon(), toneAccent,
			"Create a key", "Forge a new key pair and wire it to a Git host."), func() {
			u.navigate(scrNewKey)
		}),
		newTapSurface(iconBadge(theme.StorageIcon(), toneAccent,
			"Manage keys", "Inspect fingerprints, copy public keys, load the agent, delete."), func() {
			u.navigate(scrKeys)
		}),
		newTapSurface(iconBadge(theme.LoginIcon(), toneSuccess,
			"Test a connection", "Pick a key and a service; we shake hands and report honestly."), func() {
			u.startConnectionTestFlow()
		}),
		newTapSurface(iconBadge(theme.ComputerIcon(), toneWarning,
			"Check the agent", "See whether this session can reach your agent, and what it holds."), func() {
			u.checkAgentFlow()
		}),
	}
	return container.NewGridWithColumns(2, cards...)
}

// ---------------------------------------------------------------------------
// Create a key
// ---------------------------------------------------------------------------

// newKeyForm holds the live widgets on the create page.
type newKeyForm struct {
	name    *widget.Entry
	comment *widget.Entry
	pass    *widget.Entry
	confirm *widget.Entry
	force   *widget.Check

	algoIdx  int
	algoCard []*tapSurface
	algoRow  *fyne.Container
	aside    *fyne.Container
	errBox   *fyne.Container
}

// algoChoice is one selectable key type.
type algoChoice struct {
	label string
	value core.KeyAlgorithm
	note  string
}

var algoChoices = []algoChoice{
	{"Ed25519", core.AlgoEd25519, "Recommended. Fast, small, and the modern default."},
	{"RSA 4096", core.AlgoRSA, "For older servers and systems that predate Ed25519."},
	{"ECDSA", core.AlgoECDSA, "A middle ground; widely supported, no better than Ed25519."},
}

// newKeyContent is the create page: the form on the left, a live summary of
// what will be written to disk on the right.
func (u *appUI) newKeyContent() fyne.CanvasObject {
	f := &newKeyForm{algoIdx: 0}
	u.form = f

	f.name = widget.NewEntry()
	f.name.SetText("id_ed25519")
	f.name.PlaceHolder = "e.g. id_work_github"

	f.comment = widget.NewEntry()
	f.comment.PlaceHolder = "you@laptop — optional, stored in the key"

	f.pass = widget.NewPasswordEntry()
	f.pass.PlaceHolder = "Leave blank for no passphrase"
	f.confirm = widget.NewPasswordEntry()
	f.confirm.PlaceHolder = "Repeat the passphrase"

	f.force = widget.NewCheck("Overwrite an existing key of this name", func(on bool) {
		f.renderAside()
	})
	f.name.OnChanged = func(string) { f.renderAside() }

	// Algorithm picker: real cards rather than a radio group, so the
	// recommendation is visible instead of implied.
	f.algoRow = &fyne.Container{Layout: &vStackLayout{gap: gapSM}}
	for i, c := range algoChoices {
		idx := i
		surface := newTapSurface(algoCardContent(c), func() {
			f.selectAlgo(idx)
		}).withPad(gapMD)
		if idx == 0 {
			surface.withSelected(true)
		}
		f.algoCard = append(f.algoCard, surface)
		f.algoRow.Add(surface)
	}
	f.algoRow.Refresh()

	f.aside = &fyne.Container{Layout: &vStackLayout{gap: gapMD}}
	f.errBox = &fyne.Container{Layout: &vStackLayout{gap: gapSM}}

	form := vStack(gapLG,
		f.errBox,
		field("Key name", "The file name in ~/.ssh. Letters, numbers, dot, underscore or dash.", f.name),
		field("Key type", "Pick the algorithm. Ed25519 is the right answer for most people.", f.algoRow),
		field("Comment", "An optional label stored inside the public key.", f.comment),
		field("Passphrase", "Adds a second lock on the private key. You will be asked for it when the key is used, unless the key is in the agent.", f.pass),
		field("Confirm passphrase", "Must match the passphrase above.", f.confirm),
		inset(gapXS, 0, gapXS, 0, f.force),
	)

	generate := primary("Create key", func() { u.submitGenerate() })
	formCol := card(vStack(gapXL, form, buttonRow(generate)))

	f.renderAside()
	asideCol := vStack(gapLG, f.aside,
		noteCard(toneAccent, "Where this runs",
			"KeySmith calls ssh-keygen directly. Nothing is uploaded, and the private half never leaves ~/.ssh."))

	return splitColumns(formCol, asideCol, 0.6)
}

// selectAlgo switches the selected algorithm and refreshes both the card
// states and the summary panel.
func (f *newKeyForm) selectAlgo(idx int) {
	f.algoIdx = idx
	for i, c := range f.algoCard {
		c.setContent(algoCardContent(algoChoices[i]))
		c.withSelected(i == idx)
		c.Refresh()
	}
	f.renderAside()
}

// renderAside rebuilds the "what will be created" summary from the current
// form values.
func (f *newKeyForm) renderAside() {
	if f.aside == nil {
		return
	}
	f.aside.Objects = f.asideObjects()
	f.aside.Refresh()
}

func (f *newKeyForm) asideObjects() []fyne.CanvasObject {
	name := strings.TrimSpace(f.name.Text)
	algo := algoChoices[f.algoIdx]

	rows := []fyne.CanvasObject{
		inset(gapSM, 0, 0, 0, sectionLabel("What will be created")),
		vStack(gapMD,
			rowPair("Name", mono(name, keyInkBody)),
			rowPair("Type", mono(algo.label, keyInkBody)),
			rowPair("Location", mono("~/.ssh", keyMuted)),
		),
		rule(),
	}

	if name == "" {
		rows = append(rows, noteCard(toneWarning, "Name required",
			"Enter a name for the key files before creating the key."))
	} else if !core.ValidKeyName(name) {
		rows = append(rows, noteCard(toneDanger, "Invalid name",
			"Use only letters, numbers, dot (.), underscore (_) or dash (-)."))
	} else if keyExists(name) {
		rows = append(rows, noteCard(toneWarning, "A key with this name already exists",
			"Creating it again replaces both files. Turn on the overwrite option if that is what you want."))
	} else {
		rows = append(rows, noteCard(toneSuccess, name+" is free",
			"Creating this key writes "+name+" and "+name+".pub into ~/.ssh."))
	}

	if f.pass.Text != "" && f.pass.Text != f.confirm.Text {
		rows = append(rows, noteCard(toneDanger, "Passphrases do not match",
			"Retype the passphrase in the confirmation field."))
	} else if f.pass.Text != "" {
		rows = append(rows, noteCard(toneWarning, "Passphrase set",
			"After creating this key, load it into the agent so you are not asked for the passphrase on every use."))
	}

	rows = append(rows, vStack(gapXS,
		sectionLabel("Algorithm notes"),
		muted(algo.note)))

	return rows
}

// algoCardContent is the body of one algorithm card: the name, the note, and
// an optional "Recommended" pill on the right. The note must get the full
// width, so the pill is pinned rather than placed in an HBox.
func algoCardContent(c algoChoice) fyne.CanvasObject {
	name := canvas.NewText(c.label, resolve(keyInk))
	name.TextStyle = fyne.TextStyle{Bold: true}
	name.TextSize = theme.TextSize() + 1

	var badge fyne.CanvasObject
	if c.value == core.AlgoEd25519 {
		badge = container.NewCenter(chipText(toneAccent, "Recommended"))
	}
	return container.NewBorder(nil, nil, nil, badge,
		vStack(gapXS, name, rich(c.note, keyMuted, false)))
}

// field is a labelled form control with help text underneath.
func field(labelText, help string, control fyne.CanvasObject) fyne.CanvasObject {
	return vStack(gapXS, label(labelText), control, caption(help))
}

// keyExists reports whether a key pair with this name is already on disk.
func keyExists(name string) bool {
	for _, k := range core.ListKeys() {
		if k.Name == name {
			return true
		}
	}
	return false
}

// submitGenerate validates the form and starts key generation.
func (u *appUI) submitGenerate() {
	f := u.form
	if f == nil {
		return
	}
	name := strings.TrimSpace(f.name.Text)

	switch {
	case name == "":
		u.formError("Key name cannot be empty.")
		return
	case !core.ValidKeyName(name):
		u.formError("Use only letters, numbers, dot (.), underscore (_) or dash (-).")
		return
	case f.pass.Text != f.confirm.Text:
		u.formError("Passphrase and confirmation do not match.")
		return
	}
	u.formError("")

	algo := algoChoices[f.algoIdx].value
	comment := strings.TrimSpace(f.comment.Text)
	passphrase := f.pass.Text
	force := f.force.Checked

	u.runOp("Creating "+name+"…", func(ctx context.Context) {
		u.lastGenResult = core.GenerateKeyContext(ctx, algo, name, comment, passphrase, force)
	}, func() {
		res := u.lastGenResult
		if !res.OK {
			u.formError(res.Message)
			u.warn(res.Message)
			return
		}
		f.pass.SetText("")
		f.confirm.SetText("")
		u.selected = name
		u.checkedKey = name
		u.keys = core.ListKeys()
		u.setStatus("Created "+name+". Next: add it to a Git service.", toneSuccess)
		u.navigate(scrService)
	})
}

// formError shows a validation message above the form, and clears it.
func (u *appUI) formError(msg string) {
	f := u.form
	if f == nil || f.errBox == nil {
		return
	}
	if msg == "" {
		f.errBox.Objects = nil
		f.errBox.Refresh()
		return
	}
	f.errBox.Objects = []fyne.CanvasObject{tintedCard(toneDanger,
		iconRow(theme.ErrorIcon(), 16, body(msg)))}
	f.errBox.Refresh()
}

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

// keysContent is the key wall: a list of keys on the left, a detail panel with
// every action for the highlighted key on the right.
func (u *appUI) keysContent() fyne.CanvasObject {
	keys := core.ListKeys()
	u.keys = keys
	if len(keys) == 0 {
		return card(emptyState(theme.StorageIcon(), "No keys yet",
			"KeySmith looks for key pairs in ~/.ssh. There are none here yet.",
			primary("Create a key", func() { u.navigate(scrNewKey) })))
	}
	if keyIndex(keys, u.selected) < 0 {
		u.selected = keys[0].Name
	}

	rows := make([]fyne.CanvasObject, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, u.keyRow(k))
	}

	return splitColumns(
		inset(0, 0, 0, gapLG,
			vStack(gapSM, sectionLabel(strconv.Itoa(len(keys))+" keys"), vStack(gapSM, rows...))),
		vStack(gapSM, u.keyDetailObjects()...),
		0.36,
	)
}

// keyRow is one row on the key wall: a state dot, the key name, its algorithm,
// and where it has got to.
func (u *appUI) keyRow(k core.KeyInfo) fyne.CanvasObject {
	disp, _ := elideName(k.Name)
	name := canvas.NewText(disp, resolve(keyInk))
	name.TextStyle = fyne.TextStyle{Bold: u.selected == k.Name}
	name.TextSize = theme.TextSize()

	kindLabel := "unknown type"
	if kind, ok := core.KeyKind(k.Name); ok {
		kindLabel = kind
	}
	sub := vStack(0, name, captionLine(kindLabel))

	return newTapSurface(
		container.NewBorder(nil, nil, statusDot(u.keyTone(k.Name)), nil,
			inset(0, 0, gapMD, 0, sub)),
		func() {
			u.selected = k.Name
			u.checkedKey = ""
			u.navigate(scrKeys)
		},
	).withSelected(u.selected == k.Name).withPadXY(gapSM, gapMD)
}

// keyDetailObjects is everything about the selected key: identity, workflow
// state, and the actions that act on it.
func (u *appUI) keyDetailObjects() []fyne.CanvasObject {
	name := u.selected
	if name == "" {
		return []fyne.CanvasObject{card(muted("Select a key to see its details."))}
	}

	disp, full := elideName(name)
	heading := canvas.NewText(disp, resolve(keyInk))
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.TextSize = 20

	pathText := keyPathLabel(name)
	if disp != full {
		pathText = full + " at " + keyPathLabel(name)
	}

	// Identity block: algorithm plus fingerprint, with the fingerprint
	// in monospace because it is something people compare character by
	// character.
	kindText := "Unknown type"
	if kind, ok := core.KeyKind(name); ok {
		kindText = kind
	}
	fp := core.ShortFingerprint(core.Fingerprint(name))
	if fp == "" {
		fp = "Fingerprint unavailable"
	}

	identity := vStack(gapMD,
		chipRow(
			chipText(toneAccent, kindText),
			chip(u.keyTone(name), u.keyStatusLine(name)),
		),
		vStack(gapXS,
			sectionLabel("Fingerprint"),
			codeBlock(fp),
		),
	)

	// Workflow: three checkpoints with real state, so the panel doubles as
	// progress towards a working setup.
	workflow := vStack(gapSM,
		sectionLabel("Progress"),
		dotRow(boolTone(u.store.CopiedKeys[name]), "Public key copied to the clipboard"),
		dotRow(boolTone(u.store.AgentLoadedKeys[name]), "Private key loaded in the agent"),
		dotRow(boolTone(u.store.TestedKeysOK[name]), "Verified against a Git service"),
	)

	pub := core.PublicKey(name)
	preview := vStack(gapXS,
		sectionLabel("Public key"),
		codeBlock(elideMiddle(pub, 220)),
		caption("The .pub file is the shareable half. The private key is not."),
	)

	actions := container.NewGridWithColumns(2,
		primary("Copy public key", func() { u.copyPubKey(name) }),
		secondary("Add to agent", func() { u.addToAgentFlow(name) }),
		widget.NewButton("Test this key", func() {
			u.checkedKey = name
			u.startConnectionTestFlow()
		}),
		destructive("Delete…", func() { u.deleteFlow(name) }),
	)

	return []fyne.CanvasObject{
		card(vStack(gapLG, vStack(gapXS, heading, caption(pathText)), identity, rule(), workflow, preview, rule(), actions)),
	}
}

// ---------------------------------------------------------------------------
// SSH agent
// ---------------------------------------------------------------------------

// agentContent reports whether this session can reach the SSH agent, and what
// it is holding if so.
func (u *appUI) agentContent() fyne.CanvasObject {
	status := vStack(gapMD,
		sectionLabel("Status"),
		muted("Not checked yet. Run the check to see whether this session can reach your agent."),
	)

	keysCard := card(vStack(gapMD,
		sectionLabel("Loaded keys"),
		muted("The agent holds private keys so you are not asked for a passphrase on every connection."),
	))

	if u.agentChecked {
		if u.agent.Reachable {
			stateText := "Agent reachable"
			stateName := loadedKeysLabel(len(u.agent.Keys))
			status = vStack(gapMD,
				chipRow(chip(toneSuccess, stateText), chipText(toneNeutral, stateName)),
				body("ssh-add answered, so this session can use keys held by the agent."),
			)

			if len(u.agent.Keys) == 0 {
				keysCard = card(vStack(gapMD,
					sectionLabel("Loaded keys"),
					muted("The agent is running but holds no keys. Add one from the Keys page."),
				))
			} else {
				rows := []fyne.CanvasObject{}
				for _, k := range u.agent.Keys {
					rows = append(rows, u.agentKeyRow(k))
				}
				keysCard = card(vStack(gapMD, sectionLabel("Loaded keys"),
					&fyne.Container{Layout: &vStackLayout{gap: gapSM}, Objects: rows}))
			}
		} else {
			status = vStack(gapMD,
				chipRow(chip(toneWarning, "Agent unreachable")),
				body("This session cannot talk to an SSH agent."),
			)
			keysCard = noteCard(toneWarning, "How to start an agent",
				"Run eval $(ssh-agent) in your terminal, or start your desktop key manager, then check again. "+
					"Note that testing a key still works without an agent — the agent only matters when your key has a passphrase.")
		}
	}

	return vStack(gapLG,
		card(vStack(gapLG, status, buttonRow(primary("Check now", u.checkAgentFlow)))),
		keysCard,
		caption("On Windows KeySmith also starts the OpenSSH Authentication Agent service for you."),
	)
}

// agentKeyRow is one key currently held by the agent.
func (u *appUI) agentKeyRow(k core.AgentKey) fyne.CanvasObject {
	name := canvas.NewText(k.Fingerprint, resolve(keyInkBody))
	name.TextStyle = fyne.TextStyle{Monospace: true}
	name.TextSize = theme.CaptionTextSize()

	sub := k.Comment
	if sub == "" {
		sub = "no comment"
	}
	kind := k.Kind
	if kind == "" {
		kind = "KEY"
	}

	left := vStack(0, name, captionLine(sub))
	return container.NewBorder(nil, nil, nil, container.NewCenter(chipText(toneNeutral, kind)),
		inset(0, 0, gapMD, 0, left))
}

// ---------------------------------------------------------------------------
// Service selection
// ---------------------------------------------------------------------------

// serviceContent picks the Git host to add the current key to.
func (u *appUI) serviceContent() fyne.CanvasObject {
	key := u.subjectKey()

	name := canvas.NewText(key, resolve(keyInk))
	name.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	name.TextSize = theme.TextSize() + 2

	top := card(vStack(gapSM,
		sectionLabel("Key to connect"),
		name,
		caption("Only the public half is ever shared with a service. Pick a host to continue."),
	))

	cards := []fyne.CanvasObject{}
	for _, s := range core.Services {
		svc := s
		cards = append(cards, newTapSurface(serviceCardContent(svc), func() {
			u.svc = svc
			u.openServicePage()
			u.navigate(scrInstructions)
		}))
	}

	return vStack(gapXL,
		top,
		sectionLabel("Choose a service"),
		container.NewGridWithColumns(2, cards...),
		buttonRow(secondary("Skip for now", func() { u.navigate(scrOverview) })),
	)
}

// serviceCardContent describes one Git host: its letter mark, the SSH host,
// and the port-443 fallback when it has one.
func serviceCardContent(svc core.Service) fyne.CanvasObject {
	name := canvas.NewText(svc.Name, resolve(keyInk))
	name.TextStyle = fyne.TextStyle{Bold: true}
	name.TextSize = theme.TextSize() + 2

	host := canvas.NewText("git@"+svc.Host, resolve(keyMuted))
	host.TextSize = theme.CaptionTextSize()

	extras := vStack(0, host)
	if svc.AltHost != "" {
		alt := canvas.NewText("Port 443 fallback: "+svc.AltHost, resolve(keyMuted))
		alt.TextSize = theme.CaptionTextSize()
		extras.Add(alt)
	}

	return container.NewBorder(nil, nil, inset(0, 0, gapMD, 0, monogram(initialOf(svc.Name))), nil,
		vStack(gapXS, name, extras))
}

// initialOf is the first letter of a service name, used as its mark.
func initialOf(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// ---------------------------------------------------------------------------
// Instructions
// ---------------------------------------------------------------------------

// instructionsContent walks through adding the key on the host's website, with
// the key material kept visible on the right.
func (u *appUI) instructionsContent() fyne.CanvasObject {
	rows := make([]fyne.CanvasObject, 0, len(u.svc.Steps))
	for i, s := range u.svc.Steps {
		rows = append(rows, stepRow(i+1, stripStepNumber(s)))
	}

	test := primary("I added it — test the connection", u.runTestFlow)
	open := secondary("Open the "+u.svc.Name+" page", u.openServicePage)

	// The two actions stack rather than share a row: side by side they fight
	// over a narrow column and the labels start truncating.
	left := card(vStack(gapLG,
		sectionLabel("Add the key to "+u.svc.Name),
		vStack(gapMD, rows...),
		rule(),
		test,
		open,
	))

	key := u.subjectKey()
	pub := core.PublicKey(key)
	if pub == "" {
		pub = "The public key could not be read for " + key + "."
	}

	aside := vStack(gapLG,
		card(vStack(gapMD,
			sectionLabel("Your public key"),
			codeBlock(elideMiddle(pub, 400)),
			caption("Copied automatically when you opened the page. Paste this into the host's key field."),
			secondary("Copy again", func() { u.copyPubKey(key) }),
		)),
		noteCard(toneAccent, "If the test fails",
			"The result page explains the most likely cause and exactly how to fix it, instead of leaving you with a raw SSH error."),
	)

	return splitColumns(left, aside, 0.64)
}

// stripStepNumber removes the leading "3. " from a core instruction, because
// the GUI renders its own numbered badge. The plain text stays intact for the
// TUI, which relies on it.
func stripStepNumber(step string) string {
	if i := strings.Index(step, ". "); i > 0 && i <= 2 {
		if _, err := strconv.Atoi(step[:i]); err == nil {
			return strings.TrimSpace(step[i+2:])
		}
	}
	return step
}

// stepRow is one numbered instruction: a badge on the left, wrapping copy on
// the right.
func stepRow(n int, text string) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		inset(0, 0, gapMD, 0, numberedBadge(n, toneAccent)), nil,
		inset(gapXS, 0, 0, 0, body(text)))
}

// ---------------------------------------------------------------------------
// Result
// ---------------------------------------------------------------------------

// resultContent is the verdict for the connection test, plus the diagnosis
// when it failed.
func (u *appUI) resultContent() fyne.CanvasObject {
	key := u.subjectKey()

	if u.success {
		return vStack(gapLG,
			verdictCard(toneSuccess, theme.ConfirmIcon(), "Connected",
				key+" authenticated with "+u.svc.Name+"."),
			card(vStack(gapMD,
				sectionLabel("What the server said"),
				codeBlock(firstLineOfDetail(u.lastTest.Output)),
			)),
			nextSteps(u, key),
		)
	}

	items := []fyne.CanvasObject{
		verdictCard(toneDanger, theme.ErrorIcon(), "Not connected",
			key+" did not authenticate with "+u.svc.Name+"."),
	}
	for _, dg := range core.Diagnose(u.lastTest.Output) {
		items = append(items, diagnosisCard(dg))
	}
	items = append(items,
		card(vStack(gapMD,
			sectionLabel("Raw SSH output"),
			codeBlock(u.lastTest.Output),
		)),
		nextSteps(u, key),
	)
	return vStack(gapLG, items...)
}

// boolTone maps a done/not-done flag to the app's status colors.
func boolTone(done bool) tone {
	if done {
		return toneSuccess
	}
	return toneNeutral
}

// verdictCard is the big pass/fail statement at the top of the result page.
func verdictCard(t tone, res fyne.Resource, heading, explanation string) fyne.CanvasObject {
	headline := canvas.NewText(heading, resolve(keyInk))
	headline.TextStyle = fyne.TextStyle{Bold: true}
	headline.TextSize = 22

	_, bg := t.pair()
	g := canvas.NewLinearGradient(resolve(bg), resolve(keySurface), 0)

	return container.NewStack(g, inset(gapXL, gapXL, gapXL, gapXL,
		iconRow(res, 32, vStack(gapXS, headline, body(explanation)))))
}

// diagnosisCard is one cause-and-fix pair from the core diagnoser.
func diagnosisCard(dg core.Diagnosis) fyne.CanvasObject {
	return tintedCard(toneWarning, vStack(gapMD,
		iconRow(theme.WarningIcon(), 18, rich(dg.Cause, keyInk, true)),
		inset(gapMD, 0, 0, 0, body(dg.Fix)),
	))
}

// nextSteps is the action row at the bottom of the result page.
func nextSteps(u *appUI, key string) fyne.CanvasObject {
	return card(vStack(gapMD,
		sectionLabel("What next"),
		container.NewGridWithColumns(2,
			primary("Run the test again", u.runTestFlow),
			widget.NewButton("See the instructions", func() { u.navigate(scrInstructions) }),
			widget.NewButton("Create another key", func() { u.navigate(scrNewKey) }),
			widget.NewButton("Back to overview", func() { u.navigate(scrOverview) }),
		),
		caption("Testing "+key+" again is safe; it only makes a read-only SSH connection."),
	))
}

// ---------------------------------------------------------------------------
// small shared pieces
// ---------------------------------------------------------------------------

// elideMiddle shortens a long single-line string from the middle, which keeps
// both the key prefix and its fingerprint tail readable.
func elideMiddle(s string, max int) string {
	runes := []rune(s)
	if max < 8 || len(runes) <= max {
		return s
	}
	head := (max - 1) / 2
	return string(runes[:head]) + "…" + string(runes[len(runes)-head:])
}
