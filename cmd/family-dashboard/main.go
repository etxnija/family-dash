// Command family-dashboard is the composition root: it wires configured
// calendar sources to the domain and serves today's events as an HTML
// dashboard.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	dashboard "family-dashboard"
	"family-dashboard/config"
	"family-dashboard/eink"
	"family-dashboard/eink/epd"
	"family-dashboard/eink/pngpanel"
	"family-dashboard/skolmaten"
	"family-dashboard/smhi"
	"family-dashboard/web"
	"family-dashboard/webcal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	opts, err := parseFlags(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(2)
	}

	cfg, err := config.Load(opts.config)
	if err != nil {
		slog.Error("loading sources", "error", err)
		os.Exit(1)
	}

	collector, lunchSource, weatherSource := buildAdapters(cfg)

	if opts.einkPNG != "" {
		panel := pngpanel.New(opts.einkPNG)
		if err := showOnce(context.Background(), panel, collector, lunchSource, weatherSource, time.Now()); err != nil {
			slog.Error("writing e-paper frame", "path", opts.einkPNG, "error", err)
			os.Exit(1)
		}
		slog.Info("wrote e-paper frame", "path", opts.einkPNG)
		return
	}

	if opts.einkPanel {
		// A failed New leaves no panel to sleep or close.
		panel, err := epd.New(opts.cable)
		if err != nil {
			slog.Error("opening e-paper panel", "error", err)
			os.Exit(1)
		}
		if err := showOnce(context.Background(), panel, collector, lunchSource, weatherSource, time.Now()); err != nil {
			slog.Error("drawing e-paper frame", "error", err)
			os.Exit(1)
		}
		slog.Info("drew e-paper frame")
		return
	}

	http.HandleFunc("/", handleIndex(collector, lunchSource, weatherSource))

	slog.Info("starting server", "addr", opts.addr)
	if err := http.ListenAndServe(opts.addr, nil); err != nil {
		slog.Error("serving", "error", err)
		os.Exit(1)
	}
}

// options are the parsed command-line flags.
type options struct {
	config    string
	addr      string
	einkPNG   string
	einkPanel bool
	// cable is only parsed with einkPanel; otherwise it stays CableLeft.
	cable epd.Cable
}

// parseFlags parses args (without the program name). flag package errors and
// usage go to output; validation errors are logged with slog. It returns
// flag.ErrHelp for -h, and an error for a missing -config (ADR-0001),
// -eink-png together with -eink-panel, or an invalid -cable with
// -eink-panel; -cable is ignored without -eink-panel.
func parseFlags(args []string, output io.Writer) (options, error) {
	fs := flag.NewFlagSet("family-dashboard", flag.ContinueOnError)
	fs.SetOutput(output)
	var opts options
	fs.StringVar(&opts.config, "config", "", "path to the sources config file (required)")
	fs.StringVar(&opts.addr, "addr", ":8080", "address to listen on")
	fs.StringVar(&opts.einkPNG, "eink-png", "", "render the e-paper frame to this PNG file and exit")
	fs.BoolVar(&opts.einkPanel, "eink-panel", false, "draw one frame on the e-paper panel, sleep it and exit")
	cable := fs.String("cable", "left", "side of the portrait frame where the panel cable exits: left|right; only with -eink-panel")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	usageErr := func(err error) (options, error) {
		slog.Error("invalid flags", "error", err)
		fs.Usage()
		return options{}, err
	}
	if opts.config == "" {
		return usageErr(errors.New("-config is required"))
	}
	if opts.einkPNG != "" && opts.einkPanel {
		return usageErr(errors.New("-eink-png and -eink-panel are mutually exclusive"))
	}
	if opts.einkPanel {
		c, err := epd.ParseCable(*cable)
		if err != nil {
			return usageErr(fmt.Errorf("-cable: %w", err))
		}
		opts.cable = c
	}
	return opts, nil
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

// showOnce fetches the dashboard data as of now, renders the e-paper frame
// and shows it on panel with a full refresh. It always calls panel.Sleep
// and panel.Close afterwards, even if building the view or the refresh
// failed, and returns every error joined.
func showOnce(ctx context.Context, panel eink.Panel, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource, now time.Time) error {
	var errs []error
	d := fetchDashboard(ctx, collector, lunchSource, weatherSource, now)
	view, err := eink.NewView(d.Today, d.Homework, d.Upcoming, d.Owners, d.Lunch, d.Weather, now)
	if err != nil {
		errs = append(errs, fmt.Errorf("building e-paper view: %w", err))
	} else if err := panel.Full(eink.Render(view)); err != nil {
		errs = append(errs, fmt.Errorf("full refresh: %w", err))
	}
	if err := panel.Sleep(); err != nil {
		errs = append(errs, fmt.Errorf("sleeping panel: %w", err))
	}
	if err := panel.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing panel: %w", err))
	}
	return errors.Join(errs...)
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
