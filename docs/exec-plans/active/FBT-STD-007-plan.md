# FBT-STD-007 Add Opt-In Standard Backend Visualization Verification

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Back the standard visualization story with real backend evidence while keeping
base fbt service-free.

## Observation

fbt exports OpenLineage and OTLP/JSON and documents Marquez, Jaeger, Tempo,
Grafana, and OpenMetadata paths. The base checks validate payload shape and
redaction without starting services. That is correct for the core, but docs and
examples should eventually show screenshots captured from actual standard
backends, not custom diagrams or unverified mockups.

## Decision

Add opt-in backend recipes or scripts for local Marquez and OTLP ingestion.
Keep them outside `make verify` unless they can run deterministically without
heavy service requirements.

## Permanent Fix

Pending. Expected permanent fix:

- Add opt-in backend smoke instructions or scripts.
- Capture docs screenshots from real standard tools after ingestion.
- Keep `fbt export openlineage` and `fbt export otel` as the only core surface.

## Next Check

Run:

```sh
make verify
```

Expected result: standard backend verification is available on demand without
adding a custom fbt graph UI or a required service dependency.
