# FBT-STATE-002 Define Local State And Artifact Retention Hygiene

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Define the smallest safe answer for long-running local projects whose artifact
history grows every day.

## Observation

Immutable artifact versions are central to fbt's value. Daily projects that
process many source files will accumulate `.fbt/artifacts`, run results,
evaluation results, and policy decisions. The current design explains
immutability but does not define retention, archive, or pruning behavior.

## Decision

Design retention as local state hygiene, not as a metadata database, artifact
store, scheduler, or hosted service. The default should remain safe and
inspectable. Any destructive cleanup must be explicit and receipt-aware.

## Permanent Fix

Pending. Expected permanent fix:

- Decide whether the MVP needs only docs, a dry-run cleanup command, or a
  state/archive export pattern.
- Preserve current artifact pointers and lineage for retained versions.
- Add conformance coverage before any destructive operation is exposed.

## Next Check

Run:

```sh
make verify
```

Expected result: high-volume users get a clear retention story without weakening
artifact immutability.
