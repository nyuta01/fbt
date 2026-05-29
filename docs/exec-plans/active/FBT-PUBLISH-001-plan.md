# FBT-PUBLISH-001 Plan

## Task

Add external approval, publishing, and notification workflow examples.

## Observation

The production daily loop generated artifacts and evidence, but users still
needed a concrete handoff to PR review, publishing, and notifications that did
not imply fbt owns approval or publishing.

## Decision

Add a publish handoff script that creates a publish manifest, PR body, and
notification draft from the run bundle. Keep all actual review, merge,
notification, and knowledge-base publishing outside fbt.

## Permanent Fix

`daily-ops-smoke` now asserts the handoff files are created and explicitly state
that review and publishing happen outside fbt.

## Next Check

- `make daily-ops-smoke`
- `make verify`
