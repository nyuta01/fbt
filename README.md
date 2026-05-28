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

Suppose a support team wants to turn resolved tickets and agent response logs
into an official support resolution manual.

The example project is `examples/support_resolution_manual/`:

```text
data/support/tickets/*.jsonl          customer inquiries
data/support/response_logs/*.md       agent handling notes
data/reference/product_docs/*.md      product facts
data/reference/macros/*.md            approved customer language
assets/*.md                           prompt, format, style guide, rubric
transforms/support/manual.yml         output and runner definition
```

One source record looks like this:

```json
{"ticket_id":"SUP-10437","topic":"billing","summary":"Customer asks why seat count increased after SSO group sync","customer_impact":"unexpected invoice estimate","resolution_status":"resolved"}
```

One response log says what worked:

```md
## Agent Steps That Worked

1. Confirm the workspace has SSO group sync enabled.
2. Ask the admin to compare the identity provider billing group with the workspace member list.
3. Explain that removing users from the billing group takes effect after the next sync.
4. Escalate billing disputes when the customer reports a mismatch after sync.
```

The transform declares the conversion:

```yaml
name: support_resolution_manual
runner: openai.responses
inputs:
  - source: support.inquiry_tickets
  - source: support.response_logs
  - source: reference.product_docs
  - source: reference.approved_macros
outputs:
  - path: target/artifacts/support/support_resolution_manual.md
assets:
  - ref: support_resolution_prompt
  - ref: support_resolution_manual_format
  - ref: support_manual_style_guide
policy: support_manual_generation_scope
evals:
  - required_support_manual_sections
review:
  required: true
  group: support_leads
```

The format asset requires the generated manual to contain sections such as
`Audience`, `When to Use`, `Intake Checklist`, `Triage`,
`Resolution Procedure`, `Escalation`, `Customer Response Templates`, and
`Source Evidence`.

Run the workflow:

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

In this example, the runner writes:

```text
target/artifacts/support/support_resolution_manual.md
```

fbt records which source files, assets, policy, eval, runner, model, artifact
version, and approval produced that file.

In this flow, fbt is responsible for:

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

## Try It Locally

The quickstart is a small offline fixture. It is useful for checking that the
control plane works before wiring a real runner.

```bash
fbt init knowledge_ops --template support
fbt parse --project-dir knowledge_ops
fbt doctor --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir knowledge_ops --comment "Reviewed locally"
fbt build --project-dir knowledge_ops --select weekly_support_insights
fbt artifact history case_summaries --project-dir knowledge_ops
```

The full command transcript and generated files are in the
[quickstart demo](apps/docs/src/content/docs/get-started/quickstart.mdx).

## Boundaries

fbt core does not implement LLM providers, agent runtimes, OCR, document
conversion, a scheduler, daemon, metadata database, hosted UI, CMS, knowledge
base, ticket system, or document editor. Those capabilities stay outside core;
fbt coordinates them through external runners and records what happened
locally.

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

Start with the [usage guide](docs/usage-guide.md),
[manual generation guide](apps/docs/src/content/docs/get-started/manual-generation.mdx),
and [CLI reference](docs/cli-reference.md). Core contracts are the
[design doc](docs/design-doc.md), [core spec](docs/spec.md),
[project config spec](docs/project-config-spec.md),
[schema/versioning spec](docs/schema-and-versioning-spec.md),
[runner discovery spec](docs/runner-discovery-spec.md),
[runner protocol spec](docs/runner-protocol-spec.md),
[security/conformance spec](docs/security-and-conformance-spec.md), and
[standard export spec](docs/standard-export-spec.md).

The published docs site is
[nyuta01.github.io/fbt](https://nyuta01.github.io/fbt/).
