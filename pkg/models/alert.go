package models

import "time"

// AlertStatus tracks the status of an individual alert
type AlertStatus string

const (
	AlertStatusPending AlertStatus = "Pending" // Alert is scheduled
	AlertStatusAlerted AlertStatus = "Alerted" // Alert was shown
	AlertStatusSnoozed AlertStatus = "Snoozed" // Alert was snoozed
	AlertStatusMuted   AlertStatus = "Muted"   // Alert is muted (quiet time)
)

// ScheduledAlert represents a pre-computed alert for a specific event
type ScheduledAlert struct {
	ID          string      // Unique identifier for the alert (UUID)
	EventID     string      // Event ID (stable ID, not original iCal UID)
	Status      AlertStatus // Alert status
	AlertTime   time.Time   // When this alert should fire
	AlertOffset int         // negative = minutes before event start, positive = minutes from snooze time
}

// RoundToMinute rounds a time down to the nearest minute
func RoundToMinute(t time.Time) time.Time {
	return t.Truncate(time.Minute)
}
