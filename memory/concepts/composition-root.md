---
type: component
tags: [main, config, composition-root, multi-source, cmd, http, lunch, weather, eink, png]
summary: cmd/family-dashboard main() — JSON sources+school+location config, adapter construction at startup, dashboard.Collector/LunchSource/WeatherSource wiring, shared fetchDashboard helper for per-request HTML render and the one-shot -eink-png export.
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md, collector.md, skolmaten-adapter.md, smhi-adapter.md, eink-adapter.md]
updated: 2026-11-10 (fetchDashboard shared by handleIndex and -eink-png; PNG mode exits without serving)
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-21 }, { by: "human", at: 2026-09-22 }, { by: "human", at: 2026-11-10 }]
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
  which returns the collector, lunch source, and weather source. Both the
  HTTP server and the `-eink-png` export use this one wiring path, so the
  `no school lunch feed configured` / `no weather location configured` Info
  logs fire in both modes.
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
  consistent instant. `handleIndex` calls it with `r.Context()`; the PNG
  export calls it with `context.Background()`. New output targets should
  reuse it rather than re-fetch inline.
- `-eink-png <path>` (optional) switches `main()` into one-shot export mode
  after sources are built: `exportEinkPNG` fetches once, builds
  `eink.NewView` → `eink.Render`, and `writePNG` encodes the 480x800 frame
  (returning the `Close` error, joined with any encode error). If
  `png.Encode` fails after `os.Create` succeeded, `writePNG` removes the
  file so a display reading that path never sees a truncated PNG. Note that
  `os.Create` truncates first, so a previous good frame at the path is gone
  either way — there is no write-to-temp-and-rename. Success logs
  `wrote e-paper frame` and returns before the HTTP server starts (exit 0);
  any failure logs `writing e-paper frame` with the cause and exits 1.
  Nothing is written to stdout. The mode never serves HTTP, and it does
  not change `-config`: the flag is still required and parsed exactly as in
  server mode (ADR-0001). Rendering itself is in
  [eink-adapter.md](eink-adapter.md).
- The `ownerNames` free function is gone — `Collector.Owners()` replaces it,
  computed once in `NewCollector` from the same `NamedSource` pairs used for
  fetching, so the owner list and the fetch list can't drift apart.
- Per-source fetch failures (`.Events(ctx)` errors), owner-tagging, merging,
  and deriving the today/homework/upcoming views now happen inside
  `Collector.Collect` — see [collector.md](collector.md) for that logic
  (log-and-continue on a failing source, same as the old per-request loop).
- Tests split along the same boundary: `config/config_test.go` (`package
  config_test`) covers `Load`; `cmd/family-dashboard/main_test.go`
  (`package main`) covers `buildSource`, `buildAdapters`, `fetchDashboard`
  (stub lunch/weather sources: nil, failing, succeeding), and `writePNG`
  (decode-back 480x800, missing-directory error, encode failure removes the
  partial file);
  `collector_test.go` (package `dashboard`) covers the fetch/merge/filter
  logic that used to live inline in `handleIndex`.

## Gotchas & Failure Modes
- Only `config.Load` failure (missing/malformed config file) and
  `http.ListenAndServe` failure are fatal (`os.Exit(1)`); a failed
  `-eink-png` export also exits 1. Missing `-config` flag is a usage
  error (`os.Exit(2)`, per ADR-0001). Individual source
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
