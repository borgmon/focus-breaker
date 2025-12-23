package main

import (
	"fmt"
	"io"
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
	for i := range events {
		events[i].SourceID = s.ID
		// Fallback: if no iCal UID, use deterministic ID based on start time and title
		if events[i].ID == "" {
			events[i].ID = s.ID + "-" + events[i].StartTime.Format(time.RFC3339) + "-" + events[i].Title
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
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

	for {
		cal, err := decoder.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, comp := range cal.Children {
			if comp.Name != ical.CompEvent {
				continue
			}

			event := parseEvent(comp)

			if event.StartTime.IsZero() || event.EndTime.IsZero() {
				continue
			}

			startDate := event.StartTime.Format("2006-01-02")
			endDate := event.EndTime.Format("2006-01-02")
			if startDate != endDate && event.EndTime.Sub(event.StartTime) >= 24*time.Hour {
				continue
			}

			if event.StartTime.Before(tomorrow) && event.EndTime.After(now) {
				events = append(events, event)
			}
		}
	}

	return events, nil
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
		if t, err := startProp.DateTime(time.Local); err == nil {
			// Convert to local timezone for consistent comparisons
			event.StartTime = t.In(time.Local)
		}
	}

	if endProp := comp.Props.Get(ical.PropDateTimeEnd); endProp != nil {
		if t, err := endProp.DateTime(time.Local); err == nil {
			// Convert to local timezone for consistent comparisons
			event.EndTime = t.In(time.Local)
		}
	}

	if statusProp := comp.Props.Get(ical.PropStatus); statusProp != nil {
		event.Status = statusProp.Value
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
