---
type: component
tags: [main, config, composition-root, multi-source, cmd, http]
summary: cmd/family-dashboard main() — JSON sources config, adapter construction at startup, dashboard.Collector wiring, per-request Collect/render.
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md, collector.md]
updated: 2026-09-21 (rewired main.go onto dashboard.Collector)
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-21 }]
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
  fields) and `Load(path) ([]Source, error)`. It imports only
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
- `main()` loads config once at startup, then loops over `sources` calling
  `buildSource(cfg)` for each. A `buildSource` error is `slog.Error`-logged
  (source name/type, never the URL) and that source is skipped — same
  log-and-continue as before, just moved to startup instead of per-request.
  Successful builds are collected into `[]dashboard.NamedSource{Owner:
  cfg.Name, Source: source}` and passed once to `dashboard.NewCollector(...)`,
  whose result is stored and handed to `handleIndex`.
- `handleIndex(collector *dashboard.Collector)` no longer loops over sources
  or calls `TodayEvents`/`HomeworkEvents`/`UpcomingEvents`/`WithOwner`
  directly — its closure calls `collector.Collect(r.Context(), time.Now())`
  to get the three filtered event slices, then `collector.Owners()` for the
  owner list, and passes both into `web.NewPageData(today, homework,
  upcoming, owners, time.Now())` / `web.Render` (see
  [webcal-adapter.md](webcal-adapter.md) sibling pattern in the `web`
  package) to produce the HTML response. `r.Context()` still flows through
  to `Collect` so a client disconnect cancels in-flight fetches.
- The `ownerNames` free function is gone — `Collector.Owners()` replaces it,
  computed once in `NewCollector` from the same `NamedSource` pairs used for
  fetching, so the owner list and the fetch list can't drift apart.
- Per-source fetch failures (`.Events(ctx)` errors), owner-tagging, merging,
  and deriving the today/homework/upcoming views now happen inside
  `Collector.Collect` — see [collector.md](collector.md) for that logic
  (log-and-continue on a failing source, same as the old per-request loop).
- Tests split along the same boundary: `config/config_test.go` (`package
  config_test`) covers `Load`; `cmd/family-dashboard/main_test.go`
  (`package main`) covers `buildSource`, unchanged by this rewrite;
  `collector_test.go` (package `dashboard`) covers the fetch/merge/filter
  logic that used to live inline in `handleIndex`.

## Gotchas & Failure Modes
- Only `config.Load` failure (missing/malformed config file) and
  `http.ListenAndServe` failure are fatal (`os.Exit(1)`); missing `-config`
  flag is a usage error (`os.Exit(2)`, per ADR-0001). Individual source
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
