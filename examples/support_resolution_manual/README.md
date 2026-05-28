# Support Resolution Manual Example

This is the support-ops practical example.

It turns tickets and support response notes into an approved support manual:

```text
customer tickets + agent response logs + product docs + approved macros
  -> target/artifacts/support/support_resolution_manual.md
  -> .fbt receipt with sources, runner, checks, version, review, and lineage
```

Use this example when your real workflow looks like:

> Support keeps solving the same kinds of customer issues. The answers are
> spread across tickets, agent notes, product docs, and approved reply macros.
> We want one official manual that agents can follow during live customer
> conversations.

If you are new to fbt, read the incident runbook example first. This example is
equally realistic, but it has more input types.

## Input Evidence

The example covers login-domain changes and SSO billing seat sync.

| File | Role |
|---|---|
| `data/support/tickets/2026-05-12-login-and-billing.jsonl` | Customer problems and business impact. |
| `data/support/response_logs/login-domain-change.md` | Steps that resolved login-domain issues. |
| `data/support/response_logs/sso-seat-sync.md` | Steps that resolved SSO billing seat questions. |
| `data/reference/product_docs/*.md` | Product behavior the manual must not contradict. |
| `data/reference/macros/*.md` | Approved language for customer-facing responses. |

The source facts are specific enough for a real manual:

```text
Customer cannot receive password reset email after changing company domain.
Customer asks why seat count increased after SSO group sync.
Support must not update identity fields without admin verification.
Seat estimates update after group sync completes.
```

## Desired Output

The generated artifact is:

```text
target/artifacts/support/support_resolution_manual.md
```

The required format is declared in `assets/support_resolution_manual_format.md`.
It forces sections such as `Audience`, `When to Use`, `Intake Checklist`,
`Resolution Procedure`, `Customer Response Templates`, and `Source Evidence`.

The manual is meant for support agents. fbt's receipt is meant for leads and
operators who need to know:

- which tickets and notes informed the manual
- which product docs and macros constrained the answer
- which runner and model generated the version
- which checks passed
- who approved the exact artifact version

## Repeated Operation

For daily operation, keep adding ticket exports and response notes under the
declared source paths:

```text
data/support/tickets/2026-05-13.jsonl
data/support/response_logs/new-topic.md
data/reference/product_docs/updated-feature.md
```

The next `fbt plan` sees the changed source file set or content and marks
`support_resolution_manual` dirty. The next `fbt build` creates a new manual
artifact version, and the support lead reviews that exact version before it is
trusted.

If the team receives hundreds of tickets per day, split sources and transforms
by product area, customer segment, or date window. fbt tracks the build graph
and versions, but it does not schedule runs or automatically create one
artifact per incoming ticket.

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

Preview the work:

```sh
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
```

Expected first-run shape:

```text
Plan: 1 selected, 1 run, 0 skipped, 0 blocked
run transform.support_resolution_manual.support_resolution_manual
  reason: no previous successful run
  reason: output missing
```

Check runner readiness:

```sh
fbt doctor --project-dir examples/support_resolution_manual
```

Without credentials, the useful failure is:

```text
Doctor: error
error RUNNER_ENV_MISSING: runner env OPENAI_API_KEY is not set
ok RUNNER_COMMAND_OK: runner command is executable: ...
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

Inspect and approve the exact generated version:

```sh
fbt review show support_resolution_manual --project-dir examples/support_resolution_manual
fbt review approve support_resolution_manual \
  --project-dir examples/support_resolution_manual \
  --comment "Support lead approved"
```

Explain where it came from:

```sh
fbt artifact history support_resolution_manual --project-dir examples/support_resolution_manual
```

Generate local docs:

```sh
fbt docs generate --project-dir examples/support_resolution_manual
```

You get:

```text
target/docs/index.md
```
