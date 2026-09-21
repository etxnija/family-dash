// Package dashboard is the domain package: the center of the hexagon.
package dashboard

import (
	"context"
	"strings"
	"time"
)

// Event is a single calendar event.
type Event struct {
	Summary     string
	Location    string
	Description string
	Owner       string
	AllDay      bool
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

// homeworkPrefix identifies homework events by their Swedish summary prefix.
const homeworkPrefix = "Läxa "

// HomeworkEvents returns the events from events whose Summary starts with
// "Läxa " or whose Description contains "prov" (case-insensitive), and
// whose Start falls within the 7-calendar-day window from now's date
// through now's date plus 6 days (inclusive), in the Europe/Stockholm
// timezone.
func HomeworkEvents(events []Event, now time.Time) []Event {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}

	today := now.In(loc)
	year, month, day := today.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, loc)
	end := time.Date(year, month, day+6, 0, 0, 0, 0, loc)

	var result []Event
	for _, e := range events {
		isHomework := strings.HasPrefix(e.Summary, homeworkPrefix)
		isProv := strings.Contains(strings.ToLower(e.Description), "prov")
		if !isHomework && !isProv {
			continue
		}
		y, m, d := e.Start.In(loc).Date()
		eventDate := time.Date(y, m, d, 0, 0, 0, 0, loc)
		if !eventDate.Before(start) && !eventDate.After(end) {
			result = append(result, e)
		}
	}
	return result
}

// UpcomingEvents returns the events from events whose Start falls within the
// window from tomorrow through now's date plus 6 days (inclusive), in the
// Europe/Stockholm timezone, excluding events that match the homework/prov
// criteria (see HomeworkEvents).
func UpcomingEvents(events []Event, now time.Time) []Event {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}

	today := now.In(loc)
	year, month, day := today.Date()
	start := time.Date(year, month, day+1, 0, 0, 0, 0, loc)
	end := time.Date(year, month, day+6, 0, 0, 0, 0, loc)

	var result []Event
	for _, e := range events {
		isHomework := strings.HasPrefix(e.Summary, homeworkPrefix)
		isProv := strings.Contains(strings.ToLower(e.Description), "prov")
		if isHomework || isProv {
			continue
		}
		y, m, d := e.Start.In(loc).Date()
		eventDate := time.Date(y, m, d, 0, 0, 0, 0, loc)
		if !eventDate.Before(start) && !eventDate.After(end) {
			result = append(result, e)
		}
	}
	return result
}

// WithOwner returns a new slice with Owner set to owner on every event.
func WithOwner(events []Event, owner string) []Event {
	var result []Event
	for _, e := range events {
		e.Owner = owner
		result = append(result, e)
	}
	return result
}
