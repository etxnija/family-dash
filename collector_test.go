package dashboard_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dashboard "family-dashboard"
)

// fakeSource is a canned EventSource for testing Collector.
type fakeSource struct {
	events []dashboard.Event
	err    error
}

func (f fakeSource) Events(ctx context.Context) ([]dashboard.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.events, nil
}

func TestNewCollector_Owners(t *testing.T) {
	tests := []struct {
		name    string
		sources []dashboard.NamedSource
		want    []string
	}{
		{
			name: "distinct owners kept in insertion order",
			sources: []dashboard.NamedSource{
				{Owner: "bob", Source: fakeSource{}},
				{Owner: "alice", Source: fakeSource{}},
			},
			want: []string{"bob", "alice"},
		},
		{
			name: "duplicate owners are deduplicated, first occurrence order kept",
			sources: []dashboard.NamedSource{
				{Owner: "alice", Source: fakeSource{}},
				{Owner: "bob", Source: fakeSource{}},
				{Owner: "alice", Source: fakeSource{}},
			},
			want: []string{"alice", "bob"},
		},
		{
			name:    "no sources returns empty",
			sources: nil,
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dashboard.NewCollector(tt.sources).Owners()
			if len(got) != len(tt.want) {
				t.Fatalf("Owners() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Owners()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// assertEventsEqual compares got and want by Summary and Owner, in order.
func assertEventsEqual(t *testing.T, label string, got, want []dashboard.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for i := range got {
		if got[i].Summary != want[i].Summary || got[i].Owner != want[i].Owner {
			t.Errorf("%s[%d] = %+v, want %+v", label, i, got[i], want[i])
		}
	}
}

// collectorFixtureEvents returns the canned events used across TestCollector_Collect cases.
func collectorFixtureEvents(loc *time.Location) (today, homework, upcoming, outOfWindow dashboard.Event) {
	today = dashboard.Event{
		Summary: "today",
		Start:   time.Date(2026, 9, 17, 9, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 17, 10, 0, 0, 0, loc),
	}
	homework = dashboard.Event{
		Summary: "Läxa matte",
		Start:   time.Date(2026, 9, 19, 9, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 19, 10, 0, 0, 0, loc),
	}
	upcoming = dashboard.Event{
		Summary: "Bortamatch",
		Start:   time.Date(2026, 9, 18, 10, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 18, 11, 0, 0, 0, loc),
	}
	outOfWindow = dashboard.Event{
		Summary: "too far out",
		Start:   time.Date(2026, 9, 30, 9, 0, 0, 0, loc),
		End:     time.Date(2026, 9, 30, 10, 0, 0, 0, loc),
	}
	return today, homework, upcoming, outOfWindow
}

// mergeSources replicates Collector's fetch-and-tag step for computing expected results.
func mergeSources(sources []dashboard.NamedSource) []dashboard.Event {
	var merged []dashboard.Event
	for _, s := range sources {
		fetched, err := s.Source.Events(context.Background())
		if err != nil {
			continue
		}
		merged = append(merged, dashboard.WithOwner(fetched, s.Owner)...)
	}
	return merged
}

func TestCollector_Collect(t *testing.T) {
	loc := stockholm()
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, loc)
	todayEvent, homeworkEvent, upcomingEvent, outOfWindowEvent := collectorFixtureEvents(loc)

	tests := []struct {
		name    string
		sources []dashboard.NamedSource
	}{
		{
			name: "multiple sources merge and get correct owner tags",
			sources: []dashboard.NamedSource{
				{Owner: "alice", Source: fakeSource{events: []dashboard.Event{todayEvent, homeworkEvent}}},
				{Owner: "bob", Source: fakeSource{events: []dashboard.Event{upcomingEvent, outOfWindowEvent}}},
			},
		},
		{
			name: "a failing source is skipped without failing others",
			sources: []dashboard.NamedSource{
				{Owner: "alice", Source: fakeSource{events: []dashboard.Event{todayEvent}}},
				{Owner: "bob", Source: fakeSource{err: errors.New("boom")}},
				{Owner: "carol", Source: fakeSource{events: []dashboard.Event{upcomingEvent}}},
			},
		},
		{
			name:    "no sources returns empty views",
			sources: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeSources(tt.sources)
			wantToday := dashboard.TodayEvents(merged, now)
			wantHomework := dashboard.HomeworkEvents(merged, now)
			wantUpcoming := dashboard.UpcomingEvents(merged, now)

			gotToday, gotHomework, gotUpcoming := dashboard.NewCollector(tt.sources).Collect(context.Background(), now)

			assertEventsEqual(t, "today", gotToday, wantToday)
			assertEventsEqual(t, "homework", gotHomework, wantHomework)
			assertEventsEqual(t, "upcoming", gotUpcoming, wantUpcoming)
		})
	}
}
