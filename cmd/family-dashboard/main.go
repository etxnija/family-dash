// Command family-dashboard is the composition root: it wires the webcal
// adapter to the domain and prints today's events.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/webcal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	icsURL := os.Getenv("SPORTADMIN_ICS_URL")
	if icsURL == "" {
		slog.Error("SPORTADMIN_ICS_URL is not set")
		os.Exit(1)
	}

	source := webcal.New(icsURL, http.DefaultClient)

	events, err := source.Events(context.Background())
	if err != nil {
		slog.Error("fetching events", "error", err)
		os.Exit(1)
	}
	slog.Info("fetched events", "count", len(events))

	today := dashboard.TodayEvents(events, time.Now())
	slog.Info("filtered to today", "count", len(today))

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		slog.Error("loading timezone", "error", err)
		os.Exit(1)
	}

	for _, e := range today {
		fmt.Printf("%s-%s %s @ %s\n", e.Start.In(loc).Format("15:04"), e.End.In(loc).Format("15:04"), e.Summary, e.Location)
	}
}
