# Work Log

Session-to-session progress log. Distinct from AGENTS.md (stable rules).

---

## 2026-09-16 — project bootstrapped

### Done

- Bootstrapped with agent harness (lang: go)

### Next

- (add first task here)

## 2026-09-17 — LANG_PATTERNS task redirected

- Task "Add LANG_PATTERNS dict and {lang_patterns} placeholder" targeted `bootstrap/bootstrap.py`, which does not exist
  in this repo — it's the agent harness's own bootstrap tool at `/Users/nils/source/agent-work/bootstrap/bootstrap.py`
  (confirmed via plan.md's own Assumptions section). No changes were made in family-dash; the edit landed in agent-work
  instead.

**Run metrics:** 3 driver call(s), $0.7099, session c3bef2fc-7ab5-430c-b173-34eff4e36836

**Run metrics:** 8 driver call(s), $2.1093, session 42720221-5117-4679-b3ab-1fcdf18d2817

## 2026-09-17 — Task 1: domain types, today-filter, hexagonal layout

- Moved composition root from root `main.go` to `cmd/family-dashboard/main.go`; added root `package dashboard` with
  `Event`, `EventSource` port, and `TodayEvents` (Europe/Stockholm date filter) plus table-driven tests in
  `event_test.go`.

## 2026-09-17 — Task 2: SportAdmin ICS adapter with tests and composition-root wiring

- Added `sportadmin` package (`sportadmin.Client` implementing `dashboard.EventSource`) with a line-folding-aware ICS
  parser (`ParseICS`) handling TZID/bare/UTC/all-day timestamps and RFC 5545 text unescaping; fetch errors never leak
  the feed URL (`scrubURL` strips it from `*url.Error`). Wired `cmd/family-dashboard/main.go` to read
  `SPORTADMIN_ICS_URL`, fetch, filter to today, and print. Verified end-to-end against a local HTTP server serving
  `sportadmin/testdata/sample.ics`.

**Run metrics:** 6 driver call(s), $2.5562, session 2319da37-be48-4f5a-800a-b3a91d7c80f9

## 2026-09-18 — Task 1 (rename): sportadmin package renamed to webcal

- Moved `sportadmin/` → `webcal/` (files stay named `sportadmin.go`/`sportadmin_test.go`/`testdata/sample.ics`, since
  they identify the source within the adapter); changed `package sportadmin` → `package webcal`, updated imports/refs in
  `cmd/family-dashboard/main.go`; kept `sportadmin:` error prefixes and `SPORTADMIN_ICS_URL` env var unchanged. Renamed
  concept doc `sportadmin-adapter.md` → `webcal-adapter.md`. `go build`, `go vet`, `go test ./...` all pass.

## 2026-09-18 — Task 2: Update concept documentation for webcal rename

- Verified this task's scope (`sportadmin-adapter.md` → `webcal-adapter.md` rename, frontmatter/heading/body updates,
  `index.md` entry) was already completed as part of the prior "Task 1 (rename)" commit — no further changes needed.

**Run metrics:** 11 driver call(s), $3.1687, session 8920bd7c-e54a-4eae-847b-a38ed500eacf

## 2026-09-18 — Task: slog structured logging in composition root

- Replaced `fmt.Fprintln`-based stderr errors in `cmd/family-dashboard/main.go` with `slog` JSON logging (default logger
  installed at top of `main()`), and added `slog.Info` diagnostics for fetched-event and filtered-to-today counts;
  `fmt.Printf` remains for the event-printing loop. `go build`, `go vet`, `go test ./...`, and sensors (`test.sh`,
  `lint.sh`) all pass.

**Run metrics:** 3 driver call(s), $0.7386, session 719d432b-17ef-4833-96f0-aa47d48cd5ba

## 2026-09-18 — Task: Normalize Event times to UTC, convert for display

- `parseICSTime` (`webcal/sportadmin.go`) now returns UTC from both the `Z` branch and the Stockholm-local branch via
  `.UTC()`; documented `Event.Start`/`End` as always-UTC and `EventSource` as requiring UTC-normalized adapter output
  (`event.go`); `cmd/family-dashboard/main.go` now loads `Europe/Stockholm` and converts via `.In(loc)` at print time
  instead of formatting raw (implicitly local) times. Updated `webcal-adapter.md` to describe this as a cross-adapter
  UTC contract. `go build`, `go vet`, `go test ./...` all pass.

**Run metrics:** 6 driver call(s), $1.8713, session 8968eb8c-f437-4f4c-a2b1-bd9c651bbfcd

## 2026-09-18 — Task: Add Owner field and WithOwner helper to domain

- Added `Owner string` to `Event` (`event.go`) and a pure `WithOwner(events []Event, owner string) []Event` helper that
  returns a new slice with `Owner` set on every element, following the same style as `TodayEvents`; added table-driven
  tests in `event_test.go` covering non-empty, empty, and overwrite-existing-owner cases. `go build`, `go vet`,
  `go test ./...` all pass.

## 2026-09-18 — Task: Add config loading, source builder, and rewrite main()

- Added `cmd/family-dashboard/config.go` (`sourceConfig`, `loadSources`, `buildSource`) and table-driven
  `config_test.go`; rewrote `main.go` to use a required `-config` flag (ADR-0001), loop over configured sources building
  each with `buildSource`, log+`continue` on a per-source `buildSource`/`Events` failure instead of aborting, tag
  fetched events with `dashboard.WithOwner`, and print today's events with an owner prefix. Added
  `memory/concepts/composition-root.md`. `go build`, `go vet`, `go test ./... -race`, and `sensors/lint.sh` all pass.

## 2026-09-18 — Task: Add example config and .gitignore

- Added `sources.local.example.json` (placeholder single-source array matching `sourceConfig`'s JSON shape) and
  `.gitignore` ignoring `/sources.local.json` (real subscription URLs) and `/family-dashboard` (compiled binary).
  `go build`, `go vet` pass.

**Run metrics:** 7 driver call(s), $1.8704, session ded6d7c2-cf7d-4a68-a6a5-df3423c168d9

## 2026-09-18 — Task: Extract config package and relocate buildSource

- Moved config parsing out of `cmd/family-dashboard/config.go` into `config/config.go` (`package config`, exported
  `Source`/`Load`, no dependency on `family-dashboard`/`family-dashboard/webcal`); `TestLoadSources` moved to
  `config/config_test.go`. `buildSource` and its test moved into `cmd/family-dashboard/main.go`/`main_test.go` (updated
  to take `config.Source`), since wiring a config entry to a concrete adapter is composition-root work, not config
  parsing. Deleted the old `cmd/family-dashboard/config.go`/`config_test.go`. Updated `composition-root.md` to reflect
  the new package boundary. `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.

**Run metrics:** 3 driver call(s), $1.0114, session 7b12372a-6f68-4793-86b1-f4d56e169145

## 2026-09-18 — Task: Add InfoMentor source type and rename error prefixes

- Added `case "infomentor":` to `buildSource` in `cmd/family-dashboard/main.go` (identical `webcal.New` wiring to
  sportadmin); updated `webcal/sportadmin.go`'s package doc comment and renamed all 8 `"sportadmin:"` error-message
  prefixes to `"webcal:"` in `fetch`, `ParseICS`, and `build`; added a `"known type infomentor"` case to
  `TestBuildSource`; added an InfoMentor entry to `sources.local.example.json`. Updated `webcal-adapter.md` to describe
  both sources sharing `buildSource` and the source-agnostic `webcal:` error prefix. `go build`, `go vet`,
  `go test ./...` all pass.

**Run metrics:** 6 driver call(s), $2.4487, session 1e839ac9-41e6-4047-9da6-8811d387323e

## 2026-09-18 — Task: Add webcal scheme normalization

- Added `normalizeWebcalScheme` (`webcal/sportadmin.go`), called from `New` to rewrite `webcal://`/`webcals://` URLs to
  `http://`/`https://` before storing them, since `net/http` doesn't understand the `webcal(s)` pseudo-scheme used by
  calendar subscription links. Added `TestNew_NormalizesWebcalScheme` table-driven test to `webcal/sportadmin_test.go`,
  extending `stubRoundTripper` with a `capturedURL` field to assert the actual request URL. Updated `webcal-adapter.md`.
  `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.

**Run metrics:** 6 driver call(s), $2.0994, session c3b90d66-7ea4-4f04-b5d1-51a5d9b6be7e

## 2026-09-18 — Task: Create web package (HTML dashboard display adapter)

- Added `web` package (`web/page.go`, `web/templates/page.html.tmpl`, `web/page_test.go`) as a new adapter following the
  `webcal` package's adapter-imports-domain pattern: template embedded via `//go:embed` and parsed once at package init
  with `template.Must`; `EventRow`/`PageData` view-model types; `NewPageData` builds display rows from
  `dashboard.Event`, converting to Europe/Stockholm and formatting `TimeRange` as start–end (en dash) or just start when
  `End` is zero; `Render` executes the parsed template; `ownerColor` maps "Kaj"/"Lillian" to CSS custom properties with
  `"--family"` fallback, registered as the template's `ownerColor` FuncMap entry. The literal HTML/CSS markup wasn't
  present anywhere in this worktree (plan.md referenced "provided template markup" that was never supplied), so the
  template content itself was authored from the functional spec (weekday/date header, owner chips, event list, "Nothing
  scheduled today." empty state, meta-refresh, placeholder weather/school-hours/homework sections) rather than copied
  verbatim — flagged for human review. `go build`, `go vet`, `go test ./...`, and `sensors/lint.sh` all pass.

## 2026-09-18 — Task 2: Rewrite main.go to HTTP server

- Rewrote `cmd/family-dashboard/main.go`: added `-addr` flag (default `:8080`); `main()` now loads config once and
  registers `http.HandleFunc("/", handleIndex(sources))` then calls `http.ListenAndServe`, instead of
  fetching-and-printing once at startup. Added `ownerNames(sources)` (distinct `cfg.Name` in config order) and
  `handleIndex(sources) http.HandlerFunc`, which closes over `sources`/owners and, on every request, loops sources,
  builds/fetches each with `r.Context()` (log+continue per-source on error, same graceful degradation as before), tags
  with `dashboard.WithOwner`, filters via `dashboard.TodayEvents`, then renders via `web.NewPageData`/`web.Render` with
  a 500 (`http.Error`) on either failing. Removed the old `fmt.Printf` print loop and the top-level `time.LoadLocation`
  call (timezone handling now lives inside `web.NewPageData`, so a zone-load failure is per-request, not process-fatal).
  `buildSource`/`TestBuildSource` untouched. Updated `composition-root.md` for the HTTP rewrite. `go build`, `go vet`,
  `go test ./... -race`, and `sensors/lint.sh` all pass; manual `curl` smoke test was blocked by sandbox network
  permissions, so verification relied on the test/lint suite.

**Run metrics:** 7 driver call(s), $2.6203, session 6cafc61f-bba5-4888-b2d7-02ba45fcf72b

## 2026-09-18 — Task: Support all-day events end-to-end

- Added `AllDay bool` to `Event` (`event.go`); added `parseICSDate` (`webcal/sportadmin.go`), which detects all-day
  DTSTART/DTEND values by attempting `time.Parse("20060102", value)` rather than inspecting the `VALUE=DATE` param, and
  reinterprets the parsed date as midnight Europe/Stockholm converted to UTC; `build()` now produces a real (non-nil)
  `Event` for all-day VEVENTs with `AllDay: true`, and a date-only DTEND is parsed-and-discarded (leaves `End` zero,
  multi-day spans out of scope). `web.NewPageData` renders `Time`/`TimeRange` as `"All day"` when `e.AllDay`. Updated
  `TestParseICS_NormalEvents` to expect the sample feed's third (all-day "Helgcup") event; added
  `TestParseICS_AllDayWithoutValueParam`/`TestParseICS_AllDayWithValueParam` to confirm detection is parse-based, not
  param-based; added an all-day case to `TestNewPageData`. Updated `webcal-adapter.md`'s all-day section. `go build`,
  `go vet`, `go test ./...`, `gofmt -l .` all clean.

**Run metrics:** 5 driver call(s), $3.1380, session 50029c4d-c952-405f-8685-3781617539af

## 2026-09-18 — Task: Homework due panel end to end

- Added `HomeworkEvents(events, now)` to `event.go` (7-calendar-day window, `Läxa` summary prefix, Europe/Stockholm)
  with table-driven tests in `event_test.go`; added `HomeworkRow`/`HomeworkGroup`/`PageData.Homework` to `web/page.go`,
  changed `NewPageData` to take a `homeworkEvents` parameter and build per-owner groups (sorted by `Start`, day labels
  "Due today"/"Due tomorrow"/weekday name via noon-to-noon offset to dodge DST rounding), with
  `TestNewPageData_Homework` covering label/ordering/empty-owner cases; replaced the hardcoded homework panel markup in
  `web/templates/page.html.tmpl` with `{{range .Homework}}`/`{{range .Items}}` and an empty-state message; wired
  `dashboard.HomeworkEvents` + updated `web.NewPageData` call into `cmd/family-dashboard/main.go`'s handler. `go build`,
  `go vet`, `go test ./...`, and `sensors/test.sh`/`sensors/lint.sh` all pass; manually rendered the page with fixture
  homework events to confirm labels and owner grouping.

**Run metrics:** 5 driver call(s), $3.1236, session 5ad27b0f-bfef-4caa-9b8b-a76529910aeb

## 2026-09-19 — Task: Receipt-style dashboard redesign

- Added `WeekNumber`/`CurrentTime` to `PageData` (set in `NewPageData` from `localNow.ISOWeek()`/`.Format("15:04")`);
  added five unexported Swedish translation FuncMap helpers to `web/page.go` (`svWeekday`, `svDateLabel`, `svDayLabel`,
  `svTime`, `ownerInitial`), each translating at render time only so `NewPageData`'s Go-side data stays English; removed
  the now-unused `dict` helper. Rewrote `web/templates/page.html.tmpl` as a monochrome, Courier-New "receipt" layout
  (centered white sheet with corner marks, big header + week number, centered owner line, large clock + weather
  placeholder, Skolschema/Skollunch/Idag/Läxor/Kommande-dagar sections separated by `<hr>`s), removing the Google Fonts
  link, oklch color vars, and the `comingUpCard` sub-template; `<html lang="sv">`. Added
  `TestNewPageData_WeekNumberAndCurrentTime` and table-driven tests for all five new FuncMap helpers to
  `web/page_test.go`; updated `TestRender`'s empty-state expectation to `"Inget planerat idag."`; left
  `TestNewPageData`/`TestNewPageData_Homework` (English assertions) unchanged. Verified visually via a headless-Chrome
  screenshot of the rendered template. Added `memory/concepts/dashboard-page.md`. `go build`, `go vet`, `gofmt -l .`,
  `go test ./... -race`, `sensors/lint.sh` all pass.

**Run metrics:** 7 driver call(s), $3.5247, session 63d7433c-a906-4f59-9ce8-6c0befe942b2

- Wrote ADR-0002 (`docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md`) recording the decision not to
  share a rendering interface between the `web/` HTML adapter and a future `eink` e-paper adapter — each display target
  owns its own adapter package and view-model, consuming domain data independently.

**Run metrics:** 7 driver call(s), $1.6374, session 60da0967-5837-4840-af88-f5ea6e1ba6c4

## 2026-09-21 — Task: Show event description in Idag

- Added `Description string` to `EventRow` (`web/page.go`), populated from `e.Description` in `buildEventRows`; template
  renders it conditionally as a muted row after the event line in `page.html.tmpl` (`{{if .Description}}...{{end}}`, no
  markup when empty); added `wantDescription` to `pageDataTestCase` plus a new "event with description" case and
  assertion in `TestNewPageData` (`web/page_test.go`). `go build`, `go vet`, `go test ./...` all pass.

**Run metrics:** 5 driver call(s), $2.1638, session 08377e76-e26e-41e4-b1f3-c20608b2f584

**Run metrics:** 6 driver call(s), $3.4730, session e5c3b2c5-3524-463e-bdaa-f24e26b720ac

## 2026-09-21 — Task: Broaden homework filter and add prov marker

- `HomeworkEvents` (`event.go`) now also matches events whose `Description` contains "prov" (case-insensitive), in
  addition to the existing `Läxa` summary prefix, still within the same 7-day window; added a `provEvent` fixture and
  test case to `event_test.go`. Added `HomeworkRow.IsProv` (`web/page.go`), computed from
  `strings.Contains(strings.ToLower(e.Description), "prov")` for every homework row regardless of which filter branch
  matched; added test cases to `web/page_test.go`. Template heading changed to "Läxor och Prov" and rows now show a
  muted `(Prov)` marker when `IsProv` is true (`web/templates/page.html.tmpl`). `go build`, `go vet`, `go test ./...`,
  and `sensors/test.sh`/`sensors/lint.sh` all pass.

## 2026-09-21 — Task: Wire Kommande dagar end to end

- Added `UpcomingEvents(events, now)` (`event.go`): tomorrow-through-day+6 window (Europe/Stockholm), excluding
  homework/prov-matching events (checks duplicated inline, not shared with `HomeworkEvents`); added `TestUpcomingEvents`
  to `event_test.go`. Added `UpcomingRow`, `buildUpcomingRows`, and `svWeekdayAbbrev` (+ FuncMap registration) to
  `web/page.go`; `NewPageData` now takes a third `upcomingEvents` parameter and populates `PageData.Upcoming`; updated
  all `web/page_test.go` call sites and added `TestNewPageData_Upcoming`/`TestSvWeekdayAbbrev`. Replaced the static
  "Kommande dagar" placeholder rows in `web/templates/page.html.tmpl` with a data-driven `{{range .Upcoming}}` loop
  (empty-state message, abbreviated Swedish weekday, `svTime` for the heldag/HH:MM split).
  `cmd/family-dashboard/main.go` now computes `upcoming := dashboard.UpcomingEvents(events, time.Now())` and passes it
  through to `web.NewPageData`. Updated `memory/concepts/dashboard-page.md` and `memory/concepts/composition-root.md`.
  `go build`, `go vet`, `go test ./...`, and `sensors/test.sh`/`sensors/lint.sh` all pass.

**Run metrics:** 7 driver call(s), $2.5516, session ff047359-0ca8-4846-9d1f-035662a35a78

## 2026-09-21 — Task: Add Collector type and tests

- Added `collector.go` (`NamedSource`, `Collector`, `NewCollector`, `Owners()`, `Collect(ctx, now)`): merges
  owner-tagged events from multiple `EventSource`s, log-and-continue on a failing source, and derives
  today/homework/upcoming via the existing `event.go` filters — mirrors `main.go`'s current `ownerNames`/`handleIndex`
  logic but as a reusable domain type. Not yet wired into `cmd/family-dashboard/main.go` (follow-up task). Added
  `collector_test.go` (`fakeSource`, table-driven tests for owner merge/tagging, failing-source skip, parity with
  calling `TodayEvents`/`HomeworkEvents`/`UpcomingEvents` directly, and `Owners()` dedup/order). Added
  `memory/concepts/collector.md`. `go build`, `go vet`, `go test ./...`, and `sensors/test.sh`/`sensors/lint.sh` all
  pass.

## 2026-09-21 — Task: Rewire main.go to use Collector

- `cmd/family-dashboard/main.go`'s `main()` now builds each source's adapter once at startup (loop + `buildSource`,
  log-and-skip on error, same as before) into `[]dashboard.NamedSource`, then calls `dashboard.NewCollector(...)` once
  and passes the result to `handleIndex`. Removed the `ownerNames` free function — `Collector.Owners()` replaces it.
  `handleIndex(collector *dashboard.Collector)` closure now just calls `collector.Collect(r.Context(), time.Now())` for
  the three event slices and `collector.Owners()` for the owner list, then `web.NewPageData`/`web.Render` as before; no
  more per-request source loop or direct `TodayEvents`/`HomeworkEvents`/`UpcomingEvents`/`WithOwner` calls in main.go.
  Updated `memory/concepts/composition-root.md` and its `index.md` summary line to describe the startup-vs-request
  split. `go build`, `go vet`, `go test ./...`, and `sensors/test.sh`/`sensors/lint.sh` all pass.

**Run metrics:** 6 driver call(s), $2.1407, session 392df801-94f7-4389-adc4-a562424dedbf

## 2026-09-21 — Task: Exclude training keywords from UpcomingEvents

- Added package-level `excludedKeywords = []string{"träning"}` in `event.go` next to `homeworkPrefix`, and a second,
  independent `strings.Contains(strings.ToLower(e.Summary), kw)` loop/`continue` in `UpcomingEvents` (kept separate from
  the existing homework/prov check). Added `trainingEvent`/`cupEvent` fixtures and two `TestUpcomingEvents` cases
  ("training event in window is excluded", "cup event in window is included") to `event_test.go`. `go build`, `go vet`,
  `go test ./...` all pass; `gofmt -l` clean on both touched files.

**Run metrics:** 6 driver call(s), $1.9861, session d67ba3d3-f85b-4b29-80e8-56c997a201e1

## 2026-09-21 — Task 1: Restructure config to object root with school field

- `config/config.go` now defines `School{Name, RSSURL}` and `Config{Sources []Source, School School}`; `Load` returns
  `(Config, error)` instead of `([]Source, error)`. Updated `config/config_test.go` fixtures/assertions to the new
  `{"sources": [...], "school": {...}}` shape and `cfg.Sources`/`cfg.School`. Updated `sources.local.example.json` to
  wrap the source array under `"sources"` and added a `"school"` entry for "Nya Elementar". `cmd/family-dashboard/main.go`
  now loops over `cfg.Sources`. This is prep for a later task wiring the skolmaten lunch adapter (see plan.md). `go
  build`, `go test ./...` all pass.

## 2026-09-21 — Task 2: Add domain types and skolmaten adapter

- Added `lunch.go` (`package dashboard`): `LunchMenu{Dishes []string}` and `LunchSource` port
  (`Lunch(ctx) (LunchMenu, error)`), mirroring `Event`/`EventSource`. Added `skolmaten/skolmaten.go`: `Client`
  (url + `*http.Client`), `New(url, client) *Client`, and `Lunch(ctx)` implementing `dashboard.LunchSource` by
  fetching the RSS feed and parsing the first `<item>`'s `<description>` — dishes are split on literal `"<br/>"`
  text, trimmed of whitespace and a trailing comma; zero `<item>` elements yields empty dishes, not an error.
  Added `skolmaten/skolmaten_test.go` with a `stubRoundTripper` (same pattern as `webcal/sportadmin_test.go`)
  covering a 3-dish example feed, a zero-item feed, and an HTTP error. Not yet wired into `main.go`/`web.PageData`
  (next task). `go build`, `go test ./...`, `golangci-lint run ./...` all pass.

## 2026-09-21 — Task 3: Wire lunch into main.go, PageData, and template

- `cmd/family-dashboard/main.go`: after building the `Collector`, builds a `skolmaten.New(cfg.School.RSSURL,
  http.DefaultClient)` as `dashboard.LunchSource` if `cfg.School.RSSURL` is set; otherwise logs
  `slog.Info("no school lunch feed configured")` and leaves it nil. `handleIndex` now takes `lunchSource
  dashboard.LunchSource` alongside `collector`; when non-nil it calls `lunchSource.Lunch(r.Context())` and on error
  `slog.Error`-logs and continues with an empty dish list (same log-and-continue as collector source fetch
  failures), then passes the dish list to `web.NewPageData`. `web/page.go`: added `Lunch []string` field to
  `PageData` and a `lunch []string` parameter to `NewPageData` (after `owners`, before `now`), assigned straight
  through with no transformation (no display formatting needed, unlike events/homework/upcoming). Updated all
  `NewPageData` call sites in `web/page_test.go` (nil for existing cases) and added `TestNewPageData_Lunch`.
  `web/templates/page.html.tmpl`'s Skollunch section now renders `{{if not .Lunch}}` a muted "Ingen lunchmeny
  idag." row, else one `<div class="row">` per dish — no owner-initial prefix, since lunch is shared across kids,
  not per-owner. `go build`, `go test ./...`, `golangci-lint run ./...` all pass. This completes the 3-task
  plan.md lunch feature; Skolschema remains an untouched static placeholder.
- Removed the static Skolschema section (heading, placeholder note, two hardcoded schedule rows, flanking `<hr>`)
  and its now-unused `.placeholder-note` CSS from `web/templates/page.html.tmpl`; updated
  `memory/concepts/dashboard-page.md` to drop the stale "Skolschema/Skollunch remain static placeholders" note.
  `go build` and `go test ./...` pass.


**Run metrics:** 3 driver call(s), $1.4006, session a6495649-031a-46d5-9560-8694626d2214

## 2026-09-22 — Task: Wire SMHI weather into dashboard

- Added `weather.go` (`package dashboard`): `WeatherReading{Temperature float64}` and `WeatherSource` port
  (`Weather(ctx) (WeatherReading, error)`), mirroring `lunch.go`. Added `smhi/smhi.go`: `Client` (lat/lon +
  `*http.Client`), `New(lat, lon, client) *Client`, and `Weather(ctx)` implementing `dashboard.WeatherSource` by
  fetching the SMHI point-forecast JSON API and returning the first `timeSeries` entry's `air_temperature`; an
  empty `timeSeries` is an error (unlike skolmaten's empty-feed case, there's no sensible zero fallback). Added
  `smhi/smhi_test.go` (`stubRoundTripper` pattern) covering a realistic response, empty `timeSeries`, and an HTTP
  404. Added `config.Location{Lat, Lon float64}` + `Config.Location` field, updated `config/config_test.go` and
  `sources.local.example.json`. `cmd/family-dashboard/main.go` builds `smhi.New(...)` as a `dashboard.WeatherSource`
  when `cfg.Location.Lat != 0 || cfg.Location.Lon != 0` (else logs and leaves nil); `handleIndex` fetches per
  request, log-and-continue with a nil temperature on error, passes `*float64` through to `web.NewPageData`.
  `web/page.go`: added `PageData.Temperature *float64`, a `temperature *float64` param to `NewPageData` (after
  `lunch`, before `now`), and a `temperatureDisplay` FuncMap helper (rounds to nearest whole degree, `"—°"`
  fallback for nil). Updated all `NewPageData` call sites in `web/page_test.go`, added `TestNewPageData_Weather`
  and a weather case to `TestRender`. `web/templates/page.html.tmpl`'s weather block now branches on
  `{{if .Temperature}}` between live SMHI data and the original static fallback markup. Added
  `memory/concepts/smhi-adapter.md` and updated `composition-root.md`/`dashboard-page.md`/`index.md`. `go build`,
  `go vet`, `gofmt -l .`, `go test ./... -race`, and `sensors/lint.sh` all pass.


**Run metrics:** 4 driver call(s), $3.6079, session 1fc760cf-d518-441b-93cd-27c99733e1ac

## 2026-09-22 — Task: Extend weather end-to-end (low, sky, precipitation)

- `weather.go`: added `LowTemperature float64`, `Sunny bool`, `PrecipitationExpected bool` to `WeatherReading`;
  `WeatherSource.Weather` now takes a `now time.Time` param. `smhi/smhi.go`: added `SymbolCode`/`PrecipitationType`
  JSON fields to `timeSeriesEntry.Data`; `parseForecast(data, now)` parses each entry's RFC3339 `Time`, buckets
  entries whose Europe/Stockholm (UTC fallback) local date matches `now`'s into "today", and computes `Sunny` from
  the first entry's `SymbolCode` (1 or 2), `LowTemperature` as the min today `AirTemperature` (falls back to the
  first entry's temperature if no today entries match — e.g. `now` near midnight), and `PrecipitationExpected` from
  any today entry's `PrecipitationType != 0`. `smhi/smhi_test.go` expanded with distinct per-entry temperatures,
  symbol codes, and precipitation types, plus sunny/not-sunny, precipitation/no-precipitation, and near-midnight
  fallback cases. `cmd/family-dashboard/main.go` now passes `time.Now()` to `Weather` and threads a
  `*dashboard.WeatherReading` (not `*float64`) into `web.NewPageData`. `web/page.go`: `PageData.Temperature` →
  `PageData.Weather *dashboard.WeatherReading`; `NewPageData`'s `temperature` param → `weather`; `temperatureDisplay`
  dropped its nil-pointer branch (now `func(float64) string`, only called inside `{{if .Weather}}`).
  `web/templates/page.html.tmpl`'s weather block now branches on `{{if .Weather}}` and renders three extra lines
  (low temp, Sol/Mulet, regn/torrt idag) alongside the existing temperature/SMHI lines. Updated all `web/page_test.go`
  call sites and `TestNewPageData_Weather`/`TestRender`. Updated `smhi-adapter.md` and `dashboard-page.md`.
  `go build`, `go vet`, `gofmt -l .`, `go test ./... -race` all pass.


**Run metrics:** 7 driver call(s), $2.7488, session 5097678a-1e12-4cc6-a3f1-95095dec444b

## 2026-09-22 — Switch precipitation detection to probability

- `smhi/smhi.go`: `timeSeriesEntry.Data.PrecipitationType` (`predominant_precipitation_type_at_surface`) replaced with
  `ProbabilityOfPrecipitation int` (`probability_of_precipitation`) — the pcat field flagged "any" precipitation type
  as present (including trace/false-positive cases) with no magnitude signal. `parseForecast`'s today-entry loop now
  sets `PrecipitationExpected` when `ProbabilityOfPrecipitation >= precipitationProbabilityThreshold` (new
  package-level const, `30`). `smhi/smhi_test.go` updated: `exampleResponse`'s two today entries now use
  probabilities 3 (below threshold) and 50 (above), so the "example response" case now expects
  `wantPrecipitationExpected: true`; "no precipitation when today's entries all have pcat 0" renamed to "no
  precipitation when probability below threshold" (probability 3); added a boundary case at exactly 30 (inclusive,
  expects `true`). `go build`, `go vet`, `gofmt -l .`, sensors/test.sh, sensors/lint.sh all pass.

**Run metrics:** 9 driver call(s), $4.0144, session 1d998199-90dd-4791-ab20-ea6bd315e5f3

## 2026-09-22 — Add HighTemperature and daily precipitation total

- `weather.go`: added `HighTemperature float64` to `WeatherReading`, between `LowTemperature` and `Sunny`.
  `smhi/smhi.go`: `timeSeriesEntry.Data.ProbabilityOfPrecipitation int` replaced with `PrecipitationAmount float64`
  (`precipitation_amount_mean_deterministic`); `precipitationProbabilityThreshold` (30) replaced with
  `dailyPrecipitationThresholdMM = 0.2`. `parseForecast` now seeds `high`/`precipTotal` alongside `low` from the
  first entry, tracks `high` as the max today `AirTemperature` (mirroring `low`'s min), and sums each today entry's
  `PrecipitationAmount` into `precipTotal` (resetting on the first matching entry so the pre-loop seed isn't
  double-counted, same "seed as near-midnight fallback" trick as `low`/`high`); `PrecipitationExpected` is now
  `precipTotal >= dailyPrecipitationThresholdMM`. `smhi/smhi_test.go`: added `precipitation_amount_mean_deterministic`
  to `exampleResponse`'s entries and `wantHighTemp`/`HighTemperature` assertions; replaced the probability-boundary
  cases with amount-based ones ("all hours dry stays torrt", "sum crossing threshold triggers regn", "sum just below
  threshold stays torrat"). `web/templates/page.html.tmpl`: "Lägst idag" line now also shows "Högst idag" on the same
  row (middot separator). `web/page_test.go`: added `HighTemperature` to the weather fixtures and a
  `"H&ouml;gst idag"` (HTML-entity, matching the template's literal markup) assertion to `TestRender`. Updated
  `smhi-adapter.md` and `dashboard-page.md`. `go build`, `go vet`, `gofmt -l .`, `go test ./...` all pass.


**Run metrics:** 7 driver call(s), $4.9311, session 7d075ba2-a631-409d-9ddd-c317b83930b7

## 2026-11-10 — eink view-model

- Added `eink/view.go` (`NewView` → `View` of ready-to-draw Swedish strings: date/week/clock/owners header, `WeatherView`, lunch, `EventRow`/`HomeworkGroup`/`UpcomingRow`, shared heading and empty-state constants) and table-driven `eink/view_test.go`; `HomeworkGroup` also carries `Initial` so the renderer needn't derive it. `go vet`, `gofmt -l .`, `go test ./... -race`, `golangci-lint run` all pass.

## 2026-11-10 — eink 1-bit renderer

- Added the eink 1-bit renderer: `eink/layout.go` (all layout constants, `ClockRect()` at X 16–272 / Y 64–152), `eink/render.go` (`Render(View) *image.Gray` 800x480 thresholded frame, Go Mono/Go Mono Bold faces parsed once at init, per-column sub-image clipping, `fitText`/`fitParts` "…" cutting that keeps homework day labels and upcoming times, `limitRows` "+N till", low/high weather line wrapped at " · ") and table-driven `eink/render_test.go`; pinned `golang.org/x/image` v0.36.0 because v0.37.0–v0.45.0 require `go 1.25.0` and `go mod tidy` would rewrite our `go 1.25`.

## 2026-11-10 — shared fetch helper and -eink-png mode

- Extracted `fetchDashboard` (`dashboardData` struct; Collect + Owners + nil-safe lunch/weather with unchanged `slog.Error` messages, one `now` threaded through) from `handleIndex`; added `-eink-png <path>` one-shot export (`exportEinkPNG` → `eink.NewView`/`eink.Render` → `writePNG`, exit 0 on success before the server starts, `slog.Error` + exit 1 on failure, nothing on stdout) with table-driven `fetchDashboard`/`writePNG`/`buildAdapters` tests; updated `composition-root.md`. `gofmt`, `go vet`, `go test ./... -race`, `golangci-lint run` all pass.
- Deliberate deviation from the spec: extracted `buildAdapters(cfg)` (collector + lunch + weather construction) so the server and `-eink-png` share one wiring path; the "no school lunch feed configured" / "no weather location configured" Info logs now also fire on `-eink-png` runs.
- `writePNG` removes the file if `png.Encode` fails after `os.Create`, so no truncated PNG is left at the export path (covered by a 0x0-image test case).
- Manual check with a config of `{"sources": []}`: `go run ./cmd/family-dashboard -config $T/config.json -eink-png $T/frame.png` exited 0 with empty stdout and `file` reported `PNG image data, 800 x 480, 8-bit grayscale`; the same command with `-eink-png $T/missing/frame.png` logged `writing e-paper frame` and exited 1.

## 2026-11-10 — eink adapter memory and README

- Added `memory/concepts/eink-adapter.md` (renderer rules, `ClockRect` byte alignment, Go Mono/x/image pin, verified Waveshare 7.5" V2 + `epd7in5v2` v0.1.0 driver facts) with an index entry; `composition-root.md` now describes `handleIndex` via `fetchDashboard` and states that `-eink-png` exits without serving and leaves `-config` unchanged; README gains `eink/` + `-eink-png` in the Structure tree and a Phase 4 progress note. Deviation: the plan said x/image "v0.48+ needs Go 1.26"; the module cache shows `go 1.26.0` from v0.46.0 on (v0.37.0–v0.45.0 are `go 1.25.0`), so the concept file records v0.46.0+.


**Run metrics:** 27 driver call(s), $7.7441, session 4c69e1ba-bc0d-4cbb-9477-d72aea087cf4, cache read 5137752 / write 855373 / input 304 (share 86%)


**Run metrics:** 19 driver call(s), $5.1369, session 8c5df718-24dc-46e1-b48b-3a07490f94ba, cache read 3649921 / write 623177 / input 206 (share 85%)
