# Support Resolution Manual Example

This is the support-ops practical example.

It turns tickets and support response notes into a support manual:

```text
customer tickets + agent response logs + product docs + approved macros
  -> target/artifacts/support/support_resolution_manual.md
  -> .fbt receipt with sources, runner, checks, version, and lineage
```

If you are new to fbt, read the incident runbook example first. This example is
equally realistic, but it has more input types.

## What This Example Uses

This project turns repeated support handling evidence into a manual that a
support team can maintain:

| Role | File | Concrete evidence or instruction |
|---|---|---|
| Source | `data/support/tickets/2026-05-12-login-and-billing.jsonl` | Ticket topics, customer impact, and resolution status. |
| Source | `data/support/response_logs/login-domain-change.md` | Agent handling steps and approved customer language. |
| Source | `data/reference/product_docs/account-access.md` | Product constraints for account access and identity changes. |
| Source | `data/reference/macros/` | Approved reusable responses. |
| Instruction | `assets/support_resolution_prompt.md` | Generate a manual from evidence only. |
| Format | `assets/support_resolution_manual_format.md` | Required sections such as Intake Checklist, Triage, Resolution Procedure, and Source Evidence. |
| Runner | `openai.responses` in `fs_project.yml` | Calls the external OpenAI runner through fbt's runner protocol. |
| Artifact | `target/artifacts/support/support_resolution_manual.md` | The generated support manual. |

Representative input:

```json
{"ticket_id":"SUP-10422","topic":"login","summary":"Customer cannot receive password reset email after changing company domain","customer_impact":"blocked from accessing admin console","resolution_status":"resolved"}
{"ticket_id":"SUP-10437","topic":"billing","summary":"Customer asks why seat count increased after SSO group sync","customer_impact":"unexpected invoice estimate","resolution_status":"resolved"}
```

The response log contributes the usable support answer:

```text
Your workspace admin needs to update the email on your user profile before the
reset link can be delivered to the new address.
```

The generated artifact should convert those records into an operational manual:

```md
## Intake Checklist
- Confirm workspace, affected user, current profile email, identity provider
  status, and whether the workspace admin can edit the user profile.

## Resolution Procedure
- For domain-change login issues, have the workspace admin update the profile
  email before resending password reset links.

## Customer Response Templates
- Explain that reset links are sent to the email currently stored on the user
  profile.

## Source Evidence
- SUP-10422
- login-domain-change response log
- account-access product documentation
```

## Run The Workflow

Preview the work:

```sh
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
```

After setting credentials, build the manual:

```sh
export OPENAI_API_KEY=...
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
```

You get:

```text
target/artifacts/support/support_resolution_manual.md
.fbt/artifacts/<artifact_version>/content
.fbt/state/artifact_versions.json
.fbt/state/run_results.jsonl
```

Inspect where it came from:

```sh
fbt artifact show support_resolution_manual --project-dir examples/support_resolution_manual
fbt artifact history support_resolution_manual --project-dir examples/support_resolution_manual
fbt artifact explain support_resolution_manual --project-dir examples/support_resolution_manual
```

The explain output is the receipt. It shows the producer transform, current
artifact version, source fingerprints, prompt/format asset fingerprints, eval,
policy, runner, and final output path:

```text
Artifact: support_resolution_manual
Producer
  Transform        support_resolution_manual

Inputs
  ok input   support.inquiry_tickets        path=data/support/tickets/*.jsonl
  ok input   support.response_logs          path=data/support/response_logs
  ok input   reference.product_docs         path=data/reference/product_docs
  ok input   reference.approved_macros      path=data/reference/macros
  ok asset   support_resolution_manual_format
  ok asset   support_resolution_prompt
  ok runner  openai.responses

Outputs
  support_resolution_manual  target/artifacts/support/support_resolution_manual.md
```

Export standard lineage:

```sh
fbt export openlineage \
  --project-dir examples/support_resolution_manual \
  --output examples/support_resolution_manual/target/lineage/openlineage.ndjson
```

Human approval and publishing should happen in Git, PR, CI, release, or catalog
workflows outside fbt.
