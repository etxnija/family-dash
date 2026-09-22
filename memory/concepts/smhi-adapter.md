---
type: component
tags: [smhi, weather, adapter, hexagonal]
summary: weather.go's WeatherReading/WeatherSource port and the SMHI point-forecast adapter that fetches+parses today's temperature, low, sky condition, and precipitation.
links: [skolmaten-adapter.md, composition-root.md, dashboard-page.md]
updated: 2026-09-22 (extended with LowTemperature/Sunny/PrecipitationExpected, Weather now takes now time.Time)
generated: { by: worker, at: 2026-09-22T00:00:00Z }
verified: [{ by: "human", at: 2026-09-22 }]
---

# SMHI Weather Adapter (`weather.go`, `smhi/smhi.go`)

## Context & Rationale
Weather is a single shared reading for the whole family (like lunch, unlike
per-kid `Event`s): no owner, just a point-in-time temperature plus today's
forecast. It lives in the `dashboard` package as its own type family —
`WeatherReading`/`WeatherSource` in `weather.go`, separate from
`Event`/`EventSource` and `LunchMenu`/`LunchSource` — following the same
port-per-concern split as [skolmaten-adapter.md](skolmaten-adapter.md).
`smhi` is a new adapter package, named after the external system (SMHI, the
Swedish meteorological institute), using the same constructor/test pattern
as `skolmaten`/`webcal`.

## Key Rules & Mechanics
- `dashboard.WeatherReading{Temperature, LowTemperature float64; Sunny,
  PrecipitationExpected bool}` and `dashboard.WeatherSource`
  (`Weather(ctx context.Context, now time.Time) (WeatherReading, error)`)
  are the port. `now` is needed because "today" (for the low/precipitation
  fields) is derived from it, not from wall-clock time inside the adapter —
  keeps `parseForecast` pure/testable.
- `smhi.New(lat, lon float64, client *http.Client) *Client` mirrors
  `skolmaten.New`/`webcal.New` — the `*http.Client` is a constructor param so
  tests inject a stub `http.RoundTripper` instead of hitting the network.
- `Client.Weather(ctx, now)` builds the SMHI point-forecast URL from
  `lon`/`lat` (`.../geotype/point/lon/%f/lat/%f/data.json`), fetches it, and
  unmarshals the JSON body into a local `forecast{TimeSeries
  []timeSeriesEntry}` shape (`timeSeriesEntry{Time string; Data
  struct{ AirTemperature float64; SymbolCode int
  `json:"symbol_code"`; PrecipitationType int
  `json:"predominant_precipitation_type_at_surface"` }}`).
- `parseForecast(data, now)`: `Temperature` and `Sunny` come from
  `TimeSeries[0]` only (the soonest forecast point) — `Sunny` is
  `SymbolCode == 1 || SymbolCode == 2` (SMHI's "clear sky"/"nearly clear
  sky" codes). `LowTemperature` and `PrecipitationExpected` are derived from
  "today" entries: each entry's `Time` (RFC3339) is parsed and compared, in
  `Europe/Stockholm` (falling back to UTC if the zone fails to load), to
  `now`'s local date. `LowTemperature` is the min `AirTemperature` among
  today entries; `PrecipitationExpected` is true if any today entry has
  `PrecipitationType != 0`. If no entry's local date matches `now`'s (e.g.
  `now` is near midnight and the forecast's first entries are already
  "tomorrow"), `LowTemperature` falls back to `TimeSeries[0]`'s temperature
  and `PrecipitationExpected` stays false — same "no sensible per-day data,
  use the soonest point" philosophy as the original first-entry-only logic.
- An empty `timeSeries` array is still an error (unlike skolmaten's
  empty-feed case) — there's no sensible zero-value reading to fall back
  to, so `parseForecast` returns an error rather than a zero
  `WeatherReading`.
- Error messages use the `"smhi:"` prefix, same convention as `"skolmaten:"`
  and `"webcal:"`.

## Gotchas & Failure Modes
- Wired into `cmd/family-dashboard/main.go`: `main()` builds a
  `smhi.New(cfg.Location.Lat, cfg.Location.Lon, http.DefaultClient)` as a
  `dashboard.WeatherSource` only if `cfg.Location.Lat != 0 ||
  cfg.Location.Lon != 0`, else logs `slog.Info` and leaves it `nil` — same
  log-and-skip philosophy as a missing lunch feed or calendar source.
  `handleIndex` calls `Weather(r.Context(), time.Now())` per request when
  non-nil, `slog.Error`-logs and continues with a `nil` weather on failure
  (see [composition-root.md](composition-root.md)).
- `config.Location{Lat, Lon float64}` defaults to the zero value
  `{0, 0}` when absent from the config file, which is why "configured" is
  tested as "either coordinate is non-zero" rather than "Location is
  non-zero-value" — `{0,0}` isn't a plausible real location (it's the
  Gulf of Guinea), so it doubles as the "unconfigured" sentinel.
- `web.PageData.Weather *dashboard.WeatherReading` is nil when no weather
  source is configured or the fetch failed; the template's `{{if .Weather}}`
  branch is false only for a nil pointer, not for a pointed-to zero value —
  a genuine 0.0°C reading still renders. `web.temperatureDisplay` (now
  `func(float64) string`, no nil check — only ever called inside
  `{{if .Weather}}`) rounds to the nearest whole degree (`math.Round`) and
  formats as `"11°"`; the no-weather fallback is the template's own static
  markup (`&mdash;&deg;` / "ingen väderkälla"), not a function return value.
