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
| `fbt plan` | Show run/skip/block decisions before a runner is called. |
| `fbt build` | Invoke external runners, run checks, commit artifacts, and write state. |
| `fbt diff --against previous` | Compare generated versions. |
| `fbt artifact show` | Inspect path, digest, runner, model, confidence, and descriptors. |
| `fbt artifact history` | List versions for a logical artifact. |
| `fbt artifact explain` | Explain why an artifact will run, skip, or block. |
| `fbt export openlineage` | Export artifact lineage as OpenLineage NDJSON. |
| `fbt export otel` | Export local execution traces as OTLP/JSON. |

Advanced commands exist for tooling and debugging: `fbt parse`, `fbt eval`,
`fbt docs generate`, `fbt state`, and `fbt runner`. The daily path should not
need them.

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

## 6. Review And Publishing Boundary

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
