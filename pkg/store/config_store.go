package store

import (
	"encoding/json"

	"fyne.io/fyne/v2"
	"github.com/borgmon/focus-breaker/pkg/models"
)

// ConfigStore handles configuration persistence using Fyne preferences
type ConfigStore struct {
	app fyne.App
}

// NewConfigStore creates a new ConfigStore instance
func NewConfigStore(app fyne.App) *ConfigStore {
	return &ConfigStore{app: app}
}

// Load loads configuration from preferences
func (cs *ConfigStore) Load() *models.Config {
	prefs := cs.app.Preferences()

	config := &models.Config{
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
			config.ICalSources = []models.ICalSource{}
		}
	} else {
		config.ICalSources = []models.ICalSource{}
	}

	// Load quiet time ranges from JSON string
	quietTimeJSON := prefs.String("quiet_time_ranges")
	if quietTimeJSON != "" {
		if err := json.Unmarshal([]byte(quietTimeJSON), &config.QuietTimeRanges); err != nil {
			config.QuietTimeRanges = []models.TimeRange{}
		}
	} else {
		config.QuietTimeRanges = []models.TimeRange{}
	}

	return config
}

// Save saves configuration to preferences
func (cs *ConfigStore) Save(config *models.Config) {
	prefs := cs.app.Preferences()

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

	// Save quiet time ranges as JSON string
	if quietTimeJSON, err := json.Marshal(config.QuietTimeRanges); err == nil {
		prefs.SetString("quiet_time_ranges", string(quietTimeJSON))
	}
}
