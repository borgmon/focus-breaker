package calendar

import (
	"log"
	"strings"
	"time"

	"github.com/borgmon/focus-breaker/pkg/models"
)

// expandRecurringEvent expands a recurring event into instances within the time window
// This is a simplified implementation that handles DAILY and WEEKLY frequencies
// For full RRULE support, consider using github.com/teambition/rrule-go
func expandRecurringEvent(baseEvent models.Event, rrule string, startTime, endTime time.Time) []models.Event {
	events := []models.Event{}
	duration := baseEvent.EndTime.Sub(baseEvent.StartTime)

	log.Printf("  [RECURRING] Expanding RRULE: %s for event \"%s\"", rrule, baseEvent.Title)

	// Parse frequency
	if strings.Contains(rrule, "FREQ=DAILY") {
		events = expandDaily(baseEvent, duration, startTime, endTime)
	} else if strings.Contains(rrule, "FREQ=WEEKLY") {
		events = expandWeekly(baseEvent, duration, startTime, endTime)
	} else {
		log.Printf("  [RECURRING] Unsupported RRULE pattern: %s", rrule)
	}

	return events
}

func expandDaily(baseEvent models.Event, duration time.Duration, startTime, endTime time.Time) []models.Event {
	events := []models.Event{}
	current := baseEvent.StartTime

	for current.Before(endTime) {
		if current.After(startTime.Add(-24 * time.Hour)) {
			instance := baseEvent
			instance.StartTime = current
			instance.EndTime = current.Add(duration)
			instance.ID = baseEvent.ID + "-" + current.Format(time.RFC3339)
			events = append(events, instance)
			log.Printf("  [RECURRING] Generated instance at %s", current.Format("2006-01-02 15:04"))
		}
		current = current.Add(24 * time.Hour)
	}

	return events
}

func expandWeekly(baseEvent models.Event, duration time.Duration, startTime, endTime time.Time) []models.Event {
	events := []models.Event{}
	current := baseEvent.StartTime

	for current.Before(endTime) {
		if current.After(startTime.Add(-24 * time.Hour)) {
			instance := baseEvent
			instance.StartTime = current
			instance.EndTime = current.Add(duration)
			instance.ID = baseEvent.ID + "-" + current.Format(time.RFC3339)
			events = append(events, instance)
			log.Printf("  [RECURRING] Generated instance at %s", current.Format("2006-01-02 15:04"))
		}
		current = current.Add(7 * 24 * time.Hour)
	}

	return events
}
