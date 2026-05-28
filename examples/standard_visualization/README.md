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

## 4. Screenshot Rule

If you need a screenshot in docs or a runbook, capture it from Marquez, Jaeger,
Tempo, Grafana, or OpenMetadata after running the recipe above. Do not draw a
custom fbt graph and present it as product output.
