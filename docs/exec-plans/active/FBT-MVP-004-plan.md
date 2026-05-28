# FBT-MVP-004 Implement local state store, locks, and atomic writes

## Observation

Manifest and descriptor primitives exist, but there is no local state backend
for writing manifest snapshots, artifact version indexes, eval
results, policy decisions, current-state snapshots, run results, or locks.

## Decision

Implement a local filesystem state store:

- create the `.fbt/state` directory on demand
- write JSON snapshot files through temp-file plus atomic rename
- append JSON Lines records to `run_results.jsonl`
- acquire and release `.lock` files with invocation metadata
- detect active and stale locks
- provide typed helpers for state, artifact versions, evaluation
  results, and policy decisions

## Permanent Fix

Added state store tests covering atomic JSON snapshots, manifest writes,
append-only JSONL run results, lock contention/release, stale lock replacement,
and immutable/idempotent artifact version records.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
