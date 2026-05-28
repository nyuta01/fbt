# FBT-MVP-006 Implement plan and artifact CLI surfaces

Superseded note: the public `fbt parse` and `fbt state` commands described
here were removed by `FBT-UNIX-013`. Manifest writes now happen through
`fbt plan`; state inspection happens through `fbt artifact`.

## Observation

Core packages now parse project resources, generate manifests, plan dirty state,
compute descriptors, and persist local state, but the CLI still exposes only
`help` and `version`. Users cannot exercise the implemented product behavior
without writing Go tests.

## Decision

Implement the first product CLI surface:

- `fbt plan` parses, builds the current manifest, compares previous manifest and
  state when available, and reports run/skip/blocked nodes
- `fbt artifact ls|show|history` inspects artifact version records
- support `--project-dir`, `--json`, and basic `--select`
- update CLI tests and smoke coverage with a real temporary project

## Permanent Fix

Added CLI unit tests and a CLI smoke fixture that exercise `plan` and
`artifact ls` against a temporary local project.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
