// Command family-dashboard is the composition root: it wires configured
// calendar sources to the domain and serves today's events as an HTML
// dashboard.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/skolmaten"
	"family-dashboard/smhi"
	"family-dashboard/web"
	"family-dashboard/webcal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	configPath := flag.String("config", "", "path to the sources config file (required)")
	addr := flag.String("addr", ":8080", "address to listen on")
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

	http.HandleFunc("/", handleIndex(collector, lunchSource, weatherSource))

	slog.Info("starting server", "addr", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		slog.Error("serving", "error", err)
		os.Exit(1)
	}
}

// handleIndex returns an http.HandlerFunc that fetches events via collector,
// the lunch menu via lunchSource, and the weather via weatherSource on every
// request and renders the HTML dashboard. lunchSource and weatherSource may
// be nil if no school lunch feed or weather location is configured.
func handleIndex(collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		today, homework, upcoming := collector.Collect(r.Context(), time.Now())

		var dishes []string
		if lunchSource != nil {
			menu, err := lunchSource.Lunch(r.Context())
			if err != nil {
				slog.Error("fetching lunch menu", "error", err)
			} else {
				dishes = menu.Dishes
			}
		}

		var weather *dashboard.WeatherReading
		if weatherSource != nil {
			reading, err := weatherSource.Weather(r.Context(), time.Now())
			if err != nil {
				slog.Error("fetching weather", "error", err)
			} else {
				weather = &reading
			}
		}

		data, err := web.NewPageData(today, homework, upcoming, collector.Owners(), dishes, weather, time.Now())
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
