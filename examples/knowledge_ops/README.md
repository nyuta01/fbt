# knowledge_ops

Runnable local knowledge-loop example for `fbt`.

```sh
fbt parse --project-dir examples/knowledge_ops
fbt build --project-dir examples/knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir examples/knowledge_ops --comment "Reviewed locally"
fbt build --project-dir examples/knowledge_ops --select weekly_support_insights
```

The project uses bundled local protocol runners and does not call external
model providers.
