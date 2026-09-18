# AGENTS.md

Project conventions for AI agents working in this codebase.
Keep this file up to date — it is the agent's long-term memory.

## Stack
Language: go

## Patterns & Conventions
- (add project-specific patterns here)
- Domain/service logic lives in the root package (center of hexagon)
- Each adapter lives in its own package, named after what it adapts
- Ports (interfaces) are defined in the domain/service package, not the adapter
- One-way dependency: adapters import domain, never the reverse; `cmd/` wires both via DI
- Don't scaffold empty adapters/interfaces upfront — apply the structure as real integrations are added
- Use `log/slog` (stdlib) for structured logging — Info for normal operational events, Error for failures. Don't use `fmt.Println`/`fmt.Fprintln` for anything except the program's actual output.
- Structured logs go to stderr (`slog.NewJSONHandler(os.Stderr, nil)`); stdout is reserved for the program's real output, so the two never interleave.
- Log key facts as structured fields (counts, IDs, error causes), not just a bare message.

## Gotchas
- Domain code loads `Europe/Stockholm` via `time.LoadLocation`, which needs the
  IANA tzdata on the host/image. If we ever build a minimal/scratch container,
  add `_ "time/tzdata"` as a blank import so the zone is embedded.

## Testing
- Sensors (sensors/test.sh, sensors/lint.sh) run automatically after every task; you
  do not need a task for this
- New code must have tests; do not delete tests to make coverage pass
- Use table-driven tests (see existing *_test.go files for examples)

## Style
- Follow the existing code style in the repo
- Prefer simple, readable code over clever code
