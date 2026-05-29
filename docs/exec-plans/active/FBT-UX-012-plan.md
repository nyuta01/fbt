# FBT-UX-012 Show committed artifact paths after build

## Observation

`build` reported success, run ID, and committed version, but did not show the
logical output path. Users often want to open or inspect the generated file
immediately after a successful build.

## Decision

Kept `artifact show` as the detailed inspection command, but included each
committed artifact path and a contextual `fbt artifact show TARGET` next command
in human build output.

## Permanent Fix

Exposed committed artifact metadata from the build result and added CLI
coverage for the output path plus next inspection command. Updated README, CLI
reference, and docs-site quickstart output.

## Next Check

Done. `go test ./internal/build ./internal/cli`, docs/harness checks, and
`make verify` pass.
