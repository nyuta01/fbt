# FBT-MVP-010 Implement build lifecycle and idempotent commit

## Observation

Core has parser, manifest, planner, descriptors, state, runner discovery,
protocol client, and local runners, but `fbt build` still does not execute the
parse-plan-run-commit-state lifecycle.

## Decision

Implement the first build lifecycle:

- parse project files and write the manifest snapshot
- acquire the local state lock
- plan selected transforms
- invoke external runners through the protocol client
- require output candidates to stay under the invocation work directory
- compute descriptors, artifact version IDs, and artifact version records
- copy committed outputs to logical artifact paths
- update state current pointers and latest run fingerprints
- append invocation and transform-run summaries
- make re-running the same clean transform skip instead of corrupting state

## Permanent Fix

Added build lifecycle tests and CLI smoke coverage that run a protocol fake
runner, commit output candidates to logical artifact paths, record state and
artifact versions, and skip a clean second build.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
