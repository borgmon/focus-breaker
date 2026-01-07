package main

import (
	"fmt"
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (cw *ConfigWindow) buildAlertTab() fyne.CanvasObject {
	// Create Snooze Duration select with 1-min increments (1-15)
	snoozeOptions := []string{"0 min (disabled)", "1 min", "2 min", "3 min", "4 min", "5 min", "6 min", "7 min", "8 min", "9 min", "10 min", "11 min", "12 min", "13 min", "14 min", "15 min"}
	cw.snoozeTimeSelect = widget.NewSelect(snoozeOptions, func(value string) {
		cw.markChanged()
	})
	currentSnooze := cw.config.SnoozeTime
	if currentSnooze == 0 {
		cw.snoozeTimeSelect.SetSelected("0 min (disabled)")
	} else {
		cw.snoozeTimeSelect.SetSelected(strconv.Itoa(currentSnooze) + " min")
	}

	cw.notifyUnacceptedCheck = widget.NewCheck("Notify for Unaccepted Events", func(checked bool) {
		cw.markChanged()
	})
	cw.notifyUnacceptedCheck.SetChecked(cw.config.NotifyUnaccepted)

	// Create Hold Time select (1-10 seconds)
	holdTimeOptions := []string{"1 sec", "2 sec", "3 sec", "4 sec", "5 sec", "6 sec", "7 sec", "8 sec", "9 sec", "10 sec"}
	cw.holdTimeSelect = widget.NewSelect(holdTimeOptions, func(value string) {
		cw.markChanged()
	})
	currentHoldTime := cw.config.HoldTimeSeconds
	if currentHoldTime < 1 {
		currentHoldTime = 5 // Default to 5 seconds
	}
	if currentHoldTime > 10 {
		currentHoldTime = 10 // Cap at 10 seconds
	}
	cw.holdTimeSelect.SetSelected(strconv.Itoa(currentHoldTime) + " sec")

	// Initialize alert before data from config
	cw.alertBeforeData = []string{}
	if cw.config.AlertBeforeMin != "" {
		alertValues := cw.config.GetAlertMinutes()
		for _, val := range alertValues {
			if val > 0 { // Skip 0 since it's always included
				cw.alertBeforeData = append(cw.alertBeforeData, strconv.Itoa(val))
			}
		}
	}

	// Track selected item index
	var selectedIndex int = -1

	// Create alert before list
	cw.alertBeforeList = widget.NewList(
		func() int {
			return len(cw.alertBeforeData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(cw.alertBeforeData[i] + " min")
		})

	cw.alertBeforeList.OnSelected = func(id widget.ListItemID) {
		selectedIndex = id
	}

	// Create dropdown for adding new alert times
	alertTimeOptions := []string{"1 min", "2 min", "3 min", "5 min", "10 min", "15 min", "20 min", "30 min", "45 min", "60 min", "90 min", "120 min"}
	newAlertSelect := widget.NewSelect(alertTimeOptions, nil)

	// Plus button to add new alert time
	plusButton := widget.NewButton("", func() {
		if newAlertSelect.Selected == "" {
			dialog.ShowInformation("Invalid Input",
				"Please select a number of minutes.",
				cw.window)
			return
		}

		// Extract the number from the selected option (e.g., "5 min" -> "5")
		selectedValue := newAlertSelect.Selected
		var inputStr string
		var val int
		_, err := fmt.Sscanf(selectedValue, "%d min", &val)
		if err != nil || val <= 0 {
			dialog.ShowInformation("Invalid Input",
				"Please select a valid option.",
				cw.window)
			return
		}
		inputStr = strconv.Itoa(val)

		// Check for duplicates
		for _, existing := range cw.alertBeforeData {
			if existing == inputStr {
				dialog.ShowInformation("Duplicate Alert",
					"This alert time has already been added.",
					cw.window)
				newAlertSelect.ClearSelected()
				return
			}
		}

		// Add the new value
		cw.alertBeforeData = append(cw.alertBeforeData, inputStr)

		// Sort the data numerically
		sort.Slice(cw.alertBeforeData, func(i, j int) bool {
			vi, _ := strconv.Atoi(cw.alertBeforeData[i])
			vj, _ := strconv.Atoi(cw.alertBeforeData[j])
			return vi < vj
		})

		cw.alertBeforeList.Refresh()
		newAlertSelect.ClearSelected()
		cw.markChanged()
	})
	plusButton.Icon = theme.ContentAddIcon()

	// Minus button to remove selected alert time
	minusButton := widget.NewButton("", func() {
		if selectedIndex >= 0 && selectedIndex < len(cw.alertBeforeData) {
			// Remove selected item
			cw.alertBeforeData = append(cw.alertBeforeData[:selectedIndex], cw.alertBeforeData[selectedIndex+1:]...)
			cw.alertBeforeList.UnselectAll()
			selectedIndex = -1
			cw.alertBeforeList.Refresh()
			cw.markChanged()
		}
	})
	minusButton.Icon = theme.ContentRemoveIcon()

	addControls := container.NewBorder(nil, nil, nil, container.NewHBox(plusButton, minusButton), newAlertSelect)

	// Wrap list in a scroll container with minimum height
	listScroll := container.NewScroll(cw.alertBeforeList)
	listScroll.SetMinSize(fyne.NewSize(0, 150))

	// Use a bordered container with padding to create a visible boundary
	listWithBorder := container.NewBorder(
		widget.NewSeparator(), // top
		widget.NewSeparator(), // bottom
		widget.NewSeparator(), // left
		widget.NewSeparator(), // right
		listScroll,
	)

	// Create the alert container with list on top and controls on bottom in a VBox
	alertBeforeContainer := container.NewVBox(listWithBorder, addControls)
	cw.alertBeforeContainer = alertBeforeContainer

	// Create labels with help text (in gray)
	alertBeforeLabel := widget.NewLabel("Alert Before:")
	alertBeforeHelp := widget.NewLabel("Always alerts at event start (0 min). Add additional early alerts here.")
	alertBeforeHelp.Wrapping = fyne.TextWrapWord
	alertBeforeHelp.Importance = widget.MediumImportance

	snoozeLabel := widget.NewLabel("Snooze Duration:")
	snoozeHelp := widget.NewLabel("Set to 0 to disable snooze functionality")
	snoozeHelp.Importance = widget.MediumImportance

	notifyLabel := widget.NewLabel("Notify Unaccepted:")
	notifyHelp := widget.NewLabel("If unchecked, only shows alerts for events you've accepted")
	notifyHelp.Wrapping = fyne.TextWrapWord
	notifyHelp.Importance = widget.MediumImportance

	holdTimeLabel := widget.NewLabel("Button Hold Time:")
	holdTimeHelp := widget.NewLabel("How long to hold Close and Snooze buttons to activate")
	holdTimeHelp.Importance = widget.MediumImportance

	// Wrap snooze select and checkbox to control their height
	snoozeContainer := container.NewVBox(cw.snoozeTimeSelect)
	notifyContainer := container.NewVBox(cw.notifyUnacceptedCheck)
	holdTimeContainer := container.NewVBox(cw.holdTimeSelect)

	// Use FormLayout for proper label-value alignment
	form := container.New(layout.NewFormLayout(),
		container.NewVBox(alertBeforeLabel, alertBeforeHelp),
		alertBeforeContainer,

		container.NewVBox(snoozeLabel, snoozeHelp),
		snoozeContainer,

		container.NewVBox(notifyLabel, notifyHelp),
		notifyContainer,

		container.NewVBox(holdTimeLabel, holdTimeHelp),
		holdTimeContainer,
	)

	content := container.NewVBox(
		widget.NewLabel("Alert Settings"),
		widget.NewSeparator(),
		form,
	)

	return container.NewPadded(container.NewVScroll(content))
}
