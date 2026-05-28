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

`fbt` is a build tool for files generated from other files.

The simplest mental model is:

```text
sources + instructions + runner -> artifact + build receipt
```

- `sources` are files your team already has: logs, tickets, notes, product
  docs, approved replies.
- `instructions` say what to create: prompt, required format, style guide,
  checks, review rule.
- `runner` is the external worker: OpenAI, Claude Code, Codex, Gemini, a script,
  or an internal command.
- `artifact` is the generated file under `target/artifacts`.
- `build receipt` is fbt's local record of the exact inputs, runner, output
  version, checks, approval, and lineage.

Use fbt when the generated file must be reproducible, reviewable, and
explainable. Do not use it as a chat UI, CMS, ticket system, hosted knowledge
base, scheduler, or LLM provider.

## What It Solves

Without fbt, an LLM-generated manual or runbook is usually just a file. A
reviewer cannot easily answer:

- What source files was this based on?
- Which prompt and format rules were used?
- Which runner and model produced this version?
- Was this exact version reviewed?
- What should be rebuilt if an input changes?

fbt answers those questions by treating generated files like build outputs.

## Example: Incident Notes To Runbook

The easiest place to see fbt is after an incident.

An SRE team has already written down what happened:

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

What the team actually wants is not another summary. They want a runbook the
next on-call engineer can use:

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

fbt is the layer that makes this conversion controlled. The example project
already contains a recipe that says:

| Recipe part | Value |
|---|---|
| Read from | incident event logs, response notes, postmortems, existing runbooks |
| Ask | the `openai.responses` runner to draft the runbook |
| Write | `target/artifacts/runbooks/incident_response_runbook.md` |
| Check | required runbook sections are present |
| Review | SRE lead must approve the exact generated version |

So the user workflow is short:

```bash
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
```

You get a preview: fbt tells you whether the runbook will be created, skipped,
or blocked before any file is generated.

```bash
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
```

You get the runbook file plus fbt's receipt: which source files, instructions,
runner, checks, and artifact version produced it.

```bash
fbt review show incident_response_runbook --project-dir examples/incident_response_runbook
fbt review approve incident_response_runbook \
  --project-dir examples/incident_response_runbook \
  --comment "SRE lead approved"
```

You get an approval record attached to that exact generated version.

```bash
fbt artifact history incident_response_runbook --project-dir examples/incident_response_runbook
```

You get the answer to "where did this runbook come from?"

This example uses a real runner, so `build` requires the configured runner and
credentials. The quickstart below uses demo runners and works offline.

## Other Fit Cases

The same shape applies to:

| Source files | Artifact |
|---|---|
| incident logs plus response notes | incident response runbook |
| investigation notes | standard operating procedure |
| raw case records | reviewed summaries and weekly insights |

## Try It Locally

The quickstart uses deterministic demo runners, so it works without provider
credentials:

```bash
fbt init knowledge_ops --template support
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir knowledge_ops --comment "Reviewed locally"
fbt artifact history case_summaries --project-dir knowledge_ops
```

The full transcript is in the [quickstart demo](apps/docs/src/content/docs/get-started/quickstart.mdx).

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

## Documentation

Start with the [usage guide](docs/usage-guide.md), [manual generation guide](apps/docs/src/content/docs/get-started/manual-generation.mdx), and [CLI reference](docs/cli-reference.md). Core contracts are the [design doc](docs/design-doc.md), [core spec](docs/spec.md), [schema/versioning spec](docs/schema-and-versioning-spec.md), [runner discovery spec](docs/runner-discovery-spec.md), [runner protocol spec](docs/runner-protocol-spec.md), and [security/conformance spec](docs/security-and-conformance-spec.md). The published docs site is [nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).
