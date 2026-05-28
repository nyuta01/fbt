# Incident Response Runbook Example

This is the clearest practical example in the repository.

It turns incident evidence into a runbook:

```text
incident event logs + response notes + postmortem + existing runbooks
  -> target/artifacts/runbooks/incident_response_runbook.md
  -> .fbt receipt with sources, runner, checks, version, and lineage
```

Use this example when useful knowledge is scattered across logs, response
notes, and the postmortem, and you need one runbook plus a receipt explaining
where it came from.

## Run The Workflow

Preview the work before spending runner time:

```sh
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
```

After setting credentials, build the runbook:

```sh
export OPENAI_API_KEY=...
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
```

You get:

```text
target/artifacts/runbooks/incident_response_runbook.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
.fbt/state/run_results.jsonl
```

Inspect where it came from:

```sh
fbt artifact show incident_response_runbook --project-dir examples/incident_response_runbook
fbt artifact history incident_response_runbook --project-dir examples/incident_response_runbook
fbt artifact explain incident_response_runbook --project-dir examples/incident_response_runbook
```

Export standard lineage:

```sh
fbt export openlineage \
  --project-dir examples/incident_response_runbook \
  --output examples/incident_response_runbook/target/lineage/openlineage.ndjson
```

Human approval and publishing should happen in Git, PR, CI, release, or catalog
workflows outside fbt.
