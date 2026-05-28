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

`fbt` is a local-first file build tool for turning operational evidence into
reviewed, traceable knowledge artifacts.

Use it when source material already exists as files and the output is produced
by an LLM, agent, script, or provider runner:

- incident logs and response notes to an approved incident runbook
- support tickets and reply logs to a support resolution manual
- investigation notes to a repeatable operating procedure
- raw case records to reviewed summaries and weekly insights

fbt does not generate the content by itself. It gives external runners a
controlled workflow: declare inputs, plan work, run the runner, version the
output, require review, inspect lineage, and export standard records.

## Why fbt Exists

AI-generated operational documents are hard to trust if the workflow only
produces a loose Markdown file. A reviewer needs to know:

- which source files were used
- which prompt, style guide, policy, and runner produced the output
- whether required checks passed
- whether a human approved the exact generated version
- what downstream files depended on that approved version
- how to export lineage or traces into standard tools

fbt is the local control plane around that process. The model or agent can be
OpenAI, Claude Code, Codex, Gemini, a shell script, or an internal tool, as long
as it is exposed through a compatible runner.

## Concrete Example

The repository includes a production-shaped support manual example:

```text
examples/support_resolution_manual/
├── data/support/tickets/          # customer inquiries
├── data/support/response_logs/    # support replies and decisions
├── data/reference/product_docs/   # product facts
├── data/reference/macros/         # existing approved language
├── assets/                        # prompt, style guide, required format
├── policies/                      # read/write/network/review limits
├── evals/                         # deterministic section checks
└── transforms/                    # support_resolution_manual transform
```

The intended output is:

```text
target/artifacts/support/support_resolution_manual.md
```

The workflow is:

```bash
fbt parse --project-dir examples/support_resolution_manual
fbt doctor --project-dir examples/support_resolution_manual
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual

fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt review show support_resolution_manual --project-dir examples/support_resolution_manual
fbt review approve support_resolution_manual \
  --project-dir examples/support_resolution_manual \
  --comment "Support lead approved"

fbt docs generate --project-dir examples/support_resolution_manual
fbt artifact history support_resolution_manual --project-dir examples/support_resolution_manual
```

In that flow, fbt is responsible for:

| Step | What fbt does |
|---|---|
| `parse` | Reads `fs_project.yml`, sources, transforms, assets, policies, and evals. |
| `doctor` | Checks local state, runner command readiness, env requirements, and protocol capabilities. |
| `plan` | Explains what will run, skip, or block before any output is created. |
| `build` | Calls the configured runner and commits the candidate output as an immutable artifact version. |
| `review` | Binds approval to the exact artifact version, not just a path. |
| `artifact` | Shows where the file came from, what produced it, and what version is current. |
| `docs/export` | Writes local project docs, OpenLineage events, and OTLP/JSON traces. |

## What You Can Do Today

- Define a file-based transformation project in YAML.
- Use deterministic demo runners for local smoke tests.
- Use external runners for provider-backed work, including OpenAI or CLI-agent
  adapters that speak the fbt runner protocol.
- Build Markdown artifacts into `target/artifacts`.
- Store immutable generated versions under `.fbt/artifacts`.
- Gate downstream transforms on reviewed artifact versions.
- Inspect lineage locally with `artifact path`, `artifact history`, and
  `artifact explain`.
- Generate local project docs with `fbt docs generate`.
- Export OpenLineage NDJSON and OTLP/JSON traces for existing lineage and
  observability tools.

## Try the Control-Plane Demo

The quickstart is a small fixture, not the main business use case. It proves
that fbt can parse, plan, build, review, inspect, and export local artifact
state without external services.

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

The run creates:

```text
knowledge_ops/target/artifacts/support/case_summaries/index.md
knowledge_ops/target/artifacts/support/weekly_insights.md
knowledge_ops/target/docs/index.md
knowledge_ops/.fbt/state/run_results.jsonl
knowledge_ops/.fbt/state/artifact_versions.json
```

Export standard records from the same local state:

```bash
mkdir -p knowledge_ops/target/lineage knowledge_ops/target/telemetry
fbt export openlineage --project-dir knowledge_ops --output knowledge_ops/target/lineage/openlineage.ndjson
fbt export otel --project-dir knowledge_ops --output knowledge_ops/target/telemetry/otel.json
```

Expected result:

```text
OpenLineage events written to knowledge_ops/target/lineage/openlineage.ndjson
Events: 2
OTel traces written to knowledge_ops/target/telemetry/otel.json
Spans: 4
```

## Project Shape

A project is a directory with `fs_project.yml`, source declarations,
transform declarations, assets, policies, evals, `target/` outputs, and local
`.fbt/state` records.

## Boundaries

fbt core intentionally does not implement:

- LLM providers
- agent runtimes
- OCR or document conversion
- a scheduler, daemon, metadata database, or hosted UI
- a CMS, knowledge base, ticket system, or document editor

Those capabilities stay outside core. fbt coordinates them through external
runners and records what happened locally.

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

## Documentation

Start with:

- [Usage guide](docs/usage-guide.md)
- [Manual generation guide](apps/docs/src/content/docs/get-started/manual-generation.mdx)
- [CLI reference](docs/cli-reference.md)
- [Project config spec](docs/project-config-spec.md)
- [Runner protocol spec](docs/runner-protocol-spec.md)

Core design and contracts:

- [Design doc](docs/design-doc.md)
- [Core spec](docs/spec.md)
- [Schema and versioning spec](docs/schema-and-versioning-spec.md)
- [Runner discovery spec](docs/runner-discovery-spec.md)
- [Security and conformance spec](docs/security-and-conformance-spec.md)
- [Standard export spec](docs/standard-export-spec.md)

The published docs site is
[nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).
