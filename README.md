<p align="center">
  <img src="https://raw.githubusercontent.com/nyuta01/fbt/main/apps/docs/public/favicon.svg" alt="fbt" width="96" height="96" />
</p>

<h1 align="center">fbt</h1>

<p align="center">
<a href="https://github.com/nyuta01/fbt/releases/tag/v0.1.0"><img alt="Release" src="https://img.shields.io/github/v/release/nyuta01/fbt?label=fbt"></a>
<a href="https://nyuta01.github.io/fbt/"><img alt="Docs" src="https://img.shields.io/badge/docs-GitHub%20Pages-0f766e"></a>
<a href="LICENSE"><img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache--2.0-blue.svg"></a>
<a href="https://github.com/nyuta01/fbt/actions/workflows/verify.yml"><img alt="verify" src="https://img.shields.io/github/actions/workflow/status/nyuta01/fbt/verify.yml?branch=main&label=verify"></a>
</p>

Local-first file build tool for knowledge artifacts.

`fbt` is a lightweight control plane for transforming filesystem artifacts:
logs, tickets, Markdown, Word, Excel, PDFs, prompts, rubrics, investigation
notes, generated runbooks, reports, and manuals.

It is not an LLM provider, document converter, OCR engine, or agent runtime.
Those capabilities live behind external runners. fbt core owns the project
graph, planning, local state, artifact versions, evals, review gates, lineage,
and standard exports.

## What can you actually do today?

- Turn local evidence files into generated Markdown artifacts through demo or
  external protocol-compatible runners.
- Plan, build, review, and gate downstream transforms by approved artifact
  versions.
- Inspect local lineage with `artifact path`, `artifact history`, and
  `artifact explain`.
- Generate project docs and export OpenLineage NDJSON plus OTLP/JSON traces.

```text
project/
├── fs_project.yml        # project contract and paths
├── sources/              # read-only input declarations
├── transforms/           # outputs, refs, runner, assets, policy, evals
├── prompts/              # prompt assets for LLM/agent runners
├── assets/               # style guides, schemas, rubrics, examples
├── policies/             # read/write/network/review constraints
├── evals/                # deterministic checks and review gates
├── target/               # materialized artifact files
└── .fbt/state/           # manifest, run results, versions, approvals
```

## Surfaces

- **`fbt`** - Go CLI for init, parse, doctor, plan, build, eval,
  review, diff, docs, state, artifact inspection, runner diagnostics, and
  standard exports.
- **`apps/docs`** - Astro/Starlight documentation site published at
  [nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).
- **External runners** - protocol adapters for scripts, provider APIs, and CLI
  agents such as OpenAI, Claude Code, Codex, Claude, and Gemini.

## Documentation

User-facing documentation lives in [`apps/docs/`](apps/docs/README.md) and is
published at [nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).

Start with the [usage guide](docs/usage-guide.md), [design doc](docs/design-doc.md),
[core spec](docs/spec.md), and [CLI reference](docs/cli-reference.md). The main
contracts are [project config](docs/project-config-spec.md),
[schema/versioning](docs/schema-and-versioning-spec.md),
[runner discovery](docs/runner-discovery-spec.md),
[runner protocol](docs/runner-protocol-spec.md), and
[security/conformance](docs/security-and-conformance-spec.md).

## Quickstart

The quickstart is a control-plane demo. It uses a tiny support fixture and
deterministic demo runners to prove that fbt can parse, plan, build, review,
inspect, and export local artifact state. It is not a model-quality benchmark
or a realistic support-manual workflow.

```bash
fbt init knowledge_ops --template support
fbt parse --project-dir knowledge_ops
fbt doctor --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt review approve case_summaries \
  --project-dir knowledge_ops \
  --comment "Reviewed locally"
fbt build --project-dir knowledge_ops --select weekly_support_insights
fbt docs generate --project-dir knowledge_ops
fbt artifact history case_summaries --project-dir knowledge_ops
```

Captured result from the same flow, with long hashes shortened:

```text
Build: 1 selected, 1 run, 0 skipped, 0 blocked
success transform.knowledge_ops.case_summaries
  committed: artifact_version.knowledge_ops.case_summaries.sha256_a5b4...
artifact.knowledge_ops.case_summaries
  status: approved
  confidence: reviewed
success transform.knowledge_ops.weekly_support_insights
  committed: artifact_version.knowledge_ops.weekly_support_insights.sha256_49f...
```

Files created by the run:

```text
knowledge_ops/target/artifacts/support/case_summaries/index.md
knowledge_ops/target/artifacts/support/weekly_insights.md
knowledge_ops/target/docs/index.md
knowledge_ops/.fbt/state/{run_results.jsonl,artifact_versions.json}
```

```bash
mkdir -p knowledge_ops/target/lineage knowledge_ops/target/telemetry
fbt export openlineage --project-dir knowledge_ops --output knowledge_ops/target/lineage/openlineage.ndjson
fbt export otel --project-dir knowledge_ops --output knowledge_ops/target/telemetry/otel.json
```

```text
OpenLineage events written to knowledge_ops/target/lineage/openlineage.ndjson
Events: 2
OTel traces written to knowledge_ops/target/telemetry/otel.json
Spans: 4
```

The detailed walkthrough is in
[What you can do today](apps/docs/src/content/docs/get-started/what-you-can-do.mdx)
and the [Quickstart](apps/docs/src/content/docs/get-started/quickstart.mdx).
For real incident-runbook or support-manual generation, start from
[Manual generation](apps/docs/src/content/docs/get-started/manual-generation.mdx).

## Install

Download the current macOS, Linux, or Windows archive from
[GitHub Releases](https://github.com/nyuta01/fbt/releases/tag/v0.1.0) and
verify it:

```bash
shasum -a 256 -c SHA256SUMS
```

Or build from source:

```bash
git clone https://github.com/nyuta01/fbt.git
cd fbt
make build
./bin/fbt version
```

## Examples

- [Knowledge operations](examples/knowledge_ops/README.md) - local support
  knowledge loop with demo runners.
- [Incident response runbook](examples/incident_response_runbook/README.md) -
  incident logs and response notes to a reviewed runbook.
- [Support resolution manual](examples/support_resolution_manual/README.md) -
  inquiry and response logs to a support manual.

The practical examples can use the optional OpenAI Responses runner under
`runners/openai` when `OPENAI_API_KEY` is set.

## Lineage and standards

fbt records local lineage for sources, transform assets, runs, output
candidates, artifact versions, evals, approvals, and policy decisions.

```bash
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact history case_summaries --project-dir knowledge_ops
fbt export openlineage --project-dir knowledge_ops
fbt export otel --project-dir knowledge_ops
```

OpenLineage output is intended for Marquez and OpenMetadata ingestion paths.
OTLP/JSON output is intended for trace backends such as Jaeger, Tempo, and
Grafana.

## Releases

The current MVP release is [`v0.1.0`](https://github.com/nyuta01/fbt/releases/tag/v0.1.0).
The tag is signed, `make verify` passed before publication, GitHub Actions
passed for `main` and `v0.1.0`, and release archives plus `SHA256SUMS` are
attached to the release.

## Repository harness

This repository follows an AI-first operating model: compact router docs,
structured task state, active execution plans, and one deterministic
verification gate.

```bash
make agent-init   # restart context for the next agent
make verify       # harness, docs, Go, CLI/e2e, conformance, dist checks
```

See [AGENTS.md](AGENTS.md),
[harness engineering](docs/methodology/harness-engineering.md), and
[self-PDCA loop](docs/methodology/self-pdca-loop.md) for the operating model.

## Non-goals

- becoming a general-purpose workflow orchestrator
- implementing document conversion, OCR, LLM providers, or agent runtimes in core
- requiring a daemon, scheduler, metadata database, web server, or cloud account
- promising full reproducibility for stochastic AI transformations
- replacing a CMS, knowledge base, ticket system, or document editor
