package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/borgmon/focus-breaker/pkg/models"
)

func (fb *FocusBreaker) setupSystemTray() {
	fb.updateSystemTrayMenu()
}

func (fb *FocusBreaker) updateSystemTrayMenu() {
	if desk, ok := fb.app.(desktop.App); ok {
		menuItems := []*fyne.MenuItem{}

		// Add upcoming alerts section at the top
		upcomingAlerts := fb.getUpcomingTodayAlerts(5)
		if len(upcomingAlerts) > 0 {
			// Add header
			headerItem := fyne.NewMenuItem("Upcoming Today:", nil)
			headerItem.Disabled = true
			menuItems = append(menuItems, headerItem)

			// Add each alert
			for _, alert := range upcomingAlerts {
				event := fb.alertStore.GetEvent(alert.EventID)
				if event != nil {
					alertText := fmt.Sprintf("  %s - %s",
						alert.AlertTime.Format("3:04 PM"),
						truncateString(event.Title, 35))

					alertItem := fyne.NewMenuItem(alertText, nil)
					alertItem.Disabled = true
					menuItems = append(menuItems, alertItem)
				}
			}

			menuItems = append(menuItems, fyne.NewMenuItemSeparator())
		}

		// Add settings and sync below
		menuItems = append(menuItems,
			fyne.NewMenuItem("Settings", func() {
				fb.showConfigWindow()
			}),
			fyne.NewMenuItem("Sync Now", func() {
				go fb.syncEvents()
			}),
		)

		menuItems = append(menuItems, fyne.NewMenuItemSeparator())
		menuItems = append(menuItems, fyne.NewMenuItem("Quit", func() {
			fb.quit()
		}))

		menu := fyne.NewMenu("Focus Breaker", menuItems...)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconTransparentPng)
	}
}

// getUpcomingTodayAlerts returns the next N alerts scheduled for today
func (fb *FocusBreaker) getUpcomingTodayAlerts(limit int) []*models.ScheduledAlert {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	allAlerts := fb.alertStore.GetAllScheduledAlerts()
	upcomingToday := []*models.ScheduledAlert{}

	for _, alert := range allAlerts {
		// Only include pending or snoozed alerts
		if alert.Status != models.AlertStatusPending && alert.Status != models.AlertStatusSnoozed {
			continue
		}

		// Only include alerts from now until end of today
		if alert.AlertTime.After(now) && alert.AlertTime.Before(todayEnd) {
			upcomingToday = append(upcomingToday, alert)
			if len(upcomingToday) >= limit {
				break
			}
		}
	}

	return upcomingToday
}

// truncateString truncates a string to maxLen characters, adding "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
