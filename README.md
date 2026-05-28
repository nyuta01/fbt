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
  docs, reference manuals.
- `instructions` say what to create: prompt, required format, style guide,
  checks, and write policy.
- `runner` is the external worker: OpenAI, Claude Code, Codex, Gemini, a script,
  or an internal command.
- `artifact` is the generated file under `target/artifacts`.
- `build receipt` is fbt's local record of the exact inputs, runner, output
  version, checks, and lineage.

`build` is deliberate: fbt treats generated files as build outputs. `plan`
previews changes; `build` produces selected artifacts, commits immutable
versions, runs checks, and writes the local receipt. The runner is external.

Use fbt when a generated file must be reproducible and explainable. Do not use
it as a chat UI, CMS, ticket system, hosted knowledge base, scheduler, approval
workflow, or LLM provider.

## What It Solves

Without fbt, an LLM-generated manual or runbook is usually just a file. A user
cannot easily answer:

- What source files was this based on?
- Which prompt and format rules were used?
- Which runner and model produced this version?
- Which checks passed before the file was committed?
- What changed from the previous generated version?
- What should be rebuilt if an input changes?

fbt answers those questions by treating generated files like build outputs.
Human approval, pull requests, releases, and publishing remain outside fbt.

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
next on-call engineer can inspect and use:

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
| Record | input fingerprints, runner metadata, artifact version, checks, and lineage |

The commands are checkpoints, not a script to memorize:

1. Preview the work before spending runner time or writing files.

   ```bash
   fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
   ```

   fbt reads the recipe and current state. You get a preview that says whether
   the runbook will run, skip, or block, and why.

   Actual output from this repository:

   ```text
   Plan: 1 selected, 1 run, 0 skipped, 0 blocked
   run transform.incident_response_runbook.incident_response_runbook
     reason: no previous successful run
     reason: output missing
   ```

2. Generate the runbook.

   ```bash
   fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
   ```

   fbt sends the allowed source files and instructions to the runner. You get
   `target/artifacts/runbooks/incident_response_runbook.md` and a local receipt
   under `.fbt/state` and `.fbt/artifacts`.

3. Inspect the generated version.

   ```bash
   fbt artifact show incident_response_runbook --project-dir examples/incident_response_runbook
   ```

   You get the exact artifact version, digest, path, runner, model metadata,
   semantic descriptor, and generating run.

4. Compare with the previous generated version when one exists.

   ```bash
   fbt diff incident_response_runbook \
     --project-dir examples/incident_response_runbook \
     --against previous
   ```

5. Explain where the current runbook came from.

   ```bash
   fbt artifact history incident_response_runbook --project-dir examples/incident_response_runbook
   fbt artifact explain incident_response_runbook --project-dir examples/incident_response_runbook
   ```

The short version: `plan` is for deciding, `build` is for producing artifacts
and receipts, `artifact` is for inspection, `diff` is for comparison, and
`export` is for standard lineage/telemetry handoff.

This example uses a real runner, so `build` requires the configured runner and
credentials. The quickstart below uses demo runners and works offline.

## Other Fit Cases

The same shape applies to:

| Source files | Artifact |
|---|---|
| incident logs plus response notes | incident response runbook |
| investigation notes | standard operating procedure |
| raw case records | summaries and weekly insights |
| daily user questions and answers | FAQ candidates and manual update |

## Try It Locally

The quickstart uses deterministic demo runners, so it works without provider
credentials:

```bash
fbt init knowledge_ops --template support
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact history case_summaries --project-dir knowledge_ops
```

The output includes the same lifecycle signals, shortened here:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked
Build: 1 selected, 1 run, 0 skipped, 0 blocked
success transform.knowledge_ops.case_summaries
  committed: artifact_version...sha256_a5b4...

artifact_version...sha256_a5b4...
  artifact: artifact.knowledge_ops.case_summaries
  current: true
  runner: runner.knowledge_ops.demo.llm
  generated_by: transform_run.run_...
  confidence: structural
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
