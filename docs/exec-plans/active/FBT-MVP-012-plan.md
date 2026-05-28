# FBT-MVP-012 Implement evals, review gates, approvals, and confidence

## Observation

Build can commit artifact versions and planner can block on confidence/review
requirements, but eval execution and approval state are not yet connected to
the lifecycle or CLI.

## Decision

Implement the MVP eval/review loop:

- run deterministic evals against output candidates before official commit
- record evaluation results in local state
- commit review-required artifacts as pending review
- store approval records by artifact version
- promote current artifact pointers to `approved` and `reviewed` on approval
- expose `fbt eval` and `fbt review status|approve|reject`

## Permanent Fix

Added deterministic eval, approval, build, CLI, planner, state, and smoke tests
covering eval pass/fail records, pending review commits, review approval
promotion, and downstream reviewed/approved blocking behavior.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
