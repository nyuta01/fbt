# FBT-UNIX-003 Define Stable Source-Window Operations

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Document and guard the recommended repeated-operation pattern for daily or
high-volume source growth without turning fbt into a scheduler or watermark
engine.

## Observation

The daily QA example used stable inbox paths, but the docs did not clearly
separate fbt's responsibility from upstream ingestion. Users could expect fbt
to decide date windows, readiness, partitions, or new-items-only semantics.

## Decision

Keep fbt's contract simple: declared source paths are stable, fbt fingerprints
the resolved file set and content, and external systems prepare the current
window before fbt runs.

## Permanent Fix

- Added daily-operation patterns to the usage guide and examples index.
- Clarified source-window semantics in the project config spec.
- Expanded the daily QA example with stable-path, cumulative, new-items-only,
  and readiness-check guidance.
- Added practical smoke coverage that appends a new question file under the
  stable inbox and asserts `fbt plan` becomes dirty because the source
  descriptor changed.

## Next Check

Run:

```sh
make verify
```

Expected result: practical examples smoke proves source file-set growth makes
the dependent transform dirty.
