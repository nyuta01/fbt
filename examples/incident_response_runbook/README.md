# Incident Response Runbook Example

This example is a production-shaped fbt project for turning incident logs,
response notes, and postmortems into an approved incident response runbook.

It does not use the bundled demo runners. It expects an external fbt-compatible
LLM runner such as `fbt-runner-openai` to be installed and configured.

## Workflow

1. Drop new incident event JSONL files into `data/incidents/events/`.
2. Drop response-room notes into `data/incidents/response_logs/`.
3. Drop postmortems into `data/incidents/postmortems/`.
4. Keep existing approved runbooks under `data/reference/runbooks/`.
5. Run:

```sh
fbt parse --project-dir examples/incident_response_runbook
fbt doctor --project-dir examples/incident_response_runbook
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt review show incident_response_runbook --project-dir examples/incident_response_runbook
fbt review approve incident_response_runbook --project-dir examples/incident_response_runbook --comment "SRE lead approved"
fbt docs generate --project-dir examples/incident_response_runbook
fbt export openlineage --project-dir examples/incident_response_runbook --output examples/incident_response_runbook/target/lineage/openlineage.ndjson
```

`fbt doctor` and `fbt build` require the external runner and credentials. The
example is intentionally configured for a real runner boundary:

```yaml
runners:
  - name: openai.responses
    command: fbt-runner-openai
    args: ["responses"]
    env: ["OPENAI_API_KEY"]
```

## Output

The generated official artifact is:

```text
target/artifacts/runbooks/incident_response_runbook.md
```

The required format is defined in `assets/incident_runbook_format.md`. The
deterministic eval in `evals/incident.yml` checks that required sections are
present before approval.
