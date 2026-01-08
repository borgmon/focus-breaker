package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/borgmon/focus-breaker/pkg/models"
)

func (cw *ConfigWindow) buildEventsTab() fyne.CanvasObject {
	// Get initial events data
	cw.eventsData = cw.getEventsDisplayInfo()

	// Create table widget
	table := widget.NewTable(
		func() (rows int, cols int) {
			return len(cw.eventsData), 5
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)

			if id.Row >= len(cw.eventsData) {
				label.SetText("")
				return
			}

			displayInfo := cw.eventsData[id.Row]
			event := displayInfo.event

			// Get source name
			sourceName := ""
			for _, source := range cw.config.ICalSources {
				if source.ID == event.SourceID {
					sourceName = source.Name
					break
				}
			}

			// Set cell content based on column
			switch id.Col {
			case 0:
				label.SetText(event.Title)
			case 1:
				label.SetText(sourceName)
			case 2:
				label.SetText(event.StartTime.Format("Mon Jan 2, 3:04 PM"))
			case 3:
				label.SetText(displayInfo.alertStatus)
			case 4:
				label.SetText(displayInfo.reason)
			}

			// Gray out past events
			if event.StartTime.Before(time.Now()) {
				label.Importance = widget.LowImportance
			} else {
				label.Importance = widget.MediumImportance
			}

			// Color code alert status
			if id.Col == 3 {
				switch displayInfo.alertStatus {
				case "Alerted":
					label.Importance = widget.SuccessImportance
				case "Pending":
					label.Importance = widget.WarningImportance
				case "Filtered":
					label.Importance = widget.LowImportance
				}
			}

			label.TextStyle.Bold = false
		},
	)

	// Enable header row
	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		label := widget.NewLabel("Header")
		label.TextStyle.Bold = true
		return label
	}
	table.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)
		switch id.Col {
		case 0:
			label.SetText("Event")
		case 1:
			label.SetText("Calendar")
		case 2:
			label.SetText("Start Time")
		case 3:
			label.SetText("Alert Status")
		case 4:
			label.SetText("Reason")
		}
	}

	// Calculate column widths
	cw.updateEventsColumnWidths(table)

	// Store reference for refresh
	cw.eventsTable = table

	refreshButton := widget.NewButton("Refresh", func() {
		cw.refreshEventsData()
	})
	refreshButton.Icon = theme.ViewRefreshIcon()

	helpText := widget.NewLabel("Shows all events from your calendars with their alert status. 'Alerted' means alerts are scheduled, 'Filtered' means alerts are suppressed, and 'Pending' means alerts are waiting to fire.")
	helpText.Wrapping = fyne.TextWrapWord
	helpText.Importance = widget.MediumImportance

	buttonContainer := container.NewHBox(refreshButton)

	headerContent := container.NewVBox(
		widget.NewLabel("Events"),
		widget.NewSeparator(),
		helpText,
		buttonContainer,
	)

	// Check if there are no events
	var mainContent fyne.CanvasObject
	if len(cw.eventsData) == 0 {
		emptyStateText := widget.NewLabel("No events yet.\n\nTo get started:\n1. Add calendar sources in the Calendar tab\n2. Click 'Sync Now' to fetch events\n3. Events will appear here")
		emptyStateText.Wrapping = fyne.TextWrapWord
		emptyStateText.Importance = widget.MediumImportance
		mainContent = container.NewPadded(emptyStateText)
	} else {
		mainContent = table
	}

	// Store the container for dynamic updates
	cw.eventsContainer = container.NewBorder(
		headerContent,
		nil,
		nil,
		nil,
		mainContent,
	)

	return container.NewPadded(cw.eventsContainer)
}

func (cw *ConfigWindow) refreshEventsData() {
	if cw.eventsContainer == nil {
		return
	}

	// Update the events data
	cw.eventsData = cw.getEventsDisplayInfo()

	// If we have a table, recalculate column widths and refresh it
	if cw.eventsTable != nil {
		cw.updateEventsColumnWidths(cw.eventsTable)
		cw.eventsTable.Refresh()
	}

	// Update the main content based on whether we have data
	var mainContent fyne.CanvasObject
	if len(cw.eventsData) == 0 {
		emptyStateText := widget.NewLabel("No events yet.\n\nTo get started:\n1. Add calendar sources in the Calendar tab\n2. Click 'Sync Now' to fetch events\n3. Events will appear here")
		emptyStateText.Wrapping = fyne.TextWrapWord
		emptyStateText.Importance = widget.MediumImportance
		mainContent = container.NewPadded(emptyStateText)
	} else {
		mainContent = cw.eventsTable
	}

	// Update the container's content
	cw.eventsContainer.Objects[0] = mainContent
	cw.eventsContainer.Refresh()
}

func (cw *ConfigWindow) updateEventsColumnWidths(table *widget.Table) {
	// Calculate maximum width needed for each column
	headers := []string{"Event", "Calendar", "Start Time", "Alert Status", "Reason"}
	columnWidths := make([]float32, 5)

	charWidth := float32(8)
	padding := float32(20)

	// Initialize with header widths
	for i, header := range headers {
		columnWidths[i] = float32(len(header))*charWidth + padding
	}

	// Calculate widths based on actual data
	for _, displayInfo := range cw.eventsData {
		sourceName := ""
		for _, source := range cw.config.ICalSources {
			if source.ID == displayInfo.event.SourceID {
				sourceName = source.Name
				break
			}
		}

		widths := []int{
			len(displayInfo.event.Title),
			len(sourceName),
			len(displayInfo.event.StartTime.Format("Mon Jan 2, 3:04 PM")),
			len(displayInfo.alertStatus),
			len(displayInfo.reason),
		}

		for i, width := range widths {
			contentWidth := float32(width)*charWidth + padding
			if contentWidth > columnWidths[i] {
				columnWidths[i] = contentWidth
			}
		}
	}

	// Set minimum and maximum widths
	minWidths := []float32{150, 100, 180, 100, 200}
	maxWidths := []float32{400, 200, 200, 120, 400}

	for i := range columnWidths {
		if columnWidths[i] < minWidths[i] {
			columnWidths[i] = minWidths[i]
		}
		if columnWidths[i] > maxWidths[i] {
			columnWidths[i] = maxWidths[i]
		}
		table.SetColumnWidth(i, columnWidths[i])
	}
}

func (cw *ConfigWindow) getEventsDisplayInfo() []eventDisplayInfo {
	if cw.alertStore == nil {
		return []eventDisplayInfo{}
	}

	// Get all scheduled alerts
	allAlerts := cw.alertStore.GetAllScheduledAlerts()

	// Build a map of event ID to alert information
	eventAlerts := make(map[string][]models.AlertStatus)
	for _, alert := range allAlerts {
		eventAlerts[alert.EventID] = append(eventAlerts[alert.EventID], alert.Status)
	}

	// Get all events from alert store
	result := []eventDisplayInfo{}
	now := time.Now()
	cutoffTime := now.Add(-12 * time.Hour)

	// We need to access the internal events map - let's iterate through alerts to find events
	seenEvents := make(map[string]bool)
	for _, alert := range allAlerts {
		if seenEvents[alert.EventID] {
			continue
		}
		seenEvents[alert.EventID] = true

		event := cw.alertStore.GetEvent(alert.EventID)
		if event == nil || event.StartTime.Before(cutoffTime) {
			continue
		}

		displayInfo := cw.determineEventStatus(event, eventAlerts[alert.EventID])
		result = append(result, displayInfo)
	}

	return result
}

func (cw *ConfigWindow) determineEventStatus(event *models.Event, alertStatuses []models.AlertStatus) eventDisplayInfo {
	now := time.Now()

	// Check if event is in the past
	if event.StartTime.Before(now) {
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Alerted",
			reason:      "Event has passed",
		}
	}

	// Check if event is unaccepted and we're not notifying for unaccepted
	if event.Status == "NEEDS-ACTION" && !cw.config.NotifyUnaccepted {
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Filtered",
			reason:      "Event not accepted (NEEDS-ACTION)",
		}
	}

	// Analyze alert statuses
	hasPending := false
	hasMuted := false
	hasSnoozed := false
	hasAlerted := false

	for _, status := range alertStatuses {
		switch status {
		case models.AlertStatusPending:
			hasPending = true
		case models.AlertStatusMuted:
			hasMuted = true
		case models.AlertStatusSnoozed:
			hasSnoozed = true
		case models.AlertStatusAlerted:
			hasAlerted = true
		}
	}

	// Determine status based on alerts
	if hasPending {
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Pending",
			reason:      fmt.Sprintf("Alert scheduled (%d alert(s) pending)", countStatus(alertStatuses, models.AlertStatusPending)),
		}
	}

	if hasSnoozed {
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Pending",
			reason:      "Alert snoozed",
		}
	}

	if hasMuted {
		// Check if it's because of quiet time
		alertMinutes := cw.config.GetAlertMinutes()
		for _, minutes := range alertMinutes {
			alertTime := event.StartTime.Add(-time.Duration(minutes) * time.Minute)
			if cw.config.IsTimeInQuietTime(alertTime) {
				return eventDisplayInfo{
					event:       event,
					alertStatus: "Filtered",
					reason:      "In quiet time",
				}
			}
		}
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Filtered",
			reason:      "Alert muted",
		}
	}

	if hasAlerted {
		return eventDisplayInfo{
			event:       event,
			alertStatus: "Alerted",
			reason:      fmt.Sprintf("Alert shown (%d alert(s) fired)", countStatus(alertStatuses, models.AlertStatusAlerted)),
		}
	}

	// No alerts scheduled
	return eventDisplayInfo{
		event:       event,
		alertStatus: "Filtered",
		reason:      "No alerts scheduled",
	}
}

func countStatus(statuses []models.AlertStatus, target models.AlertStatus) int {
	count := 0
	for _, status := range statuses {
		if status == target {
			count++
		}
	}
	return count
}
