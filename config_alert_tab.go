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

	// Initialize quiet time data from config
	cw.quietTimeData = make([]TimeRange, len(cw.config.QuietTimeRanges))
	copy(cw.quietTimeData, cw.config.QuietTimeRanges)

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

	// Create quiet time UI
	var selectedQuietIndex int = -1

	cw.quietTimeList = widget.NewList(
		func() int {
			return len(cw.quietTimeData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			tr := cw.quietTimeData[i]
			timeStr := fmt.Sprintf("%02d:%02d - %02d:%02d", tr.StartHour, tr.StartMinute, tr.EndHour, tr.EndMinute)
			o.(*widget.Label).SetText(timeStr)
		})

	cw.quietTimeList.OnSelected = func(id widget.ListItemID) {
		selectedQuietIndex = id
	}

	// Create time entry widgets for adding quiet times
	startHourEntry := widget.NewEntry()
	startHourEntry.SetPlaceHolder("HH")
	startMinuteEntry := widget.NewEntry()
	startMinuteEntry.SetPlaceHolder("MM")
	endHourEntry := widget.NewEntry()
	endHourEntry.SetPlaceHolder("HH")
	endMinuteEntry := widget.NewEntry()
	endMinuteEntry.SetPlaceHolder("MM")

	// Plus button to add new quiet time
	quietPlusButton := widget.NewButton("", func() {
		startHour, err1 := strconv.Atoi(startHourEntry.Text)
		startMinute, err2 := strconv.Atoi(startMinuteEntry.Text)
		endHour, err3 := strconv.Atoi(endHourEntry.Text)
		endMinute, err4 := strconv.Atoi(endMinuteEntry.Text)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			dialog.ShowInformation("Invalid Input", "Please enter valid numbers for all time fields.", cw.window)
			return
		}

		if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
			dialog.ShowInformation("Invalid Hour", "Hours must be between 0 and 23.", cw.window)
			return
		}

		if startMinute < 0 || startMinute > 59 || endMinute < 0 || endMinute > 59 {
			dialog.ShowInformation("Invalid Minute", "Minutes must be between 0 and 59.", cw.window)
			return
		}

		newRange := TimeRange{
			StartHour:   startHour,
			StartMinute: startMinute,
			EndHour:     endHour,
			EndMinute:   endMinute,
		}

		cw.quietTimeData = append(cw.quietTimeData, newRange)
		cw.quietTimeList.Refresh()
		startHourEntry.SetText("")
		startMinuteEntry.SetText("")
		endHourEntry.SetText("")
		endMinuteEntry.SetText("")
		cw.markChanged()
	})
	quietPlusButton.Icon = theme.ContentAddIcon()

	// Minus button to remove selected quiet time
	quietMinusButton := widget.NewButton("", func() {
		if selectedQuietIndex >= 0 && selectedQuietIndex < len(cw.quietTimeData) {
			cw.quietTimeData = append(cw.quietTimeData[:selectedQuietIndex], cw.quietTimeData[selectedQuietIndex+1:]...)
			cw.quietTimeList.UnselectAll()
			selectedQuietIndex = -1
			cw.quietTimeList.Refresh()
			cw.markChanged()
		}
	})
	quietMinusButton.Icon = theme.ContentRemoveIcon()

	quietTimeInputs := container.NewHBox(
		widget.NewLabel("From:"),
		startHourEntry,
		widget.NewLabel(":"),
		startMinuteEntry,
		widget.NewLabel("To:"),
		endHourEntry,
		widget.NewLabel(":"),
		endMinuteEntry,
		quietPlusButton,
		quietMinusButton,
	)

	quietListScroll := container.NewScroll(cw.quietTimeList)
	quietListScroll.SetMinSize(fyne.NewSize(0, 100))

	quietListWithBorder := container.NewBorder(
		widget.NewSeparator(),
		widget.NewSeparator(),
		widget.NewSeparator(),
		widget.NewSeparator(),
		quietListScroll,
	)

	quietTimeContainer := container.NewVBox(quietListWithBorder, quietTimeInputs)

	quietTimeLabel := widget.NewLabel("Quiet Time:")
	quietTimeHelp := widget.NewLabel("Alerts will not be shown during these time ranges (24-hour format)")
	quietTimeHelp.Wrapping = fyne.TextWrapWord
	quietTimeHelp.Importance = widget.MediumImportance

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

		container.NewVBox(quietTimeLabel, quietTimeHelp),
		quietTimeContainer,
	)

	content := container.NewVBox(
		widget.NewLabel("Alert Settings"),
		widget.NewSeparator(),
		form,
	)

	return container.NewPadded(container.NewVScroll(content))
}
