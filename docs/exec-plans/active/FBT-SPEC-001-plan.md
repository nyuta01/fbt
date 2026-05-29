# FBT-SPEC-001 Reconcile Spec Statuses And Remaining Implementation Questions

## Observation

The core implementation has moved past the original draft baseline, but some
source-of-truth specs still say `Draft` or contain broad remaining-question
sections that now mix resolved items with real future work.

## Decision

Audit the core specs and reconcile status labels, remaining questions, and
future-work language with the current implemented MVP. Do not broaden scope in
the cleanup; convert any still-valid open concern into a concrete task instead.

## Permanent Fix

Keep specs as the source of truth by making status and remaining-question
sections mechanically consistent with implemented behavior and structured task
state.

## Next Check

Run docs validation and `make verify`; remaining questions should either be
current and intentionally future-facing or have matching backlog tasks.

