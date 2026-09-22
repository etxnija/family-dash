---
type: component
tags: [smhi, weather, adapter, hexagonal]
summary: weather.go's WeatherReading/WeatherSource port and the SMHI point-forecast adapter that fetches+parses the current temperature.
links: [skolmaten-adapter.md, composition-root.md, dashboard-page.md]
updated: 2026-09-22 (initial version, wired into main.go/PageData/template)
generated: { by: worker, at: 2026-09-22T00:00:00Z }
verified: [{ by: "human", at: 2026-09-22 }]
---

# SMHI Weather Adapter (`weather.go`, `smhi/smhi.go`)

## Context & Rationale
Weather is a single shared reading for the whole family (like lunch, unlike
per-kid `Event`s): no owner, just a point-in-time temperature. It lives in
the `dashboard` package as its own type family — `WeatherReading`/
`WeatherSource` in `weather.go`, separate from `Event`/`EventSource` and
`LunchMenu`/`LunchSource` — following the same port-per-concern split as
[skolmaten-adapter.md](skolmaten-adapter.md). `smhi` is a new adapter
package, named after the external system (SMHI, the Swedish
meteorological institute), using the same constructor/test pattern as
`skolmaten`/`webcal`.

## Key Rules & Mechanics
- `dashboard.WeatherReading{Temperature float64}` and `dashboard.WeatherSource`
  (`Weather(ctx context.Context) (WeatherReading, error)`) are the port.
- `smhi.New(lat, lon float64, client *http.Client) *Client` mirrors
  `skolmaten.New`/`webcal.New` — the `*http.Client` is a constructor param so
  tests inject a stub `http.RoundTripper` instead of hitting the network.
- `Client.Weather(ctx)` builds the SMHI point-forecast URL from `lon`/`lat`
  (`.../geotype/point/lon/%f/lat/%f/data.json`), fetches it, and unmarshals
  the JSON body into a local `forecast{TimeSeries []timeSeriesEntry}` shape
  (`timeSeriesEntry{Time string; Data struct{ AirTemperature float64 }}`).
  Only `TimeSeries[0]` (the soonest forecast point) is used.
- An empty `timeSeries` array is an error (unlike skolmaten's empty-feed
  case) — there's no sensible zero-value temperature to fall back to, so
  `parseForecast` returns an error rather than a zero `WeatherReading`.
- Error messages use the `"smhi:"` prefix, same convention as `"skolmaten:"`
  and `"webcal:"`.

## Gotchas & Failure Modes
- Wired into `cmd/family-dashboard/main.go`: `main()` builds a
  `smhi.New(cfg.Location.Lat, cfg.Location.Lon, http.DefaultClient)` as a
  `dashboard.WeatherSource` only if `cfg.Location.Lat != 0 ||
  cfg.Location.Lon != 0`, else logs `slog.Info` and leaves it `nil` — same
  log-and-skip philosophy as a missing lunch feed or calendar source.
  `handleIndex` calls `Weather(r.Context())` per request when non-nil,
  `slog.Error`-logs and continues with a `nil` temperature on failure (see
  [composition-root.md](composition-root.md)).
- `config.Location{Lat, Lon float64}` defaults to the zero value
  `{0, 0}` when absent from the config file, which is why "configured" is
  tested as "either coordinate is non-zero" rather than "Location is
  non-zero-value" — `{0,0}` isn't a plausible real location (it's the
  Gulf of Guinea), so it doubles as the "unconfigured" sentinel.
- `web.PageData.Temperature *float64` is nil when no weather source is
  configured or the fetch failed; the template's `{{if .Temperature}}`
  branch is false only for a nil pointer, not for a pointed-to zero value —
  a genuine 0.0°C reading still renders. `web.temperatureDisplay` rounds to
  the nearest whole degree (`math.Round`) and formats as `"11°"`; the
  no-data fallback is `"—°"` and the template's own static fallback markup
  (`&mdash;&deg;` / "ingen väderkälla") mirrors that string exactly.
