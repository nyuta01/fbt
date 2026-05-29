# fbt examples

Use these examples in this order.

| Example | Use it for | Runner | Best first command |
|---|---|---|---|
| [`knowledge_ops`](knowledge_ops/) | Verify the local fbt control plane end to end. | Demo runners, no credentials. | `fbt init knowledge_ops --template support` |
| first own-files path | Replace sample support data with your files and build one receipt-backed artifact. | Demo first, external runner later. | `docs/examples/first-own-files-success-path.md` |
| [`daily_qa_ops`](daily_qa_ops/) | See a daily batch workflow with Markdown sources and multiple outputs. | Demo runners, no credentials. | `fbt plan --project-dir examples/daily_qa_ops --select tag:daily_qa` |
| [`markdown_toolchain`](markdown_toolchain/) | See fbt wrap remark/Pandoc-style CLI tools without owning document processing. | Command runner, no credentials. | `fbt plan --project-dir examples/markdown_toolchain --select tag:document_toolchain` |
| [`data_tool_interop`](data_tool_interop/) | See fbt consume dbt/DataChain output files and turn them into a versioned human brief. | Command runner, no credentials. | `fbt plan --project-dir examples/data_tool_interop --select data_tool_brief` |
| [`runner_adapters`](runner_adapters/) | Inspect source-checkout runner adapter examples used by demos and practical flows. | Protocol adapters, optional credentials. | `python3 tests/runner-conformance/run.py --runner-command 'go run ./examples/runner_adapters/demo_llm' --strict` |
| [`runner_adapter_scaffold`](runner_adapter_scaffold/) | Copy the smallest useful external runner adapter and test it with conformance. | Python stdlib, no credentials. | `python3 tests/runner-conformance/run.py --runner-command examples/runner_adapter_scaffold/bin/fbt-runner-example --strict --agent-adapter` |
| [`semantic_eval_boundary`](semantic_eval_boundary/) | See how to keep semantic and evidence-quality checks outside fbt core. | Command runner, no credentials. | `fbt plan --project-dir examples/semantic_eval_boundary --select tag:quality_boundary` |
| [`standard_visualization`](standard_visualization/) | Send fbt OpenLineage and OTel exports to standard visualization backends. | External Marquez/Jaeger/Tempo/Grafana, optional. | `make standard-backend-smoke` |
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

Use stable source paths for the fbt project and keep windowing outside fbt:

| Operation | Keep stable in fbt | Change outside fbt |
|---|---|---|
| New-items-only daily batch | `data/qa/inbox/questions/` and `data/qa/inbox/answers/` | Replace those directories with today's prepared files before running fbt. |
| Cumulative evidence base | The same source directories | Append new files and let fbt detect the changed file set. |
| Date/service/customer partitions | The currently selected source directory | Use cron, CI, Airflow, Dagster, or another orchestrator to prepare that directory. |

The detailed daily operations guide is
[`docs/examples/daily-source-operations.md`](../docs/examples/daily-source-operations.md).

## Copying Examples

The checked-in examples are easiest to run from this repository checkout. Some
demo wrappers use `go run` against repository-local adapter code so the examples
stay small and do not vendor provider or runner binaries.

If you copy an example to a temporary directory, declare and set
`FBT_SOURCE_ROOT` for the copied runner config:

```yaml
runners:
  - name: demo.llm
    command: bin/fbt-demo-llm-runner
    env:
      - FBT_SOURCE_ROOT
```

Then run:

```sh
FBT_SOURCE_ROOT=/path/to/fbt fbt doctor --project-dir /tmp/daily_qa_ops
FBT_SOURCE_ROOT=/path/to/fbt fbt build --project-dir /tmp/daily_qa_ops --select tag:daily_qa
```

Without that declared environment variable, a copied demo wrapper can fail
before protocol initialization because it cannot find the repository `go.mod`.
Real installed adapters such as `fbt-runner-openai` do not need
`FBT_SOURCE_ROOT`.

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
