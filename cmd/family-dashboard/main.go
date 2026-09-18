// Command family-dashboard is the composition root: it wires configured
// calendar sources to the domain and prints today's events.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/webcal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	configPath := flag.String("config", "", "path to the sources config file (required)")
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

	ctx := context.Background()
	var events []dashboard.Event
	for _, cfg := range sources {
		source, err := buildSource(cfg)
		if err != nil {
			slog.Error("building source", "name", cfg.Name, "type", cfg.Type, "error", err)
			continue
		}

		fetched, err := source.Events(ctx)
		if err != nil {
			slog.Error("fetching events", "source", cfg.Name, "error", err)
			continue
		}
		slog.Info("fetched events", "source", cfg.Name, "count", len(fetched))

		events = append(events, dashboard.WithOwner(fetched, cfg.Name)...)
	}

	today := dashboard.TodayEvents(events, time.Now())
	slog.Info("filtered to today", "count", len(today))

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		slog.Error("loading timezone", "error", err)
		os.Exit(1)
	}

	for _, e := range today {
		fmt.Printf("%s: %s-%s %s @ %s\n", e.Owner, e.Start.In(loc).Format("15:04"), e.End.In(loc).Format("15:04"), e.Summary, e.Location)
	}
}

// buildSource constructs the dashboard.EventSource adapter for cfg.
func buildSource(cfg config.Source) (dashboard.EventSource, error) {
	switch cfg.Type {
	case "sportadmin":
		return webcal.New(cfg.URL, http.DefaultClient), nil
	default:
		return nil, fmt.Errorf("config: unknown source type %q for %q", cfg.Type, cfg.Name)
	}
}
