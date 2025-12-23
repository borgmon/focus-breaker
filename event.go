package main

import (
	"time"
)

type Event struct {
	ID          string // iCal event UID
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	MeetingLink string
	Status      string
	SourceID    string // ID of the iCal source this event came from
}
