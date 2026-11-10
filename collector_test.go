package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
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

			got := dashboard.NewCollector(tt.sources).Collect(context.Background(), now)

			assertEventsEqual(t, "today", got.Today, wantToday)
			assertEventsEqual(t, "homework", got.Homework, wantHomework)
			assertEventsEqual(t, "upcoming", got.Upcoming, wantUpcoming)
		})
	}
}

// switchSource is an EventSource whose result can be changed between
// Collect calls. It is safe for concurrent use.
type switchSource struct {
	mu     sync.Mutex
	events []dashboard.Event
	err    error
}

func (s *switchSource) set(events []dashboard.Event, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events, s.err = events, err
}

func (s *switchSource) Events(ctx context.Context) ([]dashboard.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	return s.events, nil
}

// fetchResult is what one source returns in one staleStep.
type fetchResult struct {
	events []dashboard.Event
	err    error
}

// staleStep is one Collect call: the per-source results to return (in
// source order), the now to collect at, and the expected outcome.
type staleStep struct {
	now          time.Time
	results      []fetchResult
	wantUpcoming []dashboard.Event
	wantProblems []dashboard.FeedProblem
}

// upcomingEvent returns an event named summary on 2026-11-13, inside the
// upcoming window of every now used by TestCollector_Collect_StaleCache*.
func upcomingEvent(summary string) dashboard.Event {
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, stockholm())
	return dashboard.Event{Summary: summary, Start: start, End: start.Add(time.Hour)}
}

// owned returns e tagged with owner.
func owned(e dashboard.Event, owner string) dashboard.Event {
	e.Owner = owner
	return e
}

// staleCacheT0 is the first now of every TestCollector_Collect_StaleCache* step.
func staleCacheT0() time.Time {
	return time.Date(2026, 11, 10, 8, 0, 0, 0, stockholm())
}

var errGone = errors.New("http 410")

// ok returns a successful fetch of events.
func ok(events ...dashboard.Event) fetchResult {
	return fetchResult{events: events}
}

// failed returns a failed fetch.
func failed() fetchResult {
	return fetchResult{err: errGone}
}

// aliceOK is a step where alice's only source returns e.
func aliceOK(now time.Time, e dashboard.Event) staleStep {
	return staleStep{now: now, results: []fetchResult{ok(e)}, wantUpcoming: []dashboard.Event{owned(e, "alice")}}
}

// aliceFails is a step where alice's only source fails, serving cached (if
// any) and reporting problem.
func aliceFails(now time.Time, cached []dashboard.Event, problem dashboard.FeedProblem) staleStep {
	problem.Owner = "alice"
	return staleStep{now: now, results: []fetchResult{failed()}, wantUpcoming: cached, wantProblems: []dashboard.FeedProblem{problem}}
}

func TestCollector_Collect_StaleCache(t *testing.T) {
	t0 := staleCacheT0()
	a1, a2 := upcomingEvent("a1"), upcomingEvent("a2")
	cachedA1 := []dashboard.Event{owned(a1, "alice")}

	tests := []struct {
		name  string
		steps []staleStep
	}{
		{"success records no problem", []staleStep{aliceOK(t0, a1)}},
		{"failure 3h after a success serves the cache", []staleStep{
			aliceOK(t0, a1),
			aliceFails(t0.Add(3*time.Hour), cachedA1, dashboard.FeedProblem{Stale: true, LastSuccess: t0}),
		}},
		{"failure exactly StaleLimit after a success is still stale", []staleStep{
			aliceOK(t0, a1),
			aliceFails(t0.Add(dashboard.StaleLimit), cachedA1, dashboard.FeedProblem{Stale: true, LastSuccess: t0}),
		}},
		{"failure past StaleLimit drops the cache", []staleStep{
			aliceOK(t0, a1),
			aliceFails(t0.Add(dashboard.StaleLimit+time.Minute), nil, dashboard.FeedProblem{LastSuccess: t0}),
		}},
		{"success after a failure clears the problem", []staleStep{
			aliceFails(t0, nil, dashboard.FeedProblem{}),
			aliceOK(t0.Add(time.Hour), a1),
		}},
		{"failure serves the newest cached success", []staleStep{
			aliceOK(t0, a1),
			aliceOK(t0.Add(time.Hour), a2),
			aliceFails(t0.Add(2*time.Hour), []dashboard.Event{owned(a2, "alice")}, dashboard.FeedProblem{Stale: true, LastSuccess: t0.Add(time.Hour)}),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStaleSteps(t, []string{"alice"}, tt.steps)
		})
	}
}

func TestCollector_Collect_StaleCacheMultiSource(t *testing.T) {
	t0 := staleCacheT0()
	a1, a2, b1 := upcomingEvent("a1"), upcomingEvent("a2"), upcomingEvent("b1")

	tests := []struct {
		name   string
		owners []string
		steps  []staleStep
	}{
		{"failure with no cache drops the source and leaves others alone", []string{"alice", "bob"}, []staleStep{{
			now:          t0,
			results:      []fetchResult{failed(), ok(b1)},
			wantUpcoming: []dashboard.Event{owned(b1, "bob")},
			wantProblems: []dashboard.FeedProblem{{Owner: "alice"}},
		}}},
		{"problems are reported in source order", []string{"alice", "bob", "carol"}, []staleStep{{
			now:          t0,
			results:      []fetchResult{ok(a1), failed(), ok(b1)},
			wantUpcoming: []dashboard.Event{owned(a1, "alice"), owned(b1, "carol")},
			wantProblems: []dashboard.FeedProblem{{Owner: "bob"}},
		}, {
			now:          t0.Add(time.Hour),
			results:      []fetchResult{failed(), failed(), ok(b1)},
			wantUpcoming: []dashboard.Event{owned(a1, "alice"), owned(b1, "carol")},
			wantProblems: []dashboard.FeedProblem{{Owner: "alice", Stale: true, LastSuccess: t0}, {Owner: "bob"}},
		}}},
		{"two sources with the same owner are cached separately", []string{"alice", "alice"}, []staleStep{{
			now:          t0,
			results:      []fetchResult{ok(a1), ok(a2)},
			wantUpcoming: []dashboard.Event{owned(a1, "alice"), owned(a2, "alice")},
		}, {
			now:          t0.Add(time.Hour),
			results:      []fetchResult{ok(b1), failed()},
			wantUpcoming: []dashboard.Event{owned(b1, "alice"), owned(a2, "alice")},
			wantProblems: []dashboard.FeedProblem{{Owner: "alice", Stale: true, LastSuccess: t0}},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStaleSteps(t, tt.owners, tt.steps)
		})
	}
}

// runStaleSteps builds a collector with one switchSource per owner and runs
// steps against it in order, checking each step's upcoming events and problems.
func runStaleSteps(t *testing.T, owners []string, steps []staleStep) {
	t.Helper()
	fakes := make([]*switchSource, len(owners))
	named := make([]dashboard.NamedSource, len(owners))
	for i, owner := range owners {
		fakes[i] = &switchSource{}
		named[i] = dashboard.NamedSource{Owner: owner, Source: fakes[i]}
	}
	c := dashboard.NewCollector(named)

	for i, step := range steps {
		for j, r := range step.results {
			fakes[j].set(r.events, r.err)
		}
		got := c.Collect(context.Background(), step.now)

		assertEventsEqual(t, fmt.Sprintf("step %d upcoming", i), got.Upcoming, step.wantUpcoming)
		if !slices.Equal(got.Problems, step.wantProblems) {
			t.Errorf("step %d Problems = %+v, want %+v", i, got.Problems, step.wantProblems)
		}
		if step.wantProblems == nil && got.Problems != nil {
			t.Errorf("step %d Problems = %#v, want nil", i, got.Problems)
		}
	}
}

func TestCollector_Collect_Concurrent(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, stockholm())
	fake := &switchSource{}
	c := dashboard.NewCollector([]dashboard.NamedSource{
		{Owner: "alice", Source: fake},
		{Owner: "bob", Source: fakeSource{events: []dashboard.Event{upcomingEvent("b1")}}},
	})

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 50 {
				if (g+i)%2 == 0 {
					fake.set([]dashboard.Event{upcomingEvent("a1")}, nil)
				} else {
					fake.set(nil, errors.New("http 503"))
				}
				got := c.Collect(context.Background(), now.Add(time.Duration(i)*time.Minute))
				if len(got.Problems) > 1 || (len(got.Problems) == 1 && got.Problems[0].Owner != "alice") {
					t.Errorf("Problems = %+v, want at most one for alice", got.Problems)
				}
				if n := len(got.Upcoming); n < 1 || n > 2 {
					t.Errorf("len(Upcoming) = %d, want 1 or 2", n)
				}
			}
		}()
	}
	wg.Wait()
}
