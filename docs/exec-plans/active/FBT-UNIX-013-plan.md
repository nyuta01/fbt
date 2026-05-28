# FBT-UNIX-013 Prune Public CLI Command Surface

Status: done  
Owner: agent  
Updated: 2026-05-29

## Goal

Keep fbt as a small Unix-style file build tool by removing user-facing commands
that expose internal phases or duplicate another command's job.

## Observation

After review removal and strict argument handling, the CLI still exposed
`parse`, `eval`, `docs`, `state`, `runner`, and `artifact versions`. These made
the product feel like a toolbox of internal operations instead of a focused
build-control loop.

## Decision

Keep the public command set to `init`, `doctor`, `plan`, `build`, `artifact`,
`diff`, `export`, `version`, and `help`. Parsing belongs inside `doctor`,
`plan`, and `build`; evals run as part of build; state is inspected through
`artifact`; runner readiness is reported through `doctor`; docs and dashboards
should consume standard exports or `.fbt/state` externally.

## Permanent Fix

- Removed public routing and CLI package implementations for `parse`, `eval`,
  `docs`, `state`, and `runner`.
- Removed the duplicate `artifact versions` alias; `artifact history` is the
  version-inspection command.
- Updated CLI tests, smoke scripts, conformance checks, product docs, docs
  site content, examples, and progress notes to use the smaller surface.
- Added regression coverage that pruned commands return the normal unknown
  command error.

## Next Check

Run:

```sh
make verify
```

Expected result: all checks pass, and pruned commands remain unknown.
