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

User-facing documentation lives in [`apps/docs/`](apps/docs/README.md).

```bash
cd apps/docs
npm ci
npm run dev      # -> http://127.0.0.1:4321/fbt
```

Canonical source docs:

- [Usage guide](docs/usage-guide.md) - end-to-end local workflow
- [Design doc](docs/design-doc.md) - product principles and architecture
- [Core spec](docs/spec.md) - overall fbt semantics
- [CLI reference](docs/cli-reference.md) - commands, flags, selectors, exits
- [Project config spec](docs/project-config-spec.md) - `fs_project.yml` and resources
- [Schema and versioning spec](docs/schema-and-versioning-spec.md) - config and artifact version rules
- [Runner discovery spec](docs/runner-discovery-spec.md) - runner resolution and plugins
- [Runner protocol spec](docs/runner-protocol-spec.md) - core/runner boundary
- [Runner authoring guide](docs/runner-authoring-guide.md) - implementation checklist
- [Security and conformance spec](docs/security-and-conformance-spec.md) - trust boundary and MVP checks
- [Standard export spec](docs/standard-export-spec.md) - OpenLineage and OTel contracts
- [Practical examples](docs/examples/practical-manual-generation-examples.md) - incident runbooks and support manuals

## Quickstart

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

The generated support project uses deterministic demo runners so the loop runs
offline from a source checkout. Replace those runners with external adapters for
provider-backed execution.

## Install

Download the current release from
[GitHub Releases](https://github.com/nyuta01/fbt/releases/tag/v0.1.0).

| Platform | Asset |
|---|---|
| macOS (Apple Silicon) | `fbt_0.1.0_darwin_arm64.tar.gz` |
| macOS (Intel) | `fbt_0.1.0_darwin_amd64.tar.gz` |
| Linux (arm64) | `fbt_0.1.0_linux_arm64.tar.gz` |
| Linux (amd64) | `fbt_0.1.0_linux_amd64.tar.gz` |
| Windows (arm64) | `fbt_0.1.0_windows_arm64.zip` |
| Windows (amd64) | `fbt_0.1.0_windows_amd64.zip` |

Verify a download:

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

The current MVP release is
[`v0.1.0`](https://github.com/nyuta01/fbt/releases/tag/v0.1.0).

Release integrity:

- the release tag is signed with Git SSH signing
- `make verify` passed before publication
- GitHub Actions `verify` passed for `main` and `v0.1.0`
- release archives and `SHA256SUMS` are attached to the GitHub Release

Future releases should be cut from a clean tree:

```bash
make verify
git tag -s vX.Y.Z -m "fbt vX.Y.Z"
git push origin main
git push origin vX.Y.Z
```

## Repository harness

This repository follows an AI-first operating model: compact router docs,
structured task state, active execution plans, and one deterministic
verification gate.

```bash
make agent-init   # restart context for the next agent
make verify       # harness + drift + docs + Go + CLI/e2e smokes
                  # + docs-site build + runner/product conformance
                  # + local release-binary smoke
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
