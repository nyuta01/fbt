# FBT-DOCS-DRIFT-001 Remove Stale Core-Boundary References

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Remove stale current-state references that contradict the simplified fbt product
boundary.

## Observation

The current docs mostly say fbt is not a review system and does not expose
debug/internal commands. A few stale references remain, including a runner
protocol line that says core owns approval state and an internal README line
that lists removed public commands.

## Decision

Clean current-state docs and add a small deterministic guard for removed
concepts where possible. Historical exec plans and failure logs may keep
superseded references when clearly marked as history.

## Permanent Fix

Pending. Expected permanent fix:

- Update `docs/runner-protocol-spec.md` and `internal/README.md`.
- Audit source-of-truth docs and docs site for current-state references to
  removed review/approval and debug command surfaces.
- Add a targeted grep/drift check that allows historical notes but rejects
  current-state drift.

## Next Check

Run:

```sh
make verify
```

Expected result: current-state docs no longer imply approval state, removed CLI
commands, or custom docs/graph generation are part of core.
