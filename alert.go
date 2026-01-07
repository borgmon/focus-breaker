package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertStatus tracks the status of an individual alert
type AlertStatus string

const (
	AlertStatusPending AlertStatus = "Pending"
	AlertStatusAlerted AlertStatus = "Alerted"
	AlertStatusSnoozed AlertStatus = "Snoozed"
	AlertStatusMuted   AlertStatus = "Muted"
)

// ScheduledAlert represents a pre-computed alert for a specific event
type ScheduledAlert struct {
	ID          string      // Unique identifier for the alert (UUID)
	EventID     string      // Event ID (stable ID, not original iCal UID)
	Status      AlertStatus // Alert status (Pending, Alerted, Snoozed)
	AlertTime   time.Time   // When this alert should fire
	AlertOffset int         // negative = minutes before event start, positive = minutes from snooze time
}

// AlertStore manages events and their scheduled alerts
type AlertStore struct {
	mu sync.RWMutex

	// Map of event ID to Event
	events map[string]*Event

	// Map of timestamp (minute precision) to list of scheduled alerts
	// Key format: Unix timestamp rounded to minute
	alertsByTime map[int64][]*ScheduledAlert

	// Map of alert ID to scheduled alert for quick lookup
	// Alert ID format: "eventID-minutesBefore"
	alertsById map[string]*ScheduledAlert
}

func NewAlertStore() *AlertStore {
	return &AlertStore{
		events:       make(map[string]*Event),
		alertsByTime: make(map[int64][]*ScheduledAlert),
		alertsById:   make(map[string]*ScheduledAlert),
	}
}

// generateAlertID creates a unique ID for an alert
func generateAlertID(eventID string, alertOffset int) string {
	return fmt.Sprintf("%s-%d", eventID, alertOffset)
}

// roundToMinute rounds a time down to the nearest minute
func roundToMinute(t time.Time) time.Time {
	return t.Truncate(time.Minute)
}

// UpdateEvents updates the alert store with new events from calendar sync
// It handles additions, updates, and keeps existing alert statuses
func (as *AlertStore) UpdateEvents(newEvents []Event, alertMinutes []int) {
	as.UpdateEventsWithConfig(newEvents, alertMinutes, nil)
}

// UpdateEventsWithConfig updates the alert store with new events and applies quiet time config
func (as *AlertStore) UpdateEventsWithConfig(newEvents []Event, alertMinutes []int, config *Config) {
	as.mu.Lock()
	defer as.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-12 * time.Hour)

	// Track which event IDs we've seen in this sync
	seenEventIDs := make(map[string]bool)

	for _, event := range newEvents {
		// Skip past events (before cutoff)
		if event.StartTime.Before(cutoffTime) {
			continue
		}

		eventID := event.ID
		seenEventIDs[eventID] = true

		// Check if event exists
		existingEvent, exists := as.events[eventID]

		if exists {
			// Update event details (but preserve alert statuses)
			existingEvent.Title = event.Title
			existingEvent.Description = event.Description
			existingEvent.StartTime = event.StartTime
			existingEvent.EndTime = event.EndTime
			existingEvent.MeetingLink = event.MeetingLink
			existingEvent.Status = event.Status

			// Update alert times if event time changed
			as.updateAlertsForEvent(eventID, &event, alertMinutes)
		} else {
			// New event - add it and create alerts
			as.events[eventID] = &event
			as.createAlertsForEventWithConfig(eventID, &event, alertMinutes, config)
		}
	}

	// Remove events that are no longer in the calendar (but keep recent ones within 12 hours)
	for eventID, event := range as.events {
		if !seenEventIDs[eventID] && event.StartTime.Before(now) && event.StartTime.Before(cutoffTime) {
			as.removeEvent(eventID)
		}
	}

	// Clean up old events and alerts older than 12 hours
	as.cleanupOldAlerts(cutoffTime)
}

// createAlertsForEvent creates all scheduled alerts for a new event
func (as *AlertStore) createAlertsForEvent(eventID string, event *Event, alertMinutes []int) {
	now := time.Now()

	for _, minutes := range alertMinutes {
		alertTime := event.StartTime.Add(-time.Duration(minutes) * time.Minute)

		// Skip creating alerts in the past
		if alertTime.Before(now) {
			continue
		}

		alert := &ScheduledAlert{
			ID:          uuid.New().String(),
			EventID:     eventID,
			Status:      AlertStatusPending,
			AlertTime:   alertTime,
			AlertOffset: -minutes, // Negative for pre-event alerts
		}

		alertID := generateAlertID(eventID, -minutes)
		as.alertsById[alertID] = alert

		// Add to time-based index
		timeKey := roundToMinute(alertTime).Unix()
		as.alertsByTime[timeKey] = append(as.alertsByTime[timeKey], alert)
	}
}

// createAlertsForEventWithConfig creates all scheduled alerts for a new event, checking quiet time
func (as *AlertStore) createAlertsForEventWithConfig(eventID string, event *Event, alertMinutes []int, config *Config) {
	now := time.Now()

	for _, minutes := range alertMinutes {
		alertTime := event.StartTime.Add(-time.Duration(minutes) * time.Minute)

		// Skip creating alerts in the past
		if alertTime.Before(now) {
			continue
		}

		// Determine initial status based on quiet time
		status := AlertStatusPending
		if config != nil && config.IsTimeInQuietTime(alertTime) {
			status = AlertStatusMuted
		}

		alert := &ScheduledAlert{
			ID:          uuid.New().String(),
			EventID:     eventID,
			Status:      status,
			AlertTime:   alertTime,
			AlertOffset: -minutes, // Negative for pre-event alerts
		}

		alertID := generateAlertID(eventID, -minutes)
		as.alertsById[alertID] = alert

		// Add to time-based index
		timeKey := roundToMinute(alertTime).Unix()
		as.alertsByTime[timeKey] = append(as.alertsByTime[timeKey], alert)
	}
}

// updateAlertsForEvent updates alerts when event time changes
func (as *AlertStore) updateAlertsForEvent(eventID string, event *Event, alertMinutes []int) {
	for _, minutes := range alertMinutes {
		alertID := generateAlertID(eventID, -minutes)
		if alert, exists := as.alertsById[alertID]; exists {
			// Update existing alert
			// Remove from old time slot
			oldTimeKey := roundToMinute(alert.AlertTime).Unix()
			as.removeAlertFromTimeIndex(oldTimeKey, alertID)

			// Update alert time
			alert.AlertTime = event.StartTime.Add(-time.Duration(minutes) * time.Minute)
			// Keep existing Status (don't reset to Pending)

			// Add to new time slot
			newTimeKey := roundToMinute(alert.AlertTime).Unix()
			as.alertsByTime[newTimeKey] = append(as.alertsByTime[newTimeKey], alert)
		} else {
			// Alert doesn't exist, create new one (happens when alertMinutes config changes)
			alertTime := event.StartTime.Add(-time.Duration(minutes) * time.Minute)

			// Skip creating alerts in the past
			now := time.Now()
			if alertTime.Before(now) {
				continue
			}

			newAlert := &ScheduledAlert{
				ID:          uuid.New().String(),
				EventID:     eventID,
				Status:      AlertStatusPending,
				AlertTime:   alertTime,
				AlertOffset: -minutes, // Negative for pre-event alerts
			}

			as.alertsById[alertID] = newAlert
			timeKey := roundToMinute(alertTime).Unix()
			as.alertsByTime[timeKey] = append(as.alertsByTime[timeKey], newAlert)
		}
	}

	// Remove alerts that are no longer in alertMinutes config
	// Only remove negative offset alerts (pre-event), keep positive offset alerts (snoozed)
	alertMinutesMap := make(map[int]bool)
	for _, m := range alertMinutes {
		alertMinutesMap[m] = true
	}

	for alertID, alert := range as.alertsById {
		if alert.EventID == eventID && alert.AlertOffset < 0 && !alertMinutesMap[-alert.AlertOffset] {
			// This alert's offset is no longer in config, remove it
			timeKey := roundToMinute(alert.AlertTime).Unix()
			as.removeAlertFromTimeIndex(timeKey, alertID)
			delete(as.alertsById, alertID)
		}
	}
}

// removeAlertFromTimeIndex removes an alert from the time-based index
func (as *AlertStore) removeAlertFromTimeIndex(timeKey int64, alertID string) {
	alerts := as.alertsByTime[timeKey]
	for i, alert := range alerts {
		if generateAlertID(alert.EventID, alert.AlertOffset) == alertID {
			as.alertsByTime[timeKey] = append(alerts[:i], alerts[i+1:]...)
			break
		}
	}
	if len(as.alertsByTime[timeKey]) == 0 {
		delete(as.alertsByTime, timeKey)
	}
}

// removeEvent removes an event and all its alerts
func (as *AlertStore) removeEvent(eventID string) {
	delete(as.events, eventID)

	// Remove all alerts for this event (both pre-event and snoozed)
	for alertID, alert := range as.alertsById {
		if alert.EventID == eventID {
			timeKey := roundToMinute(alert.AlertTime).Unix()
			as.removeAlertFromTimeIndex(timeKey, alertID)
			delete(as.alertsById, alertID)
		}
	}
}

// cleanupOldAlerts removes alerts older than cutoff time
func (as *AlertStore) cleanupOldAlerts(cutoffTime time.Time) {
	cutoffKey := roundToMinute(cutoffTime).Unix()

	// Remove old time slots
	for timeKey := range as.alertsByTime {
		if timeKey < cutoffKey {
			// Remove all alerts in this time slot from alertsById
			for _, alert := range as.alertsByTime[timeKey] {
				alertID := generateAlertID(alert.EventID, alert.AlertOffset)
				delete(as.alertsById, alertID)
			}
			delete(as.alertsByTime, timeKey)
		}
	}

	// Remove old events
	for eventID, event := range as.events {
		if event.StartTime.Before(cutoffTime) {
			delete(as.events, eventID)
		}
	}
}

// GetAlertsForCurrentMinute returns all alerts scheduled for the current minute
func (as *AlertStore) GetAlertsForCurrentMinute(notifyUnaccepted bool) []*ScheduledAlert {
	as.mu.RLock()
	defer as.mu.RUnlock()

	now := time.Now()
	timeKey := roundToMinute(now).Unix()

	alerts := as.alertsByTime[timeKey]
	result := make([]*ScheduledAlert, 0)

	for _, alert := range alerts {
		// Skip if pending and event is unaccepted
		if !notifyUnaccepted {
			event := as.events[alert.EventID]
			if event != nil && event.Status == "NEEDS-ACTION" {
				continue
			}
		}

		// Only return pending or snoozed alerts (snoozed alerts are already moved to correct time slot)
		if alert.Status == AlertStatusPending || alert.Status == AlertStatusSnoozed {
			result = append(result, alert)
		}
	}

	return result
}

// MarkAlertStatus updates the status of an alert
func (as *AlertStore) MarkAlertStatus(eventID string, alertOffset int, status AlertStatus, snoozedUntil *time.Time) {
	as.mu.Lock()
	defer as.mu.Unlock()

	alertID := generateAlertID(eventID, alertOffset)
	if alert, exists := as.alertsById[alertID]; exists {
		// If snoozed, mark original as snoozed and create a new alert with positive offset
		if status == AlertStatusSnoozed && snoozedUntil != nil {
			alert.Status = AlertStatusSnoozed

			// Calculate snooze minutes from now
			snoozeMinutes := int(snoozedUntil.Sub(time.Now()).Minutes())
			if snoozeMinutes < 0 {
				snoozeMinutes = 0
			}

			// Create new snoozed alert with positive offset
			newAlert := &ScheduledAlert{
				ID:          uuid.New().String(),
				EventID:     alert.EventID,
				Status:      AlertStatusPending,
				AlertTime:   *snoozedUntil,
				AlertOffset: snoozeMinutes, // Positive for snoozed alerts
			}

			newAlertID := generateAlertID(eventID, snoozeMinutes)
			as.alertsById[newAlertID] = newAlert

			// Add to time-based index
			timeKey := roundToMinute(*snoozedUntil).Unix()
			as.alertsByTime[timeKey] = append(as.alertsByTime[timeKey], newAlert)
		} else {
			// Just update status for non-snooze cases
			alert.Status = status
		}
	}
}

func (as *AlertStore) GetAllScheduledAlerts() []*ScheduledAlert {
	as.mu.RLock()
	defer as.mu.RUnlock()

	now := time.Now()
	result := make([]*ScheduledAlert, 0, len(as.alertsById))

	for _, alert := range as.alertsById {
		// Include future alerts and recent past alerts (within 12 hours)
		if alert.AlertTime.After(now.Add(-12 * time.Hour)) {
			result = append(result, alert)
		}
	}

	// Sort by alert time
	sort.Slice(result, func(i, j int) bool {
		return result[i].AlertTime.Before(result[j].AlertTime)
	})
	return result
}

// GetEvent returns an event by ID
func (as *AlertStore) GetEvent(eventID string) *Event {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.events[eventID]
}

// AddManualAlert adds a manual event and its alert to the store
func (as *AlertStore) AddManualAlert(event *Event, alertTime time.Time) {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Add the event
	as.events[event.ID] = event

	// Create the alert
	alert := &ScheduledAlert{
		ID:          uuid.New().String(),
		EventID:     event.ID,
		Status:      AlertStatusPending,
		AlertTime:   alertTime,
		AlertOffset: 0, // Manual alarms have 0 offset
	}

	alertID := generateAlertID(event.ID, 0)
	as.alertsById[alertID] = alert

	// Add to time-based index
	timeKey := roundToMinute(alertTime).Unix()
	as.alertsByTime[timeKey] = append(as.alertsByTime[timeKey], alert)
}

// UpdateMutedStatusForQuietTime checks all alerts and updates their muted status based on quiet time ranges
func (as *AlertStore) UpdateMutedStatusForQuietTime(config *Config) {
	as.mu.Lock()
	defer as.mu.Unlock()

	for _, alert := range as.alertsById {
		// Only update alerts that are currently Pending or Muted
		if alert.Status != AlertStatusPending && alert.Status != AlertStatusMuted {
			continue
		}

		isInQuietTime := config.IsTimeInQuietTime(alert.AlertTime)

		if isInQuietTime && alert.Status == AlertStatusPending {
			// Mark as muted
			alert.Status = AlertStatusMuted
		} else if !isInQuietTime && alert.Status == AlertStatusMuted {
			// Unmute - return to pending
			alert.Status = AlertStatusPending
		}
	}
}
