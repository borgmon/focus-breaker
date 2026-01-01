package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
)

type Config struct {
	AutoStart        bool         `json:"auto_start"`
	ICalSources      []ICalSource `json:"ical_sources"`
	UpdateInterval   int          `json:"update_interval"`
	SnoozeTime       int          `json:"snooze_time"`
	NotifyUnaccepted bool         `json:"notify_unaccepted"`
	AlertBeforeMin   string       `json:"alert_before_min"`
	HoldTimeSeconds  int          `json:"hold_time_seconds"`
}

func loadConfig(app fyne.App) *Config {
	prefs := app.Preferences()

	config := &Config{
		AutoStart:        prefs.BoolWithFallback("auto_start", false),
		UpdateInterval:   prefs.IntWithFallback("update_interval", 30),
		SnoozeTime:       prefs.IntWithFallback("snooze_time", 4),
		NotifyUnaccepted: prefs.BoolWithFallback("notify_unaccepted", false),
		AlertBeforeMin:   prefs.StringWithFallback("alert_before_min", "5,15"),
		HoldTimeSeconds:  prefs.IntWithFallback("hold_time_seconds", 5),
	}

	// Load iCal sources from JSON string
	icalSourcesJSON := prefs.String("ical_sources")
	if icalSourcesJSON != "" {
		if err := json.Unmarshal([]byte(icalSourcesJSON), &config.ICalSources); err != nil {
			config.ICalSources = []ICalSource{}
		}
	} else {
		config.ICalSources = []ICalSource{}
	}

	return config
}

func saveConfig(app fyne.App, config *Config) {
	prefs := app.Preferences()

	prefs.SetBool("auto_start", config.AutoStart)
	prefs.SetInt("update_interval", config.UpdateInterval)
	prefs.SetInt("snooze_time", config.SnoozeTime)
	prefs.SetBool("notify_unaccepted", config.NotifyUnaccepted)
	prefs.SetString("alert_before_min", config.AlertBeforeMin)
	prefs.SetInt("hold_time_seconds", config.HoldTimeSeconds)

	// Save iCal sources as JSON string
	if icalSourcesJSON, err := json.Marshal(config.ICalSources); err == nil {
		prefs.SetString("ical_sources", string(icalSourcesJSON))
	}
}

func (c *Config) NeedsConfiguration() bool {
	return len(c.ICalSources) == 0
}

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
