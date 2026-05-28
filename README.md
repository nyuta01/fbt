# fbt

Status: Draft  
Created: 2026-05-28  
Service name: `fbt`  
Scope: a filesystem transformation control plane for unstructured and semi-structured documents

`fbt` stands for **file build tool**. It is designed as a lightweight, local-first control plane for transforming filesystem artifacts such as Markdown, Word, Excel, PDF, logs, chats, tickets, investigation notes, and generated reports.

`fbt` does not implement document conversion, OCR, LLM calls, or agent runtimes inside the core. Transform execution is delegated to external runners and plugins. The core manages project definitions, dependency graphs, state, artifact versions, evals, review and approval state, lineage, and the runner protocol.

## Quick Start

Target user experience:

```sh
fbt init knowledge_ops --template support
cd knowledge_ops
fbt parse
fbt plan --select tag:support
fbt build --select case_summaries
fbt diff case_summaries --against last-approved
fbt review approve case_summaries --comment "Reviewed"
fbt docs generate
```

Implementation status: the current CLI implements `help`, `version`, the first
product inspection commands (`parse`, `plan`, `state`, `artifact`), and runner
discovery diagnostics (`runner list`, `runner doctor`, `runner validate`).
The JSON-RPC stdio runner protocol client exists but is not wired into build
execution yet. Local fake and command runners are available for tests and local
MVP wiring. Build execution, evals, review, diff, and docs generation are
specified but not implemented yet.

The base runtime should work with only the local filesystem.

- No daemon
- No scheduler
- No required metadata database
- No required web server
- No required cloud account
- Runners and plugins are externalized only as needed

## Documentation Map

Start here:

| Document | Purpose |
|---|---|
| [Usage Guide](docs/usage-guide.md) | End-to-end workflow from project initialization to build, review, and docs generation |
| [Knowledge Loop Example](docs/examples/knowledge-loop-example.md) | Customer support, incident response, and agile management examples |
| [Design Doc](docs/design-doc.md) | Background, principles, architecture, roadmap, and remaining decisions |

Specifications:

| Document | Purpose |
|---|---|
| [Core Spec](docs/spec.md) | Overall `fbt` semantics |
| [Project Config Spec](docs/project-config-spec.md) | `fs_project.yml` and resource YAML definitions |
| [CLI Reference](docs/cli-reference.md) | Commands, flags, selection syntax, exit codes |
| [Manifest Spec](docs/manifest-spec.md) | Canonical graph metadata produced by parsing |
| [State and Run Results Spec](docs/state-and-run-results-spec.md) | Local state, run results, artifact versions, approvals, eval and policy records |
| [Runner Protocol Spec](docs/runner-protocol-spec.md) | JSON-RPC protocol between `fbt-core` and external runners |
| [Schema and Versioning Spec](docs/schema-and-versioning-spec.md) | Config versioning, schema compatibility, artifact type registry, descriptor rules |
| [Runner Discovery Spec](docs/runner-discovery-spec.md) | Runner resolution, plugin manifest shape, and diagnostics |
| [Security and Conformance Spec](docs/security-and-conformance-spec.md) | Core security model and MVP conformance scenarios |

Research:

| Document | Purpose |
|---|---|
| [dbt Core Overview Report](docs/research/dbt-core-overview-report.md) | dbt-core architecture and sources of advantage |
| [Related Landscape Report](docs/research/related-landscape-report.md) | Similar tools, adjacent categories, and differentiation |
| [Naming and Standards Research](docs/research/fbt-naming-and-spec-standards-research.md) | Naming, standards, lineage/provenance, runner protocol research |

## Core Concepts

| Concept | Meaning |
|---|---|
| `source` | External input file or directory |
| `artifact` | Logical output managed by the project |
| `artifact_version` | Immutable content snapshot identified by `descriptor.digest` |
| `transform` | Contract that turns input artifacts into output artifacts |
| `transform_run` | Concrete execution activity for a transform |
| `transform_asset` | Prompt, template, script, style guide, rubric, example, schema, or config that affects a transform |
| `policy` | Read/write scope, tool, network, cost, and review constraints |
| `eval` | Deterministic, semantic, LLM-judge, or human-review quality check |
| `runner` | External process or plugin that executes transform logic |

## Standard Project Layout

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

Minimal project config:

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
```

## Positioning

`fbt` is not an AI agent runner. It is the control plane for filesystem artifacts generated or updated by transforms, including AI agent transforms.

Mapping to dbt:

| dbt | fbt |
|---|---|
| Warehouse relation | Filesystem artifact |
| Model | Transform |
| Adapter / materialization | Runner / plugin |
| SQL | Transform assets, code, LLM, agent, converter |
| `ref()` | Upstream artifact reference |
| `source()` | External file source |
| Data test | Artifact eval, document eval, LLM judge, human approval |
| `manifest.json` | Filesystem transformation manifest |
| `run_results.json` | Transform and eval execution results |
| Docs | Artifact lineage, eval report, review state |

## Non-Goals

- Becoming a general-purpose workflow orchestrator
- Implementing document conversion, OCR, LLM providers, or agent runtimes in core
- Requiring a daemon, scheduler, metadata database, web server, or cloud account
- Promising full reproducibility for stochastic AI transformations
- Replacing a CMS, knowledge base, ticket system, or document editor

## Current Status

This repository currently contains design and specification drafts, a baseline
AI-first engineering harness, a Go CLI scaffold, project/resource parsing,
manifest graph generation, descriptor and state primitives, dirty-state
planning, initial CLI inspection commands, and runner discovery diagnostics.
The JSON-RPC stdio runner protocol client is implemented, but build execution,
evals, review, diff, and docs generation are still pending. Local fake and
command runners exist for deterministic protocol and future build tests.
