# FBT-OPS-003 Plan

## Task

Add a production-shaped daily support knowledge reference loop without
expanding fbt core into scheduling, review, publishing, or runner execution.

## Observation

The `daily_qa_ops` example already proves stable source windows and multiple
artifacts, but a production user still has to infer how to combine fbt with
source readiness checks, CI, retention inspection, standard exports, and
external approval/publishing workflows.

## Decision

Keep fbt focused on the build control plane. Add an ops wrapper under the
example that runs `doctor`, `plan`, `build`, artifact inspection, retention,
OpenLineage export, and OTel export into a CI-friendly run bundle. Document
that ingestion, scheduling, approval, publishing, and notifications remain
outside fbt.

## Permanent Fix

`scripts/smoke-daily-ops.sh` runs the new ops wrapper against a copied example
and asserts the run bundle, lineage export, trace export, and artifact explain
evidence exist. This prevents the production reference loop from drifting into
documentation-only guidance.

## Next Check

- `make daily-ops-smoke`
- `make verify`
