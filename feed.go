package dashboard

import "time"

// StaleLimit is how long a Collector keeps serving a failing source's last
// successfully fetched events before dropping them.
const StaleLimit = 24 * time.Hour

// FeedProblem reports a calendar source whose latest fetch failed.
type FeedProblem struct {
	// Owner is the owner name of the failing source.
	Owner string
	// Stale is true when the source's events were served from the cache
	// of its last successful fetch (at most StaleLimit old), and false
	// when the source contributed no events at all.
	Stale bool
	// LastSuccess is the now of the source's last successful fetch, or
	// the zero time if it has never succeeded since the process started.
	LastSuccess time.Time
}

// Collection is the result of one Collector.Collect: the derived event
// views plus any problems with the feeds they were built from.
type Collection struct {
	Today, Homework, Upcoming []Event
	// Problems lists the failing sources in source order; nil when every
	// source fetched successfully.
	Problems []FeedProblem
}
