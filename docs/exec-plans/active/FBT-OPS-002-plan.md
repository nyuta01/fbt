# FBT-OPS-002 Define Failed-Run Recovery And Failure-Focused Selection UX

## Observation

Failed build receipts are recorded, but day-to-day operation still needs a
clear way to inspect failed artifacts and rerun only the failed work without
turning fbt into a workflow engine.

## Decision

Define the smallest build-tool UX for failure recovery: inspection of failed
runs, failure-focused selection or retry semantics, and clear next commands
while keeping receipts append-only.

## Permanent Fix

Make failed-run recovery predictable for local and CI usage without adding job
queues, retries, schedulers, or approval workflow.

## Next Check

Specify CLI behavior, add conformance around failed-run selection/inspection,
and run `make verify`.
