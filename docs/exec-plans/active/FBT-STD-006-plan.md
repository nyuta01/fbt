# FBT-STD-006 Document standard visualization integrations

## Observation

fbt now exports OpenLineage and OTel-compatible payloads, but users still need
clear guidance for viewing those files in standard tools without expecting fbt
core to host a graph UI or trace backend.

## Decision

Document visualization as external integration recipes: Marquez for
OpenLineage lineage graphs, Jaeger for OTLP/JSON trace inspection, and
Tempo/Grafana through an OpenTelemetry Collector or Grafana Alloy pipeline.
Keep OpenMetadata documented as reserved pending the catalog evaluation task.

## Permanent Fix

Added `docs/standard-visualization-guide.md` and linked it from the README,
CLI reference, usage guide, and runnable knowledge-loop example. The guide
includes export commands, ingestion examples, redaction expectations, and
troubleshooting checks while keeping fbt core lightweight.

## Next Check

Run:

```sh
make verify
```
