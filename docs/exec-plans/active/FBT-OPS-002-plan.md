# FBT-OPS-002 Define Failed-Run Recovery And Failure-Focused Selection UX

## Observation

Failed build receipts were recorded, but day-to-day operation still needed a
clear way to inspect failed transforms and rerun only the failed work without
turning fbt into a workflow engine.

## Decision

Define the smallest build-tool UX for failure recovery: inspection of failed
runs, failure-focused selection or retry semantics, and clear next commands
while keeping receipts append-only. Use explicit `plan --failed` and
`build --failed` commands rather than automatic retry loops.

## Permanent Fix

Make failed-run recovery predictable for local and CI usage without adding job
queues, automatic retries, schedulers, or approval workflow. The `state:failed`
selector and `--failed` flag read `state.json` latest-run status, add
`latest run failed` as a visible run reason, and append retry receipts without
mutating prior failed receipts.

## Next Check

`make verify` must continue to pass. Future recovery UX should keep retries as
explicit CLI invocations over local state rather than background orchestration.
