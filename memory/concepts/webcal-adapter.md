---
type: component
tags: [webcal, sportadmin, infomentor, ics, adapter, hexagonal]
summary: webcal ICS calendar adapter (SportAdmin and InfoMentor sources) — parsing rules, timezone handling, and URL-secrecy discipline.
links: [AGENTS.md]
updated: 2026-09-18
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-17 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }]
---

# webcal ICS Adapter

## Context & Rationale
`webcal` is the first real adapter in the hexagon (`webcal/sportadmin.go`),
implementing `dashboard.EventSource` by fetching and parsing an ICS calendar
feed. It sets the pattern later adapters should follow. The package is named
`webcal` after the adapter *type* (ICS/webcal feeds in general); SportAdmin
was the first concrete source and InfoMentor is the second — both are plain
ICS feeds, so both are built via the same `webcal.New` constructor from
`cmd/family-dashboard/main.go`'s `buildSource` (one `case` per source type,
same client construction). The file is still named `sportadmin.go` since it
predates the second source, but error messages now use a source-agnostic
`"webcal:"` prefix rather than naming SportAdmin specifically. A source with
feed quirks the others don't share would get its own file (e.g.
`webcal/otherfeed.go`) alongside this one.

## Key Rules & Mechanics
- `New(url, *http.Client)` takes the client as a parameter so tests inject a
  stub `http.RoundTripper` instead of hitting the network.
- `ParseICS` is a standalone function, not a method, so tests can feed it
  fixture bytes directly without constructing a `Client`.
- ICS line folding (RFC 5545 §3.1) is unfolded before parsing: continuation
  lines start with a single space or tab.
- SportAdmin's feed uses **local Stockholm time**, not UTC: both
  `DTSTART;TZID=Europe/Stockholm:...` and the bare `20260917T180000` format
  (no trailing `Z`) are parsed as Europe/Stockholm. `parseICSTime` normalizes
  both branches (the `Z` branch and the local-time branch) to UTC via `.UTC()`
  before returning — this is a cross-adapter contract: every `EventSource`
  adapter must return UTC-normalized `Event.Start`/`End`, and callers convert
  to a display timezone only at the presentation layer (see
  `cmd/family-dashboard/main.go`).
- All-day events (`DTSTART;VALUE=DATE:...`) are skipped silently (no error,
  no event emitted) — the dashboard only needs to-the-minute events.
- **The feed URL must never appear in an error message** (it embeds a private
  subscription token). `scrubURL` strips the URL out of net/http's
  `*url.Error` before wrapping it, since `url.Error.Error()` otherwise
  includes the full request URL.

## Gotchas & Failure Modes
- A VEVENT missing `DTSTART`, an unparsable timestamp, or an unterminated
  `BEGIN:VEVENT`/`END:VEVENT` pair are treated as malformed input and return
  an error. A feed with zero VEVENT blocks is *not* an error — it's a valid
  empty result.
- If wrapping any error from `http.Client.Do` or `NewRequestWithContext`
  without scrubbing, the URL (and thus the token) leaks into logs — always
  route those errors through `scrubURL` first.
