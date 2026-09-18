// Package dashboard is the domain package: the center of the hexagon.
package dashboard

import (
	"context"
	"time"
)

// Event is a single calendar event.
type Event struct {
	Summary     string
	Location    string
	Description string
	// Start is always UTC.
	Start time.Time
	// End is always UTC.
	End time.Time
}

// EventSource is a port for fetching events; adapters implement it.
// Adapters must return UTC-normalized Start/End times.
type EventSource interface {
	Events(ctx context.Context) ([]Event, error)
}

// TodayEvents returns the events from events whose Start falls on the same
// calendar date as now, in the Europe/Stockholm timezone.
func TodayEvents(events []Event, now time.Time) []Event {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}

	today := now.In(loc)
	year, month, day := today.Date()

	var result []Event
	for _, e := range events {
		y, m, d := e.Start.In(loc).Date()
		if y == year && m == month && d == day {
			result = append(result, e)
		}
	}
	return result
}
