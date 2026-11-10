---
type: component
tags: [collector, domain, multi-source, owner-tagging, stale-cache, feed-problems]
summary: collector.go's Collector type — merges owner-tagged events from multiple EventSources, serves a failing source's cached events for up to 24 h (feed.go StaleLimit), and returns a Collection of today/homework/upcoming views plus FeedProblems.
links: [composition-root.md]
updated: 2026-11-10 (per-source stale cache, Collection, FeedProblem)
generated: { by: worker, at: 2026-09-21T00:00:00Z }
verified: [{ by: "human", at: 2026-09-21 }, { by: "human", at: 2026-11-10 }]
---

# Collector (`collector.go`)

## Context & Rationale
`cmd/family-dashboard/main.go`'s `handleIndex` closure originally did the
fetch-tag-merge-filter work itself, coupling that domain logic to the HTTP
handler. `Collector` extracts it into the domain package so it can be built
once and reused without an `http.Request` in scope; `main.go`'s
`fetchDashboard` calls it for every render.

## Key Rules & Mechanics
- `NamedSource{Owner string; Source EventSource}` is the exported pairing
  type passed to `NewCollector([]NamedSource) *Collector`. The unexported
  `source` struct is `Collector`'s internal storage — kept separate so the
  exported constructor input isn't tied to the internal field names.
- `NewCollector` derives `owners []string` the same way `main.go:ownerNames`
  does for `config.Source`: distinct `Owner` values, first-occurrence order.
  `Owners()` returns that slice.
- `Collect(ctx, now) Collection` (types in `feed.go`): `Collection{Today,
  Homework, Upcoming []Event; Problems []FeedProblem}`. For each source, in
  order, it calls `Events(ctx)` **outside** the lock, then takes
  `Collector.mu` (a `sync.Mutex`) only to read/update that source's entry in
  `cache` — a `[]cacheEntry{events, at}` parallel to `sources`, allocated in
  `NewCollector`. The cache is keyed by source index, not owner, so two
  sources with the same owner are cached independently.
  - **Success:** the entry becomes `{fetched, now}`; `fetched events` Info
    log; no problem.
  - **Failure, `at` set and `now.Sub(at) <= StaleLimit`** (`StaleLimit = 24 *
    time.Hour`, boundary inclusive): the cached events are used and
    `FeedProblem{Owner, Stale: true, LastSuccess: at}` is appended.
  - **Failure otherwise:** the source contributes no events;
    `FeedProblem{Owner, Stale: false, LastSuccess: at}` — `LastSuccess` is
    zero if the source never succeeded. The expired cache is kept (not
    cleared), so `LastSuccess` stays reported.
  - Every failure logs `slog.Error("fetching events", "source", owner,
    "error", err, "serving_stale", bool, "last_success", RFC3339 or "")` —
    owner name only, never the URL.
- `Problems` is in source order and `nil` when every source succeeded. The
  views are derived from the merged events with `TodayEvents`/
  `HomeworkEvents`/`UpcomingEvents`, with the same per-stage `slog.Info`
  counts as before. `Collector` is safe for concurrent `Collect` calls
  (HTTP handler + tests under `-race`); with concurrent calls the last one
  to take the lock wins the cache entry.

## Gotchas & Failure Modes
- The cache is in memory only. A restart while a feed is dead shows
  `Stale=false` (zero `LastSuccess`) at once — nothing survives to serve.
- Staleness is measured against the `now` passed to `Collect`, not the wall
  clock, so callers must pass a consistent `now` (tests use explicit ones).
