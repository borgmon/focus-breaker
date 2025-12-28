package main

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
)

func (cw *ConfigWindow) buildCalendarTab() fyne.CanvasObject {
	// Initialize iCal sources data from config
	cw.icalSourcesData = []ICalSource{}
	for _, source := range cw.config.ICalSources {
		cw.icalSourcesData = append(cw.icalSourcesData, source)
	}

	// Track selected item index
	var selectedIndex int = -1

	// Create iCal sources list
	cw.icalSourcesList = widget.NewList(
		func() int {
			return len(cw.icalSourcesData)
		},
		func() fyne.CanvasObject {
			nameLabel := widget.NewLabel("Name")
			nameLabel.TextStyle.Bold = true
			urlLabel := widget.NewLabel("URL")
			urlLabel.Importance = widget.MediumImportance
			return container.NewVBox(nameLabel, urlLabel)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			vbox := o.(*fyne.Container)
			nameLabel := vbox.Objects[0].(*widget.Label)
			urlLabel := vbox.Objects[1].(*widget.Label)

			source := cw.icalSourcesData[i]
			nameLabel.SetText(source.Name)

			// Truncate long URLs for display
			displayURL := source.URL
			if len(displayURL) > 60 {
				displayURL = displayURL[:57] + "..."
			}
			urlLabel.SetText(displayURL)
		})

	cw.icalSourcesList.OnSelected = func(id widget.ListItemID) {
		selectedIndex = id
	}

	// Plus button to add new iCal source
	plusButton := widget.NewButton("", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("e.g., Work Calendar")
		nameEntry.Validator = func(s string) error {
			if s == "" {
				return fmt.Errorf("name is required")
			}
			return nil
		}

		urlEntry := widget.NewMultiLineEntry()
		urlEntry.SetPlaceHolder("https://calendar.example.com/ical/...")
		urlEntry.Wrapping = fyne.TextWrapBreak
		urlEntry.SetMinRowsVisible(5)
		urlEntry.Validator = func(s string) error {
			if s == "" {
				return fmt.Errorf("URL is required")
			}

			// Basic URL validation - check if it starts with http:// or https://
			if len(s) < 10 {
				return fmt.Errorf("please enter a valid URL (http:// or https://)")
			}
			hasValidPrefix := false
			if len(s) >= 7 && s[:7] == "http://" {
				hasValidPrefix = true
			}
			if len(s) >= 8 && s[:8] == "https://" {
				hasValidPrefix = true
			}
			if !hasValidPrefix {
				return fmt.Errorf("URL must start with http:// or https://")
			}

			// Check for duplicate URLs
			for _, existing := range cw.icalSourcesData {
				if existing.URL == s {
					return fmt.Errorf("this calendar URL has already been added")
				}
			}

			return nil
		}

		formItems := []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("URL", urlEntry),
		}

		addDialog := dialog.NewForm("Add iCal Source", "Add", "Cancel", formItems, func(confirmed bool) {
			if !confirmed {
				return
			}

			// Add the new source with generated UUID
			cw.icalSourcesData = append(cw.icalSourcesData, ICalSource{
				ID:   uuid.New().String(),
				Name: nameEntry.Text,
				URL:  urlEntry.Text,
			})

			cw.icalSourcesList.Refresh()
			cw.markChanged()
		}, cw.window)

		// Resize the dialog to be larger
		addDialog.Resize(fyne.NewSize(600, 300))
		addDialog.Show()
	})
	plusButton.Icon = theme.ContentAddIcon()

	// Minus button to remove selected iCal source
	minusButton := widget.NewButton("", func() {
		if selectedIndex >= 0 && selectedIndex < len(cw.icalSourcesData) {
			sourceName := cw.icalSourcesData[selectedIndex].Name
			dialog.ShowConfirm("Remove Calendar",
				fmt.Sprintf("Are you sure you want to remove '%s'?", sourceName),
				func(confirmed bool) {
					if confirmed {
						// Remove selected item
						cw.icalSourcesData = append(cw.icalSourcesData[:selectedIndex], cw.icalSourcesData[selectedIndex+1:]...)
						cw.icalSourcesList.UnselectAll()
						selectedIndex = -1
						cw.icalSourcesList.Refresh()
						cw.markChanged()
					}
				}, cw.window)
		}
	})
	minusButton.Icon = theme.ContentRemoveIcon()

	addControls := container.NewHBox(plusButton, minusButton)

	// Wrap list in a scroll container with minimum height
	listScroll := container.NewScroll(cw.icalSourcesList)
	listScroll.SetMinSize(fyne.NewSize(0, 200))

	// Use a bordered container with padding to create a visible boundary
	listWithBorder := container.NewBorder(
		widget.NewSeparator(), // top
		widget.NewSeparator(), // bottom
		widget.NewSeparator(), // left
		widget.NewSeparator(), // right
		listScroll,
	)

	// Create the iCal sources container with list on top and controls on bottom in a VBox
	icalSourcesContainer := container.NewVBox(listWithBorder, addControls)

	// Create Update Interval select with 15-min increments (8 options: 15, 30, 45, 60, 75, 90, 105, 120)
	intervalOptions := []string{"15 min", "30 min", "45 min", "60 min", "75 min", "90 min", "105 min", "120 min"}
	cw.updateIntervalSelect = widget.NewSelect(intervalOptions, func(value string) {
		cw.markChanged()
	})
	// Set current value
	currentInterval := cw.config.UpdateInterval
	cw.updateIntervalSelect.SetSelected(strconv.Itoa(currentInterval) + " min")

	syncStatusLabel := widget.NewLabel("")
	syncStatusLabel.Importance = widget.MediumImportance

	cw.syncNowButton = widget.NewButton("Sync Now", func() {
		if cw.onSave != nil {
			cw.syncNowButton.Disable()
			syncStatusLabel.SetText("Syncing calendars...")
			syncStatusLabel.Importance = widget.MediumImportance
			syncStatusLabel.Refresh()

			newConfig := cw.getConfigFromUI()
			go func() {
				cw.onSave(newConfig)
				fyne.Do(func() {
					syncStatusLabel.SetText("Sync completed successfully")
					syncStatusLabel.Importance = widget.SuccessImportance
					syncStatusLabel.Refresh()
					cw.syncNowButton.Enable()
					// Refresh schedules tab after sync
					cw.refreshSchedulesData()

					// Clear sync message after 3 seconds
					go func() {
						time.Sleep(3 * time.Second)
						fyne.Do(func() {
							if syncStatusLabel.Text == "Sync completed successfully" {
								syncStatusLabel.SetText("")
								syncStatusLabel.Refresh()
							}
						})
					}()
				})
			}()
		}
	})
	cw.syncNowButton.Icon = theme.ViewRefreshIcon()

	// Create labels with help text
	icalSourcesLabel := widget.NewLabel("iCal Sources:")
	icalSourcesHelp := widget.NewLabel("Add one or more named calendar sources. Events from all calendars will be synced.")
	icalSourcesHelp.Wrapping = fyne.TextWrapWord
	icalSourcesHelp.Importance = widget.MediumImportance

	updateIntervalLabel := widget.NewLabel("Update Interval:")
	updateIntervalHelp := widget.NewLabel("How often to sync calendar events from all iCal sources")
	updateIntervalHelp.Importance = widget.MediumImportance

	syncLabel := widget.NewLabel("Sync Calendars:")
	syncHelp := widget.NewLabel("Manually sync all calendar sources to fetch the latest events")
	syncHelp.Importance = widget.MediumImportance

	// Wrap update interval select to control its height
	updateIntervalContainer := container.NewVBox(cw.updateIntervalSelect)

	// Sync button container with status
	syncButtonContainer := container.NewVBox(
		container.NewHBox(cw.syncNowButton, syncStatusLabel),
	)

	// Use FormLayout for proper label-value alignment
	form := container.New(layout.NewFormLayout(),
		container.NewVBox(icalSourcesLabel, icalSourcesHelp),
		icalSourcesContainer,

		container.NewVBox(updateIntervalLabel, updateIntervalHelp),
		updateIntervalContainer,

		container.NewVBox(syncLabel, syncHelp),
		syncButtonContainer,
	)

	content := container.NewVBox(
		widget.NewLabel("Calendar Settings"),
		widget.NewSeparator(),
		form,
	)

	return container.NewPadded(container.NewVScroll(content))
}
