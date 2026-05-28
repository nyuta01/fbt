# knowledge_ops

This is the offline example. Use it to verify that fbt's local control plane
works before wiring a real provider or agent.

It demonstrates:

- source files becoming versioned artifacts
- downstream work waiting for a current upstream artifact
- local build receipts for generated files
- OpenLineage and OTel export files

It does not demonstrate model quality. The runners are deterministic demo
runners, and the generated text is intentionally small.

## Run It

Prefer creating a fresh copy through the template so your run starts clean:

```sh
fbt init knowledge_ops --template support
cd knowledge_ops
```

Preview the workflow:

```sh
fbt plan --select tag:support
```

Expected shape:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked
run transform.knowledge_ops.case_summaries
blocked transform.knowledge_ops.weekly_support_insights
  blocked: requires artifact.knowledge_ops.case_summaries current artifact
  next: fbt build --select case_summaries
```

Build and inspect the first artifact:

```sh
fbt build --select case_summaries
fbt artifact show case_summaries
fbt artifact explain case_summaries
```

You get:

```text
target/artifacts/support/case_summaries/index.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
```

Now the downstream transform can run:

```sh
fbt build --select weekly_support_insights
```

Inspect and export what happened:

```sh
fbt artifact history case_summaries
fbt export openlineage --output target/lineage/openlineage.ndjson
fbt export otel --output target/telemetry/otel.json
```

## What To Look At

- `transforms/support/case_summaries.yml`: a first LLM-style transform from
  source tickets to an artifact.
- `transforms/support/weekly_insights.yml`: a downstream agent-style transform
  that requires the structural `case_summaries` artifact.
- `.fbt/state/`: the local build receipt.
- `target/artifacts/`: the files users would consume.
