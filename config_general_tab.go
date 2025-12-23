package main

import (
	"log"
	"os/exec"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (cw *ConfigWindow) buildGeneralTab() fyne.CanvasObject {
	// Auto Start checkbox
	cw.autoStartCheck = widget.NewCheck("Auto Start on System Boot", func(checked bool) {
		cw.markChanged()
	})
	cw.autoStartCheck.SetChecked(cw.config.AutoStart)

	// Get storage root URI
	storageRootURI := cw.app.Storage().RootURI().String()

	// Storage root URI display (read-only)
	storageURIEntry := widget.NewEntry()
	storageURIEntry.SetText(storageRootURI)
	storageURIEntry.Disable()

	// Button to open file manager
	openStorageButton := widget.NewButton("Open in File Manager", func() {
		path := cw.app.Storage().RootURI().Path()
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			cmd = exec.Command("explorer", path)
		case "linux":
			cmd = exec.Command("xdg-open", path)
		default:
			log.Printf("Unsupported OS: %s", runtime.GOOS)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Error opening file manager: %v", err)
		}
	})

	// Create labels with help text
	autoStartLabel := widget.NewLabel("Auto Start:")
	autoStartHelp := widget.NewLabel("Launch Focus Breaker automatically when your system starts")
	autoStartHelp.Importance = widget.MediumImportance

	storageLabel := widget.NewLabel("Storage Location:")
	storageHelp := widget.NewLabel("Application data and settings are stored here")
	storageHelp.Wrapping = fyne.TextWrapWord
	storageHelp.Importance = widget.MediumImportance

	// Container for storage URI and button
	storageContainer := container.NewBorder(
		nil,
		container.NewPadded(openStorageButton),
		nil,
		nil,
		storageURIEntry,
	)

	// Use FormLayout for proper label-value alignment
	form := container.New(layout.NewFormLayout(),
		container.NewVBox(autoStartLabel, autoStartHelp),
		cw.autoStartCheck,

		container.NewVBox(storageLabel, storageHelp),
		storageContainer,
	)

	content := container.NewVBox(
		widget.NewLabel("General Settings"),
		widget.NewSeparator(),
		form,
	)

	return container.NewPadded(container.NewVScroll(content))
}
