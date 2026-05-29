# FBT-CONFIG-001 Remove Or Implement Inert Config Fields

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Ensure every accepted project config field either has real behavior or is
explicitly reserved with actionable diagnostics.

## Observation

`execution.max_workers`, `execution.fail_fast`, `defaults.cache`,
`defaults.confidence`, and transform-level `cache` appear in the project
contract. Some are decoded and fingerprinted but do not currently change
execution behavior. In a declarative CLI, placebo config is worse than missing
config because users will trust settings that do not act.

## Decision

Audit the config surface through the Unix lens: keep only the smallest set of
controls needed for the build-receipt model. Implement fields that are essential
now. Mark future fields as reserved or remove them from the public contract.

## Permanent Fix

Pending. Expected permanent fix:

- Decide field-by-field whether to implement, reserve, or remove.
- Align docs, examples, parser diagnostics, and tests with that decision.
- Prefer one explicit rebuild control and simple confidence/eval behavior over
  a hidden cache engine.

## Next Check

Run:

```sh
make verify
```

Expected result: project YAML no longer accepts no-op controls as if they were
active behavior.
