# FBT-UNIX-016 Clarify Or Remove Plan Side Effects

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make `fbt plan` behavior predictable for users and scripts.

## Observation

`fbt plan` wrote `.fbt/state/manifest.json`. That behavior came from the old
public `fbt parse` command: after `parse` was removed from the public surface,
its manifest-write behavior effectively remained in `plan`. This made a
preview command mutate local state.

## Decision

Make `fbt plan` read-only. `build` already owns write operations and still
writes the manifest snapshot used for later dirty-state comparison.

## Permanent Fix

- Removed `ctx.Store.WriteManifest(ctx.Manifest)` from `runPlan`.
- Updated CLI tests to assert `plan` does not write `manifest.json`.
- Updated smoke and dist checks to expect no manifest after `plan`, and a
  manifest after `build`.
- Updated CLI, usage, design, core spec, and state docs to describe read-only
  `plan` behavior.

## Next Check

Run:

```sh
make verify
```

Expected result: all checks pass with `plan` read-only.
