package calendar

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/borgmon/focus-breaker/pkg/models"
	"github.com/emersion/go-ical"
)

// FetchEvents fetches and parses events from an iCal source
func FetchEvents(source models.ICalSource) ([]models.Event, error) {
	events, err := fetchAndParseICal(source.URL)
	if err != nil {
		return nil, err
	}

	// Set the source ID for all events
	eventsWithoutUID := 0
	for i := range events {
		events[i].SourceID = source.ID
		// Fallback: if no iCal UID, use deterministic ID based on start time and title
		if events[i].ID == "" {
			events[i].ID = source.ID + "-" + events[i].StartTime.Format(time.RFC3339) + "-" + events[i].Title
			eventsWithoutUID++
		}
	}

	if eventsWithoutUID > 0 {
		log.Printf("Generated fallback IDs for %d events without UID", eventsWithoutUID)
	}

	return events, nil
}

func fetchAndParseICal(icalURL string) ([]models.Event, error) {
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

	// Validate response format
	if err := validateICalFormat(bodyStr); err != nil {
		return nil, err
	}

	decoder := ical.NewDecoder(strings.NewReader(bodyStr))
	events := []models.Event{}
	seenEventIDs := make(map[string]bool)
	seenEventKeys := make(map[string]bool) // key: title + start time

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	// Tracking filtered events
	stats := &filterStats{}

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
			stats.totalComponents++
			if comp.Name != ical.CompEvent {
				log.Printf("  [DEBUG] Skipping non-event component: %s", comp.Name)
				continue
			}
			stats.totalEvents++

			event := parseEvent(comp)

			// Check if this is a recurring event
			if rruleProp := comp.Props.Get(ical.PropRecurrenceRule); rruleProp != nil {
				log.Printf("  [RECURRING] Event: \"%s\" has RRULE: %s", event.Title, rruleProp.Value)
				recurringEvents := expandRecurringEvent(event, rruleProp.Value, now, tomorrow)
				for _, recEvent := range recurringEvents {
					if shouldIncludeEvent(recEvent, now, tomorrow, stats) {
						if !isDuplicate(recEvent, seenEventIDs, seenEventKeys, stats) {
							events = append(events, recEvent)
						}
					}
				}
				continue
			}

			// Process single event
			if shouldIncludeEvent(event, now, tomorrow, stats) {
				if !isDuplicate(event, seenEventIDs, seenEventKeys, stats) {
					events = append(events, event)
				}
			}
		}
	}

	// Log filtering summary
	stats.logSummary(len(events))

	return events, nil
}

func validateICalFormat(bodyStr string) error {
	// Check if response is HTML instead of iCalendar
	upperBody := strings.ToUpper(strings.TrimSpace(bodyStr))
	if strings.HasPrefix(upperBody, "<!DOCTYPE") || strings.HasPrefix(upperBody, "<HTML") {
		return fmt.Errorf("received HTML instead of iCalendar data - check if URL requires authentication")
	}

	// Check if it starts with BEGIN:VCALENDAR
	if !strings.HasPrefix(strings.TrimSpace(bodyStr), "BEGIN:VCALENDAR") {
		previewLen := 100
		if len(bodyStr) < previewLen {
			previewLen = len(bodyStr)
		}
		return fmt.Errorf("invalid iCalendar format - expected BEGIN:VCALENDAR, got: %s",
			strings.TrimSpace(bodyStr[:previewLen]))
	}

	return nil
}

func isDuplicate(event models.Event, seenEventIDs, seenEventKeys map[string]bool, stats *filterStats) bool {
	// Check for duplicates by ID
	if seenEventIDs[event.ID] {
		stats.filteredDuplicates++
		log.Printf("  [FILTERED] Duplicate (ID) - Event: \"%s\" (ID: %s)", event.Title, event.ID)
		return true
	}

	// Check for duplicates by title + start time
	eventKey := event.Title + "|" + event.StartTime.Format(time.RFC3339)
	if seenEventKeys[eventKey] {
		stats.filteredDuplicates++
		log.Printf("  [FILTERED] Duplicate (Title+Time) - Event: \"%s\" (Start: %s)",
			event.Title, event.StartTime.Format("2006-01-02 15:04"))
		return true
	}

	seenEventIDs[event.ID] = true
	seenEventKeys[eventKey] = true
	return false
}

type filterStats struct {
	totalComponents      int
	totalEvents          int
	filteredMissingTime  int
	filteredCancelled    int
	filteredAllDay       int
	filteredOutsideWindow int
	filteredDuplicates   int
}

func (s *filterStats) logSummary(includedCount int) {
	totalFiltered := s.filteredMissingTime + s.filteredCancelled + s.filteredAllDay + s.filteredOutsideWindow + s.filteredDuplicates
	log.Printf("  [SUMMARY] Total components: %d, Events: %d, Included: %d, Filtered: %d",
		s.totalComponents, s.totalEvents, includedCount, totalFiltered)
	if totalFiltered > 0 {
		log.Printf("  Filtered breakdown: %d cancelled, %d all-day, %d outside window, %d missing time, %d duplicates",
			s.filteredCancelled, s.filteredAllDay, s.filteredOutsideWindow, s.filteredMissingTime, s.filteredDuplicates)
	}
}
