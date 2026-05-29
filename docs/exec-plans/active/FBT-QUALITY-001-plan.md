# FBT-QUALITY-001 Guard Quality Score Next-Task References Against Completed Tasks

## Observation

`docs/QUALITY_SCORE.md` is useful as a lightweight risk ledger, but its
`Next Task` column can drift by pointing at tasks that are already done.

The quality table had already accumulated completed-task references, which made
the next-action signal less reliable than the structured feature list.

## Decision

Add a deterministic guard or docs check that validates task IDs referenced from
the quality score. References to completed tasks should either be removed,
changed to `post-MVP`, or replaced with an open task.

Put the guard in `scripts/harness_check.py` so it runs at the front of
`make verify` with the rest of the repository harness checks.

## Permanent Fix

Update the quality score and harness so stale completed-task references fail
locally before they become misleading handoff state.

`harness-check` now parses `docs/QUALITY_SCORE.md`, validates referenced
`FBT-*` task IDs, rejects unknown IDs, rejects completed-task references, and
requires low scores to name an open task. The methodology now documents that
rule.

## Next Check

`make verify` passed with the new quality-score guard.
