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

## 2. Marquez / OpenLineage

Use OpenLineage when you want to inspect:

- which transform produced an artifact
- which input sources or artifacts fed that transform
- run IDs, job names, dataset names, approvals, confidence, evals, and fbt
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

References:

- OpenLineage getting started: <https://openlineage.io/getting-started/>
- Marquez project: <https://marquezproject.ai/>

## 3. Jaeger / OTLP JSON

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

References:

- Jaeger API docs: <https://www.jaegertracing.io/docs/latest/architecture/apis/>
- OpenTelemetry Collector configuration: <https://opentelemetry.io/docs/collector/configuration/>

## 4. Grafana Tempo / Grafana

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

References:

- Grafana Tempo tracing setup: <https://grafana.com/docs/tempo/latest/set-up-for-tracing/>
- Grafana Tempo collector setup: <https://grafana.com/docs/tempo/latest/set-up-for-tracing/instrument-send/set-up-collector/>

## 5. OpenMetadata

OpenMetadata is reserved for catalog and governance integration. Until the
OpenMetadata evaluation task is complete, prefer the standard OpenLineage path:

```text
fbt export openlineage -> OpenLineage-compatible ingestion -> catalog UI
```

fbt does not use OpenMetadata as its internal state model.

## 6. Troubleshooting

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
