package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ListManager provides a reusable component for managing lists with add/remove functionality
type ListManager struct {
	list         *widget.List
	data         []string
	selectedIdx  int
	onAdd        func(string) error // Returns error if validation fails
	onRemove     func(int)
	onChange     func()
	renderItem   func(int) string
	addControl   fyne.CanvasObject // Custom add control
}

// ListManagerConfig configures the list manager
type ListManagerConfig struct {
	RenderItem   func(int) string          // Renders a data item as string for display
	OnAdd        func(string) error        // Called when adding an item
	OnRemove     func(int)                 // Called when removing an item
	OnChange     func()                    // Called when list changes
	AddControl   fyne.CanvasObject         // Custom add control (optional)
}

// NewListManager creates a new list manager component
func NewListManager(data []string, config ListManagerConfig) (*ListManager, *fyne.Container) {
	lm := &ListManager{
		data:       data,
		selectedIdx: -1,
		onAdd:      config.OnAdd,
		onRemove:   config.OnRemove,
		onChange:   config.OnChange,
		renderItem: config.RenderItem,
		addControl: config.AddControl,
	}

	// Create list widget
	lm.list = widget.NewList(
		func() int {
			return len(lm.data)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i < len(lm.data) {
				text := lm.data[i]
				if lm.renderItem != nil {
					text = lm.renderItem(i)
				}
				label.SetText(text)
			}
		})

	lm.list.OnSelected = func(id widget.ListItemID) {
		lm.selectedIdx = id
	}

	// Create control buttons
	plusButton := widget.NewButton("", func() {
		if config.OnAdd != nil {
			// OnAdd should handle the UI for getting input
		}
	})
	plusButton.Icon = theme.ContentAddIcon()

	minusButton := widget.NewButton("", func() {
		if lm.selectedIdx >= 0 && lm.selectedIdx < len(lm.data) {
			if lm.onRemove != nil {
				lm.onRemove(lm.selectedIdx)
			}
			lm.data = append(lm.data[:lm.selectedIdx], lm.data[lm.selectedIdx+1:]...)
			lm.list.UnselectAll()
			lm.selectedIdx = -1
			lm.list.Refresh()
			if lm.onChange != nil {
				lm.onChange()
			}
		}
	})
	minusButton.Icon = theme.ContentRemoveIcon()

	// Build controls section
	var addControls *fyne.Container
	if lm.addControl != nil {
		addControls = container.NewBorder(nil, nil, nil,
			container.NewHBox(plusButton, minusButton),
			lm.addControl)
	} else {
		addControls = container.NewHBox(plusButton, minusButton)
	}

	// Wrap list in scroll with border
	listScroll := container.NewScroll(lm.list)
	listScroll.SetMinSize(fyne.NewSize(0, 150))

	listWithBorder := container.NewBorder(
		widget.NewSeparator(),
		widget.NewSeparator(),
		widget.NewSeparator(),
		widget.NewSeparator(),
		listScroll,
	)

	// Combine list and controls
	listContainer := container.NewVBox(listWithBorder, addControls)

	return lm, listContainer
}

// Refresh refreshes the list display
func (lm *ListManager) Refresh() {
	lm.list.Refresh()
}

// GetData returns the current data
func (lm *ListManager) GetData() []string {
	return lm.data
}

// SetData updates the data and refreshes
func (lm *ListManager) SetData(data []string) {
	lm.data = data
	lm.list.Refresh()
}

// AddItem adds an item to the list
func (lm *ListManager) AddItem(item string) {
	lm.data = append(lm.data, item)
	lm.list.Refresh()
	if lm.onChange != nil {
		lm.onChange()
	}
}

// RemoveSelected removes the currently selected item
func (lm *ListManager) RemoveSelected() {
	if lm.selectedIdx >= 0 && lm.selectedIdx < len(lm.data) {
		if lm.onRemove != nil {
			lm.onRemove(lm.selectedIdx)
		}
		lm.data = append(lm.data[:lm.selectedIdx], lm.data[lm.selectedIdx+1:]...)
		lm.list.UnselectAll()
		lm.selectedIdx = -1
		lm.list.Refresh()
		if lm.onChange != nil {
			lm.onChange()
		}
	}
}
