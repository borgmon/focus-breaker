package components

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

// NewHoldButton creates a new HoldButton
func NewHoldButton(text string, onHoldStart, onHoldEnd func()) *HoldButton {
	b := &HoldButton{
		Text:        text,
		OnHoldStart: onHoldStart,
		OnHoldEnd:   onHoldEnd,
	}
	b.ExtendBaseWidget(b)
	return b
}

// CreateRenderer implements fyne.Widget
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

// SetProgress updates the progress bar
func (b *HoldButton) SetProgress(progress float64) {
	b.progress = progress
	b.Refresh()
}

// Tapped implements fyne.Tappable
func (b *HoldButton) Tapped(*fyne.PointEvent) {}

// TappedSecondary implements fyne.SecondaryTappable
func (b *HoldButton) TappedSecondary(*fyne.PointEvent) {}

// MouseIn implements desktop.Hoverable
func (b *HoldButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

// MouseMoved implements desktop.Hoverable
func (b *HoldButton) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable
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

// MouseDown implements desktop.Mouseable
func (b *HoldButton) MouseDown(*desktop.MouseEvent) {
	if !b.holding {
		b.holding = true
		b.Refresh()
		if b.OnHoldStart != nil {
			b.OnHoldStart()
		}
	}
}

// MouseUp implements desktop.Mouseable
func (b *HoldButton) MouseUp(*desktop.MouseEvent) {
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

	// Set minimum button size for better usability
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

func (r *holdButtonRenderer) Destroy() {}

func (r *holdButtonRenderer) BackgroundColor() color.Color {
	return theme.ButtonColor()
}
