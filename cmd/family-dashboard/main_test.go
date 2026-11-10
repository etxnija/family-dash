package main

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/eink"
)

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

func TestWritePNG(t *testing.T) {
	view, err := eink.NewView(nil, nil, nil, nil, nil, nil, time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("eink.NewView() unexpected error: %v", err)
	}
	img := eink.Render(view)
	dir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		img     image.Image // nil means the rendered frame
		wantErr bool
	}{
		{
			name: "writes decodable 480x800 frame",
			path: filepath.Join(dir, "frame.png"),
		},
		{
			name:    "missing directory",
			path:    filepath.Join(dir, "missing", "frame.png"),
			wantErr: true,
		},
		// png.Encode rejects images with zero width or height ("invalid
		// image size"), so this fails after os.Create has made the file.
		{
			name:    "encode failure removes partial file",
			path:    filepath.Join(dir, "empty.png"),
			img:     image.NewGray(image.Rect(0, 0, 0, 0)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in image.Image = img
			if tt.img != nil {
				in = tt.img
			}
			err := writePNG(tt.path, in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("writePNG(%q) = nil error, want error", tt.path)
				}
				if _, statErr := os.Stat(tt.path); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("os.Stat(%q) error = %v, want file not to exist", tt.path, statErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("writePNG(%q) unexpected error: %v", tt.path, err)
			}

			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatalf("opening written PNG: %v", err)
			}
			decoded, err := png.Decode(f)
			if cerr := f.Close(); cerr != nil {
				t.Fatalf("closing written PNG: %v", cerr)
			}
			if err != nil {
				t.Fatalf("png.Decode() unexpected error: %v", err)
			}
			if b := decoded.Bounds(); b.Dx() != 480 || b.Dy() != 800 {
				t.Errorf("decoded bounds = %v, want 480x800", b)
			}
		})
	}
}
