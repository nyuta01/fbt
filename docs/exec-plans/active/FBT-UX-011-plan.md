# FBT-UX-011 Make next commands copy-paste safe

## Observation

`plan`, `build`, and `artifact explain` printed useful `next` commands, but
those commands omitted `--project-dir` and `--state-dir`. When a user ran fbt
from outside the project root, the suggested command was not directly reusable.

## Decision

Kept planner-produced next steps project-agnostic, then added CLI invocation
context at the command boundary. This avoids coupling core planner semantics to
shell flags while making displayed commands copy-paste safe.

## Permanent Fix

Added CLI tests that assert `next` commands preserve `--project-dir` and
`--state-dir` for plan and artifact explain output. Updated README, CLI
reference, usage guide, and docs-site examples whose captured output includes
`--project-dir`.

## Next Check

Done. `go test ./internal/cli`, docs/harness checks, and `make verify` pass.
