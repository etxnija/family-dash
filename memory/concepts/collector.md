---
type: component
tags: [collector, domain, multi-source, owner-tagging]
summary: collector.go's Collector type — merges owner-tagged events from multiple EventSources and derives the today/homework/upcoming views.
links: [composition-root.md]
updated: 2026-09-21
generated: { by: worker, at: 2026-09-21T00:00:00Z }
verified: [{ by: "human", at: 2026-09-21 }]
---

# Collector (`collector.go`)

## Context & Rationale
`cmd/family-dashboard/main.go`'s `handleIndex` closure originally did the
fetch-tag-merge-filter work itself, coupling that domain logic to the HTTP
handler. `Collector` extracts it into the domain package so it can be built
once (like `ownerNames`) and reused without an `http.Request` in scope. This
task only adds the type and its tests; `main.go` still does the work inline
until a later task wires `Collector` in.

## Key Rules & Mechanics
- `NamedSource{Owner string; Source EventSource}` is the exported pairing
  type passed to `NewCollector([]NamedSource) *Collector`. The unexported
  `source` struct is `Collector`'s internal storage — kept separate so the
  exported constructor input isn't tied to the internal field names.
- `NewCollector` derives `owners []string` the same way `main.go:ownerNames`
  does for `config.Source`: distinct `Owner` values, first-occurrence order.
  `Owners()` returns that slice.
- `Collect(ctx, now) (today, homework, upcoming []Event)` fetches each
  source, `slog.Error`-logs and skips (continue) a source whose `Events`
  call fails rather than aborting, tags successes with `WithOwner`, merges
  them, and returns `TodayEvents`/`HomeworkEvents`/`UpcomingEvents` on the
  merged slice — same log-and-continue and per-stage `slog.Info` count
  logging as the current `handleIndex` closure.

## Gotchas & Failure Modes
- Not yet wired into `cmd/family-dashboard/main.go` — `handleIndex` still
  builds sources and filters events inline. Wiring it in is a follow-up
  task, not done here.
