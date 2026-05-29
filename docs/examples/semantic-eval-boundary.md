# Semantic Eval Boundary

Status: MVP-ready

fbt core runs deterministic evals. It does not call model judges, embed judge
SDKs, or implement semantic scoring logic.

Use this boundary:

| Need | Put it in |
|---|---|
| Required sections, required text, non-empty output | deterministic fbt eval |
| Model-based factuality, unsupported-claim review, tone judgement | external runner transform that produces a judge report artifact |
| Human approval or publish decision | Git, PRs, CI, release tooling, or knowledge-base workflow |

## Deterministic Gate In Core

```yaml
evals:
  - name: required_sections
    type: deterministic
    config:
      sections:
        - Summary
        - Decision
        - Rollback
    grants_confidence: structural
```

When this eval passes during `fbt build`, fbt records an evaluation result and
can grant `structural` confidence to the artifact version.

## Semantic Judge Outside Core

Model judging is a normal generated-artifact workflow:

```yaml
transforms:
  - name: judge_runbook_claims
    type: llm
    runner: openai.responses
    inputs:
      - ref: incident_response_runbook
        require:
          confidence: structural
    outputs:
      - name: runbook_claim_judge_report
        type: markdown
        path: target/artifacts/judges/runbook_claim_judge_report.md
    assets:
      - ref: unsupported_claims_rubric
    policy: judge_scope
    tags: ["judge"]
```

The runner owns the model call and rubric interpretation. fbt owns the output
artifact receipt: inputs, runner, output version, policy decision, and lineage.
Your CI, PR, or publishing workflow can then decide how to use the judge report.

## Delegated Eval Declarations

`type: semantic` and `type: llm_judge` remain reserved config shapes. In the
MVP, fbt records those evals as skipped during build and grants no confidence
from them. The skipped result is visible in build output, state receipts, and
`fbt artifact explain` with a hint to model the check as an external judge
transform when it must be an active gate. Do not rely on them for gating until a
delegated eval-runner protocol is specified and implemented.
