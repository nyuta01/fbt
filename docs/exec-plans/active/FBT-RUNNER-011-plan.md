# FBT-RUNNER-011 Simplify runner terminology and onboarding

## Observation

The runner boundary was powerful but terminology was spread across runner,
adapter, protocol, conformance, smoke, and provider examples. First-time users
needed one mental model before they read author-facing details.

## Decision

Presented all execution integrations as "external commands that speak the fbt
runner protocol." Kept adapter/protocol/conformance terminology available for
runner authors, but routed user-facing docs through the simpler command mental
model.

## Permanent Fix

Updated README, CLI help/reference, runner docs, runner protocol intro, and the
docs site so provider examples and agent tools use the same runner command
framing.

## Next Check

Done. Docs checks, runner conformance, and `make verify` pass.
