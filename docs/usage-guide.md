# fbt Usage Guide

Status: MVP-ready  
Created: 2026-05-28  
Audience: users running local `fbt` projects

## 1. Mental Model

```text
sources + instructions + runner -> artifact + build receipt
```

fbt owns the local build receipt: manifest, run results, artifact versions,
eval results, policy decisions, and standard exports. It does not own human
review, approval, publishing, or scheduling workflows.

Keep adjacent tools in their lane: use dbt or DataChain for data transforms,
DVC or artifact stores for data and blob versioning, Snakemake for workflow
orchestration, remark or Pandoc for document processing, and fbt for the
generated file artifact receipt that connects inputs, runner, version, checks,
and lineage.

`build` is the execution command because fbt treats generated files as build
outputs. The command does more than call a runner: it produces selected
artifacts, records immutable versions, runs checks, and writes the receipt that
later inspection and exports use.

## 2. Offline Quickstart

```sh
fbt init knowledge_ops --template support
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select tag:support
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact history case_summaries --project-dir knowledge_ops
```

Expected first plan shape:

```text
Plan
  selected  2
  run       2
  skipped   0
  blocked   0

RUN     case_summaries
        because  no previous successful run
        because  output missing
        output   case_summaries
        next     fbt build --select case_summaries --project-dir knowledge_ops

RUN     weekly_support_insights
        because  no previous successful run
        because  output missing
        because  upstream artifact selected to run
        output   weekly_support_insights
        next     fbt build --select weekly_support_insights --project-dir knowledge_ops
```

`build` executes the selected graph in dependency order. The upstream
`case_summaries` artifact is committed before `weekly_support_insights` runs.
You can still build one transform at a time:

```sh
fbt build --project-dir knowledge_ops --select weekly_support_insights
```

## 3. What Each Command Gives You

| Command | Purpose |
|---|---|
| `fbt doctor` | Check project config, local state, and runner readiness. |
| `fbt plan` | Show run/skip/block decisions before a runner is called; does not write state. |
| `fbt build` | Produce selected artifacts, run checks, commit versions, and write receipts. |
| `fbt plan --failed` / `fbt build --failed` | Inspect and rerun only transforms whose latest receipt failed. |
| `fbt diff --against previous` | Compare generated versions. |
| `fbt artifact show` | Inspect path, digest, runner, model, confidence, and semantic summary. |
| `fbt artifact history` | List versions for a logical artifact. |
| `fbt artifact explain` | Explain why an artifact will run, skip, or block. |
| `fbt artifact retention` | Report `.fbt/state` and `.fbt/artifacts` growth without deleting files. |
| `fbt export openlineage` | Write OpenLineage RunEvent NDJSON from local artifact lineage. |
| `fbt export otel` | Write OTLP/JSON traces from local run receipts. |

The public CLI is intentionally small. `doctor` owns readiness diagnostics,
`plan` previews without writes, and `build` owns runner execution, evals, state
writes, and artifact receipts.

CLI safety rule: unknown flags, extra arguments, and selectors that match no
transforms fail instead of being ignored.

## 4. Real Runner Use

Templates use deterministic demo runners. Replace project runner entries when
you want real output:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: bin/fbt-runner-openai
    args: ["responses"]
    env:
      - OPENAI_API_KEY
```

The runner owns provider SDKs, credentials, prompts sent to the model, and
model-specific behavior. fbt core owns state, policy checks, artifact version
commit, and lineage.

## 5. Daily Operation

For repeated workflows, keep source paths stable and let the upstream ingestion
step decide what files are in the current window:

```text
data/qa/inbox/questions/
data/qa/inbox/answers/
```

Then run:

```sh
fbt plan --select tag:daily_qa
fbt build --select daily_qa_candidates
fbt build --select promote_manual_update
```

fbt fingerprints the resolved source file set and content. New or changed files
make dependent transforms dirty. fbt intentionally does not include a daemon,
scheduler, watermark store, or built-in per-file partition engine.
After the first successful build records a manifest, `fbt plan` and
`fbt artifact explain` show which source files were added, changed, or removed,
so daily batches can be inspected before starting a runner.

Use one of these source-window patterns:

| Pattern | How it works | When to use it |
|---|---|---|
| Latest window | Replace the files under a stable inbox before each run. | Process only today's or this hour's new records. |
| Cumulative window | Keep appending files under the same source directory. | Rebuild an artifact from all known evidence. |
| External partition | Put dates, service names, or customer IDs in the upstream ingestion path, then point fbt at the prepared current directory. | Need retention, watermarks, or parallel scheduling without putting that logic in fbt. |

Before running fbt, let the external ingestion step perform readiness checks
such as "all expected files arrived", "batch marker exists", or "source export
completed". fbt can tell that a file set changed; it does not decide when the
business batch is complete.

Use `--force` only when you intentionally want to regenerate selected clean
artifacts:

```sh
fbt plan --select daily_qa_candidates --force
fbt build --select daily_qa_candidates --force
```

Force is not a cache engine and does not bypass upstream, confidence, policy,
or output-boundary checks.

## 6. Local Retention

fbt keeps all artifact versions and run receipts by default. This is deliberate:
lineage, diffs, and export records depend on immutable history.

Inspect local growth with:

```sh
fbt artifact retention
```

The command is read-only. It reports state bytes, immutable artifact bytes, run
records, current versions, historical versions, protected current-pointer
versions, archive unit, future prune safety flags, and missing storage
references.

For high-volume projects, archive these directories together:

```text
.fbt/state/
.fbt/artifacts/
```

Use external tools such as `tar`, `rsync`, backup jobs, object storage, or CI
artifacts for retention windows. fbt does not expose a destructive prune command
in MVP because deleting history must preserve current pointers, lineage, eval
results, policy decisions, and run receipts.

`fbt artifact retention --json` reports `archive_unit:
state_and_artifacts`, `protected_version_ids`, `prune_supported: false`, and
`dry_run_required: true`. Treat those fields as the safety contract for
external archive jobs.

The high-volume fixture is:

```sh
make retention-high-volume-smoke
```

It creates eight artifact versions, checks the read-only report, and verifies
that `.fbt/state/` and `.fbt/artifacts/` are the archive roots.

## 7. Existing Tool Composition

Use `type: command` transforms when an existing CLI already does the work.
Examples include remark for Markdown normalization and Pandoc for document
conversion. fbt's role is to pass the declared argv to a command runner, then
commit the resulting files as versioned artifacts with receipts and lineage.

```sh
fbt plan --project-dir examples/markdown_toolchain --select tag:document_toolchain
fbt build --project-dir examples/markdown_toolchain --select remark_markdown
fbt build --project-dir examples/markdown_toolchain --select pandoc_handbook
```

Use the same pattern when the existing tool writes files that fbt should treat
as sources. The dbt/DataChain interoperability example consumes dbt run
artifacts and DataChain job outputs, then builds a Markdown brief:

```sh
fbt plan --project-dir examples/data_tool_interop --select data_tool_brief
fbt build --project-dir examples/data_tool_interop --select data_tool_brief
fbt artifact explain data_tool_brief --project-dir examples/data_tool_interop
```

dbt still owns warehouse transformation. DataChain still owns dataset
materialization. fbt owns the generated file artifact receipt.

## 8. Semantic Eval Boundary

fbt core runs deterministic evals such as required sections or required text.
It does not call model judges.

For semantic judgement, create a normal runner-backed transform that produces a
judge report artifact:

```sh
fbt build --project-dir examples/semantic_eval_boundary --select tag:quality_boundary
fbt artifact show evidence_quality_report --project-dir examples/semantic_eval_boundary
fbt artifact explain evidence_quality_report --project-dir examples/semantic_eval_boundary
```

The runner owns the model call. fbt records the judge report's sources, runner,
artifact version, policy decision, and lineage. Git, CI, PRs, or publishing
tooling decide whether the report blocks release.

`semantic` and `llm_judge` eval types are reserved config shapes in the MVP.
Build records them as skipped, prints the skip reason and external-judge hint,
shows the skipped eval in `fbt artifact explain`, and grants no confidence from
them.

## 9. Review And Publishing Boundary

fbt deliberately does not implement `review`, `approve`, or `reject` commands.

Use fbt to produce reviewable material:

```sh
fbt artifact show manual_update
fbt artifact history manual_update
fbt diff manual_update --against previous
fbt export openlineage --output target/lineage/openlineage.ndjson
```

Then use Git, pull requests, CI, release tooling, OpenMetadata, or your
organization's approval system to decide whether to publish or trust the file.
