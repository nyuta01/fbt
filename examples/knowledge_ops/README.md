# knowledge_ops

This is the offline example. Use it to verify that fbt's local control plane
works before wiring a real provider or agent.

It demonstrates:

- source files becoming versioned artifacts
- review approval attached to an exact artifact version
- downstream work waiting for an approved input
- generated project docs
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

Build and approve the first artifact:

```sh
fbt build --select case_summaries
fbt review show case_summaries
fbt review approve case_summaries --comment "Reviewed locally"
```

You get:

```text
target/artifacts/support/case_summaries/index.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
.fbt/state/approvals.json
```

Now the downstream transform can run:

```sh
fbt build --select weekly_support_insights
```

You get:

```text
target/artifacts/support/weekly_insights.md
```

Inspect and export what happened:

```sh
fbt artifact history case_summaries
fbt docs generate
fbt export openlineage --output target/lineage/openlineage.ndjson
fbt export otel --output target/telemetry/otel.json
```

You get:

```text
target/docs/index.md
target/lineage/openlineage.ndjson
target/telemetry/otel.json
```

## What To Look At

- `transforms/support/case_summaries.yml`: a first LLM-style transform from
  source tickets to a reviewed artifact.
- `transforms/support/weekly_insights.yml`: a downstream agent-style transform
  that requires the reviewed `case_summaries` artifact.
- `.fbt/state/`: the local build receipt.
- `target/artifacts/`: the files users would consume.

## Replacing The Demo Runner

The project uses `demo.llm` and `demo.agent` runners:

```yaml
runners:
  - name: demo.llm
  - name: demo.agent
```

Replace those runner entries in `fs_project.yml` when you are ready to call a
real provider, Claude Code, Codex, Gemini, or an internal runner that speaks the
fbt runner protocol.
