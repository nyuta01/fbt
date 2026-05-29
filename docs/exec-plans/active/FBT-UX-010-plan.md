# FBT-UX-010 Make common CLI errors actionable

## Observation

Some common CLI failures were technically correct but left the next action
implicit. A declared artifact with no built version said "artifact not found",
empty selectors did not suggest discovery commands, and `--dry-run` did not
point users to the read-only `plan` command.

## Decision

Kept strict failure semantics and exit codes, but added short `Hint:` lines for
common user mistakes. No aliases were added and unsupported flags remain hard
failures.

## Permanent Fix

Added CLI tests that exercise declared-but-unbuilt artifacts, empty selectors,
and `--dry-run` so the actionable diagnostics stay stable. JSON errors also
carry hint fields where they flow through the shared error printer.

## Next Check

Done. `go test ./internal/cli`, docs/harness checks, and `make verify` pass.
