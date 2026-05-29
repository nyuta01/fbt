# FBT-STD-008 Clarify standard export command UX

## Observation

Standard exports are valuable because fbt avoids a custom visualization backend,
but the previous `--output` summaries did not make the standard format,
destination file, record count, or backend handoff obvious.

## Decision

Kept export commands Unix-friendly: stdout stays raw records for piping, while
`--output` now prints a concise human summary that names the standard format,
output file, record count, and next backend handoff.

## Permanent Fix

Added CLI tests, smoke checks, and docs examples for the standard export
summaries and the stdout-versus-file behavior.

## Next Check

Done. `go test ./internal/cli`, docs checks, and `make verify` pass.
