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
fbt build --project-dir knowledge_ops --select case_summaries
fbt artifact show case_summaries --project-dir knowledge_ops
fbt artifact history case_summaries --project-dir knowledge_ops
```

Expected first plan shape:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked
run transform.knowledge_ops.case_summaries
blocked transform.knowledge_ops.weekly_support_insights
  blocked: requires artifact.knowledge_ops.case_summaries current artifact
  next: fbt build --select case_summaries
```

After `case_summaries` exists, the downstream transform can run:

```sh
fbt build --project-dir knowledge_ops --select weekly_support_insights
```

## 3. What Each Command Gives You

| Command | Purpose |
|---|---|
| `fbt doctor` | Check project config, local state, and runner readiness. |
| `fbt plan` | Show run/skip/block decisions before a runner is called; does not write state. |
| `fbt build` | Produce selected artifacts, run checks, commit versions, and write receipts. |
| `fbt diff --against previous` | Compare generated versions. |
| `fbt artifact show` | Inspect path, digest, runner, model, confidence, and descriptors. |
| `fbt artifact history` | List versions for a logical artifact. |
| `fbt artifact explain` | Explain why an artifact will run, skip, or block. |
| `fbt export openlineage` | Export artifact lineage as OpenLineage NDJSON. |
| `fbt export otel` | Export local execution traces as OTLP/JSON. |

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

## 6. Existing Tool Composition

Use `type: command` transforms when an existing CLI already does the work.
Examples include remark for Markdown normalization and Pandoc for document
conversion. fbt's role is to pass the declared argv to a command runner, then
commit the resulting files as versioned artifacts with receipts and lineage.

```sh
fbt plan --project-dir examples/markdown_toolchain --select tag:document_toolchain
fbt build --project-dir examples/markdown_toolchain --select remark_markdown
fbt build --project-dir examples/markdown_toolchain --select pandoc_handbook
```

## 7. Review And Publishing Boundary

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
