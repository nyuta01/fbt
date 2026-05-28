# Incident Response Runbook Example

This is the clearest practical example in the repository.

It turns incident evidence into an approved runbook:

```text
incident event logs + response notes + postmortem + existing runbooks
  -> target/artifacts/runbooks/incident_response_runbook.md
  -> .fbt receipt with sources, runner, checks, version, review, and lineage
```

Use this example when your real workflow looks like:

> We already handled an incident. The useful knowledge is scattered across
> logs, Slack-style response notes, and the postmortem. We want one runbook the
> next on-call engineer can use, and we need to know where that runbook came
> from.

## Input Evidence

The example incident is `INC-2026-0421`, a checkout latency incident caused by
database connection pool saturation.

| File | Role |
|---|---|
| `data/incidents/events/INC-2026-0421.jsonl` | Timeline facts from pager, metrics, and incident commands. |
| `data/incidents/response_logs/INC-2026-0421-response.md` | What responders tried, what worked, and what was missing. |
| `data/incidents/postmortems/INC-2026-0421-postmortem.md` | Root cause and corrective actions. |
| `data/reference/runbooks/checkout-existing-runbook.md` | Existing runbook context the new output should not contradict. |

The key source facts are intentionally concrete:

```text
p95_latency_ms above 2400 for 10 minutes
database connection pool saturation
traffic shifted away from us-east-1 read replica
timeout rate decreased from 7.8% to 1.1%
```

## Desired Output

The generated artifact is:

```text
target/artifacts/runbooks/incident_response_runbook.md
```

The required format is declared in `assets/incident_runbook_format.md`. It
forces sections such as `Detection`, `Immediate Response`, `Mitigation`,
`Customer Communication`, and `Source Evidence`.

That output is useful only after review. fbt records both the generated version
and the approval so later users can answer:

- which evidence produced this runbook
- which runner and model generated it
- which checks passed
- who approved this exact artifact version

## Repeated Operation

For daily operation, keep dropping new incident files under the declared source
paths. For example:

```text
data/incidents/events/INC-2026-0422.jsonl
data/incidents/response_logs/INC-2026-0422-response.md
data/incidents/postmortems/INC-2026-0422-postmortem.md
```

The next `fbt plan` sees the changed source file set and marks
`incident_response_runbook` dirty. The next `fbt build` creates a new runbook
artifact version; approval for the old version does not automatically approve
the new one.

If you process many incidents per day, split the project into smaller sources
or transforms by service, severity, or date window. fbt tracks versions and
dependencies, but it does not schedule runs or automatically create one
transform per incoming file.

## Runner Requirement

This example uses the OpenAI Responses runner boundary:

```yaml
runners:
  - name: openai.responses
    command: bin/fbt-runner-openai
    args: ["responses"]
    env: ["OPENAI_API_KEY"]
```

`plan` works without credentials. `doctor` and `build` require
`OPENAI_API_KEY`.

## Run The Workflow

Preview the work before spending runner time:

```sh
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
```

Expected first-run shape:

```text
Plan: 1 selected, 1 run, 0 skipped, 0 blocked
run transform.incident_response_runbook.incident_response_runbook
  reason: no previous successful run
  reason: output missing
```

Check runner readiness:

```sh
fbt doctor --project-dir examples/incident_response_runbook
```

Without credentials, the useful failure is:

```text
Doctor: error
error RUNNER_ENV_MISSING: runner env OPENAI_API_KEY is not set
ok RUNNER_COMMAND_OK: runner command is executable: ...
```

After setting credentials, build the runbook:

```sh
export OPENAI_API_KEY=...
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
```

You get the runbook and local receipt:

```text
target/artifacts/runbooks/incident_response_runbook.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
.fbt/state/run_results.jsonl
```

Inspect and approve the exact generated version:

```sh
fbt review show incident_response_runbook --project-dir examples/incident_response_runbook
fbt review approve incident_response_runbook \
  --project-dir examples/incident_response_runbook \
  --comment "SRE lead approved"
```

Explain where it came from:

```sh
fbt artifact history incident_response_runbook --project-dir examples/incident_response_runbook
```

Generate local docs and standard lineage:

```sh
fbt docs generate --project-dir examples/incident_response_runbook
fbt export openlineage \
  --project-dir examples/incident_response_runbook \
  --output examples/incident_response_runbook/target/lineage/openlineage.ndjson
```

You get:

```text
target/docs/index.md
target/lineage/openlineage.ndjson
```
