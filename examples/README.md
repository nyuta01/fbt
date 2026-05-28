# fbt examples

Use these examples in this order.

| Example | Use it for | Runner | Best first command |
|---|---|---|---|
| [`knowledge_ops`](knowledge_ops/) | Verify the local fbt control plane end to end. | Demo runners, no credentials. | `fbt init knowledge_ops --template support` |
| [`daily_qa_ops`](daily_qa_ops/) | See a daily batch workflow with Markdown sources and multiple outputs. | Demo runners, no credentials. | `fbt plan --project-dir examples/daily_qa_ops --select tag:daily_qa` |
| [`incident_response_runbook`](incident_response_runbook/) | See the most direct practical workflow: incident evidence to a runbook. | OpenAI runner, `OPENAI_API_KEY` required for `build`. | `fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook` |
| [`support_resolution_manual`](support_resolution_manual/) | See a support-ops workflow: tickets and response notes to a support manual. | OpenAI runner, `OPENAI_API_KEY` required for `build`. | `fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual` |

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

fbt does not own human review or approval. Inspect generated files with
`artifact show`, `artifact history`, `artifact explain`, and `diff`, then use
Git, PRs, CI, release tooling, or your catalog workflow to decide whether to
publish or trust the artifact.

## Daily Source Growth

The practical examples are shaped for repeated operation. You can keep adding
new files under the declared source paths:

```text
data/incidents/events/*.jsonl
data/incidents/response_logs/
data/incidents/postmortems/
data/support/tickets/*.jsonl
data/support/response_logs/
data/qa/inbox/questions/
data/qa/inbox/answers/
```

On the next `fbt plan`, fbt fingerprints the resolved source file set and file
contents. New or changed source files make dependent transforms dirty, so the
next `fbt build` creates a new artifact version and leaves the older version in
`.fbt/artifacts/`.

## Common Commands

```sh
fbt plan --project-dir <example> --select <transform>
```

Preview whether fbt will run, skip, or block before a runner is called.

```sh
fbt build --project-dir <example> --select <transform>
```

Build the artifact and store a versioned receipt.

```sh
fbt artifact show <artifact> --project-dir <example>
fbt artifact history <artifact> --project-dir <example>
fbt artifact explain <artifact> --project-dir <example>
```

Inspect the exact artifact version, path, digest, runner, and lineage context.
