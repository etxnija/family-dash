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

	sources, err := config.Load(*configPath)
	if err != nil {
		slog.Error("loading sources", "error", err)
		os.Exit(1)
	}

	http.HandleFunc("/", handleIndex(sources))

	slog.Info("starting server", "addr", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		slog.Error("serving", "error", err)
		os.Exit(1)
	}
}

// ownerNames returns the distinct cfg.Name values from sources, in
// config-file order.
func ownerNames(sources []config.Source) []string {
	seen := make(map[string]bool, len(sources))
	names := make([]string, 0, len(sources))
	for _, cfg := range sources {
		if seen[cfg.Name] {
			continue
		}
		seen[cfg.Name] = true
		names = append(names, cfg.Name)
	}
	return names
}

// handleIndex returns an http.HandlerFunc that fetches events from sources
// on every request and renders the HTML dashboard.
func handleIndex(sources []config.Source) http.HandlerFunc {
	owners := ownerNames(sources)

	return func(w http.ResponseWriter, r *http.Request) {
		var events []dashboard.Event
		for _, cfg := range sources {
			source, err := buildSource(cfg)
			if err != nil {
				slog.Error("building source", "name", cfg.Name, "type", cfg.Type, "error", err)
				continue
			}

			fetched, err := source.Events(r.Context())
			if err != nil {
				slog.Error("fetching events", "source", cfg.Name, "error", err)
				continue
			}
			slog.Info("fetched events", "source", cfg.Name, "count", len(fetched))

			events = append(events, dashboard.WithOwner(fetched, cfg.Name)...)
		}

		today := dashboard.TodayEvents(events, time.Now())
		slog.Info("filtered to today", "count", len(today))

		data, err := web.NewPageData(today, owners, time.Now())
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
