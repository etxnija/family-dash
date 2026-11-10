package dashboard

import (
	"context"
	"log/slog"
	"sync"
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

// cacheEntry holds a source's last successfully fetched events and the now
// of that fetch; at is zero if the source has never succeeded.
type cacheEntry struct {
	events []Event
	at     time.Time
}

// Collector fetches events from a fixed set of owned sources and derives the
// dashboard's event views from the merged result. It caches each source's
// last successful fetch so a failing source can be served stale for up to
// StaleLimit. It is safe for concurrent use.
type Collector struct {
	sources []source
	owners  []string

	mu    sync.Mutex
	cache []cacheEntry // parallel to sources, guarded by mu
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
	return &Collector{sources: internal, owners: owners, cache: make([]cacheEntry, len(internal))}
}

// Owners returns the distinct owner names, in insertion order.
func (c *Collector) Owners() []string {
	return c.owners
}

// Collect fetches events from every held source, tags them with their
// owner, merges them, and derives the today/homework/upcoming views
// relative to now. A successful fetch replaces the source's cache. A source
// that fails is logged and reported in Problems: if its last success is at
// most StaleLimit before now its cached events are used (Stale), otherwise
// it contributes no events. A failing source never prevents other sources
// from contributing events.
func (c *Collector) Collect(ctx context.Context, now time.Time) Collection {
	var events []Event
	var problems []FeedProblem
	for i, s := range c.sources {
		fetched, err := s.source.Events(ctx)

		c.mu.Lock()
		entry := c.cache[i]
		if err == nil {
			entry = cacheEntry{events: fetched, at: now}
			c.cache[i] = entry
		}
		c.mu.Unlock()

		if err == nil {
			slog.Info("fetched events", "source", s.owner, "count", len(fetched))
			events = append(events, WithOwner(fetched, s.owner)...)
			continue
		}

		stale := !entry.at.IsZero() && now.Sub(entry.at) <= StaleLimit
		lastSuccess := ""
		if !entry.at.IsZero() {
			lastSuccess = entry.at.Format(time.RFC3339)
		}
		slog.Error("fetching events", "source", s.owner, "error", err, "serving_stale", stale, "last_success", lastSuccess)
		problems = append(problems, FeedProblem{Owner: s.owner, Stale: stale, LastSuccess: entry.at})
		if stale {
			events = append(events, WithOwner(entry.events, s.owner)...)
		}
	}

	today := TodayEvents(events, now)
	slog.Info("filtered to today", "count", len(today))

	homework := HomeworkEvents(events, now)
	slog.Info("filtered to homework", "count", len(homework))

	upcoming := UpcomingEvents(events, now)
	slog.Info("filtered to upcoming", "count", len(upcoming))

	return Collection{Today: today, Homework: homework, Upcoming: upcoming, Problems: problems}
}
