# fbt Standard Visualization Guide

Status: MVP-ready  
Created: 2026-05-28  
Audience: users visualizing fbt lineage and execution telemetry with standard
tools

## 1. Scope

fbt exports standard-compatible files and keeps visualization outside core. The
base CLI does not start Marquez, OpenMetadata, an OpenTelemetry Collector,
Jaeger, Tempo, Grafana, or any other service.

Use:

```sh
fbt export openlineage --output target/lineage/openlineage.ndjson
fbt export otel --output target/telemetry/otel.json
```

The first file is OpenLineage RunEvent NDJSON for artifact, job, and dataset
lineage. The second file is an OTLP/JSON trace payload for execution telemetry.

## 2. Reproducible Local Export

Create a support fixture and export both standard files:

```sh
fbt init /tmp/fbt-viz-knowledge --template support
fbt build --project-dir /tmp/fbt-viz-knowledge --select case_summaries
fbt build --project-dir /tmp/fbt-viz-knowledge --select weekly_support_insights
mkdir -p /tmp/fbt-viz-knowledge/target/lineage /tmp/fbt-viz-knowledge/target/telemetry
fbt export openlineage \
  --project-dir /tmp/fbt-viz-knowledge \
  --output /tmp/fbt-viz-knowledge/target/lineage/openlineage.ndjson
fbt export otel \
  --project-dir /tmp/fbt-viz-knowledge \
  --output /tmp/fbt-viz-knowledge/target/telemetry/otel.json
```

Quick checks:

```sh
wc -l /tmp/fbt-viz-knowledge/target/lineage/openlineage.ndjson
python3 -m json.tool /tmp/fbt-viz-knowledge/target/telemetry/otel.json >/dev/null
```

The repository also provides an opt-in smoke target:

```sh
make standard-backend-smoke
```

With no backend variables set, it creates the fixture above and validates the
OpenLineage and OTLP/JSON files locally. It does not start Marquez, Jaeger,
Tempo, Grafana, OpenMetadata, or an OpenTelemetry Collector.

When documentation needs an image, capture it from the actual standard backend
after ingesting these files. Do not use a custom fbt-drawn graph as a substitute
for backend output.

## 3. Marquez / OpenLineage

Use OpenLineage when you want to inspect:

- which transform produced an artifact
- which input sources or artifacts fed that transform
- run IDs, job names, dataset names, confidence, evals, policy decisions, and fbt
  descriptor facets

Generate the export:

```sh
fbt export openlineage --output target/lineage/openlineage.ndjson
```

Marquez accepts OpenLineage events through its lineage API. For a local Marquez
API endpoint:

```sh
MARQUEZ_URL=http://localhost:5000
while IFS= read -r event; do
  curl -sS -X POST "$MARQUEZ_URL/api/v1/lineage" \
    -H 'Content-Type: application/json' \
    --data "$event" >/dev/null
done < target/lineage/openlineage.ndjson
```

Open the Marquez UI and inspect jobs, datasets, and run metadata. fbt dataset
names are stable fbt resource IDs, and logical paths are available in `fbt_`
facets. fbt does not export raw artifact content, prompts, credentials, or
absolute project paths by default.

To verify a running Marquez endpoint with the repository smoke:

```sh
FBT_MARQUEZ_URL=http://localhost:5000 \
make standard-backend-smoke
```

References:

- OpenLineage getting started: <https://openlineage.io/getting-started/>
- Marquez project: <https://marquezproject.ai/>

## 4. Jaeger / OTLP JSON

Use OTel traces when you want to inspect:

- build invocation spans
- transform run spans
- runner progress, usage, and tool-call events as span events
- token/cost/model attributes such as `gen_ai.usage.input_tokens`

Generate the export:

```sh
fbt export otel --output target/telemetry/otel.json
```

Jaeger accepts OTLP trace data. With an OTLP HTTP endpoint enabled:

```sh
curl -sS -X POST http://localhost:4318/v1/traces \
  -H 'Content-Type: application/json' \
  --data-binary @target/telemetry/otel.json
```

Then open the Jaeger UI and search for service `fbt`.

To verify a running OTLP HTTP endpoint with the repository smoke:

```sh
FBT_OTLP_TRACES_URL=http://localhost:4318/v1/traces \
make standard-backend-smoke
```

References:

- Jaeger API docs: <https://www.jaegertracing.io/docs/latest/architecture/apis/>
- OpenTelemetry Collector configuration: <https://opentelemetry.io/docs/collector/configuration/>

## 5. Grafana Tempo / Grafana

Tempo is a trace backend and Grafana is the UI. The usual standard path is:

```text
fbt export otel -> OTLP HTTP receiver -> Tempo -> Grafana
```

The receiver can be an OpenTelemetry Collector or Grafana Alloy. Keep that
pipeline outside fbt. A collector config must define receivers, exporters, and a
`service.pipelines.traces` section before it will forward data.

Minimal shape:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 127.0.0.1:4318

exporters:
  otlphttp/tempo:
    endpoint: http://tempo:4318

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlphttp/tempo]
```

Post the fbt OTLP/JSON payload to the collector's OTLP HTTP endpoint, then use
Grafana to query Tempo for service `fbt`.

The same `FBT_OTLP_TRACES_URL` variable works when the endpoint forwards to
Tempo through an OpenTelemetry Collector or Grafana Alloy.

References:

- Grafana Tempo tracing setup: <https://grafana.com/docs/tempo/latest/set-up-for-tracing/>
- Grafana Tempo collector setup: <https://grafana.com/docs/tempo/latest/set-up-for-tracing/instrument-send/set-up-collector/>

## 6. OpenMetadata

OpenMetadata is the catalog and governance target for teams that already run
OpenMetadata. fbt does not provide a direct OpenMetadata export command in core.
Use the standard OpenLineage path:

```text
fbt export openlineage -> Kafka or Kinesis bridge -> OpenMetadata OpenLineage ingestion -> catalog UI
```

Generate the fbt export:

```sh
fbt export openlineage --output target/lineage/openlineage.ndjson
```

Then publish those NDJSON events to the Kafka topic or Kinesis stream configured
for OpenMetadata's OpenLineage connector. Keep that bridge outside fbt; it is an
environment-specific integration point.

The OpenMetadata ingestion workflow runs with the OpenMetadata ingestion
framework:

```sh
pip3 install "openmetadata-ingestion[openlineage]"
metadata ingest -c openmetadata-openlineage.yml
```

The workflow YAML must point to the OpenMetadata server and an OpenLineage
connection with Kafka or Kinesis `brokerConfig`. A minimal shape is:

```yaml
source:
  type: openlineage
  serviceName: fbt_openlineage
  serviceConnection:
    config:
      type: OpenLineage
      brokerConfig:
        brokersUrl: localhost:9092
        topicName: fbt.openlineage
  sourceConfig:
    config:
      type: PipelineMetadata
sink:
  type: metadata-rest
  config: {}
workflowConfig:
  openMetadataServerConfig:
    hostPort: http://localhost:8585/api
    authProvider: openmetadata
    securityConfig:
      jwtToken: ${OPENMETADATA_JWT_TOKEN}
```

Direct OpenMetadata publishing is intentionally left to an optional external
integration. That integration can map fbt transforms to OpenMetadata pipelines
and add owners, tags, domains, glossary terms, or custom properties through
OpenMetadata APIs when an organization needs catalog-specific enrichment.

References:

- OpenMetadata OpenLineage connector:
  <https://docs.open-metadata.org/v1.12.x/connectors/pipeline/openlineage/yaml>
- OpenMetadata external ingestion:
  <https://docs.open-metadata.org/v1.12.x/deployment/ingestion/external>
- OpenMetadata catalog export evaluation:
  [OpenMetadata Catalog Export Evaluation](research/openmetadata-catalog-export-evaluation.md)

## 7. Troubleshooting

If Marquez shows no graph:

- confirm the NDJSON file has at least one line
- confirm each POST to `/api/v1/lineage` returns a success status
- search by fbt transform ID, such as `transform.knowledge_ops.case_summaries`

If Jaeger or Tempo shows no traces:

- confirm the OTLP endpoint is listening on the port you posted to
- confirm the backend accepts OTLP/JSON over HTTP
- search for service `fbt`
- inspect `target/telemetry/otel.json` for `resourceSpans`

If sensitive text appears in an export, treat it as a bug. Default exports
should include IDs, paths, descriptors, statuses, usage, and runner metadata,
not raw source documents, prompts, model responses, credentials, or unredacted
tool-call payloads.

## 8. Evidence Capture

For docs or release evidence, save the generated export files and backend smoke
summary:

```sh
FBT_MARQUEZ_URL=http://localhost:5000 \
FBT_OTLP_TRACES_URL=http://localhost:4318/v1/traces \
FBT_STANDARD_EVIDENCE_DIR=/tmp/fbt-standard-evidence \
make standard-backend-smoke
```

The evidence directory receives `openlineage.ndjson`, `otel.json`, and
`smoke-summary.txt`. Screenshots should be captured separately from the actual
Marquez, Jaeger, Tempo, Grafana, or OpenMetadata UI after ingestion.
