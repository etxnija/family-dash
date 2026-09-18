# ADR-0001: Config file location via required CLI flag

## Status

Accepted

## Context

family-dashboard needs to know where its local sources config file (listing
SportAdmin/InfoMentor subscriptions per family member) lives. That path isn't
itself secret, but a silent default risks the program quietly reading a stale
or wrong file after a change on the host. The usual 12-factor "put config in
the environment" advice mainly exists to keep secrets out of process
arguments, which are visible to others via `ps` — that concern doesn't apply
to this path value, since the actual sensitive URLs live inside the config
file itself, not in this flag.

## Decision

The config file's path is passed as a required `-config` command-line flag.
If it's omitted, the program prints usage and exits non-zero immediately,
rather than falling back to a hardcoded default path or reading it from an
environment variable.

## Consequences

- The program's required input is visible via `-h`/`--help` and fails fast
  and loudly if misconfigured, instead of silently running against the wrong
  file.
- Deploying later (e.g. on the Raspberry Pi) needs a wrapper script or
  systemd unit line supplying `-config=<path>`, itself possibly populated
  from an environment variable at that layer — not a concern for local
  development today.
- The source list and its URLs still live inside the config file, not as
  flags or env vars: lists don't map cleanly onto flat env vars, and a
  git-ignored file is a reasonable home for secrets, arguably better than
  command-line args, which leak into `ps aux` output.
