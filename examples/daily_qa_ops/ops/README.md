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
```

The bundle contains the human command output, artifact inspection output,
retention report, OpenLineage NDJSON, and OTLP/JSON traces.

## Source Readiness

The script requires:

```text
data/qa/inbox/_READY
```

Keep the fbt source paths stable and let ingestion decide what belongs in the
current processing window:

```text
data/qa/inbox/questions/
data/qa/inbox/answers/
```

For a new-items-only daily job, replace those directories before writing
`_READY`. For a cumulative knowledge base, append files and rewrite `_READY`
after ingestion checks counts, schema, and freshness.

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
