# AGENTS.md

Project conventions for AI agents working in this codebase.
Keep this file up to date — it is the agent's long-term memory.

## Stack
Language: go

## Patterns & Conventions
- (add project-specific patterns here)

## Gotchas
- (add things that have tripped agents up)

## Testing
- Sensors (sensors/test.sh, sensors/lint.sh) run automatically after every task; you
  do not need a task for this
- New code must have tests; do not delete tests to make coverage pass
- Use table-driven tests (see existing *_test.go files for examples)

## Style
- Follow the existing code style in the repo
- Prefer simple, readable code over clever code
