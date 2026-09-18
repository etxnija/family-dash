---
type: component
tags: [main, config, composition-root, multi-source, cmd, http]
summary: cmd/family-dashboard main() — JSON sources config, HTTP server wiring, per-request fetch with per-source error isolation, and owner tagging.
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md]
updated: 2026-09-18 (homework panel wiring)
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }]
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
- `main()` loads config once at startup, then registers a single handler:
  `http.HandleFunc("/", handleIndex(sources))`. All per-request work —
  fetching, aggregating, rendering — happens inside `handleIndex`'s returned
  closure, not in `main()` itself.
- `handleIndex(sources)` precomputes `ownerNames(sources)` (distinct
  `cfg.Name` values, in config-file order) once outside the closure, since
  the owner list doesn't change per request; only the event fetch does.
- On each request, the closure loops over `sources` and treats each as
  independently failable: a `buildSource` or `.Events(r.Context())` error is
  `slog.Error`-logged (source name/type only, never the URL) and the loop
  `continue`s to the next source, rather than aborting the whole response.
  One bad feed shouldn't blank the whole dashboard. `r.Context()` is used
  (not `context.Background()`) so a client disconnect cancels in-flight
  fetches.
- Successfully fetched events are tagged via `dashboard.WithOwner(events,
  cfg.Name)` before being appended to the combined slice, then filtered two
  ways from the same combined slice: `dashboard.TodayEvents` for the
  schedule panel and `dashboard.HomeworkEvents` for the homework panel
  (7-day window, `"Läxa "`-prefixed summaries). Both filtered slices are
  handed to `web.NewPageData(today, homework, owners, time.Now())` /
  `web.Render` (see [webcal-adapter.md](webcal-adapter.md) sibling pattern
  in the `web` package) to produce the HTML response.
- Tests split along the same boundary: `config/config_test.go` (`package
  config_test`) covers `Load`; `cmd/family-dashboard/main_test.go`
  (`package main`) covers `buildSource`, which is unchanged by the HTTP
  rewrite.

## Gotchas & Failure Modes
- Only `config.Load` failure (missing/malformed config file) and
  `http.ListenAndServe` failure are fatal (`os.Exit(1)`); missing `-config`
  flag is a usage error (`os.Exit(2)`, per ADR-0001). Individual source
  failures are not fatal by design — they degrade that request's dashboard,
  not the whole server.
- Timezone loading (`Europe/Stockholm`) moved into `web.NewPageData`; a
  failure there is per-request (`http.Error` 500), not process-fatal like
  it was when `main()` loaded the zone once at startup.
- There is no caching: every request re-fetches every configured source.
  Fine for a low-traffic family dashboard; would need revisiting under
  higher request volume or rate-limited upstream feeds.
