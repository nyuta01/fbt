# fbt Design Doc

Status: Draft  
Created: 2026-05-28  
Service name: `fbt`  
Scope: a filesystem transformation control plane for unstructured and semi-structured artifacts

## 1. Overview

`fbt` is a control plane for managing transformation logic over files and directories. dbt manages SQL transformations over warehouse relations; `fbt` manages transformations over filesystem artifacts such as Markdown, Word, Excel, PDF, images, audio transcripts, HTML, JSON, source code, configuration files, logs, tickets, chats, generated reports, and agent-created drafts.

Transformations are not limited to deterministic SQL, regex, or code. `fbt` treats LLMs, vision models, OCR, AI agents, and human-in-the-loop review as first-class parts of a transformation graph.

The goal is not to build an AI workflow runner or document converter. `fbt` core does not implement transformation logic. It declares transformations, resolves dependencies, delegates execution to runners and plugins, evaluates outputs, gates review, and records lineage and state.

In short, `fbt` is a **file build tool**: a dbt-like control plane for filesystem transformations.

## 2. Background

Modern knowledge work and AI applications rely on far more than structured tables. High-value information lives in:

- Markdown documents
- Word documents
- Excel workbooks
- PDFs
- HTML pages
- slide decks
- images
- transcripts
- source code
- configuration files
- logs
- issue and ticket exports
- Slack and email exports
- RAG corpora
- generated reports
- agent-generated drafts

These files are continuously transformed by people, scripts, LLMs, agents, SaaS tools, and CI jobs. The transformations are often ad hoc, making it hard to answer:

- Which file was created from which inputs?
- Which prompt, model, tool, policy, code, or human review affected it?
- When and why should it be regenerated?
- Does it meet quality expectations?
- How should stochastic LLM outputs be compared or evaluated?
- How should local, CI, production, and shared workspaces use the same definitions?
- How should agent-written files be audited?
- What is the current approved version and what evidence supports it?

dbt brought software-engineering discipline to SQL transformation layers. `fbt` brings the same control-plane idea to filesystem artifacts and AI-era transformations.

## 3. Problem Statement

File transformation work mixes three classes of transforms.

### Deterministic Transforms

The same input and code should produce the same output.

- Markdown to HTML
- Word to PDF
- Excel workbook to summary Markdown
- JSON normalization
- YAML frontmatter generation
- Directory index generation
- Image resize or compression

### AI-Assisted Transforms

An ML or LLM model is used, but scope and output contract remain relatively clear.

- OCR PDF to Markdown
- Extract action items from transcripts
- Summarize Word documents
- Explain workbook content
- Generate FAQ candidates from Markdown
- Generate alt text from images
- Generate release notes from issues

### Agentic Transforms

An agent uses multiple steps, tools, search, and judgment to create or update outputs.

- Update a design doc after reading multiple docs
- Investigate a repository and generate an architecture report
- Create a proposal from Excel, Word, and Markdown inputs
- Apply review comments to a draft
- Evaluate and improve a RAG corpus
- Organize a directory, deduplicate content, and merge stale notes

Traditional build systems and orchestrators handle deterministic transforms reasonably well. LLM and agent transforms require additional concepts: transform assets, model identity, tools, context, evals, human approval, stochastic output handling, and regeneration policy.

## 4. Goals

1. Treat filesystem artifacts as first-class resources.
2. Manage deterministic, LLM, and agentic transforms on one DAG.
3. Prioritize the LLM / Agent transformation experience.
4. Record inputs, outputs, transform assets, model, tools, code, config, policy, and evals.
5. Express dependencies with `ref()` and `source()`.
6. Treat transform assets, policies, and evals as graph nodes.
7. Provide lineage at file, directory, and document-section levels over time.
8. Provide snapshots, semantic diff, evals, and approval for stochastic outputs.
9. Explain why transforms are dirty.
10. Propagate artifact confidence and approval to downstream transforms.
11. Separate `artifact_version` from `transform_run`.
12. Work local-first and later extend to managed collaboration, remote cache, and remote state.

## 5. Non-Goals

1. Becoming a general-purpose workflow orchestrator.
2. Becoming a generic agent automation platform.
3. Becoming an LLM application framework.
4. Building a document editor.
5. Replacing storage systems, CMSs, ticket systems, or knowledge bases.
6. Fully editing and rendering every file format in the initial version.
7. Promising full reproducibility for stochastic AI outputs.
8. Implementing Word, PDF, Excel, OCR, Markdown parsing, LLM providers, or agent runtimes in core.

## 6. Design Principles

### Control Plane First

`fbt` core delegates transformation logic to runners and plugins.

Core owns:

- project parser
- manifest
- artifact graph
- `source()` and `ref()`
- selection and planner
- dirty-state detection
- runner invocation
- input/output fingerprinting
- provenance capture
- eval orchestration
- approval state
- run results
- docs and lineage
- plugin registry
- sandbox and policy enforcement

Core does not own:

- Word / Excel / PDF conversion
- OCR
- Markdown AST transform implementation
- LLM provider APIs
- agent runtime implementation
- semantic evaluator implementation
- storage backend implementation

### Lightweight Local-First

`fbt` should behave like a developer tool, not Airflow, Prefect, or Dagster.

Base runtime requirements:

- single binary distribution
- no daemon
- no scheduler
- no metadata database
- no Docker or Kubernetes requirement
- no cloud account
- no web server
- `fbt build` works after repository checkout

### Artifact Contract First

A transform is not merely a command. It is a contract for turning input artifacts into output artifacts. The runner is replaceable; the important question is whether the output satisfies the contract, evals, and approval policy.

### Transform Assets / Policy / Eval as Graph Nodes

Prompts, templates, scripts, style guides, rubrics, examples, schemas, and config are `transform_asset` resources. Changing them should mark dependent transforms as modified.

### Confidence and Approval Propagation

Artifacts are not only success/failure. They have confidence classes and approval state. Downstream transforms may require reviewed, semantic, or structural confidence.

### Traceability Over Reproducibility

LLM and agent transforms are not fully reproducible. `fbt` instead promises traceability: which inputs, assets, models, tools, policies, evals, and approvals produced which artifact version.

### Immutable Artifact Versions

Logical artifact identity is separate from content identity. `weekly_report` may change, but each output version is immutable and identified by `descriptor.digest`.

### Idempotent Commit

Runner execution is at-least-once. Core guarantees idempotent commit: re-committing the same digest does not corrupt official state.

### Controlled Side Effects

Transforms should stay within declared inputs and outputs. Agent transforms must declare read scope, write scope, allowed tools, network policy, timeout, and cost limits.

### Stable Core, Replaceable Runners

`fbt-core` is planned in Go for single-binary distribution, startup speed, dependency control, cross-platform operation, and long-term maintainability. Heavy document and AI logic belongs in optional runners.

### AI-First Runner Experience

The core does not implement LLM or agent execution, but the protocol and UX must treat model metadata, tool calls, retrieved context, token usage, cost, trace, eval, and approval as first-class.

## 7. Core Concepts

| Concept | Meaning |
|---|---|
| Project | Directory containing filesystem artifact and transform definitions |
| Source | External input file or directory |
| Artifact | Logical output managed by the project |
| Artifact Version | Immutable content snapshot |
| Transform | Contract that produces output artifacts |
| Transform Run | Concrete execution activity |
| Runner | External executor for transform logic |
| Ref | Logical dependency on another project artifact |
| Contract | Output expectations such as format, sections, citations, or style |
| Confidence | Trust class granted by deterministic, semantic, or human checks |
| Approval | Human review state bound to artifact versions |
| Evaluation | Deterministic, semantic, LLM-judge, or human quality check |
| State | Previous manifests, run results, artifact versions, approvals, eval results |

## 8. Project Example

```text
fs_project.yml
sources/contracts.yml
sources/meeting_notes.yml
transforms/contracts/normalize_contracts.yml
transforms/contracts/summarize_contracts.yml
transforms/meetings/extract_action_items.yml
transforms/reports/generate_weekly_report.yml
prompts/contract_summary.md
prompts/weekly_report.md
evals/no_hallucinated_claims.yml
policies/legal_summary.yml
target/artifacts/
.fbt/state/
```

Example transform:

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
        path: target/artifacts/contracts/summaries/
        type: markdown_directory
    contract:
      language: en
      required_sections:
        - Summary
        - Key Terms
        - Risks
        - Questions
      citations:
        required: true
    evals:
      - no_hallucinated_claims
      - citation_coverage
    review:
      required: true
      group: legal
```

## 9. Mapping to dbt

| dbt | fbt |
|---|---|
| Warehouse relation | Filesystem artifact |
| Model | Transform |
| Adapter / materialization | Runner / plugin |
| SQL | Code, transform assets, LLM, agent plan, converter |
| `ref()` | Upstream artifact reference |
| `source()` | External file source |
| Materialization | File, directory, docx, xlsx, pdf, markdown, index |
| Data test | Artifact test, document eval, LLM judge, human approval |
| `manifest.json` | Filesystem transformation manifest |
| `run_results.json` | Transform and eval results |
| State selection | Changed files, assets, policies, evals, models, tools |
| Docs | Artifact lineage, transform asset lineage, evaluation report |

## 10. Execution Model

Execution phases:

```text
parse -> plan -> execute -> evaluate -> commit
```

### Parse

Read project files, sources, transforms, assets, policies, and evals. Produce a manifest.

### Plan

Compare the current manifest and previous state to select transforms. Detect changes to sources, transform definitions, assets, model parameters, tool sets, policies, evals, upstream artifacts, and approval state.

### Execute

Resolve the runner and invoke it through the runner protocol. Core manages inputs, policy, trace, eval, approval, and state.

### Evaluate

Run deterministic checks, semantic evals, LLM judges, and human review gates as configured.

### Commit

Runners write output candidates into a scoped work directory. Core computes descriptors, runs evals and policy checks, records immutable artifact versions, and updates logical artifact pointers if allowed.

## 11. Dirty-State Algorithm

Each transform has an effective fingerprint:

```text
transform_fingerprint = hash(
  transform_config_hash,
  input_artifact_version_descriptors,
  transform_asset_hashes,
  model_identity,
  model_parameters_hash,
  runner_identity,
  runner_config_hash,
  tool_policy_hash,
  code_hash,
  eval_contract_hash,
  declared_external_dependencies
)
```

Raw digest and semantic digest are separate. Markdown AST, normalized docx/xlsx content, and PDF extracted text may produce semantic descriptors, but raw content digest remains the primary content identity.

## 12. Runner and Plugin Boundary

The initial protocol is JSON-RPC 2.0 compatible messages over stdio with JSONL framing. In-process plugins are not part of MVP. Runners can be implemented in any language.

Runner request includes:

- transform unique ID
- resolved input artifacts
- declared outputs
- work directory
- environment
- policy
- transform asset paths and rendered assets
- model identity and parameters
- tool definitions
- state references
- writable scope
- timeout and cost limits

Runner result includes:

- status
- outputs
- logs
- trace
- metrics
- provenance
- token usage and cost
- tool-call summary
- warnings and errors

## 13. Technical Architecture

Planned Go package structure:

```text
fbt/
  cmd/
    fbt/
  internal/
    project/
    config/
    manifest/
    graph/
    planner/
    state/
    artifact/
    runner/
    eval/
    approval/
    docs/
    plugin/
    protocol/
```

Base install provides:

- CLI
- project parser
- manifest model
- graph planner
- local state store
- local artifact store
- command runner
- runner protocol
- basic fingerprinting
- basic deterministic evals
- approval file store
- static docs generator

Optional runners/plugins:

- `fbt-remark`
- `fbt-pandoc`
- `fbt-unstructured`
- `fbt-openai`
- `fbt-langgraph`
- `fbt-s3`
- `fbt-promptfoo`

## 14. Transform Types

- `command`: shell, Python, Node, pandoc, LibreOffice, or other external command
- `template`: Jinja, Mustache, or similar rendering
- `extract`: OCR, parser, converter, partitioner
- `llm`: single-step or small-step LLM generation
- `agent`: multi-step tool-using generation
- `review`: human approval, redline, comment resolution
- `compose`: combine multiple artifacts into reports, proposals, manuals, indexes

## 15. File Types

MVP priority:

- Markdown
- plain text
- Word `.docx`
- Excel `.xlsx`
- PDF
- HTML
- JSON / YAML
- directory

Post-MVP:

- PowerPoint `.pptx`
- images
- audio/video transcripts
- code repositories
- Google Docs / Sheets export
- Notion export

## 16. Tests and Evals

Deterministic checks:

- `exists`
- `not_empty`
- `file_count`
- `markdown_parseable`
- `docx_renderable`
- `xlsx_openable`
- `pdf_renderable`
- `no_broken_links`
- `required_sections`
- `required_frontmatter`
- `max_word_count`
- `min_word_count`
- `no_unresolved_comments`
- `no_forbidden_terms`
- `pii_redacted`

Semantic evals:

- summary coverage
- citation coverage
- unsupported claim detection
- style guide compliance
- meaning-preserving translation
- policy-following agent output

Human review:

- approval required
- reviewer assignment
- approval expiration
- comment resolution
- downstream blocking until approval

## 17. Immutability, Idempotency, and Side Effects

`fbt` treats data transformation principles as follows:

- Artifact versions are immutable.
- Logical artifact pointers are mutable.
- Commit is idempotent.
- Runner execution is not exactly-once.
- Failed or interrupted runs cannot corrupt official state.
- Agent side effects are constrained by declared read/write scope and tools.

## 18. Security and Governance

Core must protect:

- source files from mutation
- official artifact pointers from runner writes
- secrets from state, logs, and docs
- users from unbounded tool access, network access, cost, and output size

Agent transforms require explicit policy.

## 19. MVP

MVP focuses on LLM and agent transformation management for unstructured documents.

Must include:

- local CLI
- Go core
- YAML project config
- local state
- manifest
- runner protocol
- LLM transform
- simple agent transform
- command runner
- artifact descriptors and immutable versions
- deterministic and LLM-judge evals
- review approval
- confidence propagation
- docs

Out of MVP:

- hosted service
- production-grade MCP server
- metadata database
- remote worker
- live Google Drive sync
- plugin marketplace

## 20. Roadmap

Phase 1: Local document graph  
Parse, plan, local state, command runner, Markdown and directory outputs.

Phase 2: Agentic transform  
LLM runner, agent runner, tool-call logs, policy enforcement, evals.

Phase 3: Collaboration and review  
Approval workflows, review groups, comments, managed state.

Phase 4: Remote sources and cache  
S3, GCS, object storage, shared artifact cache.

Phase 5: Managed service  
Hosted docs, collaboration, audit, policy, remote execution.

## 21. Risks

- Becoming too much of an agent platform
- Core absorbing converter or runner responsibilities
- Losing local-first simplicity
- Overbuilding immutable object storage
- Overpromising stochastic reproducibility
- Expanding file format support too quickly
- Weak LLM-eval reliability
- Prompt injection
- Blurry differentiation from orchestrators

Mitigation: keep core as a local-first control plane; externalize heavy dependencies; emphasize traceability, eval, approval, and artifact lineage.

## 22. User-Facing Specs

| Document | Purpose |
|---|---|
| [Core Spec](spec.md) | Overall semantics |
| [Project Config Spec](project-config-spec.md) | YAML project and resource definitions |
| [CLI Reference](cli-reference.md) | Commands and flags |
| [State and Run Results Spec](state-and-run-results-spec.md) | Local state and runtime records |
| [Usage Guide](usage-guide.md) | User workflow |
| [Knowledge Loop Example](examples/knowledge-loop-example.md) | Representative use case |
| [Manifest Spec](manifest-spec.md) | Canonical graph metadata |
| [Runner Protocol Spec](runner-protocol-spec.md) | External runner protocol |
| [Schema and Versioning Spec](schema-and-versioning-spec.md) | Config versioning, schema compatibility, artifact types |
| [Runner Discovery Spec](runner-discovery-spec.md) | Runner resolution and plugin manifests |
| [Security and Conformance Spec](security-and-conformance-spec.md) | Security model and conformance scenarios |

## 23. Remaining Decisions

MVP implementation decisions:

1. Whether transform asset rendering is owned by core or runner.
2. MVP depth of semantic diff.
3. Whether MVP starts with logical path + digest or content-addressed object store.
4. How determinism class affects planning, cache, and review invalidation.
5. Which fake-runner conformance scenarios must block the first MVP release.

Post-MVP decisions:

1. Direct Word / Excel editing versus Markdown / structured intermediate.
2. Google Docs / Sheets as file export or live API source.
3. MCP server timing.
4. Distribution story for Go core and optional Python / TypeScript runners.
5. Remote state, remote runners, and managed approval provider sequencing.

## 24. Positioning

Short:

> fbt is a file build tool for unstructured files and AI transformations.

Long:

> fbt is a lightweight local-first control plane for managing transformations over Markdown, Word, Excel, PDF, and other filesystem artifacts, especially those produced by LLMs, AI agents, and human review. Transform execution is delegated to runners and plugins; fbt tracks what was built from what, which assets and tools affected it, which evals it passed, and when it must be regenerated.

Avoid:

> fbt is not an AI agent runner. It manages filesystem artifacts produced or updated by agent-including transforms.
