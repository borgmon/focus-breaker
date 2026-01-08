package models

import "time"

// Event represents a calendar event
type Event struct {
	ID          string    // iCal event UID
	Title       string    // Event title/summary
	Description string    // Event description
	StartTime   time.Time // Event start time
	EndTime     time.Time // Event end time
	MeetingLink string    // Meeting link (Zoom, Google Meet, etc.)
	Status      string    // Event status (CONFIRMED, CANCELLED, NEEDS-ACTION)
	SourceID    string    // ID of the iCal source this event came from
}
