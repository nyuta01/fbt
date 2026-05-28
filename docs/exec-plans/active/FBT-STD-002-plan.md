# FBT-STD-002 Implement Marquez-compatible OpenLineage export for artifact lineage

## Observation

fbt already records enough local state to explain artifact lineage, but the
lineage was only available through fbt-native CLI and docs surfaces. Users who
want graph visualization should not need a custom fbt UI when OpenLineage and
Marquez already define the standard ingestion and viewing path.

## Decision

Add `fbt export openlineage [--output PATH]` as an explicit export command. The
export emits OpenLineage-compatible `RunEvent` records as NDJSON, mapping fbt
transforms to jobs, transform runs to runs, source/artifact inputs to input
datasets, and artifact versions to output datasets. fbt-specific descriptor,
confidence, approval, eval, runner/model, and policy metadata stays in `fbt_`
custom facets.

## Permanent Fix

Added an `internal/lineage` exporter with deterministic event ordering and
UUIDv5-style run IDs derived from fbt transform run IDs. Wired the CLI command,
documented the implemented command surface, and added Go plus smoke coverage so
the export remains available from the source checkout.

## Next Check

Run:

```sh
make verify
```
