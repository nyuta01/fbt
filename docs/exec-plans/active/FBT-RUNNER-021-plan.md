# FBT-RUNNER-021 Make official adapter modules remotely installable

## Observation

The official adapter design promises out-of-band installation such as
`go install github.com/nyuta01/fbt/adapters/openai/cmd/fbt-runner-openai@version`.
The current adapter modules depend on `github.com/nyuta01/fbt/sdk/go v0.0.0`
with a local `replace ../../sdk/go`, so remote `go install ...@main` fails
outside the source checkout.

## Decision

Choose a release layout that keeps the monorepo but makes official adapter
commands installable from a clean environment. The fix must not pull adapter
dependencies into fbt core and must keep `make verify` deterministic.

## Permanent Fix

Add a non-workspace install smoke for at least command and OpenAI adapters, and
document the exact supported install path. Keep local workspace convenience only
where it does not contradict remote installation.

## Next Check

Run clean-environment `go install` checks for official adapters, then
`make verify`.
