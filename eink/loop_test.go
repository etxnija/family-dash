package eink

import (
	"context"
	"errors"
	"image"
	"reflect"
	"testing"
	"time"
)

// loopClock is a fake Clock. Sleep advances the time by d (or by step, if
// set) and cancels the context on sleep number ticks+1, so the loop runs
// exactly ticks ticks and is then stopped mid-Sleep.
type loopClock struct {
	now    time.Time
	step   time.Duration
	ticks  int
	sleeps int
	cancel context.CancelFunc
}

func (c *loopClock) Now() time.Time { return c.now }

func (c *loopClock) Sleep(ctx context.Context, d time.Duration) error {
	c.sleeps++
	if c.sleeps > c.ticks {
		c.cancel()
		return ctx.Err()
	}
	if c.step > 0 {
		d = c.step
	}
	c.now = c.now.Add(d)
	return ctx.Err()
}

// loopPanel is a fake Panel recording its calls. errs scripts an error per
// call index, and cancelAt (if > 0) cancels the context on call number
// cancelAt.
type loopPanel struct {
	calls    []string
	rects    []image.Rectangle
	errs     map[int]error
	cancelAt int
	cancel   context.CancelFunc
}

func (p *loopPanel) record(call string) error {
	p.calls = append(p.calls, call)
	if len(p.calls) == p.cancelAt {
		p.cancel()
	}
	return p.errs[len(p.calls)-1]
}

func (p *loopPanel) Full(image.Image) error { return p.record("Full") }

func (p *loopPanel) Partial(_ image.Image, rect image.Rectangle) error {
	p.rects = append(p.rects, rect)
	return p.record("Partial")
}

func (p *loopPanel) Sleep() error { return p.record("Sleep") }
func (p *loopPanel) Close() error { return p.record("Close") }

// fetchResult is one scripted ViewSource result.
type fetchResult struct {
	view View
	err  error
}

// loopSource is a scripted ViewSource; the last result repeats.
type loopSource struct {
	results []fetchResult
	calls   []time.Time
}

func (s *loopSource) fetch(_ context.Context, now time.Time) (View, error) {
	r := s.results[min(len(s.calls), len(s.results)-1)]
	s.calls = append(s.calls, now)
	return r.view, r.err
}

func mustView(t *testing.T, now time.Time, lunch ...string) View {
	t.Helper()
	view, err := NewView(nil, nil, nil, []string{"Kaj"}, lunch, nil, nil, now)
	if err != nil {
		t.Fatalf("NewView: %v", err)
	}
	return view
}

// calls concatenates call sequences.
func calls(parts ...[]string) []string {
	var all []string
	for _, p := range parts {
		all = append(all, p...)
	}
	return all
}

// times repeats seq n times.
func times(n int, seq ...string) []string {
	var all []string
	for range n {
		all = append(all, seq...)
	}
	return all
}

var (
	fullSleep    = []string{"Full", "Sleep"}
	partialSleep = []string{"Partial", "Sleep"}
	sleepClose   = []string{"Sleep", "Close"}
)

func TestLoop_Run(t *testing.T) {
	// loopStart is Stockholm 12:00:30, so the ticks fall on 12:01:02,
	// 12:02:02, ...
	loopStart := time.Date(2026, 9, 16, 10, 0, 30, 0, time.UTC)
	// beforeMidnight is Stockholm 23:58:30 (still 16 September in UTC
	// after Stockholm midnight).
	beforeMidnight := time.Date(2026, 9, 16, 21, 58, 30, 0, time.UTC)
	afterMidnight := time.Date(2026, 9, 16, 22, 0, 2, 0, time.UTC)

	viewA := mustView(t, loopStart, "Fisk")
	viewB := mustView(t, loopStart, "Soppa")
	fetchErr := errors.New("calendar down")
	minute := func(start time.Time, n int) time.Time {
		return start.Truncate(time.Minute).Add(time.Duration(n)*time.Minute + tickOffset)
	}

	tests := []struct {
		name      string
		start     time.Time
		settings  LoopSettings
		results   []fetchResult
		ticks     int
		step      time.Duration
		panelErrs map[int]error
		cancelAt  int
		wantCalls []string
		wantFetch []time.Time
	}{
		{
			name:      "first update full, then partial clock updates",
			start:     loopStart,
			settings:  DefaultLoopSettings(),
			results:   []fetchResult{{view: viewA}},
			ticks:     3,
			wantCalls: calls(fullSleep, times(3, partialSleep...), sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "full after FullEvery although only the clock changed",
			start:     loopStart,
			settings:  LoopSettings{FetchEvery: time.Hour, FullEvery: 3 * time.Minute, SleepEveryUpdate: true},
			results:   []fetchResult{{view: viewA}},
			ticks:     5,
			wantCalls: calls(fullSleep, times(3, partialSleep...), fullSleep, partialSleep, sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "refetch with changed content is a full refresh",
			start:     loopStart,
			settings:  LoopSettings{FetchEvery: 2 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true},
			results:   []fetchResult{{view: viewA}, {view: viewB}},
			ticks:     3,
			wantCalls: calls(fullSleep, times(2, partialSleep...), fullSleep, sleepClose),
			wantFetch: []time.Time{loopStart, minute(loopStart, 3)},
		},
		{
			name:      "refetch with identical content is a partial refresh",
			start:     loopStart,
			settings:  LoopSettings{FetchEvery: 2 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true},
			results:   []fetchResult{{view: viewA}, {view: viewA}},
			ticks:     3,
			wantCalls: calls(fullSleep, times(3, partialSleep...), sleepClose),
			wantFetch: []time.Time{loopStart, minute(loopStart, 3)},
		},
		{
			name:     "Stockholm midnight forces a refetch and a full refresh",
			start:    beforeMidnight,
			settings: DefaultLoopSettings(),
			results: []fetchResult{
				{view: mustView(t, beforeMidnight)},
				{view: mustView(t, afterMidnight)},
			},
			ticks:     2,
			wantCalls: calls(fullSleep, partialSleep, fullSleep, sleepClose),
			wantFetch: []time.Time{beforeMidnight, afterMidnight},
		},
		{
			name:     "failing refetch keeps the content and retries after FetchRetryDelay",
			start:    loopStart,
			settings: LoopSettings{FetchEvery: 2 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true},
			results:  []fetchResult{{view: viewA}, {err: fetchErr}, {view: viewA}},
			ticks:    9,
			// Fails at 12:03:02, retried at 12:08:02 (FetchRetryDelay later)
			// and not again by 12:09:02.
			wantCalls: calls(fullSleep, times(9, partialSleep...), sleepClose),
			wantFetch: []time.Time{loopStart, minute(loopStart, 3), minute(loopStart, 8)},
		},
		{
			name:      "failing Partial makes the next refresh full",
			start:     loopStart,
			settings:  DefaultLoopSettings(),
			results:   []fetchResult{{view: viewA}},
			ticks:     3,
			panelErrs: map[int]error{2: errors.New("partial failed")},
			wantCalls: calls(fullSleep, partialSleep, fullSleep, partialSleep, sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "failing first Full is retried as Full at the next tick",
			start:     loopStart,
			settings:  DefaultLoopSettings(),
			results:   []fetchResult{{view: viewA}},
			ticks:     2,
			panelErrs: map[int]error{0: errors.New("full failed")},
			wantCalls: calls(fullSleep, fullSleep, partialSleep, sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "failing Full later is retried as Full",
			start:     loopStart,
			settings:  LoopSettings{FetchEvery: 2 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true},
			results:   []fetchResult{{view: viewA}, {view: viewB}},
			ticks:     4,
			panelErrs: map[int]error{6: errors.New("full failed")},
			wantCalls: calls(fullSleep, times(2, partialSleep...), fullSleep, fullSleep, sleepClose),
			wantFetch: []time.Time{loopStart, minute(loopStart, 3)},
		},
		{
			name:      "no Sleep between updates without SleepEveryUpdate",
			start:     loopStart,
			settings:  LoopSettings{FetchEvery: 30 * time.Minute, FullEvery: time.Hour},
			results:   []fetchResult{{view: viewA}},
			ticks:     2,
			wantCalls: calls([]string{"Full", "Partial", "Partial"}, sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "cancellation between ticks",
			start:     loopStart,
			settings:  DefaultLoopSettings(),
			results:   []fetchResult{{view: viewA}},
			ticks:     100,
			cancelAt:  4, // the Sleep after the first Partial
			wantCalls: calls(fullSleep, partialSleep, sleepClose),
			wantFetch: []time.Time{loopStart},
		},
		{
			name:      "wakeups within the same minute call nothing",
			start:     time.Date(2026, 9, 16, 10, 0, 10, 0, time.UTC),
			settings:  DefaultLoopSettings(),
			results:   []fetchResult{{view: viewA}},
			ticks:     3,
			step:      20 * time.Second, // 12:00:30, 12:00:50, 12:01:10
			wantCalls: calls(fullSleep, partialSleep, sleepClose),
			wantFetch: []time.Time{time.Date(2026, 9, 16, 10, 0, 10, 0, time.UTC)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			clock := &loopClock{now: tt.start, step: tt.step, ticks: tt.ticks, cancel: cancel}
			panel := &loopPanel{errs: tt.panelErrs, cancelAt: tt.cancelAt, cancel: cancel}
			source := &loopSource{results: tt.results}

			err := NewLoop(panel, source.fetch, clock, tt.settings).Run(ctx)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if !reflect.DeepEqual(panel.calls, tt.wantCalls) {
				t.Errorf("panel calls =\n  %v\nwant\n  %v", panel.calls, tt.wantCalls)
			}
			for i, rect := range panel.rects {
				if rect != ClockRect() {
					t.Errorf("Partial %d rect = %v, want ClockRect() %v", i, rect, ClockRect())
				}
			}
			if !reflect.DeepEqual(source.calls, tt.wantFetch) {
				t.Errorf("fetch times =\n  %v\nwant\n  %v", source.calls, tt.wantFetch)
			}
		})
	}
}

func TestLoop_Run_FirstFetchFails(t *testing.T) {
	fetchErr := errors.New("calendar down")
	panel := &loopPanel{}
	source := &loopSource{results: []fetchResult{{err: fetchErr}}}
	clock := &loopClock{now: time.Date(2026, 9, 16, 10, 0, 30, 0, time.UTC), cancel: func() {}}

	err := NewLoop(panel, source.fetch, clock, DefaultLoopSettings()).Run(context.Background())
	if !errors.Is(err, fetchErr) {
		t.Fatalf("Run error = %v, want wrapping %v", err, fetchErr)
	}
	if want := []string{"Close"}; !reflect.DeepEqual(panel.calls, want) {
		t.Errorf("panel calls = %v, want %v", panel.calls, want)
	}
	if len(source.calls) != 1 {
		t.Errorf("fetches = %d, want 1", len(source.calls))
	}
}

func TestLoop_Run_ShutdownErrorsAreLogged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	panel := &loopPanel{errs: map[int]error{
		1: errors.New("sleep failed"),
		4: errors.New("sleep failed"),
		5: errors.New("close failed"),
	}}
	source := &loopSource{results: []fetchResult{{view: mustView(t, time.Date(2026, 9, 16, 10, 0, 30, 0, time.UTC))}}}
	clock := &loopClock{now: time.Date(2026, 9, 16, 10, 0, 30, 0, time.UTC), ticks: 1, cancel: cancel}

	if err := NewLoop(panel, source.fetch, clock, DefaultLoopSettings()).Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := calls(fullSleep, partialSleep, sleepClose); !reflect.DeepEqual(panel.calls, want) {
		t.Errorf("panel calls = %v, want %v", panel.calls, want)
	}
}

func TestUntilNextMinute(t *testing.T) {
	tests := []struct {
		now  time.Time
		want time.Duration
	}{
		{time.Date(2026, 9, 16, 10, 0, 30, 0, time.UTC), 30 * time.Second},
		{time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC), time.Minute},
		{time.Date(2026, 9, 16, 10, 0, 59, 500_000_000, time.UTC), 500 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := untilNextMinute(tt.now); got != tt.want {
			t.Errorf("untilNextMinute(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}
}

func TestDiffPixels(t *testing.T) {
	rect := image.Rect(2, 2, 4, 4)
	base := image.NewGray(image.Rect(0, 0, 6, 6))
	withPixels := func(points ...image.Point) *image.Gray {
		img := image.NewGray(base.Rect)
		for _, p := range points {
			img.Pix[img.PixOffset(p.X, p.Y)] = 255
		}
		return img
	}

	tests := []struct {
		name        string
		prev, next  *image.Gray
		wantOutside int
		wantInside  int
	}{
		{"identical", base, withPixels(), 0, 0},
		{"inside only", base, withPixels(image.Pt(2, 2), image.Pt(3, 3)), 0, 2},
		{"outside only", base, withPixels(image.Pt(0, 0), image.Pt(4, 4)), 2, 0},
		{"both", base, withPixels(image.Pt(3, 2), image.Pt(5, 1)), 1, 1},
		{"nothing shown", nil, withPixels(), 32, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outside, inside := diffPixels(tt.prev, tt.next, rect)
			if outside != tt.wantOutside || inside != tt.wantInside {
				t.Errorf("diffPixels = (%d, %d), want (%d, %d)", outside, inside, tt.wantOutside, tt.wantInside)
			}
		})
	}
}

func TestSystemClock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := (SystemClock{}).Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("Sleep on cancelled ctx = %v, want %v", err, context.Canceled)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Sleep on cancelled ctx took %v", elapsed)
	}
	if err := (SystemClock{}).Sleep(context.Background(), time.Millisecond); err != nil {
		t.Errorf("Sleep(1ms) = %v, want nil", err)
	}
	if now := (SystemClock{}).Now(); time.Until(now).Abs() > time.Minute {
		t.Errorf("Now() = %v, far from time.Now()", now)
	}
}
