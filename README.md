# Family Dashboard

A shared, always-visible information display for our household: two parents, two kids (11 and 13). The goal is a single
screen that answers "what's happening today?" — sports practice, family calendar events, school times, homework due,
weather — without anyone having to open four different apps or ask around out loud. Success isn't a metric, it's
simpler: fewer "what's happening today?" questions asked in the house once this is actually running on a screen someone
looks at.

## Inspiration

Sparked by
[this nyteknik.se piece](https://www.nyteknik.se/kroenikor/1-500-kronor-och-en-raspberry-pi-senare-nu-har-familjen-en-e-ink-dashboard/4487700)
on a DIY Raspberry Pi + e-ink family dashboard. The e-ink/Pi hardware build is a real future direction (see Roadmap
below) but deferred — this project starts with the data and a browser page on an iPad already at home, and is
architected so the display layer (browser now, e-ink later) and the hosting location (dev machine now, Raspberry Pi
later) can both change without rewriting the data layer.

The full product requirements (goal, success criteria, phased rollout, open questions) live in the PRD, not here — see
`~/Documents/nils/PRD/Family
dashboard.md`.

## Structure

Hexagonal architecture, kept deliberately minimal — domain in the root package, one package per adapter, composition
only in `cmd/`:

```
event.go                       domain: Event type, EventSource port, TodayEvents()
event_test.go                  domain tests

webcal/
  sportadmin.go                adapter: ICS/webcal feed → []dashboard.Event
  sportadmin_test.go           adapter tests (fixture-based, no network)
  testdata/sample.ics          sample ICS fixture used by the adapter tests

eink/
  view.go                      adapter: e-paper view-model (Swedish, Stockholm time)
  layout.go                    every layout constant of the portrait 480x800 frame
  render.go                    1-bit renderer: View → *image.Gray (Go Mono fonts)
  panel.go                     Panel port: Full/Sleep/Close of a portrait frame
  view_test.go, render_test.go adapter tests
  pngpanel/                    Panel adapter: writes each frame to a PNG file
  epd/                         Panel adapter for the Waveshare 7.5" V2 panel
    rotate.go                  pure: -cable parsing, portrait → native 800x480 rotation
    epd.go                     linux && arm only: thin wrapper over the epd7in5v2 driver
    epd_other.go               stub elsewhere: every call returns "only supported on linux/arm"

cmd/
  family-dashboard/main.go     composition root: wires the webcal adapter into
                                the domain, prints today's events to stdout;
                                `-config <file> -eink-png frame.png` writes one
                                e-paper frame as PNG and exits;
                                `-config <file> -eink-panel -cable left|right`
                                draws one frame on the panel, sleeps it and exits

AGENTS.md                      project conventions for AI agents working here
CLAUDE.md                      pointer to AGENTS.md + memory/status.md
memory/
  status.md                    chronological work log (append-only)
  concepts/                    durable, tagged notes on patterns/decisions/components
                                (see concepts/index.md)
docs/arch/adr/                 architecture decision records (one per decision, once
                                any exist)
sensors/
  lint.sh                      golangci-lint run ./...
  test.sh                      go test ./... -race -count=1
mise.toml                      pinned toolchain versions (go, golangci-lint)
```

`event.go`'s domain package owns the `EventSource` port; `webcal` is the adapter implementing it (SportAdmin is the
first concrete feed, more can be added as their own files in the same package — see `memory/concepts/webcal-adapter.md`
for why). `cmd/family-dashboard` is the only place that imports both and wires them together — that's the dependency
rule this project follows: adapters depend on the domain, never the reverse.

This project is built using the ["agent-work"](../agent-work) coding harness — `AGENTS.md` and `memory/status.md` are
its convention/history files, not optional documentation.

## Running the app

Requires [mise](https://mise.jdx.dev/) for the pinned Go toolchain:

```bash
mise install          # installs the Go version pinned in mise.toml
```

The app needs your SportAdmin calendar subscription URL as an environment variable — it's a secret (the link embeds a
private token) and must never be committed or logged:

```bash
export SPORTADMIN_ICS_URL="https://portalweb.sportadmin.se/webcal?id=..."
go run ./cmd/family-dashboard
```

This fetches the feed, filters to today's events (Europe/Stockholm time), and prints each one to stdout — there's no web
page yet, see Roadmap.

## Running tests

```bash
sensors/test.sh    # go test ./... -race -count=1
sensors/lint.sh    # golangci-lint run ./...
```

The `webcal` adapter's tests run against a checked-in fixture (`webcal/testdata/sample.ics`), not the live network — no
real feed URL or credentials are needed to run the test suite.

## Roadmap

Per the PRD's phased rollout:

- **Phase 1 (in progress):** data sources + the web page, running locally. SportAdmin (sports calendar) is done; Google
  Calendar (family events) and weather are next, then the actual browser page.
- **Phase 2:** relocate the working Phase 1 backend + page to a Raspberry Pi for always-on home hosting.
- **Phase 3:** school data source (InfoMentor), once its integration reliability/credential tradeoffs are settled.
- **Phase 4 (future, unscheduled):** swap the browser page for an e-ink display — the original inspiration, deferred for
  now.
  The `eink/` 1-bit renderer, its `-eink-png` PNG preview and a one-shot `-eink-panel` mode (draw one frame on the
  Waveshare panel, sleep it, exit) exist; the refresh loop with partial clock refreshes comes next. Build the Pi binary
  with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o family-dashboard-pi ./cmd/family-dashboard`.
