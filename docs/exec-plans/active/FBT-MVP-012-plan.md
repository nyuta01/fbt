# FBT-MVP-012 Implement evals and confidence

Superseded note: the review/approval portions originally implemented by this
task were removed by `FBT-UNIX-011`.

## Observation

Build can commit artifact versions and planner can block on confidence
requirements, but eval execution is not yet connected to the lifecycle or CLI.

## Decision

Implement the MVP eval loop:

- run deterministic evals against output candidates before official commit
- record evaluation results in local state
- grant configured confidence when evals pass
- expose `fbt eval`

## Permanent Fix

Added deterministic eval, build, CLI, planner, state, and smoke tests covering
eval pass/fail records, confidence grants, and downstream confidence blocking.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
