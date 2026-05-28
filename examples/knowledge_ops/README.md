# knowledge_ops

Runnable local knowledge-loop example for `fbt`.

```sh
fbt parse --project-dir examples/knowledge_ops
fbt build --project-dir examples/knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir examples/knowledge_ops --comment "Reviewed locally"
fbt build --project-dir examples/knowledge_ops --select weekly_support_insights
fbt export openlineage --project-dir examples/knowledge_ops --output examples/knowledge_ops/target/lineage/openlineage.ndjson
fbt export otel --project-dir examples/knowledge_ops --output examples/knowledge_ops/target/telemetry/otel.json
```

The project uses deterministic demo protocol runners (`demo.llm` and
`demo.agent`) and does not call external model providers. Replace the runner
entries in `fs_project.yml` with external commands before using real provider
or agent execution.
