# Family Dashboard

A shared, always-visible information display for our household: two parents, two kids (11 and 13). The goal is a single
screen that answers "what's happening today?" — sports practice, family calendar events, school lunch, homework due,
weather — without anyone having to open four different apps or ask around out loud. Success isn't a metric, it's
simpler: fewer "what's happening today?" questions asked in the house now that it is running on a screen someone looks
at.

## Status

**Live at home (November 2026).** A Raspberry Pi Zero W with a Waveshare 7.5" e-paper panel sits in a picture frame,
shows the dashboard in portrait, starts by itself at power-on and is updated from the dev machine with `make deploy`.
The same data also renders as a web page (`:8080`).

What the screen shows: date, week number and a large clock (refreshed every minute), the weather from SMHI, today's
school lunch (Skolmaten), today's events, homework and tests ("Läxor och prov"), and the coming days. Entries carry the
owner's initial. If a calendar feed fails, its last good entries stay for 24 hours and a line at the bottom says which
calendar is missing or old.

## Inspiration

Sparked by
[this nyteknik.se piece](https://www.nyteknik.se/kroenikor/1-500-kronor-och-en-raspberry-pi-senare-nu-har-familjen-en-e-ink-dashboard/4487700)
on a DIY Raspberry Pi + e-ink family dashboard. The project started with the data and a browser page, and is architected
so the display layer (browser, e-paper) and the hosting location (dev machine, Raspberry Pi) can change without
rewriting the data layer (ADR-0002).

The product requirements (goal, success criteria, phased rollout, open questions) live in the PRD, not here — see
`~/Documents/nils/PRD/Family dashboard.md`.

## Structure

Hexagonal architecture, kept deliberately minimal — domain in the root package, one package per adapter, composition
only in `cmd/`:

```
event.go, collector.go,        domain: Event, EventSource port, the Collector (merges sources, derives the
feed.go, lunch.go,             today/homework/upcoming views, keeps each source's last good events for 24 h),
weather.go                     FeedProblem, and the lunch and weather ports (+ *_test.go)

webcal/                        adapter: ICS/webcal feeds (SportAdmin, InfoMentor) → []dashboard.Event
skolmaten/                     adapter: school lunch RSS → dashboard.LunchSource
smhi/                          adapter: SMHI forecast → dashboard.WeatherSource
config/                        the JSON config file (sources, school, location)
web/                           adapter: the HTML dashboard page

eink/                          adapter: the e-paper display
  view.go, layout.go, render.go  view-model (Swedish), layout constants, 1-bit renderer (portrait 480x800)
  panel.go, loop.go              Panel port; refresh loop (full / partial clock / no refresh, every minute)
  pngpanel/                      Panel adapter: writes each frame to a PNG file (for the dev machine)
  epd/                           Panel adapter for the Waveshare 7.5" V2 panel (rotates to the native 800x480)

cmd/family-dashboard/main.go   composition root: flags, wiring, web server, e-paper one-shot and loop

Makefile                       build, deploy and service control for the Pi (`make help`)
deploy/                        systemd unit template for the Pi
docs/
  raspberry-pi-setup.md        Pi setup, driver notes, Wi-Fi, deploy and service (verified steps)
  arch/adr/                    architecture decision records
  mockups/                     design mockups
memory/                        project memory: status.md (log) and concepts/ (durable notes, see concepts/index.md)
sensors/                       lint.sh and test.sh, run by the agent harness after every task
AGENTS.md, CLAUDE.md           conventions for AI agents working here
mise.toml                      pinned toolchain versions (go, golangci-lint)
```

The domain package owns the ports; adapters implement them and never the reverse; `cmd/family-dashboard` is the only
place that imports both and wires them together.

This project is built using the ["agent-work"](../agent-work) coding harness — `AGENTS.md` and `memory/status.md` are
its convention/history files, not optional documentation. The deploy tooling (`Makefile`, `deploy/`) and the Pi docs are
written by hand.

## Configuration

The app needs a JSON config file (`-config`, required; ADR-0001). Copy `sources.local.example.json` to
`sources.local.json` and fill it in: one entry per calendar feed (`sportadmin` or `infomentor`, with the owner's name),
the school lunch RSS URL, and the location for the weather. The feed URLs embed private tokens: `sources.local.json` is
gitignored, must never be committed, and its URLs must never be logged or printed.

## Running on the dev machine

Requires [mise](https://mise.jdx.dev/) for the pinned Go toolchain (`mise install`).

```bash
go run ./cmd/family-dashboard -config sources.local.json                           # web page on :8080
go run ./cmd/family-dashboard -config sources.local.json -eink-png /tmp/board.png  # the e-paper frame as a PNG
```

Flags for the e-paper output: `-eink-png <file>` or `-eink-panel` (with `-cable left|right`, the side of the portrait
picture where the panel's ribbon cable exits) draw one frame and exit; add `-eink-loop` to keep either up to date until
SIGINT/SIGTERM (`-eink-fetch-every` 30m, `-eink-full-every` 1h, `-eink-sleep` true). `-eink-panel` only works on the
Pi (linux/arm).

## Running on the Pi

Everything runs from the dev machine over SSH (`make help` lists the targets; see `docs/raspberry-pi-setup.md`,
sections 7–10, for the first-time setup):

```bash
make doctor      # check SSH, the sudo rule and the groups on the Pi
make deploy      # test, lint, cross-compile (ARMv6), copy binary + config, install the service, restart
make logs        # follow the service log
make deploy-config   # copy only sources.local.json and restart (e.g. after a calendar link changed)
make stop | start | restart | status | logs-last
```

The service (`deploy/family-dashboard.service.in`) runs the loop with a full refresh at least every 30 minutes and a
partial refresh of the clock every minute, starts at power-on and restarts after a crash.

## Running tests

```bash
sensors/test.sh    # go test ./... -race -count=1
sensors/lint.sh    # golangci-lint run ./...
```

Tests run against checked-in fixtures, not the live network — no real feed URL or credentials are needed.

## Roadmap

Per the PRD's phased rollout:

- **Phase 1 — data sources and the web page: done.** SportAdmin and InfoMentor calendars (ICS), SMHI weather, Skolmaten
  lunch, and the web page. Not built: a Google Calendar (OAuth) source for family events.
- **Phase 2 — Raspberry Pi hosting: done.** Runs as a systemd service on the Pi Zero W, deployed with `make deploy`.
- **Phase 3 — school data: partly done.** Homework and tests come from the InfoMentor calendar feed. A school schedule
  ("Skolschema") section is not built; there is no data source for it yet.
- **Phase 4 — e-paper display: live.** Portrait layout on the 7.5" panel in a picture frame; partial clock refresh,
  periodic full refresh, dead-feed notice, auto-start at power-on, home Wi-Fi at boot.

Next, roughly in order:

- Quieter logs: a `-log-level` flag, and the per-minute refresh lines at debug level.
- Clear the screen on stop (for example `make clear`), for long breaks.
- Finish the frame: a mat window cut to the visible screen area, and a notch for the power cable.
- Put the real location in `sources.local.json` (it holds a placeholder), so the weather is local.
- Skolschema section, once there is a data source and the kids have said what they want to see.
- Measure ghosting over several days of running.

Parked or open: small text is a little rough on the e-paper (a test of Terminus and Spleen bitmap fonts gave little
gain; a lower threshold is the cheap thing to try next); two family members with the same initial would look the same;
a Google Calendar source.
