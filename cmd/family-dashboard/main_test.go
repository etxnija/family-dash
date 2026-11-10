package main

import (
	"context"
	"errors"
	"flag"
	"image"
	"io"
	"path/filepath"
	"slices"
	"testing"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/eink"
	"family-dashboard/eink/epd"
	"family-dashboard/eink/pngpanel"
	"family-dashboard/eink/pngpanel/pngpaneltest"
)

// parseFlagsCase is one successful parseFlags call.
type parseFlagsCase struct {
	name string
	args []string
	want options
}

func runParseFlagsCases(t *testing.T, tests []parseFlagsCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlags(tt.args, io.Discard)
			if err != nil {
				t.Fatalf("parseFlags(%q) unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("parseFlags(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	runParseFlagsCases(t, []parseFlagsCase{
		{
			name: "only config",
			args: []string{"-config", "c.json"},
			want: options{config: "c.json", addr: ":8080", cable: epd.CableLeft},
		},
		{
			name: "addr and png",
			args: []string{"-config", "c.json", "-addr", ":9090", "-eink-png", "f.png"},
			want: options{config: "c.json", addr: ":9090", einkPNG: "f.png", cable: epd.CableLeft},
		},
		{
			name: "panel default cable is left",
			args: []string{"-config", "c.json", "-eink-panel"},
			want: options{config: "c.json", addr: ":8080", einkPanel: true, cable: epd.CableLeft},
		},
		{
			name: "panel cable left",
			args: []string{"-config", "c.json", "-eink-panel", "-cable", "left"},
			want: options{config: "c.json", addr: ":8080", einkPanel: true, cable: epd.CableLeft},
		},
		{
			name: "panel cable right",
			args: []string{"-config", "c.json", "-eink-panel", "-cable", "right"},
			want: options{config: "c.json", addr: ":8080", einkPanel: true, cable: epd.CableRight},
		},
		{
			// -cable is ignored without -eink-panel, even if invalid.
			name: "invalid cable without panel is ignored",
			args: []string{"-config", "c.json", "-cable", "up"},
			want: options{config: "c.json", addr: ":8080", cable: epd.CableLeft},
		},
	})
}

func TestParseFlags_Loop(t *testing.T) {
	runParseFlagsCases(t, []parseFlagsCase{
		{
			name: "loop with png uses default loop settings",
			args: []string{"-config", "c.json", "-eink-loop", "-eink-png", "p"},
			want: options{config: "c.json", addr: ":8080", einkPNG: "p", cable: epd.CableLeft,
				einkLoop: true, loop: eink.LoopSettings{FetchEvery: 30 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true}},
		},
		{
			name: "loop with panel and cable right",
			args: []string{"-config", "c.json", "-eink-loop", "-eink-panel", "-cable", "right"},
			want: options{config: "c.json", addr: ":8080", einkPanel: true, cable: epd.CableRight,
				einkLoop: true, loop: eink.LoopSettings{FetchEvery: 30 * time.Minute, FullEvery: time.Hour, SleepEveryUpdate: true}},
		},
		{
			name: "all loop flags",
			args: []string{"-config", "c.json", "-eink-loop", "-eink-panel",
				"-eink-fetch-every", "1m", "-eink-full-every", "2h30m", "-eink-sleep"},
			want: options{config: "c.json", addr: ":8080", einkPanel: true, cable: epd.CableLeft,
				einkLoop: true, loop: eink.LoopSettings{FetchEvery: time.Minute, FullEvery: 150 * time.Minute, SleepEveryUpdate: true}},
		},
		{
			name: "loop without sleep",
			args: []string{"-config", "c.json", "-eink-loop", "-eink-png", "p", "-eink-sleep=false"},
			want: options{config: "c.json", addr: ":8080", einkPNG: "p", cable: epd.CableLeft,
				einkLoop: true, loop: eink.LoopSettings{FetchEvery: 30 * time.Minute, FullEvery: time.Hour}},
		},
	})
}

func TestParseFlags_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing config", args: []string{"-eink-panel"}},
		{name: "both eink flags", args: []string{"-config", "c.json", "-eink-png", "f.png", "-eink-panel"}},
		{name: "panel invalid cable", args: []string{"-config", "c.json", "-eink-panel", "-cable", "up"}},
		{name: "unknown flag", args: []string{"-config", "c.json", "-bogus"}},
		{name: "loop alone", args: []string{"-config", "c.json", "-eink-loop"}},
		{name: "loop with both eink flags", args: []string{"-config", "c.json", "-eink-loop", "-eink-png", "f.png", "-eink-panel"}},
		{name: "fetch-every without loop", args: []string{"-config", "c.json", "-eink-png", "f.png", "-eink-fetch-every", "30m"}},
		{name: "full-every without loop", args: []string{"-config", "c.json", "-eink-panel", "-eink-full-every", "1h"}},
		{name: "sleep without loop", args: []string{"-config", "c.json", "-eink-panel", "-eink-sleep=false"}},
		{name: "fetch-every under a minute", args: []string{"-config", "c.json", "-eink-loop", "-eink-png", "f.png", "-eink-fetch-every", "30s"}},
		{name: "full-every under a minute", args: []string{"-config", "c.json", "-eink-loop", "-eink-png", "f.png", "-eink-full-every", "59s"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := parseFlags(tt.args, io.Discard); err == nil {
				t.Fatalf("parseFlags(%q) = %+v, nil error, want error", tt.args, got)
			}
		})
	}
}

func TestParseFlags_Help(t *testing.T) {
	if _, err := parseFlags([]string{"-h"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h) error = %v, want flag.ErrHelp", err)
	}
}

func TestBuildSource(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Source
		wantErr bool
	}{
		{
			name: "known type sportadmin",
			cfg:  config.Source{Name: "alice", Type: "sportadmin", URL: "https://example.com/feed.ics"},
		},
		{
			name: "known type infomentor",
			cfg:  config.Source{Name: "alice", Type: "infomentor", URL: "https://example.com/feed.ics"},
		},
		{
			name:    "unknown type",
			cfg:     config.Source{Name: "bob", Type: "carrier-pigeon", URL: "https://example.com/feed.ics"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := buildSource(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildSource(%+v) = nil error, want error", tt.cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildSource(%+v) unexpected error: %v", tt.cfg, err)
			}
			if source == nil {
				t.Fatalf("buildSource(%+v) = nil source, want non-nil", tt.cfg)
			}
		})
	}
}

func TestBuildAdapters(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.Config
		wantOwners  []string
		wantLunch   bool
		wantWeather bool
	}{
		{
			name: "everything configured, unknown source skipped",
			cfg: config.Config{
				Sources: []config.Source{
					{Name: "alice", Type: "sportadmin", URL: "https://example.com/a.ics"},
					{Name: "bob", Type: "carrier-pigeon", URL: "https://example.com/b.ics"},
				},
				School:   config.School{RSSURL: "https://example.com/lunch.rss"},
				Location: config.Location{Lat: 59.3, Lon: 18.0},
			},
			wantOwners:  []string{"alice"},
			wantLunch:   true,
			wantWeather: true,
		},
		{
			name: "no lunch feed or weather location",
			cfg:  config.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, lunchSource, weatherSource := buildAdapters(tt.cfg)
			if got := collector.Owners(); !slices.Equal(got, tt.wantOwners) {
				t.Errorf("Owners() = %v, want %v", got, tt.wantOwners)
			}
			if (lunchSource != nil) != tt.wantLunch {
				t.Errorf("lunchSource = %v, want non-nil %v", lunchSource, tt.wantLunch)
			}
			if (weatherSource != nil) != tt.wantWeather {
				t.Errorf("weatherSource = %v, want non-nil %v", weatherSource, tt.wantWeather)
			}
		})
	}
}

type stubLunchSource struct {
	menu dashboard.LunchMenu
	err  error
}

func (s stubLunchSource) Lunch(context.Context) (dashboard.LunchMenu, error) {
	return s.menu, s.err
}

type stubWeatherSource struct {
	reading dashboard.WeatherReading
	err     error
}

func (s stubWeatherSource) Weather(context.Context, time.Time) (dashboard.WeatherReading, error) {
	return s.reading, s.err
}

func TestFetchDashboard(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	fetchErr := errors.New("upstream down")

	tests := []struct {
		name        string
		lunch       dashboard.LunchSource
		weather     dashboard.WeatherSource
		wantLunch   []string
		wantWeather *dashboard.WeatherReading
	}{
		{
			name: "no lunch or weather source configured",
		},
		{
			name:    "failing sources leave lunch and weather empty",
			lunch:   stubLunchSource{err: fetchErr},
			weather: stubWeatherSource{err: fetchErr},
		},
		{
			name:        "successful sources fill lunch and weather",
			lunch:       stubLunchSource{menu: dashboard.LunchMenu{Dishes: []string{"Fiskgratäng", "Linsgryta"}}},
			weather:     stubWeatherSource{reading: dashboard.WeatherReading{Temperature: 4.5, LowTemperature: 1, HighTemperature: 6}},
			wantLunch:   []string{"Fiskgratäng", "Linsgryta"},
			wantWeather: &dashboard.WeatherReading{Temperature: 4.5, LowTemperature: 1, HighTemperature: 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fetchDashboard(context.Background(), dashboard.NewCollector(nil), tt.lunch, tt.weather, now)

			if len(got.Today) != 0 || len(got.Homework) != 0 || len(got.Upcoming) != 0 || len(got.Owners) != 0 {
				t.Errorf("fetchDashboard() events/owners = %+v, want empty", got)
			}
			if len(got.Lunch) != len(tt.wantLunch) {
				t.Fatalf("fetchDashboard() Lunch = %v, want %v", got.Lunch, tt.wantLunch)
			}
			for i := range tt.wantLunch {
				if got.Lunch[i] != tt.wantLunch[i] {
					t.Errorf("fetchDashboard() Lunch[%d] = %q, want %q", i, got.Lunch[i], tt.wantLunch[i])
				}
			}
			switch {
			case tt.wantWeather == nil && got.Weather != nil:
				t.Errorf("fetchDashboard() Weather = %+v, want nil", *got.Weather)
			case tt.wantWeather != nil && got.Weather == nil:
				t.Errorf("fetchDashboard() Weather = nil, want %+v", *tt.wantWeather)
			case tt.wantWeather != nil && *got.Weather != *tt.wantWeather:
				t.Errorf("fetchDashboard() Weather = %+v, want %+v", *got.Weather, *tt.wantWeather)
			}
		})
	}
}

func TestShowOnce_PNGPanel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)

	if err := showOnce(context.Background(), pngpanel.New(path), dashboard.NewCollector(nil), nil, nil, now); err != nil {
		t.Fatalf("showOnce() unexpected error: %v", err)
	}
	pngpaneltest.AssertFrame(t, path)
}

// fakePanel records the calls made to it; fullErr is returned from Full,
// and onFull (if set) is called on every Full.
type fakePanel struct {
	calls   []string
	frames  []image.Image
	rects   []image.Rectangle
	fullErr error
	onFull  func()
}

func (p *fakePanel) Full(frame image.Image) error {
	p.calls = append(p.calls, "Full")
	p.frames = append(p.frames, frame)
	if p.onFull != nil {
		p.onFull()
	}
	return p.fullErr
}

func (p *fakePanel) Partial(frame image.Image, rect image.Rectangle) error {
	p.calls = append(p.calls, "Partial")
	p.frames = append(p.frames, frame)
	p.rects = append(p.rects, rect)
	return nil
}

func (p *fakePanel) Sleep() error {
	p.calls = append(p.calls, "Sleep")
	return nil
}

func (p *fakePanel) Close() error {
	p.calls = append(p.calls, "Close")
	return nil
}

func TestShowOnce_PanelCalls(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	fullErr := errors.New("panel busy")

	tests := []struct {
		name    string
		fullErr error
	}{
		{name: "successful refresh"},
		{name: "failing refresh still sleeps and closes", fullErr: fullErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := &fakePanel{fullErr: tt.fullErr}
			err := showOnce(context.Background(), panel, dashboard.NewCollector(nil), nil, nil, now)

			if tt.fullErr != nil {
				if !errors.Is(err, tt.fullErr) {
					t.Errorf("showOnce() error = %v, want it to wrap %v", err, tt.fullErr)
				}
			} else if err != nil {
				t.Errorf("showOnce() unexpected error: %v", err)
			}
			checkOneFullFrame(t, panel, []string{"Full", "Sleep", "Close"})
		})
	}
}

func TestRunLoop(t *testing.T) {
	tests := []struct {
		name      string
		sleep     bool
		wantCalls []string
	}{
		// The loop sleeps the panel after the first update and again on
		// shutdown.
		{name: "sleep after every update", sleep: true, wantCalls: []string{"Full", "Sleep", "Sleep", "Close"}},
		{name: "sleep only on shutdown", sleep: false, wantCalls: []string{"Full", "Sleep", "Close"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Cancelling in Full stops the loop before its first tick, so
			// SystemClock never really sleeps.
			panel := &fakePanel{onFull: cancel}
			settings := eink.DefaultLoopSettings()
			settings.SleepEveryUpdate = tt.sleep

			if err := runLoop(ctx, panel, eink.SystemClock{}, dashboard.NewCollector(nil), nil, nil, settings); err != nil {
				t.Fatalf("runLoop() unexpected error: %v", err)
			}
			checkOneFullFrame(t, panel, tt.wantCalls)
		})
	}
}

// checkOneFullFrame checks that panel got wantCalls and exactly one
// full-screen frame.
func checkOneFullFrame(t *testing.T, panel *fakePanel, wantCalls []string) {
	t.Helper()
	if !slices.Equal(panel.calls, wantCalls) {
		t.Errorf("panel calls = %v, want %v", panel.calls, wantCalls)
	}
	if len(panel.frames) != 1 {
		t.Fatalf("Full() called with %d frames, want 1", len(panel.frames))
	}
	if b := panel.frames[0].Bounds(); b.Dx() != eink.ScreenWidth || b.Dy() != eink.ScreenHeight {
		t.Errorf("Full() frame bounds = %v, want %dx%d", b, eink.ScreenWidth, eink.ScreenHeight)
	}
}
