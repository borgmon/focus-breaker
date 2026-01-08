package calendar

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/borgmon/focus-breaker/pkg/models"
	"github.com/emersion/go-ical"
)

func parseEvent(comp *ical.Component) models.Event {
	event := models.Event{}

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
		if t, err := parseDateTimeProperty(startProp); err == nil {
			event.StartTime = t
		}
	}

	if endProp := comp.Props.Get(ical.PropDateTimeEnd); endProp != nil {
		if t, err := parseDateTimeProperty(endProp); err == nil {
			event.EndTime = t
		}
	}

	if statusProp := comp.Props.Get(ical.PropStatus); statusProp != nil {
		event.Status = statusProp.Value
	}

	// Polyfill: If status is not CANCELLED but title indicates cancellation, set status to CANCELLED
	if event.Status != "CANCELLED" && isCancelledTitle(event.Title) {
		event.Status = "CANCELLED"
	}

	// Try to extract meeting link from location if not found in description
	if locProp := comp.Props.Get(ical.PropLocation); locProp != nil && event.MeetingLink == "" {
		event.MeetingLink = extractMeetingLink(locProp.Value)
	}

	return event
}

func parseDateTimeProperty(prop *ical.Prop) (time.Time, error) {
	// First try the standard DateTime method with local timezone
	if t, err := prop.DateTime(time.Local); err == nil {
		return t.In(time.Local), nil
	}

	// If that fails, try parsing the raw value directly
	value := prop.Value

	// Try multiple datetime formats
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

func extractMeetingLink(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^[\]` + "`" + `]+`)
	matches := urlRegex.FindAllString(text, -1)

	// Prioritize known meeting platforms
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

	// Return first URL if no known platform found
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

func isCancelledTitle(title string) bool {
	cleanTitle := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(strings.ToLower(title), "")
	return strings.HasPrefix(cleanTitle, "canceled") || strings.HasPrefix(cleanTitle, "cancelled")
}
