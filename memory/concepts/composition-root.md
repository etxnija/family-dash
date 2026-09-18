---
type: component
tags: [main, config, composition-root, multi-source, cmd]
summary: cmd/family-dashboard main() — JSON sources config, per-source error isolation, and owner tagging.
links: [docs/arch/adr/0001-config-path-via-required-flag.md, webcal-adapter.md]
updated: 2026-09-18
generated: { by: worker, at: 2026-09-18T00:00:00Z }
verified: [{ by: "human", at: 2026-09-18 }]
---

# Composition Root (`cmd/family-dashboard`)

## Context & Rationale
`main.go` wires an arbitrary number of calendar sources (not just SportAdmin)
into the domain. Source list and per-source secrets (ICS URLs) live in a
JSON config file rather than env vars or flags, per
[ADR-0001](../../docs/arch/adr/0001-config-path-via-required-flag.md); the
file's path is a required `-config` flag with no default.

## Key Rules & Mechanics
- `config.go` (`package main`) owns `sourceConfig` (`name`/`type`/`url` JSON
  fields), `loadSources(path) ([]sourceConfig, error)`, and
  `buildSource(cfg) (dashboard.EventSource, error)`. Kept in the same
  package as `main.go` rather than its own package — this is composition-root
  wiring, not a reusable component.
- `buildSource` switches on `cfg.Type`; `"sportadmin"` maps to
  `webcal.New`. Adding a second source type means adding a `case`, not a new
  package-level branch elsewhere.
- `loadSources` errors never include file contents — only the path and
  underlying cause — since the file holds secret URLs.
- `main()` loops over sources and treats each as independently failable: a
  `buildSource` or `.Events(ctx)` error is `slog.Error`-logged (source
  name/type only, never the URL) and the loop `continue`s to the next
  source, rather than aborting the whole run. One bad feed shouldn't blank
  the whole dashboard.
- Successfully fetched events are tagged via `dashboard.WithOwner(events,
  cfg.Name)` before being appended to the combined slice, so the final
  print loop can prefix each line with the owning family member.

## Gotchas & Failure Modes
- Only `loadSources` failure (missing/malformed config file) and timezone
  load failure are fatal (`os.Exit(1)`); missing `-config` flag is a usage
  error (`os.Exit(2)`, per ADR-0001). Individual source failures are not
  fatal by design.
