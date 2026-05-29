# Practical Manual Generation Examples

Status: MVP-ready with external runners
Created: 2026-05-28
Updated: 2026-05-29
Audience: teams applying fbt to operational manuals and runbooks

## 1. Purpose

These examples show fbt projects shaped for real business workflows, not demo
runner output. They use external runner configuration and project-local wrappers
that call the official `adapters/openai` implementation. `fbt doctor` and
`fbt build` require `OPENAI_API_KEY`.

| Example | Inputs | Generated Artifact |
|---|---|---|
| `examples/incident_response_runbook` | incident event logs, response notes, postmortems, existing runbooks | incident response runbook |
| `examples/support_resolution_manual` | user inquiry tickets, support response logs, product docs, macros | support resolution manual |

Both examples use the same production loop:

```text
primary records
  -> fbt doctor / plan
  -> external LLM runner
  -> deterministic section eval
  -> committed manual artifact
  -> artifact inspection, docs, and standard lineage exports
```

fbt does not own human approval or publishing. After the artifact is generated,
route it through Git, PRs, CI, release tooling, or your knowledge-base workflow.

## 2. External Runner Boundary

The examples intentionally do not use `demo.llm` or `demo.agent`. Their
`fs_project.yml` files reference:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: bin/fbt-runner-openai
    args: ["responses"]
    env:
      - OPENAI_API_KEY
```

Validate the runner before building:

```sh
FBT_RUNNER_CONFORMANCE_COMMAND='examples/incident_response_runbook/bin/fbt-runner-openai responses' make runner-conformance
OPENAI_API_KEY=... fbt doctor --project-dir examples/incident_response_runbook
```

The runner owns the provider API call and credentials. fbt core owns state,
policy/eval checks, lineage, and artifact commits. If you install a separately
packaged runner, replace the project-local `command` with the installed command.

## 3. Incident Response Runbook

Use this when the source of truth is incident evidence:

```text
data/incidents/events/*.jsonl
data/incidents/response_logs/
data/incidents/postmortems/
data/reference/runbooks/
```

Concrete source-to-artifact mapping:

| Role | File | Contribution |
|---|---|---|
| Source | `data/incidents/events/INC-2026-0421.jsonl` | Machine-readable event timeline: pager alert, database saturation, mitigation, and resolution. |
| Source | `data/incidents/response_logs/INC-2026-0421-response.md` | Human response timeline, actions that worked, and unresolved gaps. |
| Source | `data/incidents/postmortems/INC-2026-0421-postmortem.md` | Root cause and corrective actions. |
| Format | `assets/incident_runbook_format.md` | Required headings for the official runbook. |
| Prompt | `assets/incident_runbook_prompt.md` | Evidence-only generation rules and invention guardrails. |
| Runner | `openai.responses` | External runner that performs the LLM call. |
| Artifact | `target/artifacts/runbooks/incident_response_runbook.md` | Maintained incident runbook. |

Representative evidence:

```json
{"incident_id":"INC-2026-0421","source":"pager","severity":"SEV2","service":"checkout-api","event":"p95_latency_ms above 2400 for 10 minutes","customer_impact":"elevated checkout timeouts for a subset of customers"}
{"incident_id":"INC-2026-0421","source":"incident_command","event":"traffic shifted away from us-east-1 read replica","result":"checkout timeout rate decreased from 7.8% to 1.1%"}
```

Expected artifact shape:

```md
## Detection
- Confirm sustained checkout latency, timeout rate, and database connection
  pool saturation.

## Mitigation
- Shift read traffic away from the affected read replica when the evidence
  matches this failure mode.

## Customer Communication
- Acknowledge elevated checkout latency.
- Verify payment status before recommending a retry.

## Source Evidence
- INC-2026-0421 event log
- INC-2026-0421 response log
- INC-2026-0421 postmortem
```

Build flow:

```sh
fbt doctor --project-dir examples/incident_response_runbook
fbt plan --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt build --project-dir examples/incident_response_runbook --select incident_response_runbook
fbt artifact explain incident_response_runbook --project-dir examples/incident_response_runbook
```

`artifact explain` proves which sources, prompt, format asset, policy, eval, and
runner produced the artifact version:

```text
Artifact: incident_response_runbook
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

Concrete source-to-artifact mapping:

| Role | File | Contribution |
|---|---|---|
| Source | `data/support/tickets/2026-05-12-login-and-billing.jsonl` | Ticket topics, customer impact, and resolution status. |
| Source | `data/support/response_logs/login-domain-change.md` | Actual agent steps and approved customer-facing explanation. |
| Source | `data/reference/product_docs/account-access.md` | Product rules that constrain the procedure. |
| Source | `data/reference/macros/` | Reusable approved response language. |
| Format | `assets/support_resolution_manual_format.md` | Required headings for the support manual. |
| Prompt | `assets/support_resolution_prompt.md` | Evidence-only generation rules and invention guardrails. |
| Runner | `openai.responses` | External runner that performs the LLM call. |
| Artifact | `target/artifacts/support/support_resolution_manual.md` | Maintained support manual. |

Representative evidence:

```json
{"ticket_id":"SUP-10422","topic":"login","summary":"Customer cannot receive password reset email after changing company domain","customer_impact":"blocked from accessing admin console","resolution_status":"resolved"}
{"ticket_id":"SUP-10437","topic":"billing","summary":"Customer asks why seat count increased after SSO group sync","customer_impact":"unexpected invoice estimate","resolution_status":"resolved"}
```

Expected artifact shape:

```md
## Intake Checklist
- Confirm workspace, affected user, current profile email, identity provider
  status, and admin edit capability.

## Resolution Procedure
- For domain-change login issues, have the workspace admin update the profile
  email before resending reset links.

## Customer Response Templates
- Explain that reset links are sent to the email currently stored on the user
  profile.

## Source Evidence
- SUP-10422
- login-domain-change response log
- account-access product documentation
```

Build flow:

```sh
fbt doctor --project-dir examples/support_resolution_manual
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt artifact explain support_resolution_manual --project-dir examples/support_resolution_manual
```

`artifact explain` gives the same provenance receipt for the support manual:

```text
Artifact: support_resolution_manual
Inputs
  ok input   support.inquiry_tickets        path=data/support/tickets/*.jsonl
  ok input   support.response_logs          path=data/support/response_logs
  ok input   reference.product_docs         path=data/reference/product_docs
  ok asset   support_resolution_manual_format
  ok asset   support_resolution_prompt
  ok runner  openai.responses
Outputs
  support_resolution_manual  target/artifacts/support/support_resolution_manual.md
```

The output contract lives in
`examples/support_resolution_manual/assets/support_resolution_manual_format.md`.
The required sections include intake checklist, triage, resolution procedure,
escalation, customer response templates, agent notes, maintenance notes, and
source evidence.

## 5. Operating Notes

- Add new source records; do not edit generated artifacts by hand.
- Update prompt, format, style guide, and evidence checklist assets when the
  manual contract changes.
- Use `fbt plan` before building to see whether source, policy, runner, model,
  or asset changes make the manual dirty.
- Use `fbt artifact history`, `fbt artifact explain`,
  `fbt export openlineage`, and `fbt export otel` to inspect what produced a
  manual version.
