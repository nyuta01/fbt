# FBT-UX-013 Make human artifact details scannable

## Observation

`artifact show` printed semantic descriptors as dense one-line JSON, and
`artifact retention` reported raw byte counts. Those values are useful for
machines but noisy in default human output.

## Decision

Kept `--json` complete and exact. For human output, semantic descriptor counts
are summarized and retention sizes are shown in human-readable units.

## Permanent Fix

Added CLI tests and smoke/conformance expectations for the new human summaries
so dense JSON and raw byte labels do not return accidentally.

## Next Check

Done. `go test ./internal/cli`, docs/harness checks, and `make verify` pass.
