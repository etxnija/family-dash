---
type: component
tags: [eink, e-paper, view-model, i18n, adapter, hexagonal, feed-notice]
summary: eink/view.go NewView/View — e-paper view-model of ready-to-draw Swedish strings, built from domain types independently of web/.
links: [dashboard-page.md, collector.md, smhi-adapter.md, skolmaten-adapter.md]
updated: 2026-11-10 (FeedNotice from dashboard.FeedProblem)
generated: { by: worker, at: 2026-11-10T00:00:00Z }
verified: [{ by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }]
---

# E-paper View-model (`eink/view.go`)

## Context & Rationale
ADR-0002 gives each display adapter its own view-model, so `eink` does not
import `web` and re-implements the Swedish maps and day-label/initial/
temperature helpers privately. Unlike `web.PageData` (English internally,
translated in the template FuncMap), `eink.View` is already Swedish and
fully formatted: the 1-bit renderer only lays out strings.

## Key Rules & Mechanics
- `NewView` takes the same arguments in the same order as `web.NewPageData`,
  plus `problems []dashboard.FeedProblem` between `weather` and `now`, and
  errors only if `Europe/Stockholm` can't be loaded; every time is
  converted to Stockholm before formatting (date, week, clock, weekdays).
- Swedish maps are keyed by `time.Weekday`/`time.Month`, not English
  strings, so there is no pass-through-on-miss path.
- `Owners` is a single `" · "`-joined string; `Weather` is a
  `WeatherView{Temperature, Lines}` — nil weather gives `"—°"` and
  `["ingen väderkälla"]`, otherwise `"SMHI"`, low/high, Sol/Mulet,
  regn/torrt idag.
- Homework grouping/ordering mirrors `web.buildHomeworkGroups` (owners
  order, unknown owners dropped, items sorted by Start, noon-to-noon day
  offset). `HomeworkGroup.Initial` holds the uppercased first letter for
  the row prefix. Event/upcoming `Time` is `"15:04"` or `"heldag"`.
- `View.FeedNotice` is one ready-to-draw row about failing calendar
  feeds, built by `buildFeedNotice(problems, now)`; `""` when there are no
  problems. The problems are the Collector's `dashboard.FeedProblem`s
  (see [collector.md](collector.md)): `Stale` means the feed is served
  from a cache at most `dashboard.StaleLimit` (24 h) old, otherwise the
  owner's events are missing. Format:
  `"Kalender saknas: K, L · Kalender gammal: K (3 h), L (40 min)"` —
  missing part first, parts joined with `" · "`, owners with `", "`.
  Owners are collapsed per kind in first-seen order and shown through
  `initial`. The age is `now - LastSuccess` rounded down: `"N min"` under
  one hour (at least `"1 min"`), else `"N h"`; an owner with several
  stale sources shows the oldest age. Prefixes and formats are the
  `feed*` constants in `view.go`.
- `View.Clock` comes from `clockText(t, loc)`, which the refresh loop
  (`eink/loop.go`) also uses to restamp a fetched view every minute
  without fetching again.
- Section headings (`headingLunch`, …) and empty-state texts (`emptyLunch`,
  …) are unexported constants in `view.go`, shared by the renderer and
  tests. Empty sections stay empty slices/nil in `View`; the renderer
  draws the matching empty-state constant.
- Rows stay one entry each (time/weekday/initial, owner, summary,
  location, day label); the portrait renderer splits them into
  prefix/body/suffix and wraps each to at most two lines itself, so the
  view-model needed no change for the wrapping layout.
  `EventRow.Owner`/`UpcomingRow.Owner` keep the full name; the renderer
  reduces it to an initial with `initial`.

## Gotchas & Failure Modes
- The heading/empty-state constants have no in-package caller until the
  renderer lands; small tests reference them so `unused` lint stays quiet.
- Changing a display rule in `web/page.go` does not propagate here (by
  design, ADR-0002) — update both if the two displays should agree.
