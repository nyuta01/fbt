# FBT-EVAL-003 Plan

## Task

Add a production quality-gate reference workflow for the daily support
knowledge loop.

## Observation

fbt records deterministic evals and lineage, but production users still need a
clear pattern for structural gates, evidence-grounding gates, and human/domain
review handoff without putting LLM judge or approval logic into fbt core.

## Decision

Add an external CI-style quality gate script under the daily ops wrapper. It
checks generated artifacts and run-bundle evidence, writes text and JSON gate
results, and records domain review as a pending external requirement.

## Permanent Fix

`daily-ops-smoke` now executes the quality gate as part of the production
wrapper and asserts both passing gates and the pending domain-review handoff
appear in the run bundle.

## Next Check

- `make daily-ops-smoke`
- `make verify`
