package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type FocusBreaker struct {
	app          fyne.App
	config       *Config
	alertStore   *AlertStore
	syncTicker   *time.Ticker
	alertTicker  *time.Ticker
	configWindow *ConfigWindow
}

func main() {
	fb := &FocusBreaker{
		app:        app.New(),
		alertStore: NewAlertStore(),
	}

	if err := fb.initialize(); err != nil {
		log.Fatal(err)
	}

	fb.run()
}

func (fb *FocusBreaker) initialize() error {
	fb.config = loadConfig(fb.app)

	// Sync autostart state with config on startup
	if err := setupAutostart(fb.config.AutoStart); err != nil {
		log.Printf("Warning: failed to setup autostart: %v", err)
	}

	saveConfig(fb.app, fb.config)

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
		setActivationPolicy()
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
	fb.configWindow = NewConfigWindow(fb.app, fb.config, fb.alertStore, func(newConfig *Config) {
		fb.config = newConfig
		saveConfig(fb.app, fb.config)

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
	if len(fb.config.ICalSources) == 0 {
		log.Println("No iCal sources configured")
		return
	}

	// Collect events from all iCal sources
	allEvents := []Event{}
	for _, source := range fb.config.ICalSources {
		if !source.Validate() {
			continue
		}

		events, err := source.FetchEvents()
		if err != nil {
			log.Printf("Error fetching iCal source '%s' (%s): %v", source.Name, source.URL, err)
			continue
		}

		allEvents = append(allEvents, events...)
		log.Printf("Synced %d events from '%s'", len(events), source.Name)
	}

	alertMinutes := fb.config.GetAlertMinutes()
	fb.alertStore.UpdateEvents(allEvents, alertMinutes)
	log.Printf("Total synced %d events from %d iCal sources", len(allEvents), len(fb.config.ICalSources))

	// Update system tray menu with new events
	fb.updateSystemTrayMenu()
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
		fb.showAlert(alert)
	}
}

func (fb *FocusBreaker) showAlert(alert *ScheduledAlert) {
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
		func() {
			// Mark alert as alerted (closed/dismissed)
			fb.alertStore.MarkAlertStatus(alert.EventID, alert.AlertOffset, AlertStatusAlerted, nil)
			log.Printf("Alert closed for event: %s", event.Title)
		},
		func() {
			// Mark alert as snoozed and schedule new alert
			snoozeUntil := time.Now().Add(time.Duration(fb.config.SnoozeTime) * time.Minute)
			fb.alertStore.MarkAlertStatus(alert.EventID, alert.AlertOffset, AlertStatusSnoozed, &snoozeUntil)
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
