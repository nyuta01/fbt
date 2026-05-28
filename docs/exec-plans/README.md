# Execution Plans

Execution plans are the restartable work records for agent tasks. They connect
structured task state to implementation evidence.

## Files

- `feature-list.json` is the structured task list.
- `active/` contains active and recently completed plan files.

## Rules

- Every `in_progress` or `done` task must link a plan.
- Every plan under `active/` must be referenced from `feature-list.json`.
- Every plan must include Observation, Decision, Permanent Fix, and Next Check.
- Do not use chat history as the source of truth.

