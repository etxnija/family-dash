# Work Log

Session-to-session progress log. Distinct from AGENTS.md (stable rules).

---

## 2026-09-16 — project bootstrapped

### Done
- Bootstrapped with agent harness (lang: go)

### Next
- (add first task here)

## 2026-09-17 — LANG_PATTERNS task redirected

- Task "Add LANG_PATTERNS dict and {lang_patterns} placeholder" targeted `bootstrap/bootstrap.py`,
  which does not exist in this repo — it's the agent harness's own bootstrap tool at
  `/Users/nils/source/agent-work/bootstrap/bootstrap.py` (confirmed via plan.md's own
  Assumptions section). No changes were made in family-dash; the edit landed in agent-work instead.


**Run metrics:** 3 driver call(s), $0.7099, session c3bef2fc-7ab5-430c-b173-34eff4e36836


**Run metrics:** 8 driver call(s), $2.1093, session 42720221-5117-4679-b3ab-1fcdf18d2817

## 2026-09-17 — Task 1: domain types, today-filter, hexagonal layout

- Moved composition root from root `main.go` to `cmd/family-dashboard/main.go`; added root `package dashboard` with `Event`, `EventSource` port, and `TodayEvents` (Europe/Stockholm date filter) plus table-driven tests in `event_test.go`.

## 2026-09-17 — Task 2: SportAdmin ICS adapter with tests and composition-root wiring

- Added `sportadmin` package (`sportadmin.Client` implementing `dashboard.EventSource`) with a line-folding-aware ICS parser (`ParseICS`) handling TZID/bare/UTC/all-day timestamps and RFC 5545 text unescaping; fetch errors never leak the feed URL (`scrubURL` strips it from `*url.Error`). Wired `cmd/family-dashboard/main.go` to read `SPORTADMIN_ICS_URL`, fetch, filter to today, and print. Verified end-to-end against a local HTTP server serving `sportadmin/testdata/sample.ics`.


**Run metrics:** 6 driver call(s), $2.5562, session 2319da37-be48-4f5a-800a-b3a91d7c80f9

## 2026-09-18 — Task 1 (rename): sportadmin package renamed to webcal

- Moved `sportadmin/` → `webcal/` (files stay named `sportadmin.go`/`sportadmin_test.go`/`testdata/sample.ics`, since they identify the source within the adapter); changed `package sportadmin` → `package webcal`, updated imports/refs in `cmd/family-dashboard/main.go`; kept `sportadmin:` error prefixes and `SPORTADMIN_ICS_URL` env var unchanged. Renamed concept doc `sportadmin-adapter.md` → `webcal-adapter.md`. `go build`, `go vet`, `go test ./...` all pass.

## 2026-09-18 — Task 2: Update concept documentation for webcal rename

- Verified this task's scope (`sportadmin-adapter.md` → `webcal-adapter.md` rename, frontmatter/heading/body updates, `index.md` entry) was already completed as part of the prior "Task 1 (rename)" commit — no further changes needed.


**Run metrics:** 11 driver call(s), $3.1687, session 8920bd7c-e54a-4eae-847b-a38ed500eacf

## 2026-09-18 — Task: slog structured logging in composition root

- Replaced `fmt.Fprintln`-based stderr errors in `cmd/family-dashboard/main.go` with `slog` JSON logging (default logger installed at top of `main()`), and added `slog.Info` diagnostics for fetched-event and filtered-to-today counts; `fmt.Printf` remains for the event-printing loop. `go build`, `go vet`, `go test ./...`, and sensors (`test.sh`, `lint.sh`) all pass.


**Run metrics:** 3 driver call(s), $0.7386, session 719d432b-17ef-4833-96f0-aa47d48cd5ba

## 2026-09-18 — Task: Normalize Event times to UTC, convert for display

- `parseICSTime` (`webcal/sportadmin.go`) now returns UTC from both the `Z` branch and the Stockholm-local branch via `.UTC()`; documented `Event.Start`/`End` as always-UTC and `EventSource` as requiring UTC-normalized adapter output (`event.go`); `cmd/family-dashboard/main.go` now loads `Europe/Stockholm` and converts via `.In(loc)` at print time instead of formatting raw (implicitly local) times. Updated `webcal-adapter.md` to describe this as a cross-adapter UTC contract. `go build`, `go vet`, `go test ./...` all pass.


**Run metrics:** 6 driver call(s), $1.8713, session 8968eb8c-f437-4f4c-a2b1-bd9c651bbfcd

## 2026-09-18 — Task: Add Owner field and WithOwner helper to domain

- Added `Owner string` to `Event` (`event.go`) and a pure `WithOwner(events []Event, owner string) []Event` helper that returns a new slice with `Owner` set on every element, following the same style as `TodayEvents`; added table-driven tests in `event_test.go` covering non-empty, empty, and overwrite-existing-owner cases. `go build`, `go vet`, `go test ./...` all pass.

## 2026-09-18 — Task: Add config loading, source builder, and rewrite main()

- Added `cmd/family-dashboard/config.go` (`sourceConfig`, `loadSources`, `buildSource`) and table-driven `config_test.go`; rewrote `main.go` to use a required `-config` flag (ADR-0001), loop over configured sources building each with `buildSource`, log+`continue` on a per-source `buildSource`/`Events` failure instead of aborting, tag fetched events with `dashboard.WithOwner`, and print today's events with an owner prefix. Added `memory/concepts/composition-root.md`. `go build`, `go vet`, `go test ./... -race`, and `sensors/lint.sh` all pass.

## 2026-09-18 — Task: Add example config and .gitignore

- Added `sources.local.example.json` (placeholder single-source array matching `sourceConfig`'s JSON shape) and `.gitignore` ignoring `/sources.local.json` (real subscription URLs) and `/family-dashboard` (compiled binary). `go build`, `go vet` pass.


**Run metrics:** 7 driver call(s), $1.8704, session ded6d7c2-cf7d-4a68-a6a5-df3423c168d9

## 2026-09-18 — Task: Extract config package and relocate buildSource

- Moved config parsing out of `cmd/family-dashboard/config.go` into `config/config.go` (`package config`, exported `Source`/`Load`, no dependency on `family-dashboard`/`family-dashboard/webcal`); `TestLoadSources` moved to `config/config_test.go`. `buildSource` and its test moved into `cmd/family-dashboard/main.go`/`main_test.go` (updated to take `config.Source`), since wiring a config entry to a concrete adapter is composition-root work, not config parsing. Deleted the old `cmd/family-dashboard/config.go`/`config_test.go`. Updated `composition-root.md` to reflect the new package boundary. `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.


**Run metrics:** 3 driver call(s), $1.0114, session 7b12372a-6f68-4793-86b1-f4d56e169145

## 2026-09-18 — Task: Add InfoMentor source type and rename error prefixes

- Added `case "infomentor":` to `buildSource` in `cmd/family-dashboard/main.go` (identical `webcal.New` wiring to sportadmin); updated `webcal/sportadmin.go`'s package doc comment and renamed all 8 `"sportadmin:"` error-message prefixes to `"webcal:"` in `fetch`, `ParseICS`, and `build`; added a `"known type infomentor"` case to `TestBuildSource`; added an InfoMentor entry to `sources.local.example.json`. Updated `webcal-adapter.md` to describe both sources sharing `buildSource` and the source-agnostic `webcal:` error prefix. `go build`, `go vet`, `go test ./...` all pass.


**Run metrics:** 6 driver call(s), $2.4487, session 1e839ac9-41e6-4047-9da6-8811d387323e

## 2026-09-18 — Task: Add webcal scheme normalization

- Added `normalizeWebcalScheme` (`webcal/sportadmin.go`), called from `New` to rewrite `webcal://`/`webcals://` URLs to `http://`/`https://` before storing them, since `net/http` doesn't understand the `webcal(s)` pseudo-scheme used by calendar subscription links. Added `TestNew_NormalizesWebcalScheme` table-driven test to `webcal/sportadmin_test.go`, extending `stubRoundTripper` with a `capturedURL` field to assert the actual request URL. Updated `webcal-adapter.md`. `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.


**Run metrics:** 6 driver call(s), $2.0994, session c3b90d66-7ea4-4f04-b5d1-51a5d9b6be7e

## 2026-09-18 — Task: Create web package (HTML dashboard display adapter)

- Added `web` package (`web/page.go`, `web/templates/page.html.tmpl`, `web/page_test.go`) as a new adapter following the `webcal` package's adapter-imports-domain pattern: template embedded via `//go:embed` and parsed once at package init with `template.Must`; `EventRow`/`PageData` view-model types; `NewPageData` builds display rows from `dashboard.Event`, converting to Europe/Stockholm and formatting `TimeRange` as start–end (en dash) or just start when `End` is zero; `Render` executes the parsed template; `ownerColor` maps "Kaj"/"Lillian" to CSS custom properties with `"--family"` fallback, registered as the template's `ownerColor` FuncMap entry. The literal HTML/CSS markup wasn't present anywhere in this worktree (plan.md referenced "provided template markup" that was never supplied), so the template content itself was authored from the functional spec (weekday/date header, owner chips, event list, "Nothing scheduled today." empty state, meta-refresh, placeholder weather/school-hours/homework sections) rather than copied verbatim — flagged for human review. `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.

## 2026-09-18 — Task 2: Rewrite main.go to HTTP server

- Rewrote `cmd/family-dashboard/main.go`: added `-addr` flag (default `:8080`); `main()` now loads config once and registers `http.HandleFunc("/", handleIndex(sources))` then calls `http.ListenAndServe`, instead of fetching-and-printing once at startup. Added `ownerNames(sources)` (distinct `cfg.Name` in config order) and `handleIndex(sources) http.HandlerFunc`, which closes over `sources`/owners and, on every request, loops sources, builds/fetches each with `r.Context()` (log+continue per-source on error, same graceful degradation as before), tags with `dashboard.WithOwner`, filters via `dashboard.TodayEvents`, then renders via `web.NewPageData`/`web.Render` with a 500 (`http.Error`) on either failing. Removed the old `fmt.Printf` print loop and the top-level `time.LoadLocation` call (timezone handling now lives inside `web.NewPageData`, so a zone-load failure is per-request, not process-fatal). `buildSource`/`TestBuildSource` untouched. Updated `composition-root.md` for the HTTP rewrite. `go build`, `go vet`, `go test ./... -race`, and `sensors/lint.sh` all pass; manual `curl` smoke test was blocked by sandbox network permissions, so verification relied on the test/lint suite.


**Run metrics:** 7 driver call(s), $2.6203, session 6cafc61f-bba5-4888-b2d7-02ba45fcf72b
