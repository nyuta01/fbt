# FBT-SOURCE-001 Plan

## Task

Define source-window readiness and reprocessing semantics for daily support
knowledge operation.

## Observation

The daily loop had a `_READY` marker, but the marker did not say which source
window was prepared, whether the run represented new items, cumulative
evidence, corrections, deletions, or a backfill, or what minimum source files
were expected before fbt should start.

## Decision

Keep ingestion and date partitioning outside fbt. Treat `_READY` as a small
JSON handoff manifest and validate it in the example ops wrapper before any fbt
command runs. fbt still fingerprints stable source paths and records artifact
receipts; ingestion owns the meaning of the processing window.

## Permanent Fix

`examples/daily_qa_ops/ops/check-source-window.py` validates the readiness
manifest, source path containment, and minimum file counts. `daily-ops-smoke`
asserts the validation report is present in the production run bundle.

## Next Check

- `make daily-ops-smoke`
- `make verify`
