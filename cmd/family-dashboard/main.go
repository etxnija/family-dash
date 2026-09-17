// Command family-dashboard is the composition root: it wires the SportAdmin
// adapter to the domain and prints today's events.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/sportadmin"
)

func main() {
	icsURL := os.Getenv("SPORTADMIN_ICS_URL")
	if icsURL == "" {
		fmt.Fprintln(os.Stderr, "SPORTADMIN_ICS_URL is not set")
		os.Exit(1)
	}

	source := sportadmin.New(icsURL, http.DefaultClient)

	events, err := source.Events(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetching events:", err)
		os.Exit(1)
	}

	for _, e := range dashboard.TodayEvents(events, time.Now()) {
		fmt.Printf("%s-%s %s @ %s\n", e.Start.Format("15:04"), e.End.Format("15:04"), e.Summary, e.Location)
	}
}
