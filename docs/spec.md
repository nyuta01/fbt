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
where possible, official commit boundaries, and state safety. Runners are still
responsible for respecting network and tool policy. The MVP security boundary
and deterministic conformance scenarios are defined in
[Security and Conformance Spec](security-and-conformance-spec.md).

## 9. Evals and Confidence

Implemented eval execution:

- `deterministic`

Delegated/future eval types:

- `semantic`
- `llm_judge`

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

Project documentation, catalogs, dashboards, and review UIs should consume those
files or the `.fbt/state` directory from external tools instead of requiring a
built-in docs generator.

## 17. MVP Acceptance Criteria

1. `fbt build` runs in a fresh environment without additional services.
2. File and directory sources can be defined.
3. An LLM transform can generate Markdown artifacts through an external runner.
4. An agent transform can generate a report from multiple artifacts.
5. Transform assets, model, tool calls, tokens, and cost are recorded when runners report them.
6. `transform_run` and `artifact_version` are recorded in state and run results.
7. Changes to source, transform assets, or policy trigger downstream dirty selection.
8. Policy or confidence requirements can block downstream work.
9. Interrupted runs do not corrupt official artifact pointers.
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

## 19. Remaining Implementation Questions

1. How much schema-generated validation should replace hand-written parser
   checks.
2. Which source-window helper patterns can improve daily operation without
   adding scheduling or partition management to core.
3. How far semantic descriptors should go before extractor runners become the
   right boundary.
4. Which optional runner adapters should be documented next for common tools
   such as remark, Pandoc, dbt, DataChain, Codex, and Claude Code.
