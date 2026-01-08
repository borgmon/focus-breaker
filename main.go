package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/borgmon/focus-breaker/pkg/calendar"
	"github.com/borgmon/focus-breaker/pkg/models"
	"github.com/borgmon/focus-breaker/pkg/platform"
	"github.com/borgmon/focus-breaker/pkg/store"
)

type FocusBreaker struct {
	app          fyne.App
	config       *models.Config
	alertStore   *store.AlertStore
	syncTicker   *time.Ticker
	alertTicker  *time.Ticker
	configWindow *ConfigWindow
}

func main() {
	fb := &FocusBreaker{
		app:        app.New(),
		alertStore: store.NewAlertStore(),
	}

	if err := fb.initialize(); err != nil {
		log.Fatal(err)
	}

	fb.run()
}

func (fb *FocusBreaker) initialize() error {
	configStore := store.NewConfigStore(fb.app)
	fb.config = configStore.Load()

	// Sync autostart state with config on startup
	if err := platform.SetupAutostart(fb.config.AutoStart); err != nil {
		log.Printf("Warning: failed to setup autostart: %v", err)
	}

	configStore.Save(fb.config)

	fb.setupSystemTray()
	fb.startBackgroundSync() // This will sync and update the tray menu
	fb.startAlertChecker()

	if fb.config.NeedsConfiguration() {
		fb.showConfigWindow()
	}

	return nil
}

func (fb *FocusBreaker) run() {
	fb.app.Lifecycle().SetOnStarted(func() {
		platform.SetActivationPolicy()
	})
	fb.app.Run()
}

func (fb *FocusBreaker) showConfigWindow() {
	// If config window already exists and is showing, just bring it to front
	if fb.configWindow != nil && fb.configWindow.window != nil {
		fb.configWindow.window.RequestFocus()
		fb.configWindow.window.Show()
		return
	}

	// Create new config window
	fb.configWindow = NewConfigWindow(fb.app, fb.config, fb.alertStore, func(newConfig *models.Config) {
		fb.config = newConfig
		configStore := store.NewConfigStore(fb.app)
		configStore.Save(fb.config)

		// Update muted status for all alerts based on new quiet time settings
		fb.alertStore.UpdateMutedStatusForQuietTime(fb.config)

		fb.restartBackgroundSync()

		if !fb.config.NeedsConfiguration() {
			fb.syncEvents()
		}
	})

	// Set close handler to clear reference
	originalOnClosed := fb.configWindow.window.SetOnClosed
	fb.configWindow.window.SetOnClosed(func() {
		fb.configWindow = nil
		if originalOnClosed != nil {
			// Call original if it exists
		}
	})

	fb.configWindow.Show()
}

func (fb *FocusBreaker) syncEvents() {
	log.Println("=== Starting sync process ===")

	if len(fb.config.ICalSources) == 0 {
		log.Println("No iCal sources configured")
		return
	}

	log.Printf("Found %d iCal source(s) to sync", len(fb.config.ICalSources))

	// Collect events from all iCal sources
	allEvents := []models.Event{}
	successfulSources := 0
	failedSources := 0

	for i, source := range fb.config.ICalSources {
		log.Printf("Processing source %d/%d: '%s'", i+1, len(fb.config.ICalSources), source.Name)

		if !source.Validate() {
			log.Printf("Skipping invalid source '%s' (missing name or URL)", source.Name)
			failedSources++
			continue
		}

		log.Printf("Fetching events from '%s' (%s)", source.Name, source.URL)
		events, err := calendar.FetchEvents(source)
		if err != nil {
			log.Printf("Error fetching iCal source '%s' (%s): %v", source.Name, source.URL, err)
			failedSources++
			continue
		}

		allEvents = append(allEvents, events...)
		successfulSources++
		log.Printf("Successfully synced %d events from '%s'", len(events), source.Name)
	}

	log.Printf("Sync completed: %d successful, %d failed out of %d total sources",
		successfulSources, failedSources, len(fb.config.ICalSources))

	alertMinutes := fb.config.GetAlertMinutes()
	log.Printf("Updating alert store with %d total events (alert offset: %d minutes)",
		len(allEvents), alertMinutes)

	fb.alertStore.UpdateEventsWithConfig(allEvents, alertMinutes, fb.config)
	log.Printf("Alert store updated successfully")

	// Update system tray menu with new events
	log.Println("Updating system tray menu")
	fb.updateSystemTrayMenu()
	log.Println("=== Sync process completed ===")
}

func (fb *FocusBreaker) startBackgroundSync() {
	// Do initial sync synchronously to populate data before UI setup
	if len(fb.config.ICalSources) > 0 {
		fb.syncEvents()
	}

	// Start periodic background sync
	fb.syncTicker = time.NewTicker(time.Duration(fb.config.UpdateInterval) * time.Minute)
	go func() {
		for range fb.syncTicker.C {
			if !fb.config.NeedsConfiguration() {
				fb.syncEvents()
			}
		}
	}()
}

func (fb *FocusBreaker) restartBackgroundSync() {
	if fb.syncTicker != nil {
		fb.syncTicker.Stop()
	}
	fb.startBackgroundSync()
}

func (fb *FocusBreaker) startAlertChecker() {
	fb.alertTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for range fb.alertTicker.C {
			fb.checkAlerts()
		}
	}()

	go func() {
		time.Sleep(5 * time.Second)
		fb.checkAlerts()
	}()
}

func (fb *FocusBreaker) checkAlerts() {
	if fb.config.NeedsConfiguration() {
		return
	}

	alerts := fb.alertStore.GetAlertsForCurrentMinute(fb.config.NotifyUnaccepted)

	for _, alert := range alerts {
		// Skip muted alerts - they should not be shown
		if alert.Status == models.AlertStatusMuted {
			log.Printf("Skipping muted alert for event: %s", alert.EventID)
			continue
		}

		fb.showAlert(alert)
	}
}

func (fb *FocusBreaker) showAlert(alert *models.ScheduledAlert) {
	// Get the actual event from the store
	event := fb.alertStore.GetEvent(alert.EventID)
	if event == nil {
		log.Printf("Event not found for alert: %s", alert.EventID)
		return
	}

	alertWindow := NewAlertWindow(
		fb.app,
		*event,
		fb.config.SnoozeTime,
		fb.config.HoldTimeSeconds,
		func() {
			// Mark alert as alerted (closed/dismissed)
			fb.alertStore.MarkAlertStatus(alert.EventID, alert.AlertOffset, models.AlertStatusAlerted, nil)
			log.Printf("Alert closed for event: %s", event.Title)
		},
		func() {
			// Mark alert as snoozed and schedule new alert
			snoozeUntil := time.Now().Add(time.Duration(fb.config.SnoozeTime) * time.Minute)
			fb.alertStore.MarkAlertStatus(alert.EventID, alert.AlertOffset, models.AlertStatusSnoozed, &snoozeUntil)
			log.Printf("Alert snoozed for event: %s until %s", event.Title, snoozeUntil.Format(time.RFC3339))
		},
	)
	alertWindow.Show()
}

func (fb *FocusBreaker) quit() {
	if fb.syncTicker != nil {
		fb.syncTicker.Stop()
	}
	if fb.alertTicker != nil {
		fb.alertTicker.Stop()
	}
	fb.app.Quit()
}
