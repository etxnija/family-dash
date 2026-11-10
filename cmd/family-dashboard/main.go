// Command family-dashboard is the composition root: it wires configured
// calendar sources to the domain and serves today's events as an HTML
// dashboard.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/eink"
	"family-dashboard/skolmaten"
	"family-dashboard/smhi"
	"family-dashboard/web"
	"family-dashboard/webcal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	configPath := flag.String("config", "", "path to the sources config file (required)")
	addr := flag.String("addr", ":8080", "address to listen on")
	einkPNG := flag.String("eink-png", "", "render the e-paper frame to this PNG file and exit")
	flag.Parse()
	if *configPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("loading sources", "error", err)
		os.Exit(1)
	}

	collector, lunchSource, weatherSource := buildAdapters(cfg)

	if *einkPNG != "" {
		if err := exportEinkPNG(*einkPNG, collector, lunchSource, weatherSource); err != nil {
			slog.Error("writing e-paper frame", "path", *einkPNG, "error", err)
			os.Exit(1)
		}
		slog.Info("wrote e-paper frame", "path", *einkPNG)
		return
	}

	http.HandleFunc("/", handleIndex(collector, lunchSource, weatherSource))

	slog.Info("starting server", "addr", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		slog.Error("serving", "error", err)
		os.Exit(1)
	}
}

// buildAdapters constructs the event collector and the lunch and weather
// sources for cfg. A source that fails to build is logged and skipped;
// lunchSource and weatherSource are nil if no school lunch feed or weather
// location is configured.
func buildAdapters(cfg config.Config) (*dashboard.Collector, dashboard.LunchSource, dashboard.WeatherSource) {
	var namedSources []dashboard.NamedSource
	for _, src := range cfg.Sources {
		source, err := buildSource(src)
		if err != nil {
			slog.Error("building source", "name", src.Name, "type", src.Type, "error", err)
			continue
		}
		namedSources = append(namedSources, dashboard.NamedSource{Owner: src.Name, Source: source})
	}
	collector := dashboard.NewCollector(namedSources)

	var lunchSource dashboard.LunchSource
	if cfg.School.RSSURL == "" {
		slog.Info("no school lunch feed configured")
	} else {
		lunchSource = skolmaten.New(cfg.School.RSSURL, http.DefaultClient)
	}

	var weatherSource dashboard.WeatherSource
	if cfg.Location.Lat == 0 && cfg.Location.Lon == 0 {
		slog.Info("no weather location configured")
	} else {
		weatherSource = smhi.New(cfg.Location.Lat, cfg.Location.Lon, http.DefaultClient)
	}

	return collector, lunchSource, weatherSource
}

// dashboardData is everything fetched for one render of the dashboard,
// shared by the HTML handler and the e-paper export.
type dashboardData struct {
	Today, Homework, Upcoming []dashboard.Event
	Owners, Lunch             []string
	Weather                   *dashboard.WeatherReading
}

// fetchDashboard collects events via collector, the lunch menu via
// lunchSource, and the weather via weatherSource as of now. lunchSource and
// weatherSource may be nil if no school lunch feed or weather location is
// configured; a failing lunch or weather fetch is logged and left empty.
func fetchDashboard(ctx context.Context, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource, now time.Time) dashboardData {
	data := dashboardData{Owners: collector.Owners()}
	data.Today, data.Homework, data.Upcoming = collector.Collect(ctx, now)

	if lunchSource != nil {
		menu, err := lunchSource.Lunch(ctx)
		if err != nil {
			slog.Error("fetching lunch menu", "error", err)
		} else {
			data.Lunch = menu.Dishes
		}
	}

	if weatherSource != nil {
		reading, err := weatherSource.Weather(ctx, now)
		if err != nil {
			slog.Error("fetching weather", "error", err)
		} else {
			data.Weather = &reading
		}
	}

	return data
}

// handleIndex returns an http.HandlerFunc that fetches the dashboard data
// on every request and renders the HTML dashboard. lunchSource and
// weatherSource may be nil if no school lunch feed or weather location is
// configured.
func handleIndex(collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		d := fetchDashboard(r.Context(), collector, lunchSource, weatherSource, now)

		data, err := web.NewPageData(d.Today, d.Homework, d.Upcoming, d.Owners, d.Lunch, d.Weather, now)
		if err != nil {
			slog.Error("building page data", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.Render(w, data); err != nil {
			slog.Error("rendering page", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

// exportEinkPNG fetches the dashboard data once, renders the e-paper frame
// and writes it to path as a PNG.
func exportEinkPNG(path string, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource) error {
	now := time.Now()
	d := fetchDashboard(context.Background(), collector, lunchSource, weatherSource, now)

	view, err := eink.NewView(d.Today, d.Homework, d.Upcoming, d.Owners, d.Lunch, d.Weather, now)
	if err != nil {
		return fmt.Errorf("building e-paper view: %w", err)
	}
	return writePNG(path, eink.Render(view))
}

// writePNG encodes img as a PNG file at path. If encoding fails, the
// partially written file is removed so a reader never sees a truncated PNG.
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		return errors.Join(err, f.Close(), os.Remove(path))
	}
	return f.Close()
}

// buildSource constructs the dashboard.EventSource adapter for cfg.
func buildSource(cfg config.Source) (dashboard.EventSource, error) {
	switch cfg.Type {
	case "sportadmin":
		return webcal.New(cfg.URL, http.DefaultClient), nil
	case "infomentor":
		return webcal.New(cfg.URL, http.DefaultClient), nil
	default:
		return nil, fmt.Errorf("config: unknown source type %q for %q", cfg.Type, cfg.Name)
	}
}
