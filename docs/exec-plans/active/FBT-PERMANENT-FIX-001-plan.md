# FBT-PERMANENT-FIX-001 Maintain permanent-fix loop

## Observation

The task run did not reveal a repeated agent failure requiring a new permanent
guard. `docs/agent-failures.md` still has no active failures, and
`scripts/harness_drift.py` validates linked failure entries when they exist.

## Decision

Close the standing permanent-fix task for this execution pass by recording the
failure-log review and verifying that no active failure is unresolved.

## Permanent Fix

The permanent-fix loop remains guarded by `scripts/harness_drift.py`: future
failure entries must link to a known task and plan, use a valid status, and
describe a permanent fix when marked fixed.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
