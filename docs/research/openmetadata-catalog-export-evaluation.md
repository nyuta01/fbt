# OpenMetadata Catalog Export Evaluation

Status: Decision  
Created: 2026-05-28  
Task: FBT-STD-004

## Question

Should `fbt` map artifacts directly into OpenMetadata entities, or should users
reach OpenMetadata through the existing OpenLineage export?

## Sources

- OpenMetadata OpenLineage connector:
  <https://docs.open-metadata.org/v1.12.x/connectors/pipeline/openlineage/yaml>
- OpenMetadata ingestion framework external deployment:
  <https://docs.open-metadata.org/v1.12.x/deployment/ingestion/external>
- OpenMetadata OpenLineage connection schema:
  <https://github.com/open-metadata/OpenMetadata/blob/main/openmetadata-spec/src/main/resources/json/schema/entity/services/connections/pipeline/openLineageConnection.json>
- OpenMetadata workflow schema:
  <https://github.com/open-metadata/OpenMetadata/blob/main/openmetadata-spec/src/main/resources/json/schema/metadataIngestion/workflow.json>
- OpenMetadata Lineage API:
  <https://docs.open-metadata.org/v1.12.x/api-reference/lineage>
- OpenMetadata Pipeline and Pipeline Service APIs:
  <https://docs.open-metadata.org/v1.12.x/api-reference/data-assets/pipelines/create>
  and
  <https://docs.open-metadata.org/v1.12.x/api-reference/data-assets/pipeline-services/create>

## Findings

OpenMetadata already has an OpenLineage pipeline connector. The documented
connector is configured as an OpenMetadata ingestion workflow and consumes
OpenLineage events from Kafka or AWS Kinesis. Its connection schema requires a
`brokerConfig`, with Kafka requiring `brokersUrl` and `topicName`, and Kinesis
requiring `streamName` plus AWS credentials.

OpenMetadata ingestion workflows are YAML-driven and can be run outside the
OpenMetadata UI with the `openmetadata-ingestion` package and
`metadata ingest -c <config>`. This means fbt does not need an embedded
OpenMetadata client or daemon to participate in the catalog path.

OpenMetadata also exposes direct APIs for pipeline services, pipelines, and
lineage edges. Those APIs are useful for a custom publisher, but they require an
OpenMetadata server, credentials, entity lifecycle management, and a stable
mapping from fbt's file-oriented artifacts to OpenMetadata entity types.

## Options

| Option | Fit | Cost |
|---|---|---|
| OpenLineage ingestion | Best default. Reuses fbt's implemented standard export and OpenMetadata's existing connector. | Requires an external bridge from fbt NDJSON to Kafka or Kinesis when users are not already emitting OpenLineage events to a broker. |
| Direct OpenMetadata publisher | Useful for organizations that need OpenMetadata-specific owners, domains, tags, glossary terms, or custom properties. | Requires server credentials, entity upserts, lifecycle semantics, and OpenMetadata SDK/API dependency outside the lightweight core. |
| fbt-native catalog UI | Poor fit. Duplicates a catalog/governance product and conflicts with fbt's local-first core boundary. | High product and maintenance cost. |

## Decision

Do not add a base `fbt export openmetadata` command. The supported path is:

```text
fbt native state -> fbt export openlineage -> external broker or ingestion workflow -> OpenMetadata
```

OpenMetadata remains a catalog and governance visualization target, not fbt's
internal state model. fbt should keep OpenLineage as the portable lineage export
and document how to feed that export into OpenMetadata-compatible ingestion.

If direct OpenMetadata publishing becomes necessary, it should be an optional
external integration or runner. That integration may use OpenMetadata REST/SDK
APIs to create a Pipeline Service, Pipelines, lineage edges, and optional
OpenMetadata-specific tags or custom properties. It must not add OpenMetadata
server credentials, SDKs, schedulers, or entity lifecycle coupling to fbt core.

## Recommended Mapping

For the default OpenLineage path:

- fbt transforms remain OpenLineage jobs.
- fbt transform runs remain OpenLineage runs.
- fbt sources and artifact versions remain OpenLineage datasets with `fbt_`
  facets for logical paths, descriptors, confidence, approvals, evals, policy,
  runner, and model metadata.
- OpenMetadata owns the catalog representation after ingestion.

For a future optional direct publisher:

- fbt transforms map to OpenMetadata `pipeline` entities under a configured
  Pipeline Service.
- fbt transform steps may map to pipeline tasks.
- fbt artifact/source lineage can become OpenMetadata lineage edges only after
  the target entities exist in OpenMetadata.
- OpenMetadata owners, tags, domains, glossary terms, and custom properties
  remain opt-in configuration outside fbt core.
