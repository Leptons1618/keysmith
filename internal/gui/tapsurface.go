//go:build !tui

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tapSurface is the app's interactive card: a rounded, optionally bordered
// panel that reacts to hover and press. Fyne's Button cannot do this because it
// always paints its own chrome, so every selectable thing in the app (home
// actions, sidebar destinations, algorithm choices, key rows) is built on this
// instead.
type tapSurface struct {
	widget.BaseWidget

	onTap func()

	bg, hover, pressed fyne.ThemeColorName
	border             fyne.ThemeColorName
	selected           bool

	radius  float32
	pad     float32
	padX    float32
	content fyne.CanvasObject

	hovering bool
	holding  bool
}

var (
	_ fyne.Widget        = (*tapSurface)(nil)
	_ fyne.Tappable      = (*tapSurface)(nil)
	_ desktop.Hoverable  = (*tapSurface)(nil)
	_ desktop.Mouseable  = (*tapSurface)(nil)
	_ desktop.Cursorable = (*tapSurface)(nil)
)

func newTapSurface(content fyne.CanvasObject, onTap func()) *tapSurface {
	t := &tapSurface{
		onTap:   onTap,
		bg:      keySurface,
		hover:   keySurfaceAlt,
		pressed: keyAccentTint,
		border:  keyBorder,
		radius:  radiusCard,
		pad:     gapLG,
		padX:    gapLG,
		content: content,
	}
	t.ExtendBaseWidget(t)
	return t
}

// withSelected renders the surface in the accent wash, marking it as the
// current choice in a set. It is symmetric: passing false restores the resting
// state, so a group of cards can be re-selected without being rebuilt.
func (t *tapSurface) withSelected(on bool) *tapSurface {
	t.selected = on
	if on {
		t.bg = keyAccentTint
		t.hover = keyAccentTint
		t.pressed = keyAccentTint
		t.border = keyAccent
		return t
	}
	t.bg = keySurface
	t.hover = keySurfaceAlt
	t.pressed = keyAccentTint
	t.border = keyBorder
	return t
}

// withPad overrides the internal inset, for denser rows.
func (t *tapSurface) withPad(pad float32) *tapSurface {
	t.pad = pad
	return t
}

// withPadXY sets vertical and horizontal insets independently, which is what
// list rows want: tight vertically, comfortable horizontally.
func (t *tapSurface) withPadXY(vertical, horizontal float32) *tapSurface {
	t.pad = vertical
	t.padX = horizontal
	return t
}

// setContent swaps the body in place, so a selected row can change its label
// without the widget being rebuilt.
func (t *tapSurface) setContent(content fyne.CanvasObject) {
	t.content = content
	t.Refresh()
}

func (t *tapSurface) CreateRenderer() fyne.WidgetRenderer {
	t.ExtendBaseWidget(t)
	bg := canvas.NewRectangle(resolve(t.bg))
	bg.CornerRadius = t.radius
	bg.StrokeWidth = 1
	bg.StrokeColor = resolve(t.border)
	r := &tapSurfaceRenderer{
		objects: []fyne.CanvasObject{bg, t.content},
		surface: t,
		bg:      bg,
	}
	r.applyState()
	return r
}

func (t *tapSurface) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tapSurface) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (t *tapSurface) MouseIn(*desktop.MouseEvent) {
	t.hovering = true
	t.Refresh()
}

func (t *tapSurface) MouseMoved(*desktop.MouseEvent) {}

func (t *tapSurface) MouseOut() {
	t.hovering = false
	t.holding = false
	t.Refresh()
}

func (t *tapSurface) MouseDown(*desktop.MouseEvent) {
	t.holding = true
	t.Refresh()
}

func (t *tapSurface) MouseUp(*desktop.MouseEvent) {
	t.holding = false
	t.Refresh()
}

// tapSurfaceRenderer draws the rounded background and positions the body with
// the surface's inset.
type tapSurfaceRenderer struct {
	objects []fyne.CanvasObject
	surface *tapSurface
	bg      *canvas.Rectangle
}

func (r *tapSurfaceRenderer) applyState() {
	fill := resolve(r.surface.bg)
	switch {
	case r.surface.holding:
		fill = resolve(r.surface.pressed)
	case r.surface.hovering:
		fill = resolve(r.surface.hover)
	}
	r.bg.FillColor = fill
	r.bg.StrokeColor = resolve(r.surface.border)
	if r.surface.selected {
		r.bg.StrokeColor = resolve(keyAccent)
	}
	r.bg.Refresh()
}

func (r *tapSurfaceRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *tapSurfaceRenderer) Destroy() {}

func (r *tapSurfaceRenderer) Layout(_ fyne.Size) {
	if len(r.objects) < 2 || r.objects[1] == nil {
		return
	}
	size := r.surface.Size()
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	inner := fyne.NewSize(size.Width-r.surface.padX*2, size.Height-r.surface.pad*2)
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	r.objects[1].Resize(inner)
	r.objects[1].Move(fyne.NewPos(r.surface.padX, r.surface.pad))
}

func (r *tapSurfaceRenderer) MinSize() fyne.Size {
	padY, padX := r.surface.pad, r.surface.padX
	if len(r.objects) < 2 || r.objects[1] == nil {
		return fyne.NewSize(padX*2, padY*2)
	}
	inner := r.objects[1].MinSize()
	return fyne.NewSize(inner.Width+padX*2, inner.Height+padY*2)
}

func (r *tapSurfaceRenderer) Refresh() {
	r.applyState()
	if len(r.objects) >= 2 && r.objects[1] != r.surface.content {
		r.objects[1] = r.surface.content
	}
	r.Layout(fyne.NewSize(0, 0))
	for _, o := range r.objects {
		if o != nil {
			o.Refresh()
		}
	}
}

// iconBadge pairs a tinted icon tile with a bold heading and explanation. It is
// the repeated structure of every action card in the app.
func iconBadge(res fyne.Resource, t tone, heading, explanation string) fyne.CanvasObject {
	headingText := canvas.NewText(heading, resolve(keyInk))
	headingText.TextStyle = fyne.TextStyle{Bold: true}
	headingText.TextSize = theme.TextSize() + 1

	return vStack(gapSM,
		pinned(iconTile(res, t)),
		headingText,
		rich(explanation, keyMuted, false),
	)
}

// navItem is one destination in the sidebar rail.
func navItem(res fyne.Resource, name string, active bool, onTap func()) fyne.CanvasObject {
	nameText := canvas.NewText(name, resolve(keyMuted))
	nameText.TextStyle = fyne.TextStyle{Bold: active}
	nameText.TextSize = theme.TextSize()
	if active {
		nameText.Color = resolve(keyInk)
	}

	content := container.NewHBox(
		glyph(res, 18),
		inset(0, 0, gapSM, 0, nameText),
	)
	return newTapSurface(content, onTap).withPadXY(gapSM, gapMD).withSelected(active)
}

// chipRow lays out pills left-to-right without stretching them.
func chipRow(chips ...fyne.CanvasObject) fyne.CanvasObject {
	row := container.NewHBox()
	for i, c := range chips {
		if i > 0 {
			row.Add(inset(0, 0, gapSM, 0, layout.NewSpacer()))
		}
		row.Add(pinned(c))
	}
	return row
}
