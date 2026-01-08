package calendar

import (
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

// Map of common Windows timezone names to IANA timezone names
var windowsToIANA = map[string]string{
	"Pacific Standard Time":        "America/Los_Angeles",
	"Mountain Standard Time":       "America/Denver",
	"Central Standard Time":        "America/Chicago",
	"Eastern Standard Time":        "America/New_York",
	"Atlantic Standard Time":       "America/Halifax",
	"Alaskan Standard Time":        "America/Anchorage",
	"Hawaiian Standard Time":       "Pacific/Honolulu",
	"GMT Standard Time":            "Europe/London",
	"Central Europe Standard Time": "Europe/Paris",
	"China Standard Time":          "Asia/Shanghai",
	"Tokyo Standard Time":          "Asia/Tokyo",
	"India Standard Time":          "Asia/Kolkata",
	"AUS Eastern Standard Time":    "Australia/Sydney",
}

// normalizeComponentTimezones fixes Windows timezone names in a component before processing
func normalizeComponentTimezones(comp *ical.Component) {
	// Check DTSTART
	if dtstart := comp.Props.Get(ical.PropDateTimeStart); dtstart != nil {
		if tzid := dtstart.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if ianaName, ok := windowsToIANA[tzid]; ok {
				dtstart.Params.Set(ical.ParamTimezoneID, ianaName)
			}
		}
	}

	// Check DTEND
	if dtend := comp.Props.Get(ical.PropDateTimeEnd); dtend != nil {
		if tzid := dtend.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if ianaName, ok := windowsToIANA[tzid]; ok {
				dtend.Params.Set(ical.ParamTimezoneID, ianaName)
			}
		}
	}

	// Check EXDATE properties
	for _, exdate := range comp.Props.Values(ical.PropExceptionDates) {
		if tzid := exdate.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if ianaName, ok := windowsToIANA[tzid]; ok {
				exdate.Params.Set(ical.ParamTimezoneID, ianaName)
			}
		}
	}

	// Check RDATE properties
	for _, rdate := range comp.Props.Values(ical.PropRecurrenceDates) {
		if tzid := rdate.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if ianaName, ok := windowsToIANA[tzid]; ok {
				rdate.Params.Set(ical.ParamTimezoneID, ianaName)
			}
		}
	}
}

// getTimezoneFromComponent tries to determine the timezone for a component
func getTimezoneFromComponent(comp *ical.Component) *time.Location {
	// Check DTSTART for timezone
	if dtstart := comp.Props.Get(ical.PropDateTimeStart); dtstart != nil {
		if tzid := dtstart.Params.Get(ical.ParamTimezoneID); tzid != "" {
			// Try to map Windows timezone to IANA
			if ianaName, ok := windowsToIANA[tzid]; ok {
				if loc, err := time.LoadLocation(ianaName); err == nil {
					return loc
				}
			}
			// Try original timezone name
			if loc, err := time.LoadLocation(tzid); err == nil {
				return loc
			}
		}

		// Check if it's a UTC time (ends with Z)
		if strings.HasSuffix(dtstart.Value, "Z") {
			return time.UTC
		}
	}

	// Default to local timezone
	return time.Local
}
