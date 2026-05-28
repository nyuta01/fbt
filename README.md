# fbt

Status: MVP-ready local release candidate  
Created: 2026-05-28  
Service name: `fbt`  
Scope: a filesystem transformation control plane for unstructured and semi-structured documents

`fbt` stands for **file build tool**. It is designed as a lightweight, local-first control plane for transforming filesystem artifacts such as Markdown, Word, Excel, PDF, logs, chats, tickets, investigation notes, and generated reports.

`fbt` does not implement document conversion, OCR, LLM calls, or agent runtimes inside the core. Transform execution is delegated to external runners and plugins. The core manages project definitions, dependency graphs, state, artifact versions, evals, review and approval state, lineage, and the runner protocol.

## Quick Start

Runnable local MVP flow from a source checkout:

```sh
fbt init knowledge_ops --template support
fbt parse --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir knowledge_ops --comment "Reviewed locally"
fbt build --project-dir knowledge_ops --select weekly_support_insights
fbt docs generate --project-dir knowledge_ops
```

Implementation status: the CLI implements `help`, `version`, `init`, `parse`,
`plan`, `build`, `eval`, `review`, `diff`, `docs`, `state`, `artifact`, and
`runner` diagnostics. The local MVP includes protocol runners, deterministic
evals, pending review gates, approval state, confidence promotion, immutable
artifact versions, artifact diffing, static docs generation, runnable templates,
and conformance plus local release-binary checks. Optional deterministic demo
LLM and agent runner examples live under `runners/` without provider SDK
dependencies.

The base runtime works with only the local filesystem.

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
| [Runner Authoring Guide](docs/runner-authoring-guide.md) | Practical runner implementation checklist and protocol conformance harness |
| [Schema and Versioning Spec](docs/schema-and-versioning-spec.md) | Config versioning, schema compatibility, artifact type registry, descriptor rules |
| [Runner Discovery Spec](docs/runner-discovery-spec.md) | Runner resolution, plugin manifest shape, and diagnostics |
| [Security and Conformance Spec](docs/security-and-conformance-spec.md) | Core security model and MVP conformance scenarios |
| [Standard Export Spec](docs/standard-export-spec.md) | OpenLineage, OpenTelemetry, OpenMetadata, and visualization export contracts |
| [Standard Visualization Guide](docs/standard-visualization-guide.md) | Marquez/OpenLineage and OTel backend visualization recipes |

Research:

| Document | Purpose |
|---|---|
| [dbt Core Overview Report](docs/research/dbt-core-overview-report.md) | dbt-core architecture and sources of advantage |
| [Related Landscape Report](docs/research/related-landscape-report.md) | Similar tools, adjacent categories, and differentiation |
| [Naming and Standards Research](docs/research/fbt-naming-and-spec-standards-research.md) | Naming, standards, lineage/provenance, runner protocol research |
| [OpenMetadata Catalog Export Evaluation](docs/research/openmetadata-catalog-export-evaluation.md) | Decision on OpenMetadata integration through OpenLineage ingestion |

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

This repository contains the MVP implementation and source-of-truth
specifications. It includes the Go CLI, project/resource parsing, manifest graph
generation, descriptor and state primitives, dirty-state planning, runner
discovery diagnostics, protocol runners, deterministic evals, review gates,
artifact approvals, immutable artifact version storage, local templates,
artifact diffs, and static Markdown project docs. `make verify` runs harness,
docs, Go, CLI smoke, knowledge-loop smoke, runner conformance, product
conformance, and local release-binary checks.

Release publication is not complete until a maintainer configures the public
remote, signing setup, and signed `v0.1.0` tag. See `CONTRIBUTING.md` and the
release tasks in `docs/exec-plans/feature-list.json`.
