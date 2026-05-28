# Practical Manual Generation Examples

Status: MVP-ready with external runners  
Created: 2026-05-28  
Audience: teams applying fbt to operational manuals and runbooks

## 1. Purpose

These examples show fbt projects shaped for real business workflows, not demo
runner output. They use external runner configuration and require a compatible
runner such as `fbt-runner-openai` before `fbt doctor` or `fbt build` can pass.

The examples are:

| Example | Inputs | Generated Artifact |
|---|---|---|
| `examples/incident_response_runbook` | incident event logs, response notes, postmortems, existing runbooks | approved incident response runbook |
| `examples/support_resolution_manual` | user inquiry tickets, support response logs, product docs, approved macros | approved support resolution manual |

Both examples use the same production loop:

```text
primary records
  -> fbt parse / plan
  -> external LLM runner
  -> deterministic section eval
  -> human review
  -> approved official manual artifact
  -> docs and standard lineage exports
```

## 2. External Runner Boundary

The examples intentionally do not use `demo.llm` or `demo.agent`. Their
`fs_project.yml` files reference:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-runner-openai
    args: ["responses"]
    env:
      - OPENAI_API_KEY
```

Install and validate the runner before building:

```sh
FBT_RUNNER_CONFORMANCE_COMMAND='fbt-runner-openai responses' make runner-conformance
OPENAI_API_KEY=... fbt doctor --project-dir examples/incident_response_runbook
```

The runner package owns provider SDKs and credentials. fbt core owns state,
policy/eval/review checks, lineage, and official artifact commits.

## 3. Incident Response Runbook

Use this when the source of truth is incident evidence:

```text
data/incidents/events/*.jsonl
data/incidents/response_logs/
data/incidents/postmortems/
data/reference/runbooks/
```

Build flow:

```sh
fbt parse --project-dir examples/incident_response_runbook
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt review show incident_response_runbook --project-dir examples/incident_response_runbook
fbt review approve incident_response_runbook --project-dir examples/incident_response_runbook --comment "SRE lead approved"
```

The output contract lives in
`examples/incident_response_runbook/assets/incident_runbook_format.md`. The
required sections include detection, immediate response, mitigation, recovery,
customer communication, escalation, follow-up, maintenance notes, and source
evidence.

## 4. Support Resolution Manual

Use this when the source of truth is customer-support handling evidence:

```text
data/support/tickets/*.jsonl
data/support/response_logs/
data/reference/product_docs/
data/reference/macros/
```

Build flow:

```sh
fbt parse --project-dir examples/support_resolution_manual
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt review show support_resolution_manual --project-dir examples/support_resolution_manual
fbt review approve support_resolution_manual --project-dir examples/support_resolution_manual --comment "Support lead approved"
```

The output contract lives in
`examples/support_resolution_manual/assets/support_resolution_manual_format.md`.
The required sections include intake checklist, triage, resolution procedure,
escalation, customer response templates, agent notes, maintenance notes, and
source evidence.

## 5. Operating Notes

- Add new source records; do not edit generated official artifacts by hand.
- Update prompt, format, style guide, and evidence checklist assets when the
  manual contract changes.
- Use `fbt plan` before building to see whether source, policy, runner, model,
  or asset changes make the manual dirty.
- Keep review required for official procedures.
- Use `fbt artifact history`, `fbt docs generate`, `fbt export openlineage`,
  and `fbt export otel` to inspect what produced a manual version.
