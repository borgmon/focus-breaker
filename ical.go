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

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	// Tracking filtered events
	var filteredMissingTime, filteredCancelled, filteredAllDay, filteredOutsideWindow int

	for {
		cal, err := decoder.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode calendar: %w", err)
		}

		for _, comp := range cal.Children {
			if comp.Name != ical.CompEvent {
				continue
			}

			event := parseEvent(comp)

			if event.StartTime.IsZero() || event.EndTime.IsZero() {
				filteredMissingTime++
				continue
			}

			// Filter out cancelled events
			if event.Status == "CANCELLED" {
				filteredCancelled++
				continue
			}

			startDate := event.StartTime.Format("2006-01-02")
			endDate := event.EndTime.Format("2006-01-02")
			duration := event.EndTime.Sub(event.StartTime)

			if startDate != endDate && duration >= 24*time.Hour {
				filteredAllDay++
				continue
			}

			if event.StartTime.Before(tomorrow) && event.EndTime.After(now) {
				events = append(events, event)
			} else {
				filteredOutsideWindow++
			}
		}
	}

	// Log filtering summary
	totalFiltered := filteredMissingTime + filteredCancelled + filteredAllDay + filteredOutsideWindow
	if totalFiltered > 0 {
		log.Printf("  Filtered %d events: %d cancelled, %d all-day, %d outside window, %d missing time",
			totalFiltered, filteredCancelled, filteredAllDay, filteredOutsideWindow, filteredMissingTime)
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
