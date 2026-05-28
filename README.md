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

## Example: Turn Cases Into A Manual

A support lead has a practical problem:

> We solved the same SSO billing question several times. I want one approved
> procedure that the next agent can use, and I want to know which tickets and
> notes the procedure came from.

The source files already contain the answer, but not in a reusable form:

```text
ticket SUP-10437
  Customer asks why seat count increased after SSO group sync.

response log
  Confirm SSO group sync, compare the IdP billing group with workspace members,
  explain that removed users stop counting after the next sync.

product doc
  Billable seats follow synced group membership.
```

fbt turns those files into this kind of artifact:

```text
target/artifacts/support/support_resolution_manual.md

Manual sections:
- when to use this procedure
- intake checklist
- triage steps
- resolution procedure
- escalation rule
- customer response template
- source evidence
```

And fbt keeps the receipt for that generated file:

```text
sources: tickets, response logs, product docs, approved macros
instructions: prompt, manual format, style guide, evidence checklist
runner: openai.responses / gpt-5
review: support_leads approved this exact artifact version
lineage: this manual came from these files and this runner invocation
```

In plain terms, fbt does not answer the support question itself. It makes the
generation process controlled:

1. declare which files are allowed as evidence
2. declare what the runner must create
3. run the external worker
4. store the generated manual as a versioned artifact
5. require review before that version becomes trusted

The transform is the recipe for that process:

```yaml
name: support_resolution_manual
runner: openai.responses
inputs:
  - source: support.inquiry_tickets
  - source: support.response_logs
  - source: reference.product_docs
  - source: reference.approved_macros
assets:
  - ref: support_resolution_prompt
  - ref: support_resolution_manual_format
outputs:
  - path: target/artifacts/support/support_resolution_manual.md
review:
  required: true
  group: support_leads
```

The checked-in support example is a real runner workflow, so `build` requires
the configured runner and credentials. The quickstart below uses demo runners.

The commands are the user workflow:

```bash
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt review show support_resolution_manual --project-dir examples/support_resolution_manual
fbt review approve support_resolution_manual \
  --project-dir examples/support_resolution_manual \
  --comment "Support lead approved"
fbt artifact history support_resolution_manual --project-dir examples/support_resolution_manual
```

`plan` shows what will happen, `build` calls the runner, `review show` lets a
lead inspect the generated manual, `review approve` records approval for that
exact version, and `artifact history` shows how the current manual was made.

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
