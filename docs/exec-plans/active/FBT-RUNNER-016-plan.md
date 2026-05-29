# FBT-RUNNER-016 Promote command runner into official adapter module

## Observation

The command runner is the most Unix-aligned integration and should be an
official adapter module rather than a source-checkout example. Keeping it under
`examples/runner_adapters` made it look like demo code even though practical
examples depend on it.

## Decision

Moved the command runner into `adapters/command` as a nested Go module with its
own command entrypoint, plugin manifest, README, tests, and conformance target.
The adapter uses the provider-free `sdk/go` helpers instead of fbt core
internals.

## Permanent Fix

`make verify` now runs command adapter tests and command adapter conformance.
Practical example wrappers invoke `go run ./adapters/command/cmd/fbt-runner-command`.

## Next Check

Done. `make verify` passes. The next task should promote the optional OpenAI
runner into `adapters/openai`.
