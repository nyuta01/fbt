# FBT-UNIX-016 Clarify Or Remove Plan Side Effects

Status: todo  
Owner: agent  
Updated: 2026-05-29

## Goal

Make `fbt plan` behavior predictable for users and scripts.

## Observation

`fbt plan` currently writes `.fbt/state/manifest.json`. That behavior is useful
because later commands can compare against a current manifest snapshot, but it
is surprising for a command named `plan`, which many users expect to be
read-only.

## Decision

Choose one of two Unix-friendly behaviors:

- make `fbt plan` read-only and move manifest writes to `doctor` or `build`; or
- keep the manifest write but document it explicitly as a local cache/update
  side effect.

The chosen behavior must be reflected in tests, smoke checks, and CLI docs.

## Permanent Fix

Planned:

- Inspect planner/build state dependencies on manifest snapshots.
- Decide whether read-only `plan` is practical without weakening dirty-state
  behavior.
- Update CLI docs, tests, smoke scripts, and conformance checks to match.

## Next Check

Run:

```sh
go test ./internal/cli ./internal/planner ./internal/build
make verify
```
