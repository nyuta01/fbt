# FBT-STATE-006 Plan

## Task

Add a CI archive and retention handoff reference for production daily
operation.

## Observation

Retention inspection showed the correct archive unit for `.fbt/state` and
`.fbt/artifacts`, but the production daily loop also produces run-bundle
evidence under `target/ops`. Users needed a concrete handoff for storing and
restoring all evidence together.

## Decision

Keep pruning outside core. Add an ops archive script that packages
`.fbt/state`, `.fbt/artifacts`, and the specific `target/ops/runs/<run-id>`
bundle, plus a JSON manifest that states restore expectations and prune safety
flags.

## Permanent Fix

`daily-ops-smoke` now asserts the archive, archive manifest, and tar contents
exist for the production wrapper.

## Next Check

- `make daily-ops-smoke`
- `make verify`
