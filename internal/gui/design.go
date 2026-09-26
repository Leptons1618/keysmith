package gui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// --- spacing scale ---------------------------------------------------------
//
// One scale for every gap in the app. Screens never invent numbers.

const (
	gapXS  = 4
	gapSM  = 8
	gapMD  = 12
	gapLG  = 16
	gapXL  = 24
	gapXXL = 32
)

// --- radii -----------------------------------------------------------------

const (
	radiusCard    = 14
	radiusControl = 8
	radiusChip    = 6
	radiusTile    = 10
)

// --- semantic tone ---------------------------------------------------------
//
// tone is the app's one status vocabulary. A component asks for a tone and
// receives a matched foreground/background pair, so "this is a success state"
// never leaks as a hard-coded green.

type tone int

const (
	toneNeutral tone = iota
	toneAccent
	toneSuccess
	toneDanger
	toneWarning
)

// pair returns the (foreground, background) color names for a tone.
func (t tone) pair() (fg, bg fyne.ThemeColorName) {
	switch t {
	case toneAccent:
		return keyAccentInk, keyAccentTint
	case toneSuccess:
		return keySuccess, keySuccessTint
	case toneDanger:
		return keyDanger, keyDangerTint
	case toneWarning:
		return keyWarning, keyWarningTint
	default:
		return keyMuted, keySurfaceAlt
	}
}

// fgName returns the tone's foreground color name, for borders and glyphs.
func (t tone) fgName() fyne.ThemeColorName {
	fg, _ := t.pair()
	return fg
}

// resolve looks up a KeySmith semantic color for the variant the app is
// currently painting with. Going through the theme rather than the package
// constants keeps every surface on one palette.
func resolve(name fyne.ThemeColorName) color.Color {
	return theme.Current().Color(name, effectiveVariant())
}

// --- text ------------------------------------------------------------------

// title is a card or section heading.
func title(s string) fyne.CanvasObject {
	t := canvas.NewText(s, resolve(keyInk))
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 16
	return t
}

// label is a small strong label.
func label(s string) fyne.CanvasObject {
	t := canvas.NewText(s, resolve(keyInkBody))
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = theme.TextSize()
	return t
}

// body is wrapping paragraph copy in the body text color.
func body(s string) fyne.CanvasObject {
	return rich(s, keyInkBody, false)
}

// caption is wrapping secondary text. It wraps, because supporting copy is
// where long values and paths end up.
func caption(s string) fyne.CanvasObject {
	return rich(s, keyMuted, false)
}

// muted is an alias for caption, for call sites where the copy reads as
// secondary rather than as a label.
func muted(s string) fyne.CanvasObject {
	return caption(s)
}

// rich is wrapping copy in an arbitrary semantic color, optionally bold.
func rich(s string, name fyne.ThemeColorName, bold bool) fyne.CanvasObject {
	rt := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: name,
			TextStyle: fyne.TextStyle{Bold: bold},
		},
		Text: s,
	})
	rt.Wrapping = fyne.TextWrapWord
	rt.Scroll = fyne.ScrollNone
	return rt
}

// mono is monospaced copy for key material, fingerprints and paths. It breaks
// rather than wraps, because base64 blobs have no spaces to wrap on.
func mono(s string, name fyne.ThemeColorName) fyne.CanvasObject {
	rt := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: name,
			TextStyle: fyne.TextStyle{Monospace: true},
		},
		Text: s,
	})
	rt.Wrapping = fyne.TextWrapBreak
	rt.Scroll = fyne.ScrollNone
	return rt
}

// --- layout ----------------------------------------------------------------

// inset wraps content in a fixed inset.
func inset(top, bottom, left, right float32, objects ...fyne.CanvasObject) *fyne.Container {
	return &fyne.Container{
		Layout:  layout.NewCustomPaddedLayout(top, bottom, left, right),
		Objects: objects,
	}
}

// stack layers a tinted background behind padded content.
func stack(bg fyne.ThemeColorName, radius float32, bordered bool, content fyne.CanvasObject, pad float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(resolve(bg))
	rect.CornerRadius = radius
	if bordered {
		rect.StrokeWidth = 1
		rect.StrokeColor = resolve(keyBorder)
	}
	return container.NewStack(rect, inset(pad, pad, pad, pad, content))
}

// card is the standard raised surface: surface fill, hairline border, and
// generous internal padding. It is the default container for grouped content.
func card(content fyne.CanvasObject) fyne.CanvasObject {
	return stack(keySurface, radiusCard, true, content, gapLG)
}

// tintedCard is a card filled with a tone's background, for status panels.
func tintedCard(t tone, content fyne.CanvasObject) fyne.CanvasObject {
	_, bg := t.pair()
	return stack(bg, radiusCard, true, content, gapLG)
}

// rule is a hairline divider that spans the available width.
func rule() fyne.CanvasObject {
	return pinned(hDivider(0))
}

// hDivider is a fixed-width, one-pixel rule. Width 0 means "fill the row".
func hDivider(width float32) fyne.CanvasObject {
	r := canvas.NewRectangle(resolve(keyBorder))
	if width <= 0 {
		r.SetMinSize(fyne.NewSize(0, 1))
		return r
	}
	r.SetMinSize(fyne.NewSize(width, 1))
	return container.NewCenter(r)
}

// vStack lays children out vertically with a consistent gap, skipping nils so
// optional blocks can be passed straight through.
func vStack(gap float32, objects ...fyne.CanvasObject) *fyne.Container {
	items := make([]fyne.CanvasObject, 0, len(objects))
	for _, o := range objects {
		if o != nil {
			items = append(items, o)
		}
	}
	return &fyne.Container{
		Layout:  &vStackLayout{gap: gap},
		Objects: items,
	}
}

// vStackLayout is the app's vertical layout. It differs from Fyne's VBox in
// two ways that matter:
//
//   - It stretches the final child, so a stack used as a page fills its pane.
//   - It measures before it positions. Fyne's wrapping text reports a height
//     for the width it currently has, which is far too narrow when the tree
//     has just been built. A naive VBox therefore lays wrapped paragraphs out
//     on top of each other. Resizing every child to the width it is about to
//     get, then re-reading MinSize, fixes the heights in a single pass.
type vStackLayout struct {
	gap float32

	// lastWidth remembers the width of the most recent layout so MinSize can
	// re-measure wrapping children at the width they will actually be given.
	lastWidth float32
}

// measure resizes each child to width so wrapping content re-measures, then
// the caller can trust MinSize.
func (l *vStackLayout) measure(objects []fyne.CanvasObject, width float32) {
	for _, o := range objects {
		if o == nil {
			continue
		}
		h := o.MinSize().Height
		if w, ch := o.Size().Width, o.Size().Height; w != width || ch != h {
			o.Resize(fyne.NewSize(width, h))
		}
	}
}

func (l *vStackLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	if l.lastWidth > 0 {
		l.measure(objects, l.lastWidth)
	}
	first := objects[0].MinSize()
	width, height := first.Width, first.Height
	for _, o := range objects[1:] {
		s := o.MinSize()
		if s.Width > width {
			width = s.Width
		}
		height += s.Height + l.gap
	}
	return fyne.NewSize(width, height)
}

func (l *vStackLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	l.lastWidth = size.Width
	l.measure(objects, size.Width)

	y := float32(0)
	for i, o := range objects {
		h := o.MinSize().Height
		if i == len(objects)-1 {
			if remaining := size.Height - y; remaining > h {
				h = remaining
			}
		}
		o.Resize(fyne.NewSize(size.Width, h))
		o.Move(fyne.NewPos(0, y))
		y += h + l.gap
	}
}

// centered horizontally centers a child and lets it keep its natural size.
func centered(obj fyne.CanvasObject) fyne.CanvasObject {
	return container.NewCenter(obj)
}

// railLayout pins a fixed-width navigation rail on the left and gives the rest
// of the window to the content. Fyne's border and split layouts would size the
// rail from its content's minimum size, so the navigation would breathe as
// labels change length.
type railLayout struct {
	width float32
}

func (l *railLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	railWidth := l.width
	if railWidth > size.Width*0.4 {
		railWidth = size.Width * 0.4
	}
	objects[0].Resize(fyne.NewSize(railWidth, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(size.Width-railWidth, size.Height))
	objects[1].Move(fyne.NewPos(railWidth, 0))
}

func (l *railLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(l.width, 0)
	}
	body := objects[1].MinSize()
	return fyne.NewSize(l.width + body.Width, body.Height)
}

// shellFrame composes the pinned rail with the scrolling content pane.
func shellFrame(rail, body fyne.CanvasObject) fyne.CanvasObject {
	return &fyne.Container{
		Layout:  &railLayout{width: railWidth},
		Objects: []fyne.CanvasObject{rail, body},
	}
}

// numberedBadge is a small circular index marker.
func numberedBadge(n int, t tone) fyne.CanvasObject {
	bg := keySurfaceAlt
	fg := keyMuted
	if t == toneAccent {
		bg, fg = keyAccentTint, keyAccentInk
	}
	circle := canvas.NewRectangle(resolve(bg))
	circle.CornerRadius = 11
	circle.SetMinSize(fyne.NewSize(22, 22))

	number := canvas.NewText(strconv.Itoa(n), resolve(fg))
	number.TextStyle = fyne.TextStyle{Bold: true}
	number.TextSize = theme.CaptionTextSize()

	// Centred so a taller row cannot stretch the circle into a pill.
	return container.NewCenter(container.NewStack(circle, container.NewCenter(number)))
}

// splitColumns lays out two children at fixed fractions of the available width.
//
// It replaces container.NewHSplit on every page. HSplit gives the second pane
// whatever width its MinSize asks for, and Fyne's wrapping text reports a
// meaningless minimum width until it has been laid out — so panes collapsed to
// a few pixels. Fractions keep two-pane pages stable at any window size.
func splitColumns(left, right fyne.CanvasObject, leftFraction float32) fyne.CanvasObject {
	return &fyne.Container{
		Layout:  &splitLayout{fraction: leftFraction},
		Objects: []fyne.CanvasObject{left, right},
	}
}

type splitLayout struct {
	fraction float32 // share of the width given to the first child
}

func (l *splitLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	gap := float32(0)
	leftWidth := (size.Width - gap) * l.fraction
	objects[0].Resize(fyne.NewSize(leftWidth, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(size.Width-leftWidth-gap, size.Height))
	objects[1].Move(fyne.NewPos(leftWidth+gap, 0))
}

func (l *splitLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		if len(objects) == 1 {
			return objects[0].MinSize()
		}
		return fyne.NewSize(0, 0)
	}
	a, b := objects[0].MinSize(), objects[1].MinSize()
	height := a.Height
	if b.Height > height {
		height = b.Height
	}
	return fyne.NewSize(a.Width + b.Width, height)
}

// pinLayout keeps a child at its natural size, anchored top-left.
//
// It exists because Fyne's box layouts stretch children along the cross
// axis: a status dot placed in an HBox is stretched to the full row height and
// renders as a tall bar. Every small piece of chrome (dots, rules, icon tiles,
// pill badges) goes through pinned so it stays the size it is meant to be.
type pinLayout struct{}

func (pinLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

func (pinLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(objects[0].MinSize())
	objects[0].Move(fyne.NewPos(0, 0))
}

// pinned wraps a child so surrounding layouts cannot stretch it.
func pinned(obj fyne.CanvasObject) fyne.CanvasObject {
	return &fyne.Container{Layout: pinLayout{}, Objects: []fyne.CanvasObject{obj}}
}

// iconRow places a fixed-size glyph on the left and gives the body all the
// remaining width. An HBox would squeeze the body instead, which breaks
// wrapping text into one word per line.
func iconRow(res fyne.Resource, size float32, bodyObj fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, pinned(glyph(res, size)), nil,
		inset(0, 0, gapMD, 0, bodyObj))
}

// dotRow is a status dot followed by text that wraps freely.
func dotRow(t tone, text string) fyne.CanvasObject {
	return container.NewBorder(nil, nil, statusDot(t), nil,
		inset(gapXS, 0, 0, 0, body(text)))
}

// buttonRow lays out a trailing action cluster, spreading it to the right.
func buttonRow(actions ...fyne.CanvasObject) fyne.CanvasObject {
	row := container.NewHBox()
	for i, a := range actions {
		if i > 0 {
			row.Add(layout.NewSpacer())
		}
		row.Add(a)
	}
	return row
}

// --- glyphs, chips and tiles -----------------------------------------------

// glyph renders a theme icon at a fixed size. Fyne icons are vector-backed, so
// the icon is laid out to fill the box and scales with it.
func glyph(res fyne.Resource, size float32) fyne.CanvasObject {
	icon := widget.NewIcon(res)
	frame := canvas.NewRectangle(color.Transparent)
	frame.SetMinSize(fyne.NewSize(size, size))
	inner := &fyne.Container{
		Layout:  layout.NewMaxLayout(),
		Objects: []fyne.CanvasObject{icon},
	}
	return &fyne.Container{
		Layout:  layout.NewStackLayout(),
		Objects: []fyne.CanvasObject{frame, inner},
	}
}

// statusDot is a small filled circle in a tone's color. It is the repeated
// marker for key state on the key wall and in progress lists.
//
// The result is centred rather than bare so that placing it in an HBox or a
// border slot — both of which stretch their children along one axis — cannot
// squash it into a vertical bar.
func statusDot(t tone) fyne.CanvasObject {
	d := canvas.NewRectangle(resolve(t.fgName()))
	d.CornerRadius = 4
	d.SetMinSize(fyne.NewSize(8, 8))
	return container.NewCenter(d)
}

// chip is a compact status pill: a leading dot plus bold text on a tinted
// background. Use it for state that can change, such as "In agent".
func chip(t tone, text string) fyne.CanvasObject {
	fg, bg := t.pair()
	dot := canvas.NewRectangle(resolve(fg))
	dot.CornerRadius = 3
	dot.SetMinSize(fyne.NewSize(6, 6))

	lbl := canvas.NewText(text, resolve(fg))
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.TextSize = theme.CaptionTextSize()

	// The dot is centred rather than merely pinned so it stays a dot when the
	// row is taller than the pill.
	return stack(bg, radiusChip, false,
		container.NewHBox(container.NewCenter(dot), inset(0, 0, gapSM, 0, lbl)), gapSM)
}

// chipText is a pill without a leading dot, for static labels.
func chipText(t tone, text string) fyne.CanvasObject {
	fg, bg := t.pair()
	lbl := canvas.NewText(text, resolve(fg))
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.TextSize = theme.CaptionTextSize()
	return stack(bg, radiusChip, false, lbl, gapSM)
}

// iconTile is a rounded, tinted badge that anchors an action card.
func iconTile(res fyne.Resource, t tone) fyne.CanvasObject {
	_, bg := t.pair()
	return stack(bg, radiusTile, false, glyph(res, 22), gapSM)
}

// metric is a compact statistic: a large value with a caption underneath.
func metric(value, name string, t tone) fyne.CanvasObject {
	v := canvas.NewText(value, resolve(t.fgName()))
	v.TextStyle = fyne.TextStyle{Bold: true}
	v.TextSize = 22
	return vStack(gapXS, v, caption(name))
}

// sectionLabel is the small uppercase label above a group of content.
func sectionLabel(text string) fyne.CanvasObject {
	l := canvas.NewText(text, resolve(keyMuted))
	l.TextStyle = fyne.TextStyle{Bold: true}
	l.TextSize = theme.CaptionTextSize()
	return l
}

// codeBlock is a recessed monospace panel for key material, fingerprints and
// raw command output.
func codeBlock(text string) fyne.CanvasObject {
	if text == "" {
		text = "—"
	}
	return stack(keySurfaceAlt, radiusControl, true, mono(text, keyInkBody), gapMD)
}

// smallText is a single-line muted label that does not wrap. It reports a
// correct MinSize width, unlike RichText, so it is safe to use in border
// layouts that size themselves from a child's minimum width.
func smallText(s string) fyne.CanvasObject {
	t := canvas.NewText(s, resolve(keyMuted))
	t.TextSize = theme.TextSize()
	return t
}

// captionLine is a non-wrapping secondary label. Fyne's wrapping widgets carry
// an internal inset, so text placed next to them looks a few pixels out. Use
// this wherever a caption sits inline with plain text or a status dot.
func captionLine(s string) fyne.CanvasObject { return smallText(s) }

// rowPair is a name/value line inside a summary card. The label is a fixed
// single line so the border layout can size it, and the value takes all the
// remaining width, which matters for long paths and base64 blobs.
func rowPair(name string, value fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, inset(0, 0, 0, gapMD, smallText(name)), nil, value)
}

// --- buttons ----------------------------------------------------------------

// primary is the main call to action on a screen.
func primary(text string, onTap func()) *widget.Button {
	b := widget.NewButton(text, onTap)
	b.Importance = widget.HighImportance
	return b
}

// secondary is a supporting action.
func secondary(text string, onTap func()) *widget.Button {
	return widget.NewButton(text, onTap)
}

// destructive is an action that removes data.
func destructive(text string, onTap func()) *widget.Button {
	b := widget.NewButton(text, onTap)
	b.Importance = widget.DangerImportance
	return b
}

// quiet is a low-emphasis action.
func quiet(text string, onTap func()) *widget.Button {
	b := widget.NewButton(text, onTap)
	b.Importance = widget.LowImportance
	return b
}

// --- empty state ------------------------------------------------------------

// emptyState is the shared "nothing here yet" panel: an icon, a title, an
// explanation, and the one action that resolves it. The block is centred so it
// sits in the middle of whatever space the panel is given.
func emptyState(res fyne.Resource, heading, explanation string, action fyne.CanvasObject) fyne.CanvasObject {
	body := vStack(gapMD,
		centered(glyph(res, 32)),
		centered(title(heading)),
		inset(gapXL, gapMD, gapXXL, gapXXL, muted(explanation)),
	)
	if action != nil {
		body.Add(container.NewCenter(action))
	}
	return inset(gapXXL, gapXXL, gapXL, gapXL, container.NewCenter(body))
}

// monogram is a rounded, tinted tile carrying a service's initial. Git hosts
// have no theme icons, and a letter mark reads more deliberately than a
// generic glyph repeated three times.
func monogram(letter string) fyne.CanvasObject {
	text := canvas.NewText(letter, resolve(keyAccent))
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = theme.TextSize() + 6
	text.Alignment = fyne.TextAlignCenter
	return stack(keyAccentTint, radiusTile, false, container.NewCenter(text), gapSM)
}
