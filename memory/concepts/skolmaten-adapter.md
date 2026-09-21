---
type: component
tags: [skolmaten, lunch, rss, adapter, hexagonal]
summary: lunch.go's LunchMenu/LunchSource port and the skolmaten RSS adapter that fetches+parses the school lunch feed.
links: [webcal-adapter.md, composition-root.md]
updated: 2026-09-21 (wired into main.go/handleIndex)
generated: { by: worker, at: 2026-09-21T00:00:00Z }
verified: []
---

# skolmaten Lunch Adapter (`lunch.go`, `skolmaten/skolmaten.go`)

## Context & Rationale
The school lunch menu is a different shape of data than a calendar `Event`:
no owner, no start/end time, one shared feed for the whole family rather
than per-kid sources. Per ADR-0002, it still lives in the `dashboard`
package (one domain package collects all dashboard data) but as its own
type family — `LunchMenu`/`LunchSource` in `lunch.go`, separate from
`Event`/`EventSource` in `event.go`. `skolmaten` is a new adapter package,
named after the external system (skolmaten.se), following the same
constructor/test pattern as [webcal-adapter.md](webcal-adapter.md).

## Key Rules & Mechanics
- `dashboard.LunchMenu{Dishes []string}` and `dashboard.LunchSource`
  (`Lunch(ctx context.Context) (LunchMenu, error)`) are the port; adapters
  implement it the same way `EventSource` adapters implement `Events`.
- `skolmaten.New(url string, client *http.Client) *Client` mirrors
  `webcal.New` — the `*http.Client` is a constructor param so tests inject a
  stub `http.RoundTripper` instead of hitting the network.
- `Client.Lunch(ctx)` fetches the URL via HTTP GET, unmarshals the RSS XML
  (`rssFeed{Channel.Items []rssItem}`, `rssItem{Description string}`) with
  `encoding/xml`, and parses the first item's `Description`.
- The feed's `<description>` CDATA holds dish names separated by **literal**
  `"<br/>"` text (not real XML markup) — `parseDishes` splits on that
  string, then trims whitespace and a single trailing comma from each
  piece, dropping empty results.
- Zero `<item>` elements in the feed is not an error — `parseRSS` returns
  `LunchMenu{}` (nil `Dishes`), same log-and-continue philosophy as an empty
  ICS feed in `webcal`. The adapter assumes at most one `<item>` per feed;
  only `Items[0]` is read.

## Gotchas & Failure Modes
- Wired into `cmd/family-dashboard/main.go`: `main()` builds a
  `skolmaten.New(cfg.School.RSSURL, http.DefaultClient)` as a
  `dashboard.LunchSource` if `cfg.School.RSSURL` is non-empty, else logs
  `slog.Info` and leaves it `nil` — same log-and-skip philosophy as a
  missing/failing calendar source. `handleIndex` calls `Lunch(r.Context())`
  per request when non-nil, `slog.Error`-logs and continues with an empty
  dish list on failure (see [composition-root.md](composition-root.md)).
- Unlike `webcal`, the skolmaten feed URL has no known secret-token
  convention documented yet, so `skolmaten`'s error messages don't scrub the
  URL the way `webcal.scrubURL` does. Revisit if the real feed URL turns out
  to embed a credential.
