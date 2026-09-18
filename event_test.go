package dashboard_test

import (
	"testing"
	"time"

	dashboard "family-dashboard"
)

func stockholm() *time.Location {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}
	return loc
}

func todayEventsFixtures(loc *time.Location) (sameDay, previousDay, nextDay, midnightBoundary dashboard.Event) {
	sameDay = dashboard.Event{
		Summary: "same day",
		Start:   time.Date(2026, 9, 17, 9, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 17, 10, 0, 0, 0, loc),
	}
	previousDay = dashboard.Event{
		Summary: "previous day",
		Start:   time.Date(2026, 9, 16, 23, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 17, 0, 0, 0, 0, loc),
	}
	nextDay = dashboard.Event{
		Summary: "next day",
		Start:   time.Date(2026, 9, 18, 0, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 18, 1, 0, 0, 0, loc),
	}
	midnightBoundary = dashboard.Event{
		Summary: "midnight boundary",
		Start:   time.Date(2026, 9, 17, 0, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 17, 1, 0, 0, 0, loc),
	}
	return sameDay, previousDay, nextDay, midnightBoundary
}

func assertTodayEvents(t *testing.T, got, want []dashboard.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("TodayEvents() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i].Summary != want[i].Summary {
			t.Errorf("TodayEvents()[%d] = %q, want %q", i, got[i].Summary, want[i].Summary)
		}
	}
}

func TestTodayEvents(t *testing.T) {
	loc := stockholm()
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, loc)
	sameDay, previousDay, nextDay, midnightBoundary := todayEventsFixtures(loc)

	tests := []struct {
		name   string
		events []dashboard.Event
		want   []dashboard.Event
	}{
		{
			name:   "event on same day is included",
			events: []dashboard.Event{sameDay},
			want:   []dashboard.Event{sameDay},
		},
		{
			name:   "event on previous day is excluded",
			events: []dashboard.Event{previousDay},
			want:   nil,
		},
		{
			name:   "event on next day is excluded",
			events: []dashboard.Event{nextDay},
			want:   nil,
		},
		{
			name:   "empty input returns empty slice",
			events: []dashboard.Event{},
			want:   nil,
		},
		{
			name:   "event starting at midnight boundary is included",
			events: []dashboard.Event{midnightBoundary},
			want:   []dashboard.Event{midnightBoundary},
		},
		{
			name:   "mixed events filters to only today",
			events: []dashboard.Event{previousDay, sameDay, nextDay, midnightBoundary},
			want:   []dashboard.Event{sameDay, midnightBoundary},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTodayEvents(t, dashboard.TodayEvents(tt.events, now), tt.want)
		})
	}
}

func assertOwners(t *testing.T, got []dashboard.Event, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("WithOwner() = %v, want %d events", got, len(want))
	}
	for i := range got {
		if got[i].Owner != want[i] {
			t.Errorf("WithOwner()[%d].Owner = %q, want %q", i, got[i].Owner, want[i])
		}
	}
}

func TestWithOwner(t *testing.T) {
	tests := []struct {
		name   string
		events []dashboard.Event
		owner  string
		want   []string
	}{
		{
			name: "non-empty slice gets owner set on all elements",
			events: []dashboard.Event{
				{Summary: "first"},
				{Summary: "second"},
			},
			owner: "alice",
			want:  []string{"alice", "alice"},
		},
		{
			name:   "empty slice returns nil",
			events: []dashboard.Event{},
			owner:  "alice",
			want:   nil,
		},
		{
			name: "existing owner is overwritten",
			events: []dashboard.Event{
				{Summary: "first", Owner: "bob"},
			},
			owner: "alice",
			want:  []string{"alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertOwners(t, dashboard.WithOwner(tt.events, tt.owner), tt.want)
		})
	}
}
