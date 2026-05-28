# FBT-STD-000 Capture lineage standard export backlog

## Observation

Current lineage is stored in fbt-native JSON files: `manifest.json`,
`state.json`, `artifact_versions.json`, run results, eval results, approvals,
and policy decisions. These files include `metadata.fbt_schema_version`, but
they are not OpenLineage, OpenTelemetry, or OpenMetadata records.

Repository research already positions OpenLineage as the first natural lineage
export target, OpenTelemetry as a trace and GenAI telemetry reference, and
OpenMetadata as a likely catalog integration target rather than an internal
state model. Visualization should follow those ecosystems where possible:
Marquez for OpenLineage graphs, OTel-compatible backends such as Jaeger, Tempo,
or Grafana for traces, and OpenMetadata for catalog/governance views if that
mapping proves useful.

## Decision

Keep fbt-native state as the internal source of truth and add standard
integration as explicit export work:

- define export contracts and command surface first
- implement Marquez-compatible OpenLineage export for artifact lineage
- add OpenTelemetry-compatible execution telemetry export without making network
  export part of the default local runtime
- evaluate OpenMetadata catalog export after OpenLineage mapping exists
- add deterministic conformance fixtures for exported records and redaction
- document how to use standard visualization backends instead of building a
  custom fbt graph UI

## Permanent Fix

Added `FBT-STD-001` through `FBT-STD-006` to
`docs/exec-plans/feature-list.json` with dependencies, paths, verification
expectations, and standard-specific notes. Updated `AGENT_PROGRESS.md` and
`docs/QUALITY_SCORE.md` so the standards-export gap is visible outside chat.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
