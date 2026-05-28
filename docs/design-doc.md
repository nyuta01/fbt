# fbt Design Doc

Status: Draft
Created: 2026-05-28
Updated: 2026-05-29
Service name: `fbt`
Scope: a filesystem artifact build tool for local-first transformations

## 1. Overview

`fbt` is a file build tool. It manages a project graph that turns source files
and directories into generated artifacts through external runners.

The core job is intentionally narrow:

```text
parse project definitions
  -> plan changed transforms
  -> call configured runners
  -> validate and commit artifact versions
  -> record evals, policy decisions, lineage, and exports
```

The core does not implement document conversion, OCR, LLM providers, agent
runtimes, scheduling, publishing, or human approval workflow. Those belong in
external runners or adjacent tools.

That boundary is product-defining. fbt does not replace dbt, DataChain, DVC,
Snakemake, remark, Pandoc, provider SDKs, schedulers, artifact stores, or
catalogs. It composes with them by treating their files or outputs as sources,
runners, artifacts, or standard export destinations.

## 2. Problem

Operational work produces important files continuously:

- support tickets and responses
- incident logs and response notes
- Markdown, Word, Excel, PDF, and HTML documents
- prompt assets, rubrics, and style guides
- generated reports, manuals, runbooks, and drafts

Teams often transform those files with scripts, models, agents, and manual
copy-paste. The hard questions are not only "can a model write this?" but:

- Which files and assets produced this artifact?
- Which runner, model, policy, and evals were involved?
- Is the artifact stale because sources or prompts changed?
- What exact version is current?
- How can lineage be exported to existing observability or metadata tools?

`fbt` brings build-tool discipline to this file-based work without becoming the
model provider, workflow orchestrator, review app, or knowledge base.

## 3. Goals

1. Treat filesystem sources and artifacts as first-class graph resources.
2. Let users define transforms with `source` and `ref` dependencies.
3. Keep transform execution outside core behind a runner protocol.
4. Track artifact versions with content descriptors and immutable history.
5. Explain why transforms will run, skip, or block.
6. Run deterministic evals and record confidence granted by those evals.
7. Enforce local path, policy, and output-candidate boundaries before commit.
8. Export lineage and traces in standard-compatible formats.
9. Stay small enough to use as a normal CLI in local and CI workflows.

## 4. Non-Goals

1. General-purpose workflow orchestration.
2. A hosted scheduler or daemon.
3. A metadata database or SaaS backend.
4. Built-in OCR, document conversion, LLM provider clients, or agent runtimes.
5. Human review, assignment, approval, comments, publishing, or release
   workflow.
6. A custom lineage visualization backend.
7. Full reproducibility guarantees for stochastic model outputs.
8. Reimplementing dbt, DataChain, DVC, Snakemake, remark, Pandoc, provider
   SDKs, schedulers, artifact stores, or metadata catalogs.

Human approval is intentionally outside fbt. Use Git, PRs, CI, release tooling,
ticketing systems, or knowledge-base publishing workflows to decide whether a
generated artifact should be used.

## 5. Design Principles

### One Job

`fbt` should do one thing well: build versioned filesystem artifacts from
declared source files through external runners.

`build` is the primary execution verb for that reason. It means "produce the
declared artifact outputs and record their receipt," not merely "run the
configured worker."

### Local First

The base tool is a CLI and local state directory. It has no daemon, scheduler,
database, cloud account, or required server.

### Runner Boundary

Runners own transformation logic. A runner may call OpenAI, Claude, Gemini,
Codex, Claude Code, a shell command, a converter, or a custom service. fbt core
only requires that the runner speak the protocol and write output candidates to
the assigned work directory.

### Inspectable State

Core records enough information to answer what ran, why it ran, what it read,
what it wrote, which evals passed, and which artifact version is current.

### Standard Exports

fbt does not need a custom graph UI to be useful. It exports OpenLineage events
and OpenTelemetry-compatible traces so users can inspect runs in existing tools.

## 6. Core Concepts

| Concept | Meaning |
|---|---|
| Project | Directory containing fbt definitions |
| Source | Read-only input file, glob, or directory |
| Artifact | Logical output managed by fbt |
| Artifact version | Immutable content snapshot with descriptor and digest |
| Transform | Contract that produces one or more artifacts |
| Runner | External process that performs transform logic |
| Asset | Prompt, template, style guide, rubric, schema, or example |
| Policy | Path, tool, network, timeout, size, and cost boundary |
| Eval | Deterministic or delegated quality check |
| Confidence | Trust class granted by successful evals |
| Lineage | Relationship among sources, assets, transforms, runs, and artifacts |
| State | Local manifest snapshots, run results, versions, evals, and decisions |

## 7. Execution Model

```text
parse -> doctor -> plan -> build -> inspect/export
```

### Parse

Read project YAML files and build the manifest graph.

### Doctor

Check local readiness: project config, state lock, runner protocol, capabilities,
and required runner environment variables.

### Plan

Compare the current manifest with local state. A transform runs when sources,
assets, policies, evals, runner identity, model parameters, upstream artifact
versions, or declared outputs changed. A transform blocks when required upstream
artifacts do not exist or do not meet configured confidence/eval requirements.
Planning is read-only: it does not write fbt state or start runners.

### Build

Invoke the runner, stage output candidates, enforce boundaries, run evals,
commit artifact versions, and update local state. Build is the only normal
command that writes artifact receipts.

### Inspect And Export

Use `fbt artifact`, `fbt diff`, `fbt export openlineage`, and `fbt export otel`
to inspect the result or pass metadata to existing tools.

## 8. Runner Protocol

The MVP runner protocol is JSON-RPC 2.0 over stdio with JSONL framing.

Runner requests include:

- transform ID and type
- resolved source and artifact inputs
- declared outputs
- asset paths
- model metadata
- policy context
- work directories
- timeout and size limits
- previous state references

Runner results include:

- output candidates
- logs and warnings
- runner events
- usage and cost metadata
- provenance and trace summaries

Core computes authoritative artifact descriptors; runner-supplied digests are
advisory.

## 9. Package Shape

```text
cmd/fbt              CLI entrypoint
internal/project    project discovery
internal/config     YAML decoding and validation
internal/parser     resource parsing
internal/manifest   manifest generation
internal/graph      dependency graph and selectors
internal/planner    dirty-state planning
internal/build      run, eval, commit lifecycle
internal/state      local state store
internal/artifact   descriptors and artifact store
internal/runner     runner discovery and protocol client
internal/eval       eval execution
internal/lineage    OpenLineage and OTel exports
internal/diff       artifact diffs
runners/            optional external runner implementations
```

## 10. Current MVP

Implemented MVP scope:

- local CLI
- YAML project config
- project templates
- manifest and parser diagnostics
- selectors
- planner
- build lifecycle
- external runner protocol
- deterministic demo runners
- optional OpenAI runner
- artifact versions and local state
- deterministic evals
- artifact path/show/history/explain
- docs generation
- OpenLineage and OTLP/JSON export
- smoke, conformance, docs, and distribution checks

Explicitly removed from core:

- `fbt review`
- approval state
- review gates
- `human_review` evals
- approval facets in standard exports

## 11. Roadmap

Near-term improvements should preserve the single-job boundary:

1. Better source-window ergonomics for daily file growth.
2. Stronger parser diagnostics for unsupported or legacy fields.
3. More useful artifact explanations and diffs.
4. More runner adapters, especially CLI-agent adapters.
5. Richer standard export coverage without adding a custom backend.

Post-MVP collaboration features should integrate with external systems rather
than move review, scheduling, or publishing into core.

## 12. Positioning

Short:

> fbt is a file build tool for AI-era filesystem artifacts.

Long:

> fbt is a lightweight local-first CLI for transforming source files into
> versioned artifacts through external runners. It records what was built from
> what, which prompts and policies were used, which evals passed, and exports
> lineage to standard tools.

Avoid:

> fbt is not an AI agent runner, review system, scheduler, CMS, ticket system,
> or hosted knowledge base.
