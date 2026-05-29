# FBT-STATE-004 Specify Orphaned Resource And Artifact Semantics

## Observation

Long-running projects will delete or rename sources, transforms, and artifacts.
The current local state preserves history, but the user-facing semantics for
orphaned current artifacts and historical versions are not explicit enough.

## Decision

Define how fbt reports artifacts whose producing transform or declaration no
longer exists, how history remains inspectable, and how standard exports treat
historical lineage without pretending the resource is still declared.

## Permanent Fix

Make deleted/renamed resource behavior explicit in state, CLI inspection, and
export semantics before adding any destructive cleanup workflow.

## Next Check

Add conformance for removed transform/artifact declarations and verify
`artifact show`, `artifact history`, and standard exports remain coherent.
