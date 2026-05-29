# FBT-UX-014 Group doctor runner diagnostics for scanability

## Observation

`fbt doctor` already checked project config, local state, runner command
availability, and runner protocol initialization, but the human output was a
flat list. Projects with multiple runners made it hard to see which readiness
diagnostic belonged to which area or runner.

## Decision

Kept `doctor --json` unchanged as the automation surface. Grouped only the
human output into Project, State, and Runners sections, with runner diagnostics
nested under the runner name.

## Permanent Fix

Added CLI coverage and docs-site examples for grouped human output while
preserving existing diagnostic codes and JSON structure.

## Next Check

Done. `go test ./internal/cli`, docs checks, `scripts/smoke-cli.sh`, and
`make verify` pass.
