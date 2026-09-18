---
type: component
tags: [main, config, composition-root, multi-source, cmd]
summary: cmd/family-dashboard main() — JSON sources config, per-source error isolation, and owner tagging.
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md]
updated: 2026-09-18
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }, { by: "human", at: 2026-09-18 }]
---

# Composition Root (`cmd/family-dashboard`)

## Context & Rationale
`main.go` wires an arbitrary number of calendar sources (not just SportAdmin)
into the domain. Source list and per-source secrets (ICS URLs) live in a
JSON config file rather than env vars or flags, per
[ADR-0001](../../docs/arch/adr/0001-config-path-via-required-flag.md); the
file's path is a required `-config` flag with no default.

## Key Rules & Mechanics
- Config parsing lives in `config/config.go` (`package config`), not in
  `cmd/family-dashboard`: exported `Source` (`Name`/`Type`/`URL` JSON
  fields) and `Load(path) ([]Source, error)`. It imports only
  `encoding/json`, `fmt`, `os` — no dependency on `family-dashboard` or
  `family-dashboard/webcal`, since parsing shouldn't need to know about
  adapters.
- `buildSource(cfg config.Source) (dashboard.EventSource, error)` lives in
  `cmd/family-dashboard/main.go` itself, not in the `config` package —
  it wires a parsed `config.Source` to a concrete adapter (`webcal.New`),
  which is composition-root work, not config parsing. Adding a second
  source type means adding a `case` to `buildSource`, not a new
  package-level branch elsewhere.
- `config.Load` errors never include file contents — only the path and
  underlying cause — since the file holds secret URLs.
- `main()` loops over `config.Load`'s sources and treats each as
  independently failable: a `buildSource` or `.Events(ctx)` error is
  `slog.Error`-logged (source name/type only, never the URL) and the loop
  `continue`s to the next source, rather than aborting the whole run. One
  bad feed shouldn't blank the whole dashboard.
- Successfully fetched events are tagged via `dashboard.WithOwner(events,
  cfg.Name)` before being appended to the combined slice, so the final
  print loop can prefix each line with the owning family member.
- Tests split along the same boundary: `config/config_test.go` (`package
  config_test`) covers `Load`; `cmd/family-dashboard/main_test.go`
  (`package main`) covers `buildSource`.

## Gotchas & Failure Modes
- Only `config.Load` failure (missing/malformed config file) and timezone
  load failure are fatal (`os.Exit(1)`); missing `-config` flag is a usage
  error (`os.Exit(2)`, per ADR-0001). Individual source failures are not
  fatal by design.
