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
	"os/signal"
	"syscall"
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

	if opts.einkLoop {
		runEinkLoop(opts, collector, lunchSource, weatherSource)
		return
	}
	if opts.einkPNG != "" || opts.einkPanel {
		showEinkOnce(opts, collector, lunchSource, weatherSource)
		return
	}

	http.HandleFunc("/", handleIndex(collector, lunchSource, weatherSource))

	slog.Info("starting server", "addr", opts.addr)
	if err := http.ListenAndServe(opts.addr, nil); err != nil {
		slog.Error("serving", "error", err)
		os.Exit(1)
	}
}

// openPanel returns the panel for -eink-png or -eink-panel. A failed epd.New
// leaves no panel to sleep or close.
func openPanel(opts options) (eink.Panel, error) {
	if opts.einkPNG != "" {
		return pngpanel.New(opts.einkPNG), nil
	}
	panel, err := epd.New(opts.cable)
	if err != nil {
		return nil, err
	}
	return panel, nil
}

// runEinkLoop keeps the -eink-png or -eink-panel output up to date until
// SIGINT or SIGTERM; it exits 1 on failure.
func runEinkLoop(opts options, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	panel, err := openPanel(opts)
	if err != nil {
		slog.Error("opening e-paper panel", "error", err)
		stop()
		os.Exit(1)
	}
	if err := runLoop(ctx, panel, eink.SystemClock{}, collector, lunchSource, weatherSource, opts.loop); err != nil {
		slog.Error("running e-paper loop", "error", err)
		stop()
		os.Exit(1)
	}
	slog.Info("e-paper loop stopped")
}

// showEinkOnce shows one frame on the -eink-png or -eink-panel output; it
// exits 1 on failure.
func showEinkOnce(opts options, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource) {
	panel, err := openPanel(opts)
	if err != nil {
		slog.Error("opening e-paper panel", "error", err)
		os.Exit(1)
	}
	err = showOnce(context.Background(), panel, collector, lunchSource, weatherSource, time.Now())
	switch {
	case err != nil && opts.einkPNG != "":
		slog.Error("writing e-paper frame", "path", opts.einkPNG, "error", err)
		os.Exit(1)
	case err != nil:
		slog.Error("drawing e-paper frame", "error", err)
		os.Exit(1)
	case opts.einkPNG != "":
		slog.Info("wrote e-paper frame", "path", opts.einkPNG)
	default:
		slog.Info("drew e-paper frame")
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
	// einkLoop keeps the -eink-png or -eink-panel output up to date
	// instead of showing one frame.
	einkLoop bool
	// loop is only set with einkLoop; otherwise it stays zero.
	loop eink.LoopSettings
}

// minLoopInterval is the shortest -eink-fetch-every and -eink-full-every
// accepted; the loop ticks once a minute, so anything shorter is moot.
const minLoopInterval = time.Minute

// parseFlags parses args (without the program name). flag package errors and
// usage go to output; validation errors are logged with slog. It returns
// flag.ErrHelp for -h, and an error for a missing -config (ADR-0001),
// -eink-png together with -eink-panel, an invalid -cable with -eink-panel,
// -eink-loop without one of -eink-png or -eink-panel, a loop flag
// (-eink-fetch-every, -eink-full-every, -eink-sleep) without -eink-loop, or
// a loop interval under a minute; -cable is ignored without -eink-panel.
func parseFlags(args []string, output io.Writer) (options, error) {
	fs := flag.NewFlagSet("family-dashboard", flag.ContinueOnError)
	fs.SetOutput(output)
	var opts options
	fs.StringVar(&opts.config, "config", "", "path to the sources config file (required)")
	fs.StringVar(&opts.addr, "addr", ":8080", "address to listen on")
	fs.StringVar(&opts.einkPNG, "eink-png", "", "render the e-paper frame to this PNG file and exit; with -eink-loop, keep it up to date")
	fs.BoolVar(&opts.einkPanel, "eink-panel", false, "draw one frame on the e-paper panel, sleep it and exit; with -eink-loop, keep it up to date")
	cable := fs.String("cable", "left", "side of the portrait frame where the panel cable exits: left|right; only with -eink-panel")
	loop := eink.DefaultLoopSettings()
	fs.BoolVar(&opts.einkLoop, "eink-loop", false, "keep the -eink-png or -eink-panel output up to date until SIGINT or SIGTERM")
	fs.DurationVar(&loop.FetchEvery, "eink-fetch-every", loop.FetchEvery, "how often -eink-loop fetches the dashboard data (at least 1m)")
	fs.DurationVar(&loop.FullEvery, "eink-full-every", loop.FullEvery, "longest time between full refreshes in -eink-loop (at least 1m)")
	fs.BoolVar(&loop.SleepEveryUpdate, "eink-sleep", loop.SleepEveryUpdate, "put the panel to sleep after every -eink-loop update")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

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
	if err := validateLoopFlags(opts, loop, set); err != nil {
		return usageErr(err)
	}
	if opts.einkLoop {
		opts.loop = loop
	}
	return opts, nil
}

// validateLoopFlags checks the loop settings parsed from the flags against
// opts. set holds the names of the flags given.
func validateLoopFlags(opts options, loop eink.LoopSettings, set map[string]bool) error {
	if !opts.einkLoop {
		for _, name := range []string{"eink-fetch-every", "eink-full-every", "eink-sleep"} {
			if set[name] {
				return fmt.Errorf("-%s requires -eink-loop", name)
			}
		}
		return nil
	}
	if opts.einkPNG == "" && !opts.einkPanel {
		return errors.New("-eink-loop requires -eink-png or -eink-panel")
	}
	if loop.FetchEvery < minLoopInterval {
		return fmt.Errorf("-eink-fetch-every %v is under %v", loop.FetchEvery, minLoopInterval)
	}
	if loop.FullEvery < minLoopInterval {
		return fmt.Errorf("-eink-full-every %v is under %v", loop.FullEvery, minLoopInterval)
	}
	return nil
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

// runLoop keeps panel up to date with the dashboard data until ctx is
// cancelled, fetching through fetchDashboard on clock's schedule. The loop
// owns panel and closes it when runLoop returns; it returns nil on
// cancellation and an error if the first view cannot be built.
func runLoop(ctx context.Context, panel eink.Panel, clock eink.Clock, collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource, settings eink.LoopSettings) error {
	views := func(ctx context.Context, now time.Time) (eink.View, error) {
		d := fetchDashboard(ctx, collector, lunchSource, weatherSource, now)
		return eink.NewView(d.Today, d.Homework, d.Upcoming, d.Owners, d.Lunch, d.Weather, now)
	}
	return eink.NewLoop(panel, views, clock, settings).Run(ctx)
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
