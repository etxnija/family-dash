---
type: component
tags: [main, config, composition-root, multi-source, cmd, http, lunch, weather, eink, png, epd, flags, loop, signal]
summary: cmd/family-dashboard main() — JSON sources+school+location config, adapter construction at startup, dashboard.Collector/LunchSource/WeatherSource wiring, fetchDashboard shared by the HTML handler and the one-shot showOnce helper (any eink.Panel, always Sleep+Close) behind -eink-png (pngpanel) and -eink-panel -cable left|right (epd); -eink-loop runs eink.Loop on either panel via runLoop under signal.NotifyContext; parseFlags validates flags (mutually exclusive eink modes, loop-flag rules, exit 2).
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md, collector.md, skolmaten-adapter.md, smhi-adapter.md, eink-adapter.md]
updated: 2026-11-10 (-eink-loop docs: loop flag rules, runLoop, SIGINT/SIGTERM exit codes)
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-21 }, { by: "human", at: 2026-09-22 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }]
---

# Composition Root (`cmd/family-dashboard`)

## Context & Rationale
`main.go` wires an arbitrary number of calendar sources (not just SportAdmin)
into the domain and serves the result as a stateless HTTP dashboard: every
request re-fetches all sources live and renders fresh HTML via the `web`
package — there is no caching or background refresh. Source list and
per-source secrets (ICS URLs) live in a JSON config file rather than env vars
or flags, per
[ADR-0001](../../docs/arch/adr/0001-config-path-via-required-flag.md); the
file's path is a required `-config` flag with no default. Listen address is
a separate `-addr` flag (default `:8080`).

Flags are parsed by `parseFlags(args, output io.Writer) (options, error)`
with its own `flag.FlagSet` (`ContinueOnError`), not the global `flag`
package, so validation is table-tested (`TestParseFlags`). `options` holds
`config`, `addr`, `einkPNG`, `einkPanel`, `cable epd.Cable`, `einkLoop` and
`loop eink.LoopSettings`. It errors on
a missing `-config` (ADR-0001), `-eink-png` together with `-eink-panel`, an
invalid `-cable` when `-eink-panel` is set (via `epd.ParseCable`), and
unknown flags. Loop rules: `-eink-loop` needs exactly one of `-eink-png` /
`-eink-panel`; `-eink-fetch-every`, `-eink-full-every` and `-eink-sleep` are
errors without `-eink-loop` (detected with `fs.Visit`, so even an explicit
default like `-eink-sleep=true` counts); both intervals must be at least
`minLoopInterval` (1m). Their defaults come from `eink.DefaultLoopSettings()`
(30m, 1h, true), and the parsed settings are checked by the pure
`validateLoopFlags(opts, loop, set)`, then stored in `opts.loop` by
`parseFlags` only with `-eink-loop` (zero otherwise, like `cable`). For its own checks it logs the error with
`slog.Error("invalid flags", ...)` (JSON on stderr) and prints usage to
`output` itself. Usage printing stays in `parseFlags` on purpose: the
`FlagSet` already prints usage for `fs.Parse` errors, so both error paths
print it from the same place, and `main()` only maps errors to exit codes.
`-cable` (default `left`) is not parsed without `-eink-panel`, so
`-cable up` alone is accepted and `cable` stays `CableLeft`. `main()` exits
2 on any error and 0 on `flag.ErrHelp` (`-h`).

Adapter construction and event collection are now split across startup and
request time: `main()` builds each source's adapter once and wires them into
a `dashboard.Collector` ([collector.md](collector.md)); `handleIndex` just
asks the collector to `Collect` on every request. Fetching, owner-tagging,
merging, and deriving today/homework/upcoming views all live in
`Collector.Collect` now, not in `main.go`.

## Key Rules & Mechanics
- Config parsing lives in `config/config.go` (`package config`), not in
  `cmd/family-dashboard`: exported `Source` (`Name`/`Type`/`URL` JSON
  fields), `School` (`Name`/`RSSURL`, JSON `rssUrl`), `Location`
  (`Lat`/`Lon float64`), `Config` (`Sources []Source`, `School School`,
  `Location Location`), and `Load(path) (Config, error)`. The config file's
  JSON root is `{"sources": [...], "school": {...}, "location": {...}}`,
  not a bare array — `school` and `location` carry shared, non-per-kid
  config (lunch feed, weather coordinates). It imports only
  `encoding/json`, `fmt`, `os` — no dependency on `family-dashboard` or
  `family-dashboard/webcal`, since parsing shouldn't need to know about
  adapters.
- `buildSource(cfg config.Source) (dashboard.EventSource, error)` lives in
  `cmd/family-dashboard/main.go` itself, not in the `config` package —
  it wires a parsed `config.Source` to a concrete adapter (`webcal.New`),
  which is composition-root work, not config parsing. Adding a second
  source type means adding a `case` to `buildSource`, not a new
  package-level branch elsewhere.
- `config.Load` errors never include file contents — only the path and
  underlying cause — since the file holds secret URLs.
- `main()` loads config once at startup and passes it to `buildAdapters(cfg)`,
  which returns the collector, lunch source, and weather source. The HTTP
  server, the `-eink-png` export and the `-eink-panel` draw all use this one
  wiring path, so the `no school lunch feed configured` / `no weather
  location configured` Info logs fire in every mode.
- `buildAdapters` loops over `cfg.Sources` calling `buildSource(src)` for
  each. A `buildSource` error is `slog.Error`-logged (source name/type,
  never the URL) and that source is skipped — same log-and-continue as
  before, just moved to startup instead of per-request. Successful builds
  are collected into `[]dashboard.NamedSource{Owner: src.Name, Source:
  source}` and passed once to `dashboard.NewCollector(...)`, whose result
  is returned to `main()` and handed to `handleIndex`.
- `handleIndex(collector *dashboard.Collector, lunchSource dashboard.LunchSource, weatherSource dashboard.WeatherSource)`
  takes one `now := time.Now()` per request, calls
  `fetchDashboard(r.Context(), …, now)`, and passes the resulting
  today/homework/upcoming/owners/lunch/weather plus the same `now` into
  `web.NewPageData` / `web.Render` to produce the HTML response.
  `r.Context()` flows through to `Collect` so a client disconnect cancels
  in-flight fetches.
- `lunchSource` is built once in `buildAdapters` (see
  [skolmaten-adapter.md](skolmaten-adapter.md)) — `nil` if
  `cfg.School.RSSURL` is empty. `fetchDashboard` calls `lunchSource.Lunch(ctx)`
  only when non-nil; a `Lunch` error is `slog.Error`-logged and the render
  continues with an empty dish list (`nil []string`), same
  log-and-continue philosophy as a failing calendar source in
  `Collector.Collect`.
- `weatherSource` is built once in `buildAdapters` (see
  [smhi-adapter.md](smhi-adapter.md)) — `nil` if `cfg.Location.Lat == 0 &&
  cfg.Location.Lon == 0`. `fetchDashboard` calls
  `weatherSource.Weather(ctx, now)` only when non-nil; a `Weather` error is
  `slog.Error`-logged and the render continues with a `nil` reading, same
  log-and-continue philosophy as the lunch source. On success,
  `dashboardData.Weather` points at the fetched reading, which
  `handleIndex` passes on to `web.NewPageData`.
- `fetchDashboard(ctx, collector, lunchSource, weatherSource, now) dashboardData`
  is the single place that fetches everything for one render: `Collect`,
  `Owners`, then the nil-safe lunch and weather fetches (same
  log-and-continue `slog.Error` messages). One `now` is threaded through
  `Collect`, `Weather`, and the view constructor so a render sees a single
  consistent instant. `handleIndex` calls it with `r.Context()`; `showOnce`
  (both eink modes) calls it with `context.Background()`. New output targets should
  reuse it rather than re-fetch inline.
- `-eink-png <path>` (optional) switches `main()` into one-shot export mode
  after sources are built: it calls `showOnce(context.Background(),
  pngpanel.New(path), …, time.Now())`. `showOnce(ctx, panel eink.Panel,
  collector, lunch, weather, now)` is the shared one-shot helper for any
  `eink.Panel`: `fetchDashboard` → `eink.NewView` → `eink.Render` →
  `panel.Full`, then it **always** calls `panel.Sleep()` and
  `panel.Close()`, even when the view build or `Full` failed, and returns
  all errors joined (`errors.Join`), each wrapped (`building e-paper view`,
  `full refresh`, `sleeping panel`, `closing panel`). The PNG writing itself
  lives in `eink/pngpanel` (see [eink-adapter.md](eink-adapter.md)).
  Success logs `wrote e-paper frame` and returns before the HTTP server
  starts (exit 0); any failure logs `writing e-paper frame` with the path
  and cause and exits 1. Nothing is written to stdout. The mode never
  serves HTTP, and it does not change `-config`: the flag is still required
  and parsed exactly as in server mode (ADR-0001).
- `-eink-panel` (bool, with `-cable left|right`) is the hardware
  counterpart: `epd.New(cable)`; if that fails it logs `opening e-paper
  panel` and exits 1 (no panel to sleep — on the Mac this is the stub's
  "only supported on linux/arm" error). Otherwise it runs the same
  `showOnce`, logs `drawing e-paper frame` + exit 1 on error, or
  `drew e-paper frame` and returns. It never starts the HTTP server.
  `-eink-png` and `-eink-panel` are mutually exclusive: giving both is a
  `parseFlags` error and exits 2 before any config is loaded. In both
  modes the panel is always slept and closed once it exists (`showOnce`
  does it even after a failed view build or `Full`); only a failed
  `epd.New` skips them, as there is no panel.
  `cmd` importing `epd` is what links the real driver on linux/arm; the host
  build only compiles the stub (`go list -deps ./cmd/family-dashboard` has
  no `epd7in5v2`). Cross-check with
  `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build ./cmd/family-dashboard`.
- The cross-compiled Pi binary, `/family-dashboard-pi` (the `-o` name in
  [eink-adapter.md](eink-adapter.md)), is in `.gitignore`.
- `-eink-loop` (with `-eink-png` or `-eink-panel`) is branched on in `main()`
  before the one-shot and HTTP paths, which are unchanged. It makes
  `signal.NotifyContext(…, os.Interrupt, syscall.SIGTERM)`, builds the panel
  (`pngpanel.New` or `epd.New`; a failed `epd.New` logs `opening e-paper
  panel` and exits 1), and calls `runLoop(ctx, panel, eink.SystemClock{},
  collector, lunch, weather, settings)`. `runLoop` is the test seam: it wraps
  `fetchDashboard` + `eink.NewView` as the `eink.ViewSource` and runs
  `eink.NewLoop(...).Run(ctx)`, which owns the panel's `Sleep`/`Close` (see
  [eink-adapter.md](eink-adapter.md)). A signal → nil → `e-paper loop
  stopped`, exit 0; an error (first fetch fails) → `running e-paper loop`,
  exit 1. `stop()` is called before `os.Exit` since the defer would not run.
- The `ownerNames` free function is gone — `Collector.Owners()` replaces it,
  computed once in `NewCollector` from the same `NamedSource` pairs used for
  fetching, so the owner list and the fetch list can't drift apart.
- Per-source fetch failures (`.Events(ctx)` errors), owner-tagging, merging,
  and deriving the today/homework/upcoming views now happen inside
  `Collector.Collect` — see [collector.md](collector.md) for that logic
  (log-and-continue on a failing source, same as the old per-request loop).
- Tests split along the same boundary: `config/config_test.go` (`package
  config_test`) covers `Load`; `cmd/family-dashboard/main_test.go`
  (`package main`) covers `parseFlags`, `buildSource`, `buildAdapters`, `fetchDashboard`
  (stub lunch/weather sources: nil, failing, succeeding), and `showOnce`
  (with `pngpanel.New(tmp)` decoding a 480x800 PNG, and with a call-recording
  fake `eink.Panel`: `Full` once with a 480x800 frame, then `Sleep`, then
  `Close`, also when `Full` fails), and `runLoop` (`fakePanel.onFull`
  cancels the context, so the real `SystemClock` never sleeps: `[Full Sleep
  Sleep Close]` with `-eink-sleep`, `[Full Sleep Close]` without — the loop
  sleeps after every update *and* on shutdown);
  `collector_test.go` (package `dashboard`) covers the fetch/merge/filter
  logic that used to live inline in `handleIndex`.

## Gotchas & Failure Modes
- Only `config.Load` failure (missing/malformed config file) and
  `http.ListenAndServe` failure are fatal (`os.Exit(1)`); a failed
  `-eink-png` export or `-eink-panel` draw also exits 1, and so does an
  `-eink-loop` whose panel cannot be opened or whose first fetch fails.
  A loop stopped by SIGINT/SIGTERM exits 0; later fetch or refresh
  failures inside the loop are logged and retried, never fatal. Any
  `parseFlags` error (missing `-config` per ADR-0001, both eink flags, bad
  `-cable`, a loop-flag rule, an interval under 1m) is a usage error
  (`os.Exit(2)`). Individual source
  build or fetch failures are not fatal by design — they degrade the
  dashboard (fewer sources), not the whole server.
- Adapter construction (`buildSource`) now happens once at startup instead
  of per-request. This is safe because `webcal.New` has no I/O — it just
  stores a URL and `*http.Client` — but it does mean a source whose config
  is wrong won't be retried without a process restart (previously each
  request retried `buildSource`).
- Timezone loading (`Europe/Stockholm`) happens inside `web.NewPageData`; a
  failure there is per-request (`http.Error` 500), not process-fatal like
  it was when `main()` loaded the zone once at startup.
- There is no caching: every request re-fetches every configured source via
  `Collector.Collect`. Fine for a low-traffic family dashboard; would need
  revisiting under higher request volume or rate-limited upstream feeds.
