# FBT-RUNNER-012 Clarify runner adapter repository layout

## Observation

The top-level `runners/` directory mixed test fixtures, demo runners, command
adapters, and the optional OpenAI adapter. Even though they were outside
`internal/`, the layout made runner implementations look like part of fbt core
instead of external protocol-compatible commands.

## Decision

Removed the top-level `runners/` directory. Moved source-checkout adapter
examples to `examples/runner_adapters/` and the fake protocol fixture to
`tests/runner_fixtures/`.

## Permanent Fix

Updated templates, example wrappers, tests, conformance defaults, docs, and
feature-list paths so runner code is described as either example adapter code
or test fixture code, not core product code.

## Next Check

Done. `go test ./...`, runner conformance, docs checks, CLI smoke, and
`make verify` pass.
