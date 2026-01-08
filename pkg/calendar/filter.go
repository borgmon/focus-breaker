package calendar

import (
	"log"
	"time"

	"github.com/borgmon/focus-breaker/pkg/models"
)

func shouldIncludeEvent(event models.Event, now, tomorrow time.Time, stats *filterStats) bool {
	// Filter events with missing time information
	if event.StartTime.IsZero() || event.EndTime.IsZero() {
		stats.filteredMissingTime++
		log.Printf("  [FILTERED] Missing time - Event: \"%s\" (Start: %v, End: %v)",
			event.Title, event.StartTime, event.EndTime)
		return false
	}

	// Filter out cancelled events
	if event.Status == "CANCELLED" {
		stats.filteredCancelled++
		log.Printf("  [FILTERED] [Cancelled] - Event: \"%s\" (Start: %s, Status: %s)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"), event.Status)
		return false
	}

	// Filter out all-day events
	if isAllDayEvent(event) {
		stats.filteredAllDay++
		log.Printf("  [FILTERED] [All-day] - Event: \"%s\" (Start: %s, End: %s, Duration: %v)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"),
			event.EndTime.Format("2006-01-02 15:04"), event.EndTime.Sub(event.StartTime))
		return false
	}

	// Include events within the time window (now to tomorrow)
	if event.StartTime.Before(tomorrow) && event.EndTime.After(now) {
		log.Printf("  [INCLUDED] Event: \"%s\" (Start: %s, End: %s)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"),
			event.EndTime.Format("2006-01-02 15:04"))
		return true
	}

	// Filter events outside the time window
	stats.filteredOutsideWindow++
	log.Printf("  [FILTERED] [Outside window] - Event: \"%s\" (Start: %s, End: %s, Now: %s, Tomorrow: %s)",
		event.Title, event.StartTime.Format("2006-01-02 15:04"),
		event.EndTime.Format("2006-01-02 15:04"), now.Format("2006-01-02 15:04"), tomorrow.Format("2006-01-02 15:04"))
	return false
}

func isAllDayEvent(event models.Event) bool {
	startDate := event.StartTime.Format("2006-01-02")
	endDate := event.EndTime.Format("2006-01-02")
	duration := event.EndTime.Sub(event.StartTime)

	// An event is considered all-day if it spans multiple days and is >= 24 hours
	return startDate != endDate && duration >= 24*time.Hour
}
