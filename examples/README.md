# fbt examples

Use these examples in this order.

| Example | Use it for | Runner | Best first command |
|---|---|---|---|
| [`knowledge_ops`](knowledge_ops/) | Verify the local fbt control plane end to end. | Demo runners, no credentials. | `fbt init knowledge_ops --template support` |
| [`incident_response_runbook`](incident_response_runbook/) | See the most direct practical workflow: incident evidence to an approved runbook. | OpenAI runner, `OPENAI_API_KEY` required for `build`. | `fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook` |
| [`support_resolution_manual`](support_resolution_manual/) | See a support-ops workflow: tickets and response notes to an approved support manual. | OpenAI runner, `OPENAI_API_KEY` required for `build`. | `fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual` |

## Which Example Should I Start With?

Start with `knowledge_ops` when you want to prove fbt works on your machine. It
uses deterministic demo runners, so it can build, review, generate docs, and
export lineage without calling an external provider. Its generated text is a
fixture, not a useful business document.

Start with `incident_response_runbook` when you want to understand the real
product value. It turns incident event logs, response notes, and a postmortem
into a runbook plus a receipt that records sources, runner, checks, version,
review, and lineage.

Use `support_resolution_manual` after that if your workflow is closer to
support operations. It is realistic, but it has more moving parts: tickets,
response notes, product docs, and approved macros.

## What All Examples Demonstrate

Every example follows the same model:

```text
sources + instructions + runner -> artifact + build receipt
```

- `sources`: files under `data/`
- `instructions`: prompts, format files, policies, and evals under `assets/`,
  `policies/`, and `evals/`
- `runner`: a demo runner or an external provider-compatible runner
- `artifact`: generated Markdown under `target/artifacts/`
- `build receipt`: local records under `.fbt/state/` and `.fbt/artifacts/`

## Daily Source Growth

The practical examples are shaped for repeated operation. You can keep adding
new files under the declared source paths:

```text
data/incidents/events/*.jsonl
data/incidents/response_logs/
data/incidents/postmortems/
data/support/tickets/*.jsonl
data/support/response_logs/
```

On the next `fbt plan`, fbt fingerprints the resolved source file set and file
contents. New or changed source files make dependent transforms dirty, so the
next `fbt build` creates a new artifact version and leaves the older version in
`.fbt/artifacts/`.

The granularity is the declared source artifact and transform. fbt does not run
a daemon, watch directories, schedule jobs, or automatically partition every
new file into its own transform. For dozens or hundreds of daily files, use an
external scheduler and partition your project by date, service, customer, or
case type when you need smaller review units.

## Common Commands

```sh
fbt plan --project-dir <example> --select <transform>
```

Preview whether fbt will run, skip, or block before a runner is called.

```sh
fbt build --project-dir <example> --select <transform>
```

Generate the artifact and store a versioned receipt.

```sh
fbt review show <artifact> --project-dir <example>
fbt review approve <artifact> --project-dir <example> --comment "Reviewed"
```

Inspect and approve the exact artifact version, not just the path.

```sh
fbt artifact history <artifact> --project-dir <example>
```

Explain which version is current and which runner produced it.
