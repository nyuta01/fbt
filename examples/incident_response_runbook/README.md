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

## What This Example Uses

This project is intentionally small enough to inspect by hand. The important
files are:

| Role | File | Concrete evidence or instruction |
|---|---|---|
| Source | `data/incidents/events/INC-2026-0421.jsonl` | Pager alert, metrics evidence, mitigation event, and resolution event. |
| Source | `data/incidents/response_logs/INC-2026-0421-response.md` | Responder timeline and support guidance. |
| Source | `data/incidents/postmortems/INC-2026-0421-postmortem.md` | Root cause and corrective actions. |
| Instruction | `assets/incident_runbook_prompt.md` | Generate an official runbook from evidence only. |
| Format | `assets/incident_runbook_format.md` | Required sections such as Detection, Mitigation, Recovery, and Source Evidence. |
| Runner | `openai.responses` in `fs_project.yml` | Calls the external OpenAI runner through fbt's runner protocol. |
| Artifact | `target/artifacts/runbooks/incident_response_runbook.md` | The generated runbook. |

Representative input:

```json
{"incident_id":"INC-2026-0421","source":"pager","severity":"SEV2","service":"checkout-api","event":"p95_latency_ms above 2400 for 10 minutes","customer_impact":"elevated checkout timeouts for a subset of customers"}
{"incident_id":"INC-2026-0421","source":"incident_command","event":"traffic shifted away from us-east-1 read replica","result":"checkout timeout rate decreased from 7.8% to 1.1%"}
```

The response log adds the human handling detail: responders confirmed database
connection pool saturation, shifted traffic away from the affected read
replica, and told support to verify payment status before asking customers to
retry. The postmortem adds the root cause: extra read load from a maintenance
task.

The generated artifact should turn that evidence into a reusable procedure:

```md
## Detection
- Confirm checkout latency, timeout rate, and database connection pool
  saturation before starting mitigation.

## Mitigation
- Shift read traffic away from the affected read replica when the evidence
  matches this incident pattern.

## Customer Communication
- Acknowledge elevated checkout latency.
- Verify payment status before recommending a retry.

## Source Evidence
- INC-2026-0421 event log
- INC-2026-0421 response log
- INC-2026-0421 postmortem
```

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

The explain output is the receipt. It shows the producer transform, current
artifact version, source fingerprints, prompt/format asset fingerprints, eval,
policy, runner, and final output path:

```text
Artifact: incident_response_runbook
Producer
  Transform        incident_response_runbook

Inputs
  ok input   incident.event_logs          path=data/incidents/events/*.jsonl
  ok input   incident.response_logs       path=data/incidents/response_logs
  ok input   incident.postmortems         path=data/incidents/postmortems
  ok asset   incident_runbook_format      path=assets/incident_runbook_format.md
  ok asset   incident_runbook_prompt      path=assets/incident_runbook_prompt.md
  ok runner  openai.responses

Outputs
  incident_response_runbook  target/artifacts/runbooks/incident_response_runbook.md
```

Export standard lineage:

```sh
fbt export openlineage \
  --project-dir examples/incident_response_runbook \
  --output examples/incident_response_runbook/target/lineage/openlineage.ndjson
```

Human approval and publishing should happen in Git, PR, CI, release, or catalog
workflows outside fbt.
