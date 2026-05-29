# FBT-PROD-001 Plan

## Task

Run a production-shaped daily support pilot with real official adapters instead
of demo runners.

## Observation

`daily_qa_ops` proved the production wrapper with deterministic demo runners.
Production readiness still needed evidence that the same wrapper can use
official adapter modules and that live provider calls remain explicit opt-in.

## Decision

Add `scripts/pilot-daily-real-adapters.sh`, which rewires a temporary copy of
`daily_qa_ops` to the official OpenAI and Codex CLI adapters. The default path
uses OpenAI fake response and Codex fixture execution so verification remains
network-free; live OpenAI execution is enabled only with
`FBT_PILOT_LIVE_OPENAI=1` and `OPENAI_API_KEY`.

## Permanent Fix

`make production-pilot-smoke` runs the real-adapter pilot and asserts the run
bundle contains protocol diagnostics, quality gates, and OpenLineage evidence
for the official adapters.

## Next Check

- `make production-pilot-smoke`
- `make verify`
