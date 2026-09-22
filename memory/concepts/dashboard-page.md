---
type: component
tags: [web, page, template, i18n, funcmap, receipt-design, lunch, weather]
summary: web/page.go PageData/Render — receipt-style monochrome template, Swedish display translations via FuncMap, data-driven Skollunch and weather panels.
links: [composition-root.md, skolmaten-adapter.md, smhi-adapter.md]
updated: 2026-09-22 (Weather field replaces Temperature; panel shows low/sky/precipitation)
generated: { by: worker, at: 2026-09-21T00:00:00Z }
verified: [{ by: "human", at: 2026-09-22 }, { by: "human", at: 2026-09-22 }]
---

# Dashboard Page (`web/page.go` + `web/templates/page.html.tmpl`)

## Context & Rationale
The dashboard renders as a monochrome, receipt-style printout (Courier New,
black-on-off-white sheet with corner marks) rather than a colorful card
layout. `PageData` and `buildEventRows`/`buildHomeworkGroups` stay in
English internally (`Weekday`, `DateLabel`, `DayLabel`, etc.) — translation
to Swedish happens only at render time, in the template, via FuncMap
helpers. This keeps `NewPageData`'s Go-side tests (`TestNewPageData`,
`TestNewPageData_Homework`) asserting English values unchanged while the
HTML output is Swedish.

## Key Rules & Mechanics
- `PageData` carries `WeekNumber int` (`localNow.ISOWeek()`) and
  `CurrentTime string` (`localNow.Format("15:04")`) alongside the existing
  `Weekday`/`DateLabel` fields, all computed once in `NewPageData`.
- Six unexported translation functions in `page.go` are registered in the
  template's `template.FuncMap`, not called from Go code directly:
  `svWeekday`, `svDateLabel`, `svDayLabel`, `svTime`, `ownerInitial`,
  `svWeekdayAbbrev`. Each passes through unrecognized input unchanged (e.g.
  `svWeekday` on an already-Swedish or unknown string) rather than erroring,
  since templates have no error-return path.
- "Kommande dagar" is a data-driven loop like Idag/Homework:
  `buildUpcomingRows` sorts `dashboard.UpcomingEvents` output by `Start` and
  flattens into `UpcomingRow` (`Weekday`, `Time`, `Owner`, `Summary`,
  `Location`); the template abbreviates `Weekday` via `svWeekdayAbbrev`
  (e.g. `"Wednesday"` → `"Ons"`) and reuses `svTime` to render `"heldag"`
  for all-day events instead of an HH:MM time.
- `svDayLabel` covers both the two fixed English phrases
  (`homeworkDayLabel`'s `"Due today"`/`"Due tomorrow"`) and its weekday-name
  fallback, by reusing the `svWeekdays` map lowercased.
- `svDateLabel` splits `DateLabel` (`"2 January"` format) on the first
  space and translates only the month-name half, so the day number passes
  through untouched.
- The template has no per-owner color coding (monochrome design) — homework
  rows use `ownerInitial` (first-letter, e.g. "K"/"L") instead of the
  `ownerColor`/`--kaj`/`--lillian` CSS custom properties used by the
  schedule panel's owner chips in the prior card-based design.
- The `dict` FuncMap helper and `{{define "comingUpCard"}}` sub-template
  were removed together: `dict` existed only to pass named args into that
  sub-template. "Kommande dagar" started as static placeholder markup (like
  Skollunch) but was later wired to `dashboard.UpcomingEvents` (see above).
- `HomeworkRow.IsProv` is computed independently of which branch of
  `dashboard.HomeworkEvents`'s filter matched (summary prefix vs.
  description) — it's always `strings.Contains(strings.ToLower(e.Description), "prov")`,
  so a "Läxa " event with "prov" in its description still gets the marker.
  The template renders it as a muted `(Prov)` suffix after `.Summary`.

- "Skollunch" is data-driven like "Kommande dagar": `PageData.Lunch []string`
  is a plain dish-name list from `dashboard.LunchMenu.Dishes` (see
  [skolmaten-adapter.md](skolmaten-adapter.md)) passed straight through by
  `NewPageData`'s new `lunch []string` parameter (positioned after `owners`,
  before `now`) with no per-row transformation — unlike Events/Homework/
  Upcoming there's no `buildXRows` helper, since a dish is already a display
  string. The template renders `{{if not .Lunch}}` a muted "Ingen
  lunchmeny idag." row (same empty-state convention as other sections),
  else one `<div class="row">{{.}}</div>` per dish — no owner-initial
  prefix, since lunch is a single shared feed, not per-kid.

- The weather panel is data-driven like Skollunch: `PageData.Weather
  *dashboard.WeatherReading` (see [smhi-adapter.md](smhi-adapter.md)) is
  passed straight through by `NewPageData`'s `weather
  *dashboard.WeatherReading` parameter (positioned after `lunch`, before
  `now`) with no transformation — the whole reading struct is handed to the
  template rather than flattened, unlike Events/Homework/Upcoming. The
  template's `{{if .Weather}}` branch is true only for a non-nil pointer (a
  genuine 0.0°C reading still renders, since Go templates treat pointer
  truth as nil-ness, not pointed-to zero-value); it calls
  `temperatureDisplay` (now `func(float64) string` — the nil check was
  dropped since it's only ever invoked inside the `{{if .Weather}}` guard,
  so the argument is always valid) on `.Weather.Temperature`,
  `.Weather.LowTemperature`, and `.Weather.HighTemperature`, plus a static
  `"SMHI"` source label and two more `{{if}}`-branched lines:
  `{{if .Weather.Sunny}}Sol{{else}}Mulet{{end}}` and
  `{{if .Weather.PrecipitationExpected}}regn idag{{else}}torrt idag{{end}}`.
  The `{{else}}` branch keeps the original static fallback markup
  (`&mdash;&deg;` / "ingen väderkälla") unchanged.

## Gotchas & Failure Modes
- Adding a new day-label or date value that doesn't map into `svWeekdays`
  or `svMonths` silently passes through as English rather than failing —
  intentional (templates can't propagate errors), but means a typo in a
  map key degrades to English text rather than a build/test failure.
- `ownerColor` is still registered in the FuncMap and defined in `page.go`,
  but as of this redesign nothing in `page.html.tmpl` calls it — left in
  place since the task scope only called for removing `dict`.
