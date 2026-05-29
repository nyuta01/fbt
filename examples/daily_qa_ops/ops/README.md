# Daily Production Loop

This directory is the production-shaped wrapper around the `daily_qa_ops`
example. It is intentionally outside fbt core: scheduling, source ingestion,
approval, publishing, and notification belong to shell, CI, Git, and your
existing knowledge-base workflow.

## What It Runs

```text
prepared source window
  -> fbt doctor
  -> fbt plan
  -> fbt build
  -> fbt artifact show/explain
  -> fbt artifact retention
  -> fbt export openlineage / otel
  -> external quality gates
  -> CI or publish workflow receives the run bundle
```

Run it locally from this repository:

```sh
examples/daily_qa_ops/ops/run-daily.sh
```

The script writes a run bundle under:

```text
examples/daily_qa_ops/target/ops/runs/<run-id>/
examples/daily_qa_ops/target/ops/latest/
examples/daily_qa_ops/target/ops/archives/<run-id>/
```

The bundle contains source-window validation, human command output, artifact
inspection output, retention report, OpenLineage NDJSON, OTLP/JSON traces, and
quality-gate output.

## Archive Handoff

`ops/archive-fbt-evidence.sh` creates:

```text
target/ops/archives/<run-id>/fbt-evidence.tar.gz
target/ops/archives/<run-id>/archive-manifest.json
```

The archive contains:

```text
.fbt/state/
.fbt/artifacts/
target/ops/runs/<run-id>/
```

Restore those roots together into the same project checkout when you need to
run `fbt artifact explain`, `fbt diff`, `fbt export openlineage`, or
`fbt export otel` against historical evidence. The archive is a handoff to CI
artifacts or external storage; fbt still does not delete historical versions
automatically.

## Approval, Publishing, And Notification

`ops/prepare-publish-handoff.sh` creates files for external workflows:

```text
target/ops/publish/<run-id>/publish-manifest.json
target/ops/publish/<run-id>/pr-body.md
target/ops/publish/<run-id>/notification.md
```

Use those files from GitHub Actions, GitLab CI, Slack automation, or a
knowledge-base publisher. fbt does not merge the PR, approve the content, send
the notification, or publish the manual; it only prepares generated artifacts
and evidence.

## Quality Gates

`ops/check-quality-gates.py` is a production-shaped CI check, not fbt core
logic. It reads generated artifacts plus the run bundle and writes:

```text
quality-gates.txt
quality-gates.json
```

The checked gates are:

| Gate | Meaning |
|---|---|
| `structural_artifacts` | Declared artifacts exist and are non-empty. |
| `evidence_lineage` | Artifact explain and OpenLineage expose the expected upstream evidence. |
| `domain_review` | Publishing still requires an external owner review. |

The first two gates fail the wrapper when they fail. `domain_review` is recorded
as `pending` because human approval belongs in Git, PRs, tickets, or your
knowledge-base workflow.

## Source Readiness

The script requires a JSON readiness manifest:

```text
data/qa/inbox/_READY
```

The manifest declares the current processing window, operation mode, and
minimum source counts. `ops/check-source-window.py` validates it before fbt
runs, so an empty or half-prepared source window fails before any runner is
called.

Keep the fbt source paths stable and let ingestion decide what belongs in the
current processing window:

```text
data/qa/inbox/questions/
data/qa/inbox/answers/
```

For a new-items-only daily job, replace those directories before writing
`_READY`. For a cumulative knowledge base, append files and rewrite `_READY`
after ingestion checks counts, schema, and freshness.

Supported modes are:

| Mode | Meaning |
|---|---|
| `new_items_only` | The inbox contains only the current batch. |
| `cumulative` | The inbox grows over time and fbt rebuilds from the full file set. |
| `correction` | Existing source files were intentionally corrected. |
| `deletion` | Existing source files were intentionally removed. |
| `backfill` | A historical partition was staged into the stable inbox paths. |

## Production Runner Swap

The checked-in project uses deterministic demo runners so the loop works
offline. A production project should replace the runner declarations in
`fs_project.yml` with installed protocol-compatible runners, for example an
OpenAI, Claude Code, Codex, Gemini, or internal adapter.

Keep these as production requirements for the runner:

- fail before execution when the source window is too large
- record provider, model, cost, token usage, and runner version
- redact secrets and raw sensitive source content from protocol events
- write only output candidates under the fbt work directory
- fail closed when policy cannot be enforced

## CI Shape

`github-actions-daily-fbt.yml` is a copyable workflow. It installs fbt, checks
the readiness marker, runs the daily loop, and uploads the run bundle as CI
evidence.

In a real repository, add steps after the fbt loop to open a pull request,
publish selected artifacts, send Slack notifications, or ingest the standard
exports into Marquez, OpenMetadata-compatible tooling, Jaeger, Tempo, or
Grafana.

## What fbt Does Not Do

fbt does not decide that "today's" files are ready. It does not approve the
manual update, publish it, notify owners, or delete old history. The run bundle
is the boundary: fbt produces explainable artifacts and evidence; your
workflow decides what to do with them.
