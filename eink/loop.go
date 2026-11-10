package eink

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"time"
)

// FetchRetryDelay is how long the loop waits after a failed fetch before it
// fetches again.
const FetchRetryDelay = 5 * time.Minute

// tickOffset is added to every wait for the next minute, so the shown
// minute is the new one even if the clock wakes up a little early.
const tickOffset = 2 * time.Second

// Refresh kinds, logged as "refresh" for every decision.
const (
	refreshFull    = "full"
	refreshPartial = "partial"
	refreshNone    = "none"
)

// Clock is the loop's source of time. Sleep returns ctx.Err() if ctx is
// cancelled before d has passed, and nil otherwise.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

// SystemClock is the real Clock.
type SystemClock struct{}

// Now returns the current time.
func (SystemClock) Now() time.Time {
	return time.Now()
}

// Sleep waits for d or until ctx is cancelled.
func (SystemClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ViewSource fetches the dashboard data for now and shapes it into a View.
type ViewSource func(ctx context.Context, now time.Time) (View, error)

// LoopSettings tunes the refresh loop.
type LoopSettings struct {
	// FetchEvery is how often the view is fetched again. A new Stockholm
	// day always fetches.
	FetchEvery time.Duration
	// FullEvery is the longest time between full refreshes, to clear the
	// ghosting partial refreshes leave.
	FullEvery time.Duration
	// SleepEveryUpdate puts the panel to sleep after every update.
	SleepEveryUpdate bool
}

// DefaultLoopSettings returns the settings the refresh policy of ADR-0003
// asks for.
func DefaultLoopSettings() LoopSettings {
	return LoopSettings{FetchEvery: 30 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true}
}

// Loop keeps a Panel up to date: it updates the clock every minute,
// fetches a new view every FetchEvery, and picks a full, partial or no
// refresh for each update.
type Loop struct {
	panel    Panel
	views    ViewSource
	clock    Clock
	settings LoopSettings

	loc           *time.Location
	view          View
	shown         *image.Gray // the frame on the panel; nil until a refresh succeeds
	needFull      bool
	lastFull      time.Time
	lastFetch     time.Time
	lastFetchDate string
	fetchFailed   bool
	failedAt      time.Time
}

// NewLoop returns a Loop showing views on panel. The loop owns panel and
// closes it when Run returns.
func NewLoop(panel Panel, views ViewSource, clock Clock, settings LoopSettings) *Loop {
	return &Loop{panel: panel, views: views, clock: clock, settings: settings}
}

// Run shows the first view with a full refresh and then updates the panel
// once a minute until ctx is cancelled. It returns nil on cancellation and
// an error if it has nothing to show (the first fetch fails). The panel is
// closed exactly once on every return.
func (l *Loop) Run(ctx context.Context) error {
	defer func() {
		if err := l.panel.Close(); err != nil {
			slog.Error("closing e-paper panel", "error", err)
		}
	}()

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return err
	}
	l.loc = loc

	now := l.clock.Now()
	view, err := l.views(ctx, now)
	if err != nil {
		return fmt.Errorf("eink: first fetch: %w", err)
	}
	l.fetched(view, now)
	frame := l.frameAt(now)
	l.updateFrame(now, refreshFull, "start", frame, len(frame.Pix))

	for {
		if ctx.Err() != nil {
			l.shutdown()
			return nil
		}
		if err := l.clock.Sleep(ctx, untilNextMinute(l.clock.Now())+tickOffset); err != nil {
			l.shutdown()
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("eink: clock sleep: %w", err)
		}
		l.tick(ctx, l.clock.Now())
	}
}

// tick fetches if due and refreshes the panel as needed for now.
func (l *Loop) tick(ctx context.Context, now time.Time) {
	if l.fetchDue(now) {
		view, err := l.views(ctx, now)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Error("fetching e-paper view", "error", err, "retry_in", FetchRetryDelay)
			l.fetchFailed = true
			l.failedAt = now
		} else {
			l.fetched(view, now)
		}
	}

	kind, reason, frame, changed := l.decide(now)
	l.updateFrame(now, kind, reason, frame, changed)
}

// fetchDue reports whether the view should be fetched again at now.
func (l *Loop) fetchDue(now time.Time) bool {
	if l.fetchFailed && now.Before(l.failedAt.Add(FetchRetryDelay)) {
		return false
	}
	return now.Sub(l.lastFetch) >= l.settings.FetchEvery || l.localDate(now) != l.lastFetchDate
}

// fetched records a successful fetch of view at now.
func (l *Loop) fetched(view View, now time.Time) {
	l.view = view
	l.lastFetch = now
	l.lastFetchDate = l.localDate(now)
	l.fetchFailed = false
}

// localDate returns now's Stockholm calendar date.
func (l *Loop) localDate(now time.Time) string {
	return now.In(l.loc).Format(time.DateOnly)
}

// frameAt renders the current view with the clock set to now.
func (l *Loop) frameAt(now time.Time) *image.Gray {
	view := l.view
	view.Clock = clockText(now, l.loc)
	return Render(view)
}

// decide renders the frame for now and picks how to show it, with the
// reason and the number of pixels that differ from the shown frame.
func (l *Loop) decide(now time.Time) (kind, reason string, frame *image.Gray, changed int) {
	frame = l.frameAt(now)
	outside, inside := diffPixels(l.shown, frame, ClockRect())
	changed = outside + inside
	switch {
	case l.needFull:
		return refreshFull, "retry", frame, changed
	case now.Sub(l.lastFull) >= l.settings.FullEvery:
		return refreshFull, "interval", frame, changed
	case outside > 0:
		return refreshFull, "content changed", frame, changed
	case inside > 0:
		return refreshPartial, "clock", frame, changed
	default:
		return refreshNone, "unchanged", frame, changed
	}
}

// updateFrame logs the decision and shows frame as kind, then puts the
// panel to sleep if the settings ask for it. A failed refresh is logged,
// keeps the shown frame and makes the next refresh a full one.
func (l *Loop) updateFrame(now time.Time, kind, reason string, frame *image.Gray, changed int) {
	slog.Info("e-paper update",
		"refresh", kind,
		"reason", reason,
		"clock", clockText(now, l.loc),
		"changed_pixels", changed,
		"since_full", now.Sub(l.lastFull),
	)

	var err error
	switch kind {
	case refreshFull:
		err = l.panel.Full(frame)
	case refreshPartial:
		err = l.panel.Partial(frame, ClockRect())
	default:
		return
	}

	if err != nil {
		slog.Error("e-paper refresh failed", "refresh", kind, "error", err)
		l.needFull = true
	} else {
		l.shown = frame
		if kind == refreshFull {
			l.needFull = false
			l.lastFull = now
		}
	}

	if l.settings.SleepEveryUpdate {
		l.sleepPanel()
	}
}

// shutdown puts the panel to sleep before Run closes it.
func (l *Loop) shutdown() {
	slog.Info("stopping e-paper loop")
	l.sleepPanel()
}

// sleepPanel puts the panel to sleep, logging any error.
func (l *Loop) sleepPanel() {
	if err := l.panel.Sleep(); err != nil {
		slog.Error("e-paper panel sleep", "error", err)
	}
}

// untilNextMinute returns the time from now to the start of the next
// minute.
func untilNextMinute(now time.Time) time.Duration {
	return now.Truncate(time.Minute).Add(time.Minute).Sub(now)
}

// diffPixels counts the pixels that differ between prev and next, split
// into those outside and inside rect. A nil prev (nothing shown yet)
// differs everywhere.
func diffPixels(prev, next *image.Gray, rect image.Rectangle) (outside, inside int) {
	b := next.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if prev != nil && prev.GrayAt(x, y) == next.GrayAt(x, y) {
				continue
			}
			if (image.Point{X: x, Y: y}).In(rect) {
				inside++
			} else {
				outside++
			}
		}
	}
	return outside, inside
}
