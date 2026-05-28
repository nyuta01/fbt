# FBT-UNIX-009 Keep Semantic Evaluation Outside Core

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Clarify how fbt should handle semantic or LLM-judge evaluation while preserving
the Unix-style boundary that core does not own model-provider logic.

## Observation

Docs listed `semantic` and `llm_judge` eval types, but the practical boundary
was easy to misread as "fbt core should call model judges." In the MVP, core
only executes deterministic evals and records semantic/LLM-judge declarations
as skipped.

## Decision

Document semantic judging as an external runner workflow that produces a normal
judge report artifact. Keep delegated eval runner support reserved until a
separate protocol exists.

## Permanent Fix

- Added a semantic eval boundary doc and example pattern.
- Clarified spec, project config, usage, runner protocol, examples, and docs
  site wording.
- Made explicit that `semantic` and `llm_judge` eval declarations are skipped
  in MVP and grant no confidence.
- Recommended external judge transforms for model-based evaluation today.

## Next Check

Run:

```sh
make verify
```

Expected result: docs validation and Go tests pass without adding model-judge
logic or provider SDKs to fbt core.
