//go:build !tui

package gui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"keysmith/internal/core"
)

// screen identifies one page of the app. Navigation is no longer a linear
// wizard: every page is reachable from the sidebar at any time.
type screen int

const (
	scrOverview screen = iota
	scrNewKey
	scrKeys
	scrAgent
	scrService
	scrInstructions
	scrResult
)

// page describes the header for one screen: its title, supporting line, and
// how far through the setup flow it sits (0 = outside the flow).
type page struct {
	title    string
	subtitle string
	step     int // 1..3, or 0 when the screen is not part of setup
}

// navItemDef is one sidebar destination. The icon is a function rather than a
// resource because theme lookups need a running app, and package-level
// variables are initialised before Run has started one.
type navItemDef struct {
	screen screen
	label  string
	icon   func() fyne.Resource
}

var navItems = []navItemDef{
	{scrOverview, "Overview", theme.HomeIcon},
	{scrNewKey, "New key", theme.ContentAddIcon},
	{scrKeys, "Keys", theme.StorageIcon},
	{scrAgent, "SSH agent", theme.ComputerIcon},
}

// stepTitles name the three stages of the guided setup flow.
var stepTitles = [3]string{"Create the key", "Connect it", "Verify"}

const (
	railWidth   = 220
	headerPadX  = gapXXL
	headerPadY  = gapLG
	contentPadX = gapXXL
	contentPadY = gapXL
)

// shell composes the persistent application frame: a navigation rail on the
// left, a page header, the content area, and a status bar pinned to the bottom.
// Every screen renders through it, which is what makes the app feel like one
// product rather than a chain of unrelated forms.
type shell struct {
	ui      *appUI
	current page
	content fyne.CanvasObject
}

func (s *shell) render() fyne.CanvasObject {
	main := container.NewBorder(
		s.header(),
		inset(gapSM, gapSM, 0, 0, s.ui.statusBar()),
		nil, nil,
		inset(contentPadY, contentPadY, contentPadX, contentPadX, s.content),
	)
	return container.NewStack(
		canvas.NewRectangle(resolve(keyCanvas)),
		shellFrame(s.rail(), container.NewVScroll(main)),
	)
}

// rail is the left navigation column: brand, destinations, and a live summary
// of the key wall.
func (s *shell) rail() fyne.CanvasObject {
	ui := s.ui

	wordmark := canvas.NewText("KeySmith", resolve(keyInk))
	wordmark.TextStyle = fyne.TextStyle{Bold: true}
	wordmark.TextSize = theme.TextSize() + 4

	tagline := canvas.NewText("SSH key workbench", resolve(keyMuted))
	tagline.TextSize = theme.CaptionTextSize()

	brand := vStack(0, wordmark, tagline)

	navList := vStack(gapXS)
	for _, item := range navItems {
		def := item
		navList.Add(navItem(def.icon(), def.label, ui.screen == def.screen, func() {
			ui.navigate(def.screen)
		}))
	}
	navList.Add(hDivider(0))
	navList.Add(sectionLabel("Quick actions"))

	// The rail carries the two flows that people reach for most often, so
	// they are always one click away regardless of where you are.
	navList.Add(navItem(theme.LoginIcon(), "Test a connection", false,
		func() { ui.startConnectionTestFlow() }))
	navList.Add(navItem(theme.ViewRefreshIcon(), "Refresh agent", false,
		func() { ui.checkAgentFlow() }))

	// Live summary: how many keys exist and how far each one has travelled.
	keys := core.ListKeys()
	summary := vStack(gapSM, sectionLabel("Your keys"))
	if len(keys) == 0 {
		summary.Add(caption("No keys in ~/.ssh yet."))
	} else {
		ready, pending := workflowCounts(ui.store, keys)
		summary.Add(container.NewHBox(
			chipText(toneAccent, verifiedLabel(ready)),
			chipText(toneNeutral, pendingLabel(pending)),
		))
	}

	brandBlock := vStack(gapXL, brand, navList, summary)
	rail := container.NewBorder(
		inset(0, 0, 0, 0, brandBlock),
		inset(0, 0, gapMD, 0, railFooter(ui)),
		nil, nil,
		layout.NewSpacer(),
	)
	railBg := canvas.NewRectangle(resolve(keySurface))
	railBg.StrokeWidth = 1
	railBg.StrokeColor = resolve(keyBorder)
	railBg.SetMinSize(fyne.NewSize(railWidth, 0))

	return container.NewStack(railBg, inset(gapXL, gapLG, gapLG, gapMD, rail))
}

// railFooter shows the agent's reachability at a glance, since that single
// fact drives most of the friction people hit with SSH.
func railFooter(ui *appUI) fyne.CanvasObject {
	t := toneNeutral
	text := "Agent not checked"
	if ui.agentChecked {
		if ui.agent.Reachable {
			t = toneSuccess
			text = "Agent reachable"
		} else {
			t = toneWarning
			text = "Agent unreachable"
		}
	}
	return vStack(gapXS,
		container.NewBorder(nil, nil, statusDot(t), nil, captionLine(text)),
		captionLine("Keys unlock via the agent."),
	)
}

// header is the page title bar. During setup it also renders the three-step
// progress tracker so the wizard's position is always visible.
func (s *shell) header() fyne.CanvasObject {
	head := vStack(gapXS, title(s.current.title))
	if s.current.subtitle != "" {
		head.Add(muted(s.current.subtitle))
	}
	if s.current.step > 0 {
		head = vStack(gapMD, head, stepTracker(s.current.step))
	}
	return inset(headerPadY, gapMD, headerPadX, headerPadX, head)
}

// stepTracker renders the setup flow as three labeled pills joined by short
// rules, with the current stage highlighted.
func stepTracker(current int) fyne.CanvasObject {
	row := container.NewHBox()
	for i := 1; i <= len(stepTitles); i++ {
		if i > 1 {
			row.Add(inset(0, 0, gapSM, 0, hDivider(28)))
		}
		t := toneNeutral
		if i < current {
			t = toneSuccess
		} else if i == current {
			t = toneAccent
		}
		row.Add(pinned(chip(t, stepTitles[i-1])))
	}
	return row
}

// statusBar is the persistent feedback strip: it replaces transient popups
// with a message that stays until the next action replaces it.
func (u *appUI) statusBar() fyne.CanvasObject {
	if u.status == "" {
		return canvas.NewRectangle(resolve(keySurface))
	}
	icon := theme.ConfirmIcon()
	switch u.statusTone {
	case toneDanger:
		icon = theme.ErrorIcon()
	case toneWarning:
		icon = theme.WarningIcon()
	case toneAccent:
		icon = theme.ContentAddIcon()
	}
	return stack(keySurface, 0, true,
		iconRow(icon, 16, muted(u.status)),
		gapMD)
}

// setStatus records a message for the status strip. Tone controls its color.
func (u *appUI) setStatus(msg string, t tone) {
	u.status = msg
	u.statusTone = t
	if u.shell != nil {
		u.win.SetContent(u.shell.render())
	}
}

// clearStatus empties the strip.
func (u *appUI) clearStatus() {
	if u.status != "" {
		u.setStatus("", toneNeutral)
	}
}

// workflowCounts splits a key set into keys that have completed the flow and
// keys that have not, so the rail can show honest progress.
func workflowCounts(store *core.Store, keys []core.KeyInfo) (verified, pending int) {
	for _, k := range keys {
		if store.TestedKeysOK[k.Name] {
			verified++
		} else {
			pending++
		}
	}
	return verified, pending
}

func verifiedLabel(n int) string {
	if n == 1 {
		return "1 verified"
	}
	return strconv.Itoa(n) + " verified"
}

func pendingLabel(n int) string {
	if n == 1 {
		return "1 pending"
	}
	return strconv.Itoa(n) + " pending"
}

// navigate switches pages and re-renders the shell.
func (u *appUI) navigate(s screen) {
	u.screen = s

	var content fyne.CanvasObject
	switch s {
	case scrNewKey:
		content = u.newKeyContent()
	case scrKeys:
		content = u.keysContent()
	case scrAgent:
		content = u.agentContent()
	case scrService:
		content = u.serviceContent()
	case scrInstructions:
		content = u.instructionsContent()
	case scrResult:
		content = u.resultContent()
	default:
		content = u.overviewContent()
	}

	// Fyne's wrapping text only knows its height once it has been given a
	// width, so a freshly built tree reports heights that are far too small.
	// Laying the content out once at the width it is about to receive makes
	// every wrapping child re-measure, which is what keeps paragraphs from
	// overlapping and gives the scroll a correct content height.
	prewarm(content, u.contentWidth())

	u.shell = &shell{ui: u, current: u.pageFor(s), content: content}
	u.win.SetContent(u.shell.render())
}

// contentWidth is the width the content pane will occupy, used to pre-warm the
// page before the canvas lays it out.
func (u *appUI) contentWidth() float32 {
	width := float32(1180) - railWidth - contentPadX*2
	if u.win != nil {
		if w := u.win.Canvas().Size().Width; w > railWidth+contentPadX*2 {
			width = w - railWidth - contentPadX*2
		}
	}
	if width < 360 {
		width = 360
	}
	return width
}

// prewarm lays an object out at a known width so wrapping descendants report
// their real heights.
func prewarm(obj fyne.CanvasObject, width float32) {
	if obj == nil || width <= 0 {
		return
	}
	height := obj.MinSize().Height
	if height < 400 {
		height = 400
	}
	obj.Resize(fyne.NewSize(width, height))
	obj.MinSize()
}

// pageFor maps a screen to its header definition.
func (u *appUI) pageFor(s screen) page {
	switch s {
	case scrOverview:
		return page{title: "Overview", subtitle: "Everything KeySmith can do for your SSH keys."}
	case scrNewKey:
		return page{title: "Create a key", subtitle: "Generate a new key pair in ~/.ssh.", step: 1}
	case scrKeys:
		return page{title: "Keys", subtitle: "Inspect, copy, and manage the keys in ~/.ssh."}
	case scrAgent:
		return page{title: "SSH agent", subtitle: "Check whether this session can reach your agent."}
	case scrService:
		return page{title: "Connect a service", subtitle: "Add this key to a Git host, then verify it.", step: 2}
	case scrInstructions:
		return page{title: "Add the key", subtitle: "Your public key is already on the clipboard.", step: 2}
	case scrResult:
		return page{title: "Verification", subtitle: "The connection test tells you what actually happened.", step: 3}
	}
	return page{title: "KeySmith"}
}
