# FBT-STATE-005 Specify Archive-Safe Retention And Pruning Workflow

## Observation

`keep_all` is the safest MVP retention posture, but high-volume projects will
eventually need an explicit archive and pruning story that cannot corrupt
current pointers, lineage, or receipts.

## Decision

Specify archive-safe retention behavior before implementing destructive cleanup.
Prefer report/dry-run-first semantics and make the archive unit explicit.

## Permanent Fix

Define how state and immutable artifact history can be archived together, how
current versions are protected, and what conformance must prove before any prune
command exists.

## Next Check

Add docs and conformance for archive units, current pointer protection, and
dry-run/report semantics, then run `make verify`.
