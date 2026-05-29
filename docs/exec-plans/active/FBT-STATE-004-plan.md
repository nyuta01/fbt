# FBT-STATE-004 Specify Orphaned Resource And Artifact Semantics

## Observation

Long-running projects will delete or rename sources, transforms, and artifacts.
The local state preserved history, but the user-facing semantics for orphaned
current artifacts and historical versions were not explicit enough.

## Decision

Define how fbt reports artifacts whose producing transform or declaration no
longer exists, how history remains inspectable, and how standard exports treat
historical lineage without pretending the resource is still declared.

## Permanent Fix

Deleted/renamed resource behavior is explicit in state, CLI inspection, and
export semantics before any destructive cleanup workflow exists. Recorded
artifact versions whose current declaration is gone are marked orphaned in
`artifact show` / `artifact history` and JSON output, while OpenLineage emits
orphaned artifact-version events with material inputs when available.

## Next Check

Done. Unit, CLI, and conformance coverage remove transform declarations after a
successful build and verify `artifact show`, `artifact history`, JSON output,
and OpenLineage export remain coherent. `make verify` passed.
