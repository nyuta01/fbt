# fbt Standard Export Spec

Status: Contract  
Created: 2026-05-28  
Audience: implementers of OpenLineage, OpenTelemetry, OpenMetadata, and
visualization integrations

## 1. Overview

`fbt` keeps fbt-native manifest and state files as the local source of truth.
Standard integrations are explicit exports, not replacements for
`manifest.json`, `state.json`, `artifact_versions.json`, `run_results.jsonl`,
`evaluation_results.json` or `policy_decisions.json`.

The export contract covers:

- OpenLineage-compatible artifact lineage events
- OpenTelemetry-compatible execution telemetry
- OpenMetadata catalog integration through OpenLineage ingestion unless a
  direct catalog export proves necessary
- Visualization through standard-compatible tools instead of a custom fbt graph
  UI

External references:

- OpenLineage specification: <https://github.com/OpenLineage/OpenLineage/blob/main/spec/OpenLineage.md>
- OpenLineage home: <https://openlineage.io/>
- Marquez project: <https://marquezproject.ai/>
- OpenTelemetry specification overview: <https://opentelemetry.io/docs/specs/otel/overview/>
- OTLP specification: <https://opentelemetry.io/docs/specs/otlp/>
- OpenMetadata OpenLineage ingestion: <https://docs.open-metadata.org/v1.12.x/connectors/pipeline/openlineage/yaml>
- OpenMetadata catalog export evaluation:
  [OpenMetadata Catalog Export Evaluation](research/openmetadata-catalog-export-evaluation.md)

## 2. Non-Goals

- No daemon, scheduler, metadata database, web server, or cloud account in core.
- No custom lineage graph UI in fbt core.
- No required OpenLineage, OpenTelemetry, Marquez, or OpenMetadata dependency in
  the base binary.
- No raw artifact content, raw prompts, raw model responses, credentials, or
  unredacted tool-call payloads in default exports.
- No change to artifact version identity. `descriptor.digest` remains the fbt
  content identity.

## 3. Command Surface

Implemented export commands:

```sh
fbt export openlineage [--output PATH]
fbt export otel [--output PATH]
```

OpenLineage and OTel are implemented first. FBT-STD-004 decided not to add a
base `fbt export openmetadata` command. OpenMetadata integration uses the
OpenLineage export plus external OpenMetadata ingestion unless a future optional
publisher proves necessary outside core. OpenLineage default output is
newline-delimited JSON on stdout unless `--output` is set. OTel default output
is the OTLP/JSON payload on stdout unless `--output` is set.
`--json` returns an fbt command envelope with summary counts and the output
path; the exported records themselves remain in the selected standard format.

Envelope for file-based exports:

```json
{
  "metadata": {
    "fbt_export_schema_version": "https://schemas.fbt.dev/fbt/standard-export/v1.json",
    "fbt_version": "0.1.0",
    "project_name": "knowledge_ops",
    "format": "openlineage",
    "generated_at": "2026-05-28T10:00:00Z",
    "source_state": {
      "manifest": ".fbt/state/manifest.json",
      "state": ".fbt/state/state.json",
      "run_results": ".fbt/state/run_results.jsonl",
      "artifact_versions": ".fbt/state/artifact_versions.json"
    }
  },
  "records": []
}
```

When a target standard already defines a top-level protocol payload, the fbt
envelope is used only for `--json` summaries and fixtures. The exported file or
stdout stream uses the target standard's own top-level shape.

## 4. OpenLineage Mapping

OpenLineage is the first lineage export target. The official model is based on
run events with `run`, `job`, input/output datasets, event type, event time,
producer, schema URL, and facets. Marquez is the first visualization target
because it accepts OpenLineage-compatible metadata and provides a lineage UI.

Mapping:

| fbt | OpenLineage |
|---|---|
| `transform` | `job` |
| `transform_run` | `run` |
| `source` | input dataset |
| input `artifact` / `artifact_version` | input dataset with version facet |
| output `artifact` / `artifact_version` | output dataset with version and descriptor facets |
| `eval` / `evaluation_result` | custom fbt run or output facets |
| `policy` / `policy_decision` | custom fbt run facet |
| runner/model metadata | job/run facets |

OpenLineage constraints:

- `run.runId` must be a UUID. fbt exports derive a deterministic UUIDv5 from
  the fbt `transform_run` ID and include the original fbt ID in a custom facet.
- `job.namespace` is `fbt:<project_name>` by default.
- `job.name` is the fbt transform unique ID.
- Dataset namespace is `file://<project_root>` by default when absolute paths
  are allowed, otherwise `fbt:<project_name>`.
- Dataset names use stable fbt resource IDs, with logical paths in facets.
- Custom facets use the `fbt_` key prefix and immutable schema URLs.

Minimum OpenLineage event:

```json
{
  "eventType": "COMPLETE",
  "eventTime": "2026-05-28T10:08:00Z",
  "run": {
    "runId": "00000000-0000-5000-8000-000000000000",
    "facets": {
      "fbt_run": {
        "_schemaURL": "https://schemas.fbt.dev/openlineage/fbt-run-facet/v1.json",
        "transform_run_id": "transform_run.run_01H"
      }
    }
  },
  "job": {
    "namespace": "fbt:knowledge_ops",
    "name": "transform.knowledge_ops.case_summaries"
  },
  "inputs": [],
  "outputs": [],
  "producer": "https://github.com/nyuta01/fbt",
  "schemaURL": "https://openlineage.io/spec/1-0-0/OpenLineage.json#/definitions/RunEvent"
}
```

## 5. OpenTelemetry Mapping

OpenTelemetry export is for execution telemetry, not artifact lineage graph
identity. fbt exports OTLP/JSON-compatible trace payloads and may later add
metrics or logs. No network exporter is enabled by default.

Mapping:

| fbt | OpenTelemetry |
|---|---|
| invocation | root span |
| `transform_run` | child span |
| runner protocol request | child span or span event |
| policy decision | span event |
| eval result | span event |
| usage/cost | span attributes and optional metrics |
| runner/model metadata | span attributes, including `gen_ai.*` where applicable |

Resource attributes:

- `service.name`: `fbt`
- `service.version`: fbt version
- `fbt.project.name`
- `fbt.project.root`

Span attributes:

- `fbt.invocation.id`
- `fbt.transform.id`
- `fbt.transform.name`
- `fbt.artifact.id`
- `fbt.artifact.version_id`
- `fbt.runner.id`
- `fbt.model.provider`
- `fbt.model.name`
- token and cost fields already recorded in `run_results.jsonl`

## 6. OpenMetadata Mapping

OpenMetadata is treated as a catalog and governance visualization target. The
preferred path is:

```text
fbt native state -> fbt export openlineage -> external broker or ingestion workflow -> OpenMetadata
```

The evaluated OpenMetadata OpenLineage connector consumes OpenLineage events
from Kafka or AWS Kinesis through an OpenMetadata ingestion workflow. Users can
bridge `fbt export openlineage` NDJSON into that broker path or run a separate
ingestion workflow outside fbt.

Direct OpenMetadata publishing is not part of the base CLI because it requires
OpenMetadata server credentials, entity upsert lifecycle rules, and a stable
mapping from file-oriented fbt artifacts to OpenMetadata entities. If needed, it
belongs in an optional external integration that can create Pipeline Services,
Pipelines, lineage edges, owners, tags, domains, glossary terms, or custom
properties through OpenMetadata APIs. fbt must not adopt OpenMetadata as its
internal state model.

## 7. Redaction Rules

Default exports include:

- resource IDs
- logical paths
- content digests and descriptor metadata
- runner and model names
- eval and policy status
- usage and cost summaries

Default exports exclude:

- raw artifact content
- raw prompts and raw model responses
- secret environment variable values
- unredacted tool-call arguments or outputs
- absolute paths unless a future explicit flag requests them

If a runner emits sensitive metadata, fbt export must either omit it or place it
behind an explicit opt-in flag.

## 8. Visualization Contract

fbt standard exports should be viewable with existing tools:

- Marquez for OpenLineage graph visualization
- OpenTelemetry-compatible trace backends such as Jaeger, Tempo, or Grafana for
  execution timelines
- OpenMetadata for catalog and governance views through OpenLineage ingestion
  or an optional external publisher

External docs, catalog, or dashboard tools may link to exported files, but fbt
core does not build or host an interactive lineage graph UI.
Operational recipes live in
[Standard Visualization Guide](standard-visualization-guide.md).

## 9. Conformance Fixtures

`tests/conformance/run.sh` includes generated fixtures that assert:

- OpenLineage events include the selected schema URL and required event keys
- fbt custom facets use the `fbt_` prefix and immutable schema URLs
- OTLP JSON payloads contain valid resource spans and required fbt attributes
- redaction excludes raw content and secrets
- outputs are deterministic for the same fbt state

The default conformance gate checks the payload shape and redaction invariants
without starting Marquez, an OTel collector, OpenMetadata, or any other
backend service.
