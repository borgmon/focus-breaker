package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HoldButton is a button that requires the user to hold it down
type HoldButton struct {
	widget.BaseWidget
	Text        string
	OnHoldStart func()
	OnHoldEnd   func()

	holding  bool
	hovered  bool
	progress float64
}

func NewHoldButton(text string, onHoldStart, onHoldEnd func()) *HoldButton {
	b := &HoldButton{
		Text:        text,
		OnHoldStart: onHoldStart,
		OnHoldEnd:   onHoldEnd,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *HoldButton) CreateRenderer() fyne.WidgetRenderer {
	text := canvas.NewText(b.Text, theme.ForegroundColor())
	text.Alignment = fyne.TextAlignCenter

	bg := canvas.NewRectangle(theme.ButtonColor())
	progressBar := canvas.NewRectangle(theme.PrimaryColor())

	return &holdButtonRenderer{
		button:      b,
		text:        text,
		bg:          bg,
		progressBar: progressBar,
	}
}

func (b *HoldButton) SetProgress(progress float64) {
	b.progress = progress
	b.Refresh()
}

func (b *HoldButton) Tapped(pe *fyne.PointEvent) {
	// Tapped fires on release, we don't use it for hold behavior
}

func (b *HoldButton) TappedSecondary(*fyne.PointEvent) {
}

func (b *HoldButton) MouseIn(me *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *HoldButton) MouseMoved(*desktop.MouseEvent) {
}

func (b *HoldButton) MouseOut() {
	b.hovered = false
	// Stop holding when mouse leaves
	if b.holding {
		b.holding = false
		if b.OnHoldEnd != nil {
			b.OnHoldEnd()
		}
	}
	b.Refresh()
}

func (b *HoldButton) MouseDown(me *desktop.MouseEvent) {
	if !b.holding {
		b.holding = true
		b.Refresh()
		if b.OnHoldStart != nil {
			b.OnHoldStart()
		}
	}
}

func (b *HoldButton) MouseUp(me *desktop.MouseEvent) {
	if b.holding {
		b.holding = false
		b.Refresh()
		if b.OnHoldEnd != nil {
			b.OnHoldEnd()
		}
	}
}

type holdButtonRenderer struct {
	button      *HoldButton
	text        *canvas.Text
	bg          *canvas.Rectangle
	progressBar *canvas.Rectangle
}

func (r *holdButtonRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.text.Resize(size)

	// Progress bar fills from left to right
	progressWidth := size.Width * float32(r.button.progress)
	r.progressBar.Resize(fyne.NewSize(progressWidth, size.Height))
	r.progressBar.Move(fyne.NewPos(0, 0))
}

func (r *holdButtonRenderer) MinSize() fyne.Size {
	textSize := r.text.MinSize()
	minWidth := textSize.Width + theme.Padding()*4
	minHeight := textSize.Height + theme.Padding()*2

	// Set minimum button size to be larger for better usability
	if minWidth < 300 {
		minWidth = 300
	}
	if minHeight < 80 {
		minHeight = 80
	}

	return fyne.NewSize(minWidth, minHeight)
}

func (r *holdButtonRenderer) Refresh() {
	r.text.Text = r.button.Text
	r.text.Color = theme.ForegroundColor()

	if r.button.hovered {
		r.bg.FillColor = theme.HoverColor()
	} else {
		r.bg.FillColor = theme.ButtonColor()
	}

	// Update progress bar layout
	size := r.bg.Size()
	progressWidth := size.Width * float32(r.button.progress)
	r.progressBar.Resize(fyne.NewSize(progressWidth, size.Height))

	r.bg.Refresh()
	r.progressBar.Refresh()
	r.text.Refresh()
}

func (r *holdButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.progressBar, r.text}
}

func (r *holdButtonRenderer) Destroy() {
}

func (r *holdButtonRenderer) BackgroundColor() color.Color {
	return theme.ButtonColor()
}
