# fbt Core Spec

Status: MVP-ready
Created: 2026-05-28
Audience: users and implementers of `fbt`

## 1. Overview

`fbt` is a **file build tool**: a lightweight local-first control plane for
declaring filesystem artifact transformations, resolving dependencies,
delegating execution to external runners, and recording artifact versions,
lineage, eval results, policy decisions, confidence, and state.

`fbt` is not a transform engine. PDF OCR, Word conversion, Markdown AST
transforms, LLM calls, agent runtimes, scheduling, publishing, and human review
workflows belong outside core.

It is also not a replacement for existing transformation and data tools. dbt,
DataChain, DVC, Snakemake, remark, Pandoc, provider SDKs, schedulers, artifact
stores, and metadata catalogs remain separate systems that fbt can compose with
through files, runners, or standard exports.

## 2. Scope

In scope:

- Logical graphs for file and directory artifacts
- `source` and `ref` dependencies
- Transform assets, policies, and evals as graph nodes
- Local-first CLI
- External runner invocation
- JSON-RPC 2.0 compatible runner protocol over stdio
- Artifact descriptors and digests
- Immutable artifact versions
- Idempotent commit
- Dirty-state planning
- Deterministic eval execution
- Confidence propagation from deterministic checks
- Static docs and standard lineage/telemetry exports

Out of scope:

- Built-in document converter, OCR, LLM provider, or agent runtime
- Always-on scheduler
- Required metadata database or web server
- Distributed execution as a base requirement
- Cloud account requirement
- Human review, approval, assignment, notification, or release workflow
- Warehouse transformation, dataset versioning, document conversion, Markdown
  AST processing, workflow scheduling, artifact storage, or catalog hosting

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
| `policies/` | Tool, network, write-scope, security, and cost policies |
| `evals/` | Deterministic and delegated eval definitions |
| `target/artifacts/` | Logical output artifacts |
| `.fbt/state/` | Manifest, state snapshot, run results, artifact versions, eval results, policy decisions |

## 4. Resource Types

| Type | Kind | Meaning |
|---|---|---|
| `source` | definition | External input file or directory |
| `artifact` | definition | Logical output managed by the project |
| `artifact_version` | runtime record | Immutable content snapshot identified by descriptor/digest |
| `transform` | definition | Contract for producing artifacts |
| `transform_run` | runtime record | Concrete execution activity |
| `transform_asset` | definition | Prompt, template, script, rubric, style guide, examples, schema, config |
| `policy` | definition | Tool, network, write-scope, security, and cost constraints |
| `policy_decision` | runtime record | Runtime policy evaluation result |
| `eval` | definition | Deterministic, semantic, or LLM-judge eval definition |
| `evaluation_result` | runtime record | Eval execution result |
| `runner` | definition | External transform executor reference |

The manifest primarily stores definition resources and the dependency graph.
Runtime history belongs in state and run results.

## 5. Artifact Versions

An artifact is a logical output. An artifact version is an immutable content
snapshot.

```json
{
  "version_id": "artifact_version.knowledge_ops.weekly_report.sha256_abc123",
  "artifact_id": "artifact.knowledge_ops.weekly_report",
  "logical_path": "target/artifacts/reports/weekly_report.md",
  "storage_path": ".fbt/artifacts/artifact_version.../content",
  "descriptor": {
    "media_type": "text/markdown; charset=utf-8",
    "digest": "sha256:abc123",
    "artifact_type": "fbt.artifact.markdown_document.v1"
  },
  "generated_by": "transform_run.run_01H...",
  "confidence": "structural"
}
```

Diff, lineage, eval results, and downstream dependencies are bound to artifact
versions rather than bare paths.

MVP retention policy is `keep_all`. fbt does not delete immutable artifact
versions or run receipts automatically, and it does not expose a destructive
prune command. Use `fbt artifact retention` to inspect local growth and archive
`.fbt/state/` with `.fbt/artifacts/` together when using external retention
tools.

## 6. Transform

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
      - ref: contract_summary_prompt
    inputs:
      - source: legal_docs.raw_contracts
    outputs:
      - name: contract_summaries
        type: markdown_directory
        path: target/artifacts/contracts/summaries/
    evals:
      - required_sections
```

Initial transform types:

- `command`
- `extract`
- `template`
- `llm`
- `agent`
- `compose`

`llm` and `agent` are top-priority transform types. The type is a contract with
the runner; core does not implement the transform logic.

## 7. Inputs

`source` points to an external input:

```yaml
inputs:
  - source: legal_docs.raw_contracts
```

`ref` points to a project artifact:

```yaml
inputs:
  - ref: contract_summaries
    require:
      confidence: structural
      evals:
        required_sections: pass
```

Core supports confidence and eval requirements. Human approval state is not a
core dependency condition.

## 8. Policies

Policies define what a runner is allowed to read, write, and use.

```yaml
policies:
  - name: support_agent_scope
    read:
      - data/support/
      - target/artifacts/support/
    write:
      - .fbt/work/
      - target/artifacts/support/
    network: true
    tools:
      allow:
        - read_artifact
        - search_project
      deny:
        - write_source_files
    limits:
      timeout_seconds: 300
      max_output_bytes: 10485760
```

Core enforces path normalization, scoped work directories, output-size limits
for file artifacts and aggregate directory artifact bytes, official commit
boundaries, and state safety. Runners are still responsible for respecting
network and tool policy. The MVP security boundary and deterministic conformance
scenarios are defined in
[Security and Conformance Spec](security-and-conformance-spec.md).

## 9. Evals and Confidence

Implemented eval execution:

- `deterministic`

Reserved delegated eval types:

- `semantic`
- `llm_judge`

MVP core records `semantic` and `llm_judge` evals as skipped, writes the skip
reason and external-judge-transform hint into state/build receipts, shows the
skipped eval in artifact explanation, and grants no confidence from them.
Model-based judging belongs in an external runner transform that produces a
judge report artifact, or in a future delegated eval-runner protocol. Core keeps
the receipt, confidence gate, and lineage; it does not implement model-judge
logic.

The runnable boundary example is `examples/semantic_eval_boundary`: a generated
manual artifact feeds an external evidence-quality transform, and fbt records
the resulting report as a normal artifact.

Initial confidence classes:

- `exact`
- `structural`
- `semantic`
- `experimental`

Downstream transforms can require confidence:

```yaml
require:
  confidence: structural
```

## 10. Runner

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

## 11. Execution Lifecycle

```text
parse
  -> plan
  -> run external runner
  -> eval
  -> commit
  -> write state
```

Commit records an allowed output candidate as an immutable artifact version and
atomically updates the logical artifact pointer.

When selected transforms depend on each other, `build` executes them in
dependency order within the same invocation. A downstream selected transform may
wait for an upstream selected transform to commit a current artifact, then runs
after runtime confidence and policy requirements are checked against the updated
state.

## 12. Dirty-State Semantics

A transform becomes dirty when any relevant input to the effective transform
changes:

- Source descriptor or fingerprint
- Input artifact current pointer
- Transform asset fingerprint
- Policy
- Eval
- Runner identity or config
- Model identity or parameters
- Tool identity or version
- Missing output
- Forced rebuild

The cache model is intentionally small. By default, fbt skips a transform when
the latest successful run fingerprint and current outputs are still valid.
`fbt plan --force` previews selected clean transforms as dirty with
`forced rebuild`, and `fbt build --force` regenerates selected transforms. Force
does not bypass upstream artifact requirements, confidence requirements, policy
checks, or output-candidate boundaries.

For local file, directory, and glob sources, the source fingerprint includes
the resolved file set and file content fingerprints. Adding, removing, or
changing a file under a declared source path makes dependent transforms dirty.

`fbt` does not include a daemon, scheduler, watermark store, or built-in
per-file partition engine. Large daily batches should be partitioned in project
structure or driven by an external scheduler.

## 13. Immutability and Idempotency

`fbt` does not guarantee exactly-once runner execution. It guarantees:

- Immutable output artifact versions
- Idempotent commit
- Official logical pointers updated only after policy/eval checks pass
- Failed or interrupted outputs do not corrupt official state

## 14. CLI

Core commands:

```sh
fbt init
fbt doctor
fbt plan
fbt build
fbt diff weekly_report --against previous
fbt artifact show weekly_report
fbt artifact history weekly_report
fbt artifact retention
fbt export openlineage
fbt export otel
```

## 15. Local State

`fbt plan` is read-only. Local state files are written by `fbt build` and other
explicit write operations.

```text
.fbt/
  state/
    manifest.json
    state.json
    run_results.jsonl
    artifact_versions.json
    evaluation_results.json
    policy_decisions.json
  artifacts/
    artifact_version.../
      content
  cache/
  logs/
```

Base `fbt` does not require a metadata database.

## 16. Inspect And Export

fbt core exposes local build state through a small CLI surface:

- `fbt artifact` for current paths, versions, descriptors, and lineage context
- `fbt diff` for comparing generated artifact versions
- `fbt export openlineage` for standard lineage event records
- `fbt export otel` for standard execution trace payloads

Docs sites, catalogs, dashboards, and review UIs should consume those files or
the `.fbt/state` directory from external tools instead of requiring a built-in
docs generator.

## 17. MVP Acceptance Criteria

1. `fbt build` runs in a fresh environment without additional services.
2. File and directory sources can be defined.
3. An LLM transform can generate Markdown artifacts through an external runner.
4. An agent transform can generate a report from multiple artifacts.
5. Transform assets, model, tool calls, tokens, and cost are recorded when runners report them.
6. Successful and failed `transform_run` receipts are recorded in run results.
7. Changes to source, transform assets, or policy trigger downstream dirty selection.
8. Policy or confidence requirements can block downstream work.
9. Failed, denied, cancelled, or interrupted runs do not corrupt official
   artifact pointers.
10. Artifact inspection and standard exports expose lineage and artifact state.

## 18. User-Facing Specs

| Document | Purpose |
|---|---|
| [Project Config Spec](project-config-spec.md) | YAML project and resource definitions |
| [CLI Reference](cli-reference.md) | Commands, flags, selection syntax, exit codes |
| [State and Run Results Spec](state-and-run-results-spec.md) | `.fbt/state/`, state snapshots, run results, artifact versions |
| [Usage Guide](usage-guide.md) | User workflow |
| [Manifest Spec](manifest-spec.md) | Canonical graph metadata |
| [Runner Protocol Spec](runner-protocol-spec.md) | External runner protocol |
| [Schema and Versioning Spec](schema-and-versioning-spec.md) | Config versioning, schema compatibility, artifact types, descriptor rules |
| [Runner Discovery Spec](runner-discovery-spec.md) | Runner resolution, plugin manifests, diagnostics |
| [Security and Conformance Spec](security-and-conformance-spec.md) | Security model and conformance scenarios |

## 19. Post-MVP Follow-Up Boundaries

The MVP contract above is implementation-aligned. Future work should preserve
the same core boundary instead of expanding fbt into a scheduler, converter,
provider SDK, review app, or catalog.

| Area | MVP Position | Follow-Up Boundary |
|---|---|---|
| Schema validation | Generated JSON Schemas and parser diagnostics are kept in lockstep by repository checks. | Richer migrations require a new schema/versioning task and compatibility tests. |
| Daily source windows | fbt supports normal file and glob sources; users can compose date/window selection outside core. | Scheduling, partition catalogs, retention policy, and ingestion remain external tools. |
| Semantic descriptors | Core records text/Markdown descriptors and eval confidence; richer extraction belongs to runners. | DOCX, PDF, image, OCR, embedding, and classifier descriptors should be runner-owned outputs. |
| Runner adapters | The protocol, discovery rules, SDK, conformance harness, and official adapter module pattern are documented. | New adapters must remain optional packages and pass runner conformance before being advertised. |
