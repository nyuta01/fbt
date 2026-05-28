# FBT-STD-001 Define standard lineage, telemetry, and visualization export contracts

## Observation

fbt-native manifest and state files contain lineage and telemetry material, but
there was no stable contract for mapping them to OpenLineage, OpenTelemetry, or
OpenMetadata without replacing the internal schemas or building a custom graph
UI.

## Decision

Keep fbt-native state as the source of truth. Define standard integrations as
explicit export commands: OpenLineage first for artifact lineage and Marquez
visualization, OpenTelemetry for execution traces/telemetry, and OpenMetadata
through OpenLineage ingestion unless a direct catalog export proves necessary.

## Permanent Fix

Added `docs/standard-export-spec.md` and linked it from README, manifest,
state/run-results, runner protocol, CLI reference, and usage docs. The contract
defines command surface, JSON envelope rules, OpenLineage/OTel/OpenMetadata
mappings, redaction rules, visualization non-goals, and conformance fixture
requirements.

## Next Check

Run:

```sh
make verify
```
