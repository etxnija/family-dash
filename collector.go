package dashboard

import (
	"context"
	"log/slog"
	"time"
)

// source pairs an owner name with the EventSource that fetches their events.
type source struct {
	owner  string
	source EventSource
}

// NamedSource pairs an owner name with the EventSource that fetches their
// events. It is the input to NewCollector.
type NamedSource struct {
	Owner  string
	Source EventSource
}

// Collector fetches events from a fixed set of owned sources and derives the
// dashboard's event views from the merged result.
type Collector struct {
	sources []source
	owners  []string
}

// NewCollector builds a Collector from sources, deriving the distinct list of
// owner names in the order they first appear.
func NewCollector(sources []NamedSource) *Collector {
	internal := make([]source, len(sources))
	seen := make(map[string]bool, len(sources))
	owners := make([]string, 0, len(sources))
	for i, s := range sources {
		internal[i] = source{owner: s.Owner, source: s.Source}
		if seen[s.Owner] {
			continue
		}
		seen[s.Owner] = true
		owners = append(owners, s.Owner)
	}
	return &Collector{sources: internal, owners: owners}
}

// Owners returns the distinct owner names, in insertion order.
func (c *Collector) Owners() []string {
	return c.owners
}

// Collect fetches events from every held source, tags them with their
// owner, merges them, and derives the today/homework/upcoming views
// relative to now. A source that fails to fetch is logged and skipped; it
// does not prevent other sources from contributing events.
func (c *Collector) Collect(ctx context.Context, now time.Time) (today, homework, upcoming []Event) {
	var events []Event
	for _, s := range c.sources {
		fetched, err := s.source.Events(ctx)
		if err != nil {
			slog.Error("fetching events", "source", s.owner, "error", err)
			continue
		}
		slog.Info("fetched events", "source", s.owner, "count", len(fetched))
		events = append(events, WithOwner(fetched, s.owner)...)
	}

	today = TodayEvents(events, now)
	slog.Info("filtered to today", "count", len(today))

	homework = HomeworkEvents(events, now)
	slog.Info("filtered to homework", "count", len(homework))

	upcoming = UpcomingEvents(events, now)
	slog.Info("filtered to upcoming", "count", len(upcoming))

	return today, homework, upcoming
}
