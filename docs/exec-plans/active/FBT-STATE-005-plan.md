# FBT-STATE-005 Specify Archive-Safe Retention And Pruning Workflow

## Observation

`keep_all` is the safest MVP retention posture, but high-volume projects needed
an explicit archive and pruning story that cannot corrupt current pointers,
lineage, or receipts.

## Decision

Specify archive-safe retention behavior before implementing destructive cleanup.
Prefer report/dry-run-first semantics and make the archive unit explicit. Keep
destructive cleanup out of MVP.

## Permanent Fix

Define how state and immutable artifact history can be archived together, how
current versions are protected, and what conformance must prove before any prune
command exists. The retention report now exposes `archive_unit:
state_and_artifacts`, archive roots, protected current-version IDs,
`prune_supported: false`, and `dry_run_required: true`.

## Next Check

`make verify` must continue to pass. Any future destructive prune command must
be explicit, dry-run-first, receipt-aware, current-pointer-protecting, and
conformance-covered before it removes files.
