package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (cw *ConfigWindow) showAddAlarmDialog() {
	hourEntry := widget.NewEntry()
	hourEntry.SetPlaceHolder("0")
	minEntry := widget.NewEntry()
	minEntry.SetPlaceHolder("0")

	items := []*widget.FormItem{
		widget.NewFormItem("Hours", hourEntry),
		widget.NewFormItem("Minutes", minEntry),
	}

	dialog.ShowForm("Add Alarm", "Create", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}

		// Parse hour and minute inputs
		hours := 0
		mins := 0

		if hourEntry.Text != "" {
			fmt.Sscanf(hourEntry.Text, "%d", &hours)
		}
		if minEntry.Text != "" {
			fmt.Sscanf(minEntry.Text, "%d", &mins)
		}

		// Validate inputs
		if hours < 0 || mins < 0 {
			dialog.ShowError(fmt.Errorf("Hours and minutes must be positive numbers"), cw.window)
			return
		}

		if hours == 0 && mins == 0 {
			dialog.ShowError(fmt.Errorf("Please set at least 1 minute or 1 hour"), cw.window)
			return
		}

		// Calculate alarm time
		totalMinutes := hours*60 + mins
		alarmTime := time.Now().Add(time.Duration(totalMinutes) * time.Minute)

		// Create a manual event
		event := &Event{
			ID:          fmt.Sprintf("manual-%s", time.Now().Format("20060102-150405")),
			Title:       fmt.Sprintf("Manual Alarm (%dh %dm)", hours, mins),
			Description: "Manually created countdown alarm",
			StartTime:   alarmTime,
			EndTime:     alarmTime.Add(5 * time.Minute),
			MeetingLink: "",
			Status:      "CONFIRMED",
			SourceID:    "manual",
		}

		// Add event and alert to the alert store
		cw.alertStore.AddManualAlert(event, alarmTime)

		// Refresh the schedules table
		cw.refreshSchedulesData()
	}, cw.window)
}

func (cw *ConfigWindow) showDeleteAlertDialog() {
	if len(cw.schedulesData) == 0 {
		dialog.ShowInformation("No Alerts", "There are no alerts to delete.", cw.window)
		return
	}

	// Check if a row is selected
	if cw.selectedScheduleRow < 0 || cw.selectedScheduleRow >= len(cw.schedulesData) {
		dialog.ShowInformation("No Selection", "Please select an alert from the table to delete.", cw.window)
		return
	}

	// Get the selected alert
	schedule := cw.schedulesData[cw.selectedScheduleRow]

	// Remove the event (which also removes all associated alerts)
	cw.alertStore.removeEvent(schedule.EventID)

	// Reset selection
	cw.selectedScheduleRow = -1

	// Refresh the schedules table
	cw.refreshSchedulesData()
}
