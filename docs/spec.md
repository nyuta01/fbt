# fbt Core Spec

Status: Draft  
Created: 2026-05-28  
Audience: users and implementers of `fbt`

## 1. Overview

`fbt` is a **file build tool**: a lightweight local-first control plane for declaring filesystem artifact transformations, resolving dependencies, delegating execution to external runners, and managing lineage, evals, review, approval, confidence, and state.

`fbt` is not a transform engine. PDF OCR, Word conversion, Markdown AST transforms, LLM calls, and agent runtimes are delegated to runners or plugins.

The primary value of `fbt` is making LLM and agent-driven filesystem artifact transformations safe, inspectable, repeatable enough to operate, and easy to review.

## 2. Scope

In scope:

- Logical graphs for file and directory artifacts
- `source()` and `ref()` dependencies
- Transform assets, policies, and evals as graph nodes
- Local-first CLI
- External runner invocation
- JSON-RPC 2.0 compatible runner protocol over stdio
- Artifact descriptors and digests
- Immutable artifact versions
- Idempotent commit
- Dirty-state planning
- Deterministic and semantic eval orchestration
- Review and approval state
- Confidence propagation
- Docs and lineage generation

Out of scope:

- Built-in document converter
- Built-in OCR
- Built-in LLM provider implementation
- Built-in agent runtime
- Always-on scheduler
- Required metadata database
- Required web server
- Distributed execution as a base requirement
- Cloud account requirement

## 3. Project Layout

```text
fs_project.yml
sources/
transforms/
prompts/
assets/
policies/
evals/
target/
.fbt/
```

| Path | Role |
|---|---|
| `fs_project.yml` | Project-level config |
| `sources/` | External input artifact definitions |
| `transforms/` | Transformation contracts |
| `prompts/` | Conventional prompt directory; represented as transform assets |
| `assets/` | Prompts, templates, scripts, style guides, rubrics, examples, schemas |
| `policies/` | Tool, review, security, and cost policies |
| `evals/` | Deterministic, semantic, and human review eval definitions |
| `target/artifacts/` | Logical output artifacts |
| `target/docs/` | Generated docs |
| `.fbt/state/` | Manifest, state snapshot, run results, approvals, artifact versions |
| `.fbt/cache/` | Local cache |
| `.fbt/logs/` | Local logs |

## 4. Project Config

```yaml
name: knowledge_ops
config_version: 1
version: 0.1.0

source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["prompts", "assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]

target_path: "target"
artifact_path: "target/artifacts"

state:
  backend: local
  path: .fbt/state

execution:
  mode: local
  max_workers: 4

defaults:
  review:
    required: false
```

Local state is the default. External state and artifact stores are optional extensions.

## 5. Resource Types

`fbt` resources are split into definition resources and runtime records.

| Type | Kind | Meaning |
|---|---|---|
| `source` | definition | External input file or directory |
| `artifact` | definition | Logical output managed by the project |
| `artifact_version` | runtime record | Immutable content snapshot identified by descriptor/digest |
| `transform` | definition | Contract for producing artifacts |
| `transform_run` | runtime record | Concrete execution activity |
| `transform_asset` | definition | Prompt, template, script, rubric, style guide, examples, schema, config |
| `policy` | definition | Tool scope, review, security, and cost constraints |
| `policy_decision` | runtime record | Runtime policy evaluation result |
| `eval` | definition | Deterministic, semantic, LLM-judge, or human eval definition |
| `evaluation_result` | runtime record | Eval execution result |
| `runner` | definition | External transform executor reference |

The manifest primarily stores definition resources and the dependency graph. Runtime history belongs in state and run results.

### Schema, Versioning, and Artifact Types

Project files use `config_version: 1`. Machine-readable JSON files produced by
core include `metadata.fbt_schema_version`. Artifact descriptors use fully
qualified artifact type identifiers such as
`fbt.artifact.markdown_directory.v1`.

The first implementation baseline is fixed in
[Schema and Versioning Spec](schema-and-versioning-spec.md). That spec is
authoritative for:

- project config versioning
- JSON schema compatibility
- artifact type registry
- raw and directory digest canonicalization
- semantic descriptor method names
- artifact version ID format

## 6. Source

```yaml
sources:
  - name: legal_docs
    artifacts:
      - name: raw_contracts
        type: docx_directory
        path: data/legal/contracts/*.docx
        tests:
          - exists
          - min_file_count: 1
```

Sources are read-only.

## 7. Artifact and Artifact Version

An artifact is a logical output. An artifact version is an immutable content snapshot.

```json
{
  "artifact": "weekly_report",
  "logical_path": "target/artifacts/reports/weekly_report.md",
  "current_version_id": "artifact_version.knowledge_ops.weekly_report.sha256_abc123"
}
```

Artifact version:

```json
{
  "version_id": "artifact_version.knowledge_ops.weekly_report.sha256_abc123",
  "artifact": "artifact.knowledge_ops.weekly_report",
  "descriptor": {
    "media_type": "text/markdown; charset=utf-8",
    "digest": "sha256:abc123",
    "size": 12345,
    "artifact_type": "fbt.artifact.markdown_document.v1"
  },
  "generated_by": "transform_run.run_01H...",
  "approval_state": "pending"
}
```

Approval, downstream dependency, cache, and diff are bound to artifact versions rather than paths.

## 8. Transform and Transform Run

A transform is a contract, not the implementation.

```yaml
transforms:
  - name: contract_summaries
    type: llm
    runner: openai.responses
    model:
      provider: openai
      name: gpt-5
    assets:
      - type: prompt
        path: prompts/contract_summary.md
    inputs:
      - ref: normalized_contracts
    outputs:
      - name: contract_summaries
        type: markdown_directory
        path: target/artifacts/contracts/summaries/
    evals:
      - citation_coverage
      - no_unsupported_claims
    review:
      required: true
      group: legal
```

Initial transform types:

- `command`
- `extract`
- `template`
- `llm`
- `agent`
- `compose`
- `review`

`llm` and `agent` are top-priority transform types.

`transform_run` records an actual execution:

```json
{
  "run_id": "transform_run.run_01H...",
  "transform": "transform.knowledge_ops.contract_summaries",
  "runner": "runner.knowledge_ops.openai.responses",
  "status": "success",
  "materials": [
    {
      "resource_id": "artifact.knowledge_ops.normalized_contracts",
      "digest": "sha256:abc..."
    }
  ],
  "subjects": [
    {
      "artifact": "artifact.knowledge_ops.contract_summaries",
      "digest": "sha256:out..."
    }
  ],
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

## 9. ref and source

`ref()` points to a project artifact:

```yaml
inputs:
  - ref: contract_summaries
```

`source()` points to an external input:

```yaml
inputs:
  - source: legal_docs.raw_contracts
```

`ref()` can require quality:

```yaml
inputs:
  - ref: contract_summaries
    require:
      confidence: reviewed
      evals:
        citation_coverage: pass
      review:
        status: approved
```

## 10. Transform Asset

Transform assets are graph nodes. If an asset changes, dependent transforms become dirty.

Asset types:

- `prompt`
- `template`
- `script`
- `style_guide`
- `rubric`
- `examples`
- `glossary`
- `schema`
- `config`
- `tool_manifest`

`prompt` is important for LLM and agent transforms but is still just an asset type.

## 11. Policy

```yaml
policy:
  read:
    - target/artifacts/contracts/normalized/
  write:
    - target/quarantine/contracts/summaries/
  network: true
  tools:
    allow:
      - read_artifact
      - search_project
    deny:
      - write_source_files
  limits:
    timeout_seconds: 300
    max_cost_usd: 1.00
    max_tool_calls: 20
```

Policy changes can trigger re-evaluation or regeneration.

Core enforces path normalization, scoped work directories, output-size limits
where possible, official commit boundaries, and state safety. Runners are still
responsible for respecting network and tool policy. The MVP security boundary
and deterministic conformance scenarios are defined in
[Security and Conformance Spec](security-and-conformance-spec.md).

## 12. Eval

Eval types:

- Deterministic eval
- Semantic eval
- LLM judge eval
- Human review gate

```yaml
evals:
  - name: citation_coverage
    type: semantic
    runner: openai.responses
    config:
      min: 0.9
    grants_confidence: semantic
```

## 13. Confidence

Initial confidence classes:

- `exact`
- `structural`
- `semantic`
- `reviewed`
- `experimental`

Downstream transforms can require confidence:

```yaml
require:
  confidence: reviewed
```

## 14. Review and Approval

Review is an optional gate.

```yaml
review:
  required: true
  group: legal
```

Approval is bound to `artifact_version`.

```json
{
  "artifact": "weekly_report",
  "artifact_version": "artifact_version.knowledge_ops.weekly_report.sha256_abc123",
  "digest": "sha256:abc123",
  "status": "approved",
  "approved_by": "user@example.com",
  "approved_at": "2026-05-28T10:00:00Z"
}
```

When content changes, a new artifact version is created and prior approval does not automatically carry over.

## 15. Runner

Runners are external processes.

Initial protocol:

- JSON-RPC 2.0 compatible messages over stdio
- JSONL framing for MVP
- No in-process plugins
- Scoped work directory
- Runner generates output candidates
- `fbt-core` performs official commit

Runner references are resolved through project config, plugin manifests, and
`PATH` lookup as defined in [Runner Discovery Spec](runner-discovery-spec.md).
Static runner config is advisory; the `initialize` response from the runner is
authoritative for protocol and capability compatibility.

Runner examples:

- `command.local`
- `openai.responses`
- `langgraph.agent`
- `remark.transform`
- `pandoc.convert`
- `unstructured.partition`

## 16. Execution Lifecycle

```text
parse
  -> plan
  -> run
  -> eval
  -> review gate
  -> commit
  -> write state
```

Commit records an approved or otherwise allowed output candidate as an immutable artifact version and atomically updates the logical artifact pointer.

## 17. Dirty-State Semantics

A transform becomes dirty when any relevant input to the effective transform changes:

- Source descriptor or fingerprint
- Input artifact current pointer
- Transform asset fingerprint
- Rendered transform asset fingerprint
- Policy
- Eval
- Runner identity or config
- Model identity or parameters
- Retrieved context
- Tool identity or version
- Declared external dependency
- Review invalidation
- Missing output
- Forced rebuild

`fbt plan` must show dirty reasons.

## 18. Immutability and Idempotency

`fbt` does not guarantee exactly-once runner execution. It guarantees:

- Immutable output artifact versions
- Idempotent commit
- Official logical pointers updated only after commit
- Failed or interrupted outputs do not corrupt official state
- Unreviewed outputs can be quarantined or committed as pending

## 19. Cache and Reuse

```yaml
cache:
  mode: reuse_if_same_inputs
```

Initial cache modes:

- `reuse_if_same_inputs`
- `always_regenerate`
- `require_approval_for_reuse`

LLM and agent transforms should prefer reusing approved outputs by default.

## 20. CLI

Core commands:

```sh
fbt init
fbt parse
fbt plan
fbt build
fbt run --select weekly_report+
fbt eval weekly_report
fbt diff weekly_report --against last-approved
fbt review status weekly_report
fbt review approve weekly_report
fbt review reject weekly_report
fbt docs generate
```

## 21. Local State

```text
.fbt/
  state/
    manifest.json
    state.json
    run_results.jsonl
    approvals.json
    artifact_versions.json
    evaluation_results.json
    policy_decisions.json
  cache/
  logs/
```

Base `fbt` does not require a metadata database.

## 22. Docs

`fbt docs generate` should generate static docs showing:

- Artifact graph
- Source files
- Transform DAG
- Transform asset lineage
- Policy and eval lineage
- Runner and plugin lineage
- Model and tool metadata
- Token and cost summary
- Review state
- Confidence class
- Artifact versions
- Diff links

## 23. MVP Semantics

MVP must include:

- Local CLI
- Go single-binary core
- YAML project definition
- Local state
- Parse for source / transform / transform_asset / policy / eval
- JSON-RPC runner protocol over stdio
- Project config versioning
- Artifact type registry and descriptor canonicalization
- Runner discovery and diagnostics
- LLM transform
- Simple agent transform
- Command runner
- Artifact descriptors and digests
- Immutable version records
- Idempotent commit
- Deterministic eval
- Basic LLM judge eval
- Review approval state
- Confidence propagation
- Plan / build / eval / diff / docs
- Security conformance scenarios backed by fake runners

MVP does not require:

- Remote runner
- Hosted service
- Scheduler
- Metadata database
- In-process plugin
- Google Drive live sync
- Full MCP server

## 24. Acceptance Criteria

1. `fbt build` runs in a fresh environment without additional services.
2. Word, PDF, and Markdown directories can be defined as sources.
3. An LLM transform can generate Markdown summaries.
4. An agent transform can generate a report from multiple artifacts.
5. Transform assets, model, tool calls, tokens, and cost are recorded.
6. `transform_run` and `artifact_version` are recorded in state and run results.
7. Changes to source, transform assets, or policy trigger downstream dirty selection.
8. Unapproved artifact versions can block downstream requirements.
9. Interrupted runs do not corrupt official artifact pointers.
10. Docs expose lineage and review state.

## 25. User-Facing Specs

| Document | Purpose |
|---|---|
| [Project Config Spec](project-config-spec.md) | YAML project and resource definitions |
| [CLI Reference](cli-reference.md) | Commands, flags, selection syntax, exit codes |
| [State and Run Results Spec](state-and-run-results-spec.md) | `.fbt/state/`, state snapshots, run results, artifact versions, approvals |
| [Usage Guide](usage-guide.md) | User workflow |
| [Knowledge Loop Example](examples/knowledge-loop-example.md) | Customer support, incident response, and agile management examples |
| [Manifest Spec](manifest-spec.md) | Canonical graph metadata |
| [Runner Protocol Spec](runner-protocol-spec.md) | External runner protocol |
| [Schema and Versioning Spec](schema-and-versioning-spec.md) | Config versioning, schema compatibility, artifact types, descriptor rules |
| [Runner Discovery Spec](runner-discovery-spec.md) | Runner resolution, plugin manifests, diagnostics |
| [Security and Conformance Spec](security-and-conformance-spec.md) | Security model and conformance scenarios |

## 26. Remaining Implementation Questions

User-facing commands, YAML, schema/versioning, runner discovery, state files,
security baseline, and review flow are fixed for this draft. Remaining questions
are implementation boundaries and MVP depth.

1. Should transform asset rendering be owned by core or runner?
2. Should semantic diff be in MVP or post-MVP?
3. Should MVP include a content-addressed object store or start with logical path + digest?
4. When should optimistic concurrency for remote state backends be introduced?
5. Which fake-runner conformance scenarios must block the first MVP release?
