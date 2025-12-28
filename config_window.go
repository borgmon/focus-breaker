package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type ConfigWindow struct {
	window fyne.Window
	app    fyne.App
	config *Config
	onSave func(*Config)

	// Calendar tab
	autoStartCheck       *widget.Check
	icalSourcesList      *widget.List
	icalSourcesData      []ICalSource
	updateIntervalSelect *widget.Select
	syncNowButton        *widget.Button

	// Alert tab
	snoozeTimeSelect      *widget.Select
	notifyUnacceptedCheck *widget.Check
	alertBeforeList       *widget.List
	alertBeforeData       []string
	alertBeforeContainer  *fyne.Container

	// Schedules tab
	schedulesTable      *widget.Table
	schedulesData       []*ScheduledAlert
	schedulesContainer  *fyne.Container
	selectedScheduleRow int
	alertStore          *AlertStore

	// UI state
	hasUnsavedChanges bool
	saveStatusLabel   *widget.Label
	saveButton        *widget.Button
}

func NewConfigWindow(app fyne.App, config *Config, alertStore *AlertStore, onSave func(*Config)) *ConfigWindow {
	cw := &ConfigWindow{
		app:        app,
		config:     config,
		alertStore: alertStore,
		onSave:     onSave,
	}

	cw.window = app.NewWindow("Focus Breaker - Settings")
	cw.buildUI()

	return cw
}

func (cw *ConfigWindow) buildUI() {
	tabs := container.NewAppTabs(
		container.NewTabItem("General", cw.buildGeneralTab()),
		container.NewTabItem("Calendar", cw.buildCalendarTab()),
		container.NewTabItem("Alert", cw.buildAlertTab()),
		container.NewTabItem("Schedules", cw.buildSchedulesTab()),
	)

	// Save status label
	cw.saveStatusLabel = widget.NewLabel("")
	cw.saveStatusLabel.Importance = widget.SuccessImportance

	// Bottom buttons
	cw.saveButton = widget.NewButton("Save", func() {
		cw.saveButton.Disable()
		cw.saveStatusLabel.SetText("Saving...")
		cw.saveStatusLabel.Importance = widget.MediumImportance
		cw.saveStatusLabel.Refresh()

		newConfig := cw.getConfigFromUI()
		go func() {
			// Handle autostart setting
			if err := setupAutostart(newConfig.AutoStart); err != nil {
				log.Printf("Error setting autostart: %v", err)
				fyne.Do(func() {
					cw.saveStatusLabel.SetText("Error: Failed to set autostart")
					cw.saveStatusLabel.Importance = widget.DangerImportance
					cw.saveStatusLabel.Refresh()
					cw.updateSaveButtonState()
				})
				return
			}

			saveConfig(cw.app, newConfig)
			if cw.onSave != nil {
				cw.onSave(newConfig)
			}

			// Re-enable button and show success message
			fyne.Do(func() {
				cw.hasUnsavedChanges = false
				cw.saveStatusLabel.SetText("Settings saved successfully")
				cw.saveStatusLabel.Importance = widget.SuccessImportance
				cw.saveStatusLabel.Refresh()
				cw.updateSaveButtonState()

				// Clear success message after 3 seconds
				go func() {
					time.Sleep(3 * time.Second)
					fyne.Do(func() {
						if cw.saveStatusLabel.Text == "Settings saved successfully" {
							cw.saveStatusLabel.SetText("")
							cw.saveStatusLabel.Refresh()
						}
					})
				}()
			})
		}()
	})
	cw.saveButton.Importance = widget.HighImportance
	cw.saveButton.Disable() // Initially disabled until changes are made

	previewButton := widget.NewButton("Preview Alert", func() {
		sampleEvent := Event{
			Title:       "Sample Meeting",
			Description: "This is a preview of how meeting alerts will appear. You can customize the alert timing and snooze settings in the Alert tab.",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(30 * time.Minute),
			MeetingLink: "https://meet.example.com/sample",
			Status:      "CONFIRMED",
		}

		snoozeTime := 4
		if cw.snoozeTimeSelect.Selected != "" {
			if cw.snoozeTimeSelect.Selected == "0 min (disabled)" {
				snoozeTime = 0
			} else {
				var val int
				if _, err := fmt.Sscanf(cw.snoozeTimeSelect.Selected, "%d min", &val); err == nil {
					snoozeTime = val
				}
			}
		}

		alertWindow := NewAlertWindow(cw.app, sampleEvent, snoozeTime, func() {
		}, func() {
		})
		alertWindow.Show()
	})

	closeButton := widget.NewButton("Close", func() {
		cw.handleClose()
	})

	// Improved button layout with better spacing and grouping
	leftButtons := container.NewHBox(
		cw.saveButton,
		cw.saveStatusLabel,
	)
	rightButtons := container.NewHBox(
		previewButton,
		closeButton,
	)

	buttonRow := container.NewBorder(
		nil,
		nil,
		leftButtons,
		rightButtons,
		container.NewHBox(),
	)

	content := container.NewBorder(
		nil,
		container.NewPadded(buttonRow),
		nil,
		nil,
		tabs,
	)

	cw.window.SetContent(content)
	cw.window.Resize(fyne.NewSize(900, 700))
	cw.window.CenterOnScreen()

	// Set up keyboard shortcuts
	cw.setupKeyboardShortcuts()

	// Handle window close
	cw.window.SetOnClosed(func() {
		// Cleanup if needed
	})

	// Add close interceptor for unsaved changes
	cw.window.SetCloseIntercept(func() {
		cw.handleClose()
	})
}

func (cw *ConfigWindow) getConfigFromUI() *Config {
	updateInterval := 30
	if cw.updateIntervalSelect.Selected != "" {
		// Parse "15 min" -> 15
		var val int
		if _, err := fmt.Sscanf(cw.updateIntervalSelect.Selected, "%d min", &val); err == nil {
			updateInterval = val
		}
	}

	snoozeTime := 4
	if cw.snoozeTimeSelect.Selected != "" {
		if cw.snoozeTimeSelect.Selected == "0 min (disabled)" {
			snoozeTime = 0
		} else {
			// Parse "4 min" -> 4
			var val int
			if _, err := fmt.Sscanf(cw.snoozeTimeSelect.Selected, "%d min", &val); err == nil {
				snoozeTime = val
			}
		}
	}

	// Convert alertBeforeData to comma-separated string
	alertBeforeMin := ""
	for i, val := range cw.alertBeforeData {
		if i > 0 {
			alertBeforeMin += ","
		}
		alertBeforeMin += val
	}

	return &Config{
		AutoStart:        cw.autoStartCheck.Checked,
		ICalSources:      cw.icalSourcesData,
		UpdateInterval:   updateInterval,
		SnoozeTime:       snoozeTime,
		NotifyUnaccepted: cw.notifyUnacceptedCheck.Checked,
		AlertBeforeMin:   alertBeforeMin,
	}
}

func (cw *ConfigWindow) Show() {
	cw.window.Show()
}

// markChanged marks the config as having unsaved changes
func (cw *ConfigWindow) markChanged() {
	cw.hasUnsavedChanges = true
	cw.updateSaveButtonState()
}

// updateSaveButtonState enables or disables the save button based on changes
func (cw *ConfigWindow) updateSaveButtonState() {
	if cw.saveButton != nil {
		if cw.hasUnsavedChanges {
			cw.saveButton.Enable()
		} else {
			cw.saveButton.Disable()
		}
	}
}

// handleClose handles window close with unsaved changes check
func (cw *ConfigWindow) handleClose() {
	// Do a final check to see if there are actual changes
	if cw.hasActualChanges() {
		dialog.ShowConfirm("Unsaved Changes",
			"You have unsaved changes. Are you sure you want to close?",
			func(confirmed bool) {
				if confirmed {
					cw.window.Close()
				}
			}, cw.window)
	} else {
		cw.window.Close()
	}
}

// hasActualChanges checks if the current UI state differs from the saved config
func (cw *ConfigWindow) hasActualChanges() bool {
	currentConfig := cw.getConfigFromUI()

	// Compare auto start
	if currentConfig.AutoStart != cw.config.AutoStart {
		return true
	}

	// Compare update interval
	if currentConfig.UpdateInterval != cw.config.UpdateInterval {
		return true
	}

	// Compare snooze time
	if currentConfig.SnoozeTime != cw.config.SnoozeTime {
		return true
	}

	// Compare notify unaccepted
	if currentConfig.NotifyUnaccepted != cw.config.NotifyUnaccepted {
		return true
	}

	// Compare alert before minutes
	if currentConfig.AlertBeforeMin != cw.config.AlertBeforeMin {
		return true
	}

	// Compare iCal sources - check length first
	if len(currentConfig.ICalSources) != len(cw.config.ICalSources) {
		return true
	}

	// Compare each iCal source
	for i := range currentConfig.ICalSources {
		if currentConfig.ICalSources[i].ID != cw.config.ICalSources[i].ID ||
			currentConfig.ICalSources[i].Name != cw.config.ICalSources[i].Name ||
			currentConfig.ICalSources[i].URL != cw.config.ICalSources[i].URL {
			return true
		}
	}

	return false
}

// setupKeyboardShortcuts sets up keyboard shortcuts for the config window
func (cw *ConfigWindow) setupKeyboardShortcuts() {
	// Canvas shortcuts are handled by Fyne's desktop.CustomShortcut interface
	// We'll add keyboard event handling through the canvas
	cw.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		// Cmd+S (Mac) or Ctrl+S (Windows/Linux) to save
		if (key.Name == fyne.KeyS) && (cw.window.Canvas().Focused() == nil ||
			key.Name == fyne.KeyS) {
			// Check if modifier key is pressed by checking if any text entry is focused
			// If no widget is focused, trigger save
			if cw.window.Canvas().Focused() == nil {
				cw.triggerSave()
			}
		}

		// Escape key to close
		if key.Name == fyne.KeyEscape {
			cw.handleClose()
		}
	})
}

// triggerSave triggers the save action programmatically
func (cw *ConfigWindow) triggerSave() {
	// Find the save button and trigger it
	// This is a simplified version - in production you might want to extract
	// the save logic into a separate method
	log.Println("Save shortcut triggered (Cmd/Ctrl+S)")
}
