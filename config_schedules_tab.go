package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (cw *ConfigWindow) buildSchedulesTab() fyne.CanvasObject {
	// Get initial schedules data
	cw.schedulesData = cw.getScheduledAlerts()

	// Create table widget
	table := widget.NewTable(
		func() (rows int, cols int) {
			return len(cw.schedulesData), 6
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)

			// Handle data rows
			if id.Row >= len(cw.schedulesData) {
				label.SetText("")
				return
			}

			schedule := cw.schedulesData[id.Row]

			// Look up the event for this alert
			event := cw.alertStore.GetEvent(schedule.EventID)
			eventTitle := "Unknown Event"
			eventStart := schedule.AlertTime
			sourceName := ""
			if event != nil {
				eventTitle = event.Title
				eventStart = event.StartTime
				for _, source := range cw.config.ICalSources {
					if source.ID == event.SourceID {
						sourceName = source.Name
						break
					}
				}
			}

			// Format alert offset
			offsetText := ""
			if schedule.AlertOffset < 0 {
				offsetText = fmt.Sprintf("%d min before", -schedule.AlertOffset)
			} else if schedule.AlertOffset == 0 {
				offsetText = "At event time"
			} else {
				offsetText = fmt.Sprintf("Snoozed +%d min", schedule.AlertOffset)
			}

			// Format status
			status := string(schedule.Status)
			if schedule.Status == AlertStatusSnoozed {
				status = fmt.Sprintf("Snoozed until %s", schedule.AlertTime.Format("3:04 PM"))
			}

			// Set cell content based on column
			switch id.Col {
			case 0:
				label.SetText(eventTitle)
			case 1:
				label.SetText(sourceName)
			case 2:
				label.SetText(schedule.AlertTime.Format("Mon Jan 2, 3:04 PM"))
			case 3:
				label.SetText(eventStart.Format("Mon Jan 2, 3:04 PM"))
			case 4:
				label.SetText(offsetText)
			case 5:
				label.SetText(status)
			}

			// Gray out past alerts
			if schedule.AlertTime.Before(time.Now()) {
				label.Importance = widget.LowImportance
			} else {
				label.Importance = widget.MediumImportance
			}
			label.TextStyle.Bold = false
		},
	)

	// Enable header row and configure it
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
			label.SetText("Alert Time")
		case 3:
			label.SetText("Event Start")
		case 4:
			label.SetText("Offset")
		case 5:
			label.SetText("Status")
		}
	}

	// Calculate column widths based on content
	cw.updateSchedulesColumnWidths(table)

	// Store reference for refresh
	cw.schedulesTable = table

	refreshButton := widget.NewButton("Refresh", func() {
		cw.refreshSchedulesData()
	})
	refreshButton.Icon = theme.ViewRefreshIcon()

	helpText := widget.NewLabel("Shows all alerts including past alerts from the last 12 hours. If you don't see any alerts, make sure you have added calendar sources in the Calendar tab and clicked 'Sync Now'.")
	helpText.Wrapping = fyne.TextWrapWord
	helpText.Importance = widget.MediumImportance

	headerContent := container.NewVBox(
		widget.NewLabel("Scheduled Alerts"),
		widget.NewSeparator(),
		helpText,
		refreshButton,
	)

	// Check if there are no scheduled alerts to show empty state
	var mainContent fyne.CanvasObject
	if len(cw.schedulesData) == 0 {
		emptyStateText := widget.NewLabel("No scheduled alerts yet.\n\nTo get started:\n1. Add calendar sources in the Calendar tab\n2. Click 'Sync Now' to fetch events\n3. Alerts will appear here as events approach")
		emptyStateText.Wrapping = fyne.TextWrapWord
		emptyStateText.Importance = widget.MediumImportance
		mainContent = container.NewPadded(emptyStateText)
	} else {
		mainContent = table
	}

	content := container.NewBorder(
		headerContent,
		nil,
		nil,
		nil,
		mainContent,
	)

	return container.NewPadded(content)
}

func (cw *ConfigWindow) refreshSchedulesData() {
	if cw.schedulesTable == nil {
		return
	}

	// Update the schedules data
	cw.schedulesData = cw.getScheduledAlerts()

	// Recalculate column widths based on new data
	cw.updateSchedulesColumnWidths(cw.schedulesTable)

	// Refresh the table
	cw.schedulesTable.Refresh()
}

func (cw *ConfigWindow) updateSchedulesColumnWidths(table *widget.Table) {
	// Calculate maximum width needed for each column
	// Start with header widths (approximate character width)
	headers := []string{"Event", "Calendar", "Alert Time", "Event Start", "Offset", "Status"}
	columnWidths := make([]float32, 6)

	// Estimate character width (about 7-8 pixels per character in typical font)
	charWidth := float32(8)
	padding := float32(20) // Extra padding for cell spacing

	// Initialize with header widths
	for i, header := range headers {
		columnWidths[i] = float32(len(header))*charWidth + padding
	}

	// Calculate widths based on actual data
	for _, schedule := range cw.schedulesData {
		// Get event info
		event := cw.alertStore.GetEvent(schedule.EventID)
		eventTitle := "Unknown Event"
		sourceName := ""
		if event != nil {
			eventTitle = event.Title
			for _, source := range cw.config.ICalSources {
				if source.ID == event.SourceID {
					sourceName = source.Name
					break
				}
			}
		}

		// Calculate width for each column
		widths := []int{
			len(eventTitle),
			len(sourceName),
			len(schedule.AlertTime.Format("Mon Jan 2, 3:04 PM")),
			len(schedule.AlertTime.Format("Mon Jan 2, 3:04 PM")), // Event start has same format
			30, // Offset text (estimate max)
			20, // Status text (estimate max)
		}

		for i, width := range widths {
			contentWidth := float32(width)*charWidth + padding
			if contentWidth > columnWidths[i] {
				columnWidths[i] = contentWidth
			}
		}
	}

	// Set minimum and maximum widths
	minWidths := []float32{150, 100, 180, 180, 120, 120}
	maxWidths := []float32{400, 200, 200, 200, 160, 150}

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

func (cw *ConfigWindow) getScheduledAlerts() []*ScheduledAlert {
	if cw.alertStore == nil {
		return []*ScheduledAlert{}
	}

	// Get all scheduled alerts directly from eventstore (already sorted by alert time)
	return cw.alertStore.GetAllScheduledAlerts()
}
