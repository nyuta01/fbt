# FBT-STD-007 Add Opt-In Standard Backend Visualization Verification

Status: done
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

Added `make standard-backend-smoke`, backed by
`scripts/smoke-standard-backends.sh`. The target generates the support fixture,
builds artifacts, exports OpenLineage NDJSON and OTLP/JSON, and validates those
files locally.

The target posts to real backends only when explicit environment variables are
set:

- `FBT_MARQUEZ_URL` for Marquez `/api/v1/lineage`
- `FBT_OTLP_TRACES_URL` or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` for OTLP HTTP
  traces
- `FBT_STANDARD_EVIDENCE_DIR` to copy exports and a smoke summary for release
  or docs evidence

Docs and examples now point to the opt-in smoke and keep screenshots as a
backend-capture rule. Core still exposes only `fbt export openlineage` and
`fbt export otel`; no custom graph UI or required backend dependency was added.

## Next Check

Run:

```sh
make verify
```

Latest targeted result: `make standard-backend-smoke` passed without backend
variables, and evidence-copy mode passed with `FBT_STANDARD_EVIDENCE_DIR`.
Final gate: `make verify` passed with standard backend verification remaining
opt-in.
