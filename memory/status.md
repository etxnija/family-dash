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
  (empty-state message, abbreviated Swedish weekday, `svTime` for the heldag/HH:MM split). `cmd/family-dashboard/main.go`
  now computes `upcoming := dashboard.UpcomingEvents(events, time.Now())` and passes it through to `web.NewPageData`.
  Updated `memory/concepts/dashboard-page.md` and `memory/concepts/composition-root.md`. `go build`, `go vet`,
  `go test ./...`, and `sensors/test.sh`/`sensors/lint.sh` all pass.


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
  independent `strings.Contains(strings.ToLower(e.Summary), kw)` loop/`continue` in `UpcomingEvents` (kept separate
  from the existing homework/prov check). Added `trainingEvent`/`cupEvent` fixtures and two `TestUpcomingEvents`
  cases ("training event in window is excluded", "cup event in window is included") to `event_test.go`. `go build`,
  `go vet`, `go test ./...` all pass; `gofmt -l` clean on both touched files.
