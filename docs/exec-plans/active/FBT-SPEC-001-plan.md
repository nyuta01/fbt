# FBT-SPEC-001 Reconcile Spec Status And Post-MVP Boundaries

## Observation

The source-of-truth specs still carried `Draft` status labels and broad
remaining-question sections after the MVP implementation, schema generation,
runner conformance, security profile, and adapter work had landed.

## Decision

Reconcile the core specs to `MVP-ready` where implementation and verification
exist, replace stale open-question lists with concrete post-MVP boundaries, and
leave scope-expanding work outside core unless it gets a dedicated task.

## Permanent Fix

The stale status and remaining-question language has been removed from core
specs, `docs/QUALITY_SCORE.md` no longer points at this completed cleanup task,
`validate-docs` now checks the replacement post-MVP boundary section, and
`make verify` runs the harness guard that prevents quality-score references to
completed tasks from reappearing.

## Next Check

`make verify` must pass after the task is marked done; future spec questions
should be added as bounded tasks or kept as explicit post-MVP boundaries.
