# Standard Visualization Example

fbt does not ship a graph UI. Use this example when you want to see fbt output
in standard lineage or trace tools.

## 1. Create Export Files

Use the offline support template so the commands work without provider
credentials:

```sh
fbt init /tmp/fbt-viz-knowledge --template support
fbt build --project-dir /tmp/fbt-viz-knowledge --select case_summaries
fbt build --project-dir /tmp/fbt-viz-knowledge --select weekly_support_insights

mkdir -p /tmp/fbt-viz-knowledge/target/lineage
mkdir -p /tmp/fbt-viz-knowledge/target/telemetry

fbt export openlineage \
  --project-dir /tmp/fbt-viz-knowledge \
  --output /tmp/fbt-viz-knowledge/target/lineage/openlineage.ndjson

fbt export otel \
  --project-dir /tmp/fbt-viz-knowledge \
  --output /tmp/fbt-viz-knowledge/target/telemetry/otel.json
```

Quick local checks:

```sh
wc -l /tmp/fbt-viz-knowledge/target/lineage/openlineage.ndjson
python3 -m json.tool /tmp/fbt-viz-knowledge/target/telemetry/otel.json >/dev/null
```

The same export fixture is automated by:

```sh
make standard-backend-smoke
```

Without backend environment variables, the target only verifies that standard
export files are generated and valid. It does not start services.

## 2. Send OpenLineage To Marquez

Start Marquez by following the Marquez project instructions, then post each
OpenLineage event:

```sh
MARQUEZ_URL=http://localhost:5000
while IFS= read -r event; do
  curl -sS -X POST "$MARQUEZ_URL/api/v1/lineage" \
    -H 'Content-Type: application/json' \
    --data "$event" >/dev/null
done < /tmp/fbt-viz-knowledge/target/lineage/openlineage.ndjson
```

Open the Marquez UI and search for:

```text
transform.knowledge_ops.case_summaries
transform.knowledge_ops.weekly_support_insights
artifact.knowledge_ops.case_summaries
```

## 3. Send OTLP/JSON To Jaeger Or A Collector

When an OTLP HTTP endpoint is listening:

```sh
curl -sS -X POST http://localhost:4318/v1/traces \
  -H 'Content-Type: application/json' \
  --data-binary @/tmp/fbt-viz-knowledge/target/telemetry/otel.json
```

Then open Jaeger, Tempo, or Grafana and search for service `fbt`.

## 4. Opt-In Backend Smoke

Use the opt-in smoke target when Marquez or an OTLP HTTP endpoint is already
running:

```sh
FBT_MARQUEZ_URL=http://localhost:5000 \
FBT_OTLP_TRACES_URL=http://localhost:4318/v1/traces \
FBT_STANDARD_EVIDENCE_DIR=/tmp/fbt-standard-evidence \
make standard-backend-smoke
```

Variables:

| Variable | Meaning |
|---|---|
| `FBT_MARQUEZ_URL` | Marquez base URL or full `/api/v1/lineage` URL. |
| `FBT_OTLP_TRACES_URL` | OTLP HTTP traces endpoint, usually `/v1/traces`. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Alternative OTLP traces endpoint name. |
| `FBT_STANDARD_EVIDENCE_DIR` | Optional directory for copied exports and `smoke-summary.txt`. |

The target posts fbt OpenLineage events to Marquez and fbt OTLP/JSON traces to
the configured OTLP endpoint. It remains outside `make verify`.

## 5. Screenshot Rule

If you need a screenshot in docs or a runbook, capture it from Marquez, Jaeger,
Tempo, Grafana, or OpenMetadata after running the recipe above. Do not draw a
custom fbt graph and present it as product output.
