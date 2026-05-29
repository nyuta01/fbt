# FBT-DOCS-DRIFT-001 Remove Stale Core-Boundary References

Status: done
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

`docs/runner-protocol-spec.md` no longer says core owns approval state or docs
generation. It now describes core ownership as state, official commits, artifact
descriptors, lineage metadata, standard export inputs, and runner invocation.

`internal/README.md` now lists the current Cobra command surface and actual
internal package boundaries. Removed public commands such as `parse`, `eval`,
`docs`, `state`, and `runner` are no longer presented as active CLI surfaces.

`scripts/harness_drift.py` now has a targeted guard for the exact stale
current-state phrases that caused this regression. Historical exec plans and
failure-log notes remain allowed, but source-of-truth docs cannot reintroduce
the stale approval/docs/removed-command claims silently.

`docs/agent-failures.md` marks the repeated drift mode fixed with the permanent
guard.

## Next Check

Run:

```sh
make verify
```

Latest result: `make verify` passed. Current-state docs no longer imply
approval state, removed CLI commands, or custom docs/graph generation are part
of core.
