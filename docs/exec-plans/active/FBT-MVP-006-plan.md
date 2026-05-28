# FBT-MVP-006 Implement parse, plan, state, and artifact CLI surfaces

## Observation

Core packages now parse project resources, generate manifests, plan dirty state,
compute descriptors, and persist local state, but the CLI still exposes only
`help` and `version`. Users cannot exercise the implemented product behavior
without writing Go tests.

## Decision

Implement the first product CLI surface:

- `fbt parse` parses the project, builds a manifest, and writes
  `.fbt/state/manifest.json`
- `fbt plan` parses, builds the current manifest, compares previous manifest and
  state when available, and reports run/skip/blocked nodes
- `fbt state status|ls|current` inspects local state files and current artifact
  pointers
- `fbt artifact ls|show|versions` inspects artifact version records
- support `--project-dir`, `--json`, and basic `--select`
- update CLI tests and smoke coverage with a real temporary project

## Permanent Fix

Added CLI unit tests and a CLI smoke fixture that exercise `parse`, `plan`,
`state status`, and `artifact ls` against a temporary local project.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
