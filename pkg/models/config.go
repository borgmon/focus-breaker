package models

import (
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	AutoStart        bool         `json:"auto_start"`
	ICalSources      []ICalSource `json:"ical_sources"`
	UpdateInterval   int          `json:"update_interval"`    // minutes
	SnoozeTime       int          `json:"snooze_time"`        // minutes
	NotifyUnaccepted bool         `json:"notify_unaccepted"`  // notify for unaccepted events
	AlertBeforeMin   string       `json:"alert_before_min"`   // comma-separated minutes
	HoldTimeSeconds  int          `json:"hold_time_seconds"`  // button hold time
	QuietTimeRanges  []TimeRange  `json:"quiet_time_ranges"`  // quiet time ranges
}

// ICalSource represents a named iCal calendar source
type ICalSource struct {
	ID   string `json:"id"`   // Unique identifier
	Name string `json:"name"` // Display name
	URL  string `json:"url"`  // iCal URL
}

// TimeRange represents a time range within a day
type TimeRange struct {
	StartHour   int `json:"start_hour"`   // 0-23
	StartMinute int `json:"start_minute"` // 0-59
	EndHour     int `json:"end_hour"`     // 0-23
	EndMinute   int `json:"end_minute"`   // 0-59
}

// NeedsConfiguration returns true if the config needs initial setup
func (c *Config) NeedsConfiguration() bool {
	return len(c.ICalSources) == 0
}

// GetAlertMinutes returns the list of alert minutes including 0 (event start)
func (c *Config) GetAlertMinutes() []int {
	minutes := []int{0} // Always alert at event start

	if c.AlertBeforeMin == "" {
		return minutes
	}

	parts := strings.Split(c.AlertBeforeMin, ",")
	seen := make(map[int]bool)
	seen[0] = true // Mark 0 as already added

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if min, err := strconv.Atoi(part); err == nil {
			// Skip 0 since we always add it, and skip duplicates
			if min > 0 && !seen[min] {
				minutes = append(minutes, min)
				seen[min] = true
			}
		}
	}

	return minutes
}

// IsInQuietTime returns true if current time is in a quiet time range
func (c *Config) IsInQuietTime() bool {
	return c.IsTimeInQuietTime(time.Now())
}

// IsTimeInQuietTime returns true if the given time is in a quiet time range
func (c *Config) IsTimeInQuietTime(t time.Time) bool {
	if len(c.QuietTimeRanges) == 0 {
		return false
	}

	currentMinutes := t.Hour()*60 + t.Minute()

	for _, tr := range c.QuietTimeRanges {
		startMinutes := tr.StartHour*60 + tr.StartMinute
		endMinutes := tr.EndHour*60 + tr.EndMinute

		// Handle overnight ranges (e.g., 22:00 to 08:00)
		if endMinutes < startMinutes {
			if currentMinutes >= startMinutes || currentMinutes < endMinutes {
				return true
			}
		} else {
			if currentMinutes >= startMinutes && currentMinutes < endMinutes {
				return true
			}
		}
	}

	return false
}

// Validate checks if the iCal source has required fields
func (s *ICalSource) Validate() bool {
	return s.Name != "" && s.URL != ""
}
