# FBT-EVAL-002 Make Skipped Semantic And LLM-Judge Evals Visible In User-Facing UX

## Observation

`semantic` and `llm_judge` eval declarations are intentionally recorded as
skipped and grant no confidence in core. Users can still misread those
declarations as active quality gates.

## Decision

Make the skipped behavior visible in receipts, artifact inspection, docs, and
schema descriptions. The recommended active pattern remains an external judge
transform that produces a report artifact.

## Permanent Fix

Add user-facing hints and deterministic checks so skipped semantic/LLM-judge
evals are not mistaken for enforced gates.

## Next Check

Run eval/build/CLI tests, docs/schema checks, and `make verify`.

