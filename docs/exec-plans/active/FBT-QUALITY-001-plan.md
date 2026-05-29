# FBT-QUALITY-001 Guard Quality Score Next-Task References Against Completed Tasks

## Observation

`docs/QUALITY_SCORE.md` is useful as a lightweight risk ledger, but its
`Next Task` column can drift by pointing at tasks that are already done.

## Decision

Add a deterministic guard or docs check that validates task IDs referenced from
the quality score. References to completed tasks should either be removed,
changed to `post-MVP`, or replaced with an open task.

## Permanent Fix

Update the quality score and harness so stale completed-task references fail
locally before they become misleading handoff state.

## Next Check

Run the new quality-score check and `make verify`.

