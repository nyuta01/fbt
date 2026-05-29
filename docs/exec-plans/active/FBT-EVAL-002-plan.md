# FBT-EVAL-002 Make Skipped Semantic And LLM-Judge Evals Visible In User-Facing UX

## Observation

`semantic` and `llm_judge` eval declarations are intentionally recorded as
skipped and grant no confidence in core, but the previous state shape did not
carry an explicit reason or external-judge hint into build output, receipts, and
artifact explanation.

## Decision

Persist skipped delegated-eval reason/hint fields, surface them in build output
and `artifact explain`, and update docs/schema descriptions. The recommended
active pattern remains an external judge transform that produces a report
artifact.

## Permanent Fix

Skipped semantic/LLM-judge eval declarations now produce explicit skipped
evaluation details in state and transform-run receipts, build output shows the
skip reason and active-gate hint, `artifact explain` marks the eval dependency
as skipped, and tests cover the behavior.

## Next Check

`make verify` must pass; future model-judge behavior should be implemented as
external judge transforms or a separately specified delegated eval-runner
protocol.
