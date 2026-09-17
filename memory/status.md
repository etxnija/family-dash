# Work Log

Session-to-session progress log. Distinct from AGENTS.md (stable rules).

---

## 2026-09-16 — project bootstrapped

### Done
- Bootstrapped with agent harness (lang: go)

### Next
- (add first task here)

## 2026-09-17 — LANG_PATTERNS task redirected

- Task "Add LANG_PATTERNS dict and {lang_patterns} placeholder" targeted `bootstrap/bootstrap.py`,
  which does not exist in this repo — it's the agent harness's own bootstrap tool at
  `/Users/nils/source/agent-work/bootstrap/bootstrap.py` (confirmed via plan.md's own
  Assumptions section). No changes were made in family-dash; the edit landed in agent-work instead.


**Run metrics:** 3 driver call(s), $0.7099, session c3bef2fc-7ab5-430c-b173-34eff4e36836


**Run metrics:** 8 driver call(s), $2.1093, session 42720221-5117-4679-b3ab-1fcdf18d2817

## 2026-09-17 — Task 1: domain types, today-filter, hexagonal layout

- Moved composition root from root `main.go` to `cmd/family-dashboard/main.go`; added root `package dashboard` with `Event`, `EventSource` port, and `TodayEvents` (Europe/Stockholm date filter) plus table-driven tests in `event_test.go`.

## 2026-09-17 — Task 2: SportAdmin ICS adapter with tests and composition-root wiring

- Added `sportadmin` package (`sportadmin.Client` implementing `dashboard.EventSource`) with a line-folding-aware ICS parser (`ParseICS`) handling TZID/bare/UTC/all-day timestamps and RFC 5545 text unescaping; fetch errors never leak the feed URL (`scrubURL` strips it from `*url.Error`). Wired `cmd/family-dashboard/main.go` to read `SPORTADMIN_ICS_URL`, fetch, filter to today, and print. Verified end-to-end against a local HTTP server serving `sportadmin/testdata/sample.ics`.
