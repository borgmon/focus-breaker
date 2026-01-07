package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// ICalSource represents a named iCal calendar source
type ICalSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FetchEvents fetches and parses events from this iCal source
func (s *ICalSource) FetchEvents() ([]Event, error) {
	events, err := fetchAndParseICal(s.URL)
	if err != nil {
		return nil, err
	}

	// Set the source ID for all events
	eventsWithoutUID := 0
	for i := range events {
		events[i].SourceID = s.ID
		// Fallback: if no iCal UID, use deterministic ID based on start time and title
		if events[i].ID == "" {
			events[i].ID = s.ID + "-" + events[i].StartTime.Format(time.RFC3339) + "-" + events[i].Title
			eventsWithoutUID++
		}
	}

	return events, nil
}

// Validate checks if the iCal source has required fields
func (s *ICalSource) Validate() bool {
	return s.Name != "" && s.URL != ""
}

// processEvent applies filters to an event and returns true if it should be included
func processEvent(event Event, now, tomorrow time.Time, filteredMissingTime, filteredCancelled, filteredAllDay, filteredOutsideWindow *int) bool {
	if event.StartTime.IsZero() || event.EndTime.IsZero() {
		*filteredMissingTime++
		log.Printf("  [FILTERED] Missing time - Event: \"%s\" (Start: %v, End: %v)",
			event.Title, event.StartTime, event.EndTime)
		return false
	}

	// Filter out cancelled events
	if event.Status == "CANCELLED" {
		*filteredCancelled++
		log.Printf("  [FILTERED] [Cancelled] - Event: \"%s\" (Start: %s, Status: %s)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"), event.Status)
		return false
	}

	startDate := event.StartTime.Format("2006-01-02")
	endDate := event.EndTime.Format("2006-01-02")
	duration := event.EndTime.Sub(event.StartTime)

	if startDate != endDate && duration >= 24*time.Hour {
		*filteredAllDay++
		log.Printf("  [FILTERED] [All-day] - Event: \"%s\" (Start: %s, End: %s, Duration: %v)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"),
			event.EndTime.Format("2006-01-02 15:04"), duration)
		return false
	}

	if event.StartTime.Before(tomorrow) && event.EndTime.After(now) {
		log.Printf("  [INCLUDED] Event: \"%s\" (Start: %s, End: %s)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"),
			event.EndTime.Format("2006-01-02 15:04"))
		return true
	}

	*filteredOutsideWindow++
	log.Printf("  [FILTERED] [Outside window] - Event: \"%s\" (Start: %s, End: %s, Now: %s, Tomorrow: %s)",
		event.Title, event.StartTime.Format("2006-01-02 15:04"),
		event.EndTime.Format("2006-01-02 15:04"), now.Format("2006-01-02 15:04"), tomorrow.Format("2006-01-02 15:04"))
	return false
}

// expandRecurringEvent expands a recurring event into instances within the time window
func expandRecurringEvent(baseEvent Event, rrule string, startTime, endTime time.Time) []Event {
	// Parse RRULE - this is a simplified implementation
	// For full RRULE support, consider using github.com/teambition/rrule-go
	events := []Event{}

	log.Printf("  [RECURRING] Expanding RRULE: %s for event \"%s\"", rrule, baseEvent.Title)

	// Simple daily recurrence check
	if strings.Contains(rrule, "FREQ=DAILY") {
		duration := baseEvent.EndTime.Sub(baseEvent.StartTime)

		// Start from the base event time and generate instances
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
	} else if strings.Contains(rrule, "FREQ=WEEKLY") {
		duration := baseEvent.EndTime.Sub(baseEvent.StartTime)

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
	} else {
		log.Printf("  [RECURRING] Unsupported RRULE pattern: %s", rrule)
	}

	return events
}

func fetchAndParseICal(icalURL string) ([]Event, error) {
	resp, err := http.Get(icalURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)

	// Check if response is HTML instead of iCalendar
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(bodyStr)), "<!DOCTYPE") ||
		strings.HasPrefix(strings.TrimSpace(strings.ToUpper(bodyStr)), "<HTML") {
		return nil, fmt.Errorf("received HTML instead of iCalendar data - check if URL requires authentication")
	}

	// Check if it starts with BEGIN:VCALENDAR
	if !strings.HasPrefix(strings.TrimSpace(bodyStr), "BEGIN:VCALENDAR") {
		previewLen := 100
		if len(bodyStr) < previewLen {
			previewLen = len(bodyStr)
		}
		return nil, fmt.Errorf("invalid iCalendar format - expected BEGIN:VCALENDAR, got: %s",
			strings.TrimSpace(bodyStr[:previewLen]))
	}

	decoder := ical.NewDecoder(strings.NewReader(bodyStr))
	events := []Event{}
	seenEventIDs := make(map[string]bool)
	seenEventKeys := make(map[string]bool) // key: title + start time

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	// Tracking filtered events
	var filteredMissingTime, filteredCancelled, filteredAllDay, filteredOutsideWindow, filteredDuplicates int
	var totalComponents, totalEvents int

	for {
		cal, err := decoder.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode calendar: %w", err)
		}

		log.Printf("  [DEBUG] Calendar decoded with %d children", len(cal.Children))

		for _, comp := range cal.Children {
			totalComponents++
			if comp.Name != ical.CompEvent {
				log.Printf("  [DEBUG] Skipping non-event component: %s", comp.Name)
				continue
			}
			totalEvents++

			event := parseEvent(comp)

			// Check if this is a recurring event
			rruleProp := comp.Props.Get(ical.PropRecurrenceRule)
			if rruleProp != nil {
				log.Printf("  [RECURRING] Event: \"%s\" has RRULE: %s", event.Title, rruleProp.Value)
				// Expand recurring events within our time window
				recurringEvents := expandRecurringEvent(event, rruleProp.Value, now, tomorrow)
				for _, recEvent := range recurringEvents {
					if processEvent(recEvent, now, tomorrow, &filteredMissingTime, &filteredCancelled, &filteredAllDay, &filteredOutsideWindow) {
						// Check for duplicates by ID
						if seenEventIDs[recEvent.ID] {
							filteredDuplicates++
							log.Printf("  [FILTERED] Duplicate (ID) - Event: \"%s\" (ID: %s)",
								recEvent.Title, recEvent.ID)
							continue
						}

						// Check for duplicates by title + start time
						eventKey := recEvent.Title + "|" + recEvent.StartTime.Format(time.RFC3339)
						if seenEventKeys[eventKey] {
							filteredDuplicates++
							log.Printf("  [FILTERED] Duplicate (Title+Time) - Event: \"%s\" (Start: %s)",
								recEvent.Title, recEvent.StartTime.Format("2006-01-02 15:04"))
							continue
						}

						seenEventIDs[recEvent.ID] = true
						seenEventKeys[eventKey] = true
						events = append(events, recEvent)
					}
				}
				continue
			}

			// Process single event
			if processEvent(event, now, tomorrow, &filteredMissingTime, &filteredCancelled, &filteredAllDay, &filteredOutsideWindow) {
				// Check for duplicates by ID
				if seenEventIDs[event.ID] {
					filteredDuplicates++
					log.Printf("  [FILTERED] Duplicate (ID) - Event: \"%s\" (ID: %s)",
						event.Title, event.ID)
					continue
				}

				// Check for duplicates by title + start time
				eventKey := event.Title + "|" + event.StartTime.Format(time.RFC3339)
				if seenEventKeys[eventKey] {
					filteredDuplicates++
					log.Printf("  [FILTERED] Duplicate (Title+Time) - Event: \"%s\" (Start: %s)",
						event.Title, event.StartTime.Format("2006-01-02 15:04"))
					continue
				}

				seenEventIDs[event.ID] = true
				seenEventKeys[eventKey] = true
				events = append(events, event)
			}
		}
	}

	// Log filtering summary
	totalFiltered := filteredMissingTime + filteredCancelled + filteredAllDay + filteredOutsideWindow + filteredDuplicates
	log.Printf("  [SUMMARY] Total components: %d, Events: %d, Included: %d, Filtered: %d",
		totalComponents, totalEvents, len(events), totalFiltered)
	if totalFiltered > 0 {
		log.Printf("  Filtered breakdown: %d cancelled, %d all-day, %d outside window, %d missing time, %d duplicates",
			filteredCancelled, filteredAllDay, filteredOutsideWindow, filteredMissingTime, filteredDuplicates)
	}

	return events, nil
}

// parseDateTimeProperty attempts to parse a datetime property with multiple strategies
func parseDateTimeProperty(prop *ical.Prop) (time.Time, error) {
	// First try the standard DateTime method with local timezone
	if t, err := prop.DateTime(time.Local); err == nil {
		return t.In(time.Local), nil
	}

	// If that fails, try parsing the raw value directly
	// Format: 20260129T120500 (basic iCalendar datetime format)
	value := prop.Value

	// Try parsing as local time (without timezone)
	formats := []string{
		"20060102T150405",     // Basic format: YYYYMMDDTHHMMSS
		"20060102T150405Z",    // UTC format
		time.RFC3339,          // Standard RFC3339
		"2006-01-02T15:04:05", // ISO 8601 without timezone
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, value, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime value: %s", value)
}

func parseEvent(comp *ical.Component) Event {
	event := Event{}

	// Extract iCal UID for stable event identification
	if uidProp := comp.Props.Get(ical.PropUID); uidProp != nil {
		event.ID = uidProp.Value
	}

	if summaryProp := comp.Props.Get(ical.PropSummary); summaryProp != nil {
		event.Title = summaryProp.Value
	}

	if descProp := comp.Props.Get(ical.PropDescription); descProp != nil {
		event.Description = descProp.Value
		event.MeetingLink = extractMeetingLink(descProp.Value)
	}

	if startProp := comp.Props.Get(ical.PropDateTimeStart); startProp != nil {
		t, err := parseDateTimeProperty(startProp)
		if err == nil {
			event.StartTime = t
		}
	}

	if endProp := comp.Props.Get(ical.PropDateTimeEnd); endProp != nil {
		t, err := parseDateTimeProperty(endProp)
		if err == nil {
			event.EndTime = t
		}
	}

	if statusProp := comp.Props.Get(ical.PropStatus); statusProp != nil {
		event.Status = statusProp.Value
	}

	// Polyfill: If status is not CANCELLED but title indicates cancellation, set status to CANCELLED
	if event.Status != "CANCELLED" {
		cleanTitle := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(strings.ToLower(event.Title), "")
		if strings.HasPrefix(cleanTitle, "canceled") || strings.HasPrefix(cleanTitle, "cancelled") {
			event.Status = "CANCELLED"
		}
	}

	if locProp := comp.Props.Get(ical.PropLocation); locProp != nil && event.MeetingLink == "" {
		event.MeetingLink = extractMeetingLink(locProp.Value)
	}

	return event
}

func extractMeetingLink(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^[\]` + "`" + `]+`)
	matches := urlRegex.FindAllString(text, -1)

	for _, match := range matches {
		lower := strings.ToLower(match)
		if strings.Contains(lower, "zoom") ||
			strings.Contains(lower, "meet.google") ||
			strings.Contains(lower, "teams.microsoft") ||
			strings.Contains(lower, "webex") ||
			strings.Contains(lower, "gotomeeting") {
			return match
		}
	}

	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}
