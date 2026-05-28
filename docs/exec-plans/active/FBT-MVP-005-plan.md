# FBT-MVP-005 Implement planner and dirty-state semantics

## Observation

The repository can parse resources, build a manifest graph, compute descriptors,
and persist local state, but it cannot yet decide which transforms should run,
which are clean, or which are blocked by review/confidence requirements.

## Decision

Implement a planner baseline:

- evaluate transform nodes from the current manifest
- compare current state latest-run fingerprints and output pointers
- compare a previous manifest when available for source, asset, policy, eval,
  runner, transform config, and model changes
- report dirty reasons deterministically
- report blocked reasons for missing inputs, required confidence, and required
  approval status
- support a preselected transform ID set for later CLI selector integration

## Permanent Fix

Added planner tests for missing outputs, clean skips, manifest-driven dirty
reasons, selected transform sets, and review/confidence blocking.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
