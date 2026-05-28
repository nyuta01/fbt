# fbt Naming and Spec Standards Research

Created: 2026-05-28  
Audience: fbt specification and protocol design

## 1. Purpose

This report evaluates terminology and standards relevant to `fbt`:

- resource names
- manifest structure
- runner protocol
- lineage and provenance
- content-addressed artifact identity
- AI/agent metadata
- policy and eval representation

The goal is to avoid inventing incompatible terms when useful standards exist, while preserving `fbt`'s product-specific clarity.

## 2. Summary Recommendations

### Overall Direction

- Use `artifact` for logical filesystem outputs.
- Use `artifact_version` for immutable content snapshots.
- Use `transform` for the declared transformation contract.
- Use `transform_run` for a concrete execution activity.
- Use `transform_asset` for prompt, template, script, rubric, style guide, examples, schema, config, and tool manifests.
- Keep `prompt` as an asset type, not a top-level resource type.
- Use JSON-RPC 2.0 compatible messages over stdio for the runner protocol.
- Use OCI-like descriptors for digest-bearing artifact identity.
- Use OpenTelemetry GenAI semantic conventions where practical for AI metadata.
- Treat PROV, OpenLineage, in-toto, and SLSA as export/mapping references rather than the internal object model.

### Canonical Resource Types

Definition resources:

- `source`
- `artifact`
- `transform`
- `transform_asset`
- `policy`
- `eval`
- `runner`

Runtime records:

- `artifact_version`
- `transform_run`
- `policy_decision`
- `evaluation_result`

## 3. Standards Reviewed

## 3.1 W3C PROV

PROV distinguishes:

- `Entity`: data or object
- `Activity`: process that acts over time
- `Agent`: responsible actor

Mapping:

| PROV | fbt |
|---|---|
| Entity | artifact_version, transform_asset |
| Activity | transform_run |
| Agent | runner, human reviewer, service account |

PROV is useful conceptually, but too generic for the internal user-facing model.

## 3.2 OpenLineage

OpenLineage defines jobs, runs, datasets, and facets. It is useful for exporting lineage.

Mapping:

| OpenLineage | fbt |
|---|---|
| Dataset | artifact / artifact_version |
| Job | transform |
| Run | transform_run |
| Facet | media type, schema, confidence, approval, eval, AI metadata |

OpenLineage should be an export target, not the internal schema.

## 3.3 in-toto Attestation and SLSA Provenance

in-toto and SLSA are strong references for build provenance:

- subject digests
- materials
- builder identity
- invocation metadata
- reproducibility and trust boundaries

Mapping:

| SLSA/in-toto | fbt |
|---|---|
| subject | artifact_version |
| materials | source artifacts and transform assets |
| builder | runner |
| invocation | transform_run |
| predicate | fbt transform provenance |

These standards are useful for signed/exported provenance.

## 3.4 OCI Descriptor and Content Addressability

OCI descriptors use `mediaType`, `digest`, and `size`. This is a good pattern for artifact identity.

`fbt` should use:

```json
{
  "media_type": "text/markdown; charset=utf-8",
  "digest": "sha256:abc...",
  "size": 12345,
  "artifact_type": "fbt.artifact.markdown.v1"
}
```

Raw content digest and semantic descriptor should remain separate.

## 3.5 JSON Schema 2020-12

Manifest, run results, runner protocol payloads, and YAML config should be validated using JSON Schema 2020-12. YAML should be parsed into the JSON data model and validated with the same schemas.

Recommendations:

- Every machine-readable artifact should have schema/version metadata.
- Schema URIs should be stable.
- `v0.x` can be draft; `v1` should define compatibility rules.

## 3.6 JSON-RPC 2.0 and LSP

JSON-RPC provides:

- request/response
- notifications
- structured errors
- request IDs

LSP adds useful patterns:

- `initialize`
- capability negotiation
- cancellation
- progress notifications
- content-length framing

Recommendation: use JSON-RPC 2.0 compatible messages over stdio. Use JSONL framing for MVP, with possible future LSP-style framing.

## 3.7 MCP

MCP exposes resources, tools, and prompts to model clients. It is relevant for agent access to `fbt` project artifacts.

Recommendation:

- Do not use MCP as the runner protocol.
- Consider MCP as an integration layer for exposing artifact graph, docs, resources, and build tools to agents.
- Do not make `prompt` a top-level fbt resource just because MCP has prompts.

## 3.8 CloudEvents

CloudEvents is useful for event export, but not necessary for the internal runner protocol. `fbt/event` notifications can later be mapped to CloudEvents for external systems.

## 3.9 Trace Context and OpenTelemetry

Use W3C Trace Context and OpenTelemetry concepts for traces. Use OpenTelemetry GenAI semantic conventions for AI metadata where practical:

- provider name
- model name
- token usage
- tool calls
- cost estimate
- trace/span IDs

Do not store raw prompts or raw model outputs by default.

## 3.10 CWL / WDL / Nextflow

These workflow languages are useful references for staged inputs, declared outputs, and reproducibility. They are too workflow-centric for `fbt`'s core model.

Recommendation: do not adopt them directly, but keep command runner adapters possible.

## 3.11 Pandoc Filters / unified / unist / CommonMark

These are important references for Markdown and document AST transformations.

Recommendation:

- Do not own a core AST schema in `fbt`.
- Integrate via runners/plugins.
- Track scripts, configs, and schemas as transform assets.

## 3.12 ODRL / OPA Rego

ODRL and OPA/Rego are useful references for policy, but too heavy for MVP.

Recommendation:

- MVP policy should be declarative YAML constraints.
- Future advanced policy engines can be integrated.

## 4. fbt Core Vocabulary

| Resource | Meaning | Standards reference |
|---|---|---|
| `source` | External input | dbt source, OpenLineage input dataset |
| `artifact` | Logical output | dataset/entity |
| `artifact_version` | Immutable content snapshot | OCI descriptor, in-toto subject |
| `transform` | Declared contract/plan | dbt model, OpenLineage job |
| `transform_run` | Execution activity | PROV activity, OpenLineage run |
| `transform_asset` | Asset affecting behavior | PROV entity, partial plan |
| `policy` | Constraints | ODRL/OPA reference |
| `policy_decision` | Runtime policy result | policy decision record |
| `eval` | Quality check definition | dbt test, eval framework |
| `evaluation_result` | Runtime eval result | test/eval result |
| `runner` | External executor | builder/agent |

## 5. Naming Rules

### Field Naming

Use `snake_case`:

```yaml
source_paths: ["sources"]
transform_paths: ["transforms"]
artifact_path: "target/artifacts"
```

Draft-period kebab-case aliases may be accepted, but canonical output should use `snake_case`.

### ID Naming

```text
<resource_type>.<project_name>.<name>
```

For sources:

```text
source.<project>.<source_name>.<artifact_name>
```

### Type Naming

Use stable, lower-snake-case type names:

- `markdown_directory`
- `docx_directory`
- `llm_judge`
- `style_guide`
- `tool_manifest`

## 6. Runner Protocol Implications

Use JSON-RPC request/response/notification:

- `initialize`
- `initialized`
- `fbt/runTransform`
- `fbt/validate`
- `fbt/event`
- `fbt/outputCandidate`
- `$/cancelRequest`

Run request should include:

- invocation ID
- transform run ID
- transform definition snapshot
- resolved inputs
- output contracts
- transform assets
- model parameters
- tools
- policy
- state references
- work directories

Run result should include:

- status
- output candidates
- usage
- provenance
- warnings/errors

## 7. Manifest Implications

Top-level manifest shape:

```json
{
  "metadata": {},
  "sources": {},
  "artifacts": {},
  "artifact_versions": {},
  "transforms": {},
  "transform_assets": {},
  "policies": {},
  "evals": {},
  "runners": {},
  "parent_map": {},
  "child_map": {},
  "state_snapshot": {},
  "files": {}
}
```

`artifact_version` history should stay in state. The manifest may include only the current snapshot required for planning and docs.

## 8. Distinctive Standardization Opportunity

`fbt` can define a useful vocabulary for:

1. Filesystem artifact transformation graphs.
2. Prompts, templates, rubrics, schemas, style guides, and tool manifests as transform dependencies.
3. LLM/agent transform provenance with model, tool, context, usage, eval, and approval metadata.
4. Artifact-version approval semantics.
5. Local-first AI document transformation workflows.

## 9. Do Not Adopt Directly

Avoid making these the core internal model:

- Full PROV object model
- Full OpenLineage schema
- Full in-toto/SLSA predicate
- MCP prompts as top-level resources
- OPA/Rego as required policy language
- A core document AST schema

Use them as export or integration targets.

## 10. Recommended Next Decisions

1. Freeze canonical resource names.
2. Define artifact descriptor format.
3. Define runner protocol versioning.
4. Define JSON Schema strategy.
5. Define semantic descriptor method registry timing.
6. Define OpenLineage / OpenTelemetry export mapping later, not in MVP.

## 11. Sources

- W3C PROV: https://www.w3.org/TR/prov-overview/
- OpenLineage: https://openlineage.io/
- in-toto: https://in-toto.io/
- SLSA: https://slsa.dev/
- OCI Image Spec: https://github.com/opencontainers/image-spec
- JSON Schema 2020-12: https://json-schema.org/draft/2020-12
- JSON-RPC 2.0: https://www.jsonrpc.org/specification
- Language Server Protocol: https://microsoft.github.io/language-server-protocol/
- Model Context Protocol: https://modelcontextprotocol.io/
- CloudEvents: https://cloudevents.io/
- OpenTelemetry: https://opentelemetry.io/
- Common Workflow Language: https://www.commonwl.org/
- Pandoc: https://pandoc.org/
- unified: https://unifiedjs.com/

