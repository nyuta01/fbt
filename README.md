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

`fbt` is a local-first build tool for generated files.

Use it when your team has source files such as logs, tickets, support replies,
incident notes, product docs, or reference manuals, and wants to generate a
file artifact that is reproducible, inspectable, and explainable.

The mental model is:

```text
source files + instructions + external runner
  -> generated artifact
  -> versioned build receipt, checks, diff, lineage, and standard exports
```

fbt does not generate content by itself. It calls an external runner command.
That command can wrap OpenAI, Claude Code, Codex, Gemini, a script, a document
converter, or an internal service, as long as it speaks the fbt runner protocol.

## What You Get

Without fbt, an LLM or agent-generated manual is often just a file in a folder.
After a few days, nobody can quickly answer what it came from or whether it
should be regenerated.

fbt gives generated files build-tool behavior:

| Question | fbt command |
|---|---|
| Is this project ready to run? | `fbt doctor` |
| What would be regenerated, skipped, or blocked? | `fbt plan` |
| Generate the selected artifacts. | `fbt build` |
| Where is the current artifact and version? | `fbt artifact show` |
| Why did this artifact run, skip, or block? | `fbt artifact explain` |
| What changed from the previous version? | `fbt diff` |
| How do I send lineage or telemetry elsewhere? | `fbt export openlineage`, `fbt export otel` |

Human review, approval, pull requests, releases, publishing, scheduling,
provider SDKs, and artifact storage stay outside fbt. fbt owns the local build
control plane for generated file artifacts.

## Project Anatomy

A project declares what can be built. The common shape is:

```text
my_project/
  fs_project.yml        # project name, paths, runners, selectors
  sources/              # YAML declarations for existing input files
  transforms/           # recipes: inputs + runner + outputs
  assets/               # prompts, format guides, style guides
  policies/             # read/write scope for runner work
  evals/                # deterministic checks for generated outputs
  target/artifacts/     # current generated files
  .fbt/state/           # local receipts, manifests, run results
  .fbt/artifacts/       # immutable artifact version snapshots
```

Only the source files and recipe matter to the runner. fbt records the result:
which files were read, which runner was called, which output was committed,
which checks ran, and which artifact version is now current.

## Install

Download the current macOS, Linux, or Windows archive from
[GitHub Releases](https://github.com/nyuta01/fbt/releases/tag/v0.1.0), or build
from source:

```bash
git clone https://github.com/nyuta01/fbt.git
cd fbt
make build
./bin/fbt version
```

## First Successful Loop

This quickstart uses deterministic demo runners, so it works offline.

```bash
fbt init knowledge_ops --template support
fbt doctor --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select tag:support
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact explain case_summaries --project-dir knowledge_ops
```

What those commands prove:

| Step | What happens |
|---|---|
| `init` | Creates a runnable project with sample sources and demo runners. |
| `doctor` | Checks project config, local state, and runner readiness. |
| `plan` | Shows which transforms will run and why before writing files. |
| `build` | Calls runners, validates outputs, commits artifact versions, and writes receipts. |
| `artifact show` | Shows the current artifact path, version, digest, runner, and checks. |
| `artifact explain` | Shows the source fingerprints, upstream state, and run/skip/block reason. |

Shortened output from the offline project:

```text
Plan
  selected  2
  run       2

RUN     case_summaries
        because  output missing

RUN     weekly_support_insights
        because  upstream artifact selected to run

SUCCESS case_summaries
        output     case_summaries -> target/artifacts/support/case_summaries
        committed  case_summaries@sha256:a5b4dfd91df7
        next       fbt artifact show case_summaries --project-dir knowledge_ops

Artifact: case_summaries
  Path        target/artifacts/support/case_summaries
  Version     case_summaries@sha256:a5b4dfd91df7
```

At this point, generated files live under `target/artifacts`, immutable snapshots under `.fbt/artifacts`, and receipts under `.fbt/state`.

## Real Workflow Example

Imagine an SRE team has incident evidence:

```text
event log
  checkout-api p95 latency stayed above 2400 ms.
  database connection pool saturation was observed.

response notes
  Shift checkout read traffic away from the saturated replica.
  Ask support to verify payment status before asking customers to retry.

postmortem
  Add an alert at 80% connection usage.
  Document the traffic shift procedure.
```

The desired artifact is not another chat response. It is a runbook:

```text
target/artifacts/runbooks/incident_response_runbook.md

Purpose
Detection
Immediate Response
Investigation
Mitigation
Recovery
Customer Communication
Source Evidence
```

The example project `examples/incident_response_runbook` declares:

| Recipe part | Value |
|---|---|
| Read from | incident event logs, response notes, postmortems, existing runbooks |
| Ask | the `openai.responses` runner to draft the runbook |
| Write | `target/artifacts/runbooks/incident_response_runbook.md` |
| Check | required runbook sections are present |
| Record | input fingerprints, runner metadata, artifact version, checks, lineage |

The workflow is the same as the offline loop:

```bash
fbt doctor --project-dir examples/incident_response_runbook
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt artifact show incident_response_runbook --project-dir examples/incident_response_runbook
fbt artifact explain incident_response_runbook --project-dir examples/incident_response_runbook
```

This example uses a real runner, so `build` requires the configured command and credentials. fbt adds the controlled loop: preview, generate, check, version, inspect, diff, and export.

## Where fbt Fits

fbt is intentionally small. It composes with existing tools instead of replacing
them:

- dbt, DataChain, DVC, and Snakemake can prepare upstream data files.
- OpenAI, Claude Code, Codex, Gemini, scripts, or internal services can be
  wrapped as runners.
- Git, PRs, CI, release tools, and knowledge-base workflows can own approval
  and publishing.
- OpenLineage, Marquez, OpenTelemetry, Jaeger, Tempo, Grafana, and
  OpenMetadata-compatible workflows can own visualization and cataloging.

fbt stays focused on one job: build generated files from declared filesystem
inputs and leave an inspectable local record.

## Read Next

Start with the [usage guide](docs/usage-guide.md) for the command loop. Use the [manual generation guide](apps/docs/src/content/docs/get-started/manual-generation.mdx) for realistic examples and the [CLI reference](docs/cli-reference.md) for exact commands.

For implementation contracts, read the [design doc](docs/design-doc.md),
[core spec](docs/spec.md), [schema/versioning spec](docs/schema-and-versioning-spec.md),
[runner discovery spec](docs/runner-discovery-spec.md),
[runner protocol spec](docs/runner-protocol-spec.md), and
[security/conformance spec](docs/security-and-conformance-spec.md). The
published docs site is [nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).
