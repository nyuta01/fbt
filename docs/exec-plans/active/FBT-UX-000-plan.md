# FBT-UX-000 Capture user-facing UX hardening backlog

## Observation

The local MVP demo works end to end, but from a user perspective the CLI still
requires too much inference after blocked/skipped decisions and generated
artifact discovery.

Comparable tools make these flows explicit: DVC exposes status/repro/diff/list
and artifact access paths, dbt has debug/build/ls/artifact surfaces, Dagster
centers asset metadata and lineage, Bazel exposes dependency explanation
queries, MLflow exposes artifact/model version inspection, and
Terraform/Pulumi emphasize inspect-before-apply workflows.

## Decision

Create prioritized post-MVP backlog items for user-facing workflow hardening:

- actionable next-step guidance for blocked and skipped work
- artifact explanation for plan decisions
- artifact path, show, and history commands
- top-level project doctor checks
- stronger YAML validation and authoring diagnostics

## Permanent Fix

Added `FBT-UX-001` through `FBT-UX-006` to
`docs/exec-plans/feature-list.json` with dependencies, paths, verification
expectations, and research-backed notes. Updated `AGENT_PROGRESS.md` and
`docs/QUALITY_SCORE.md` so the user-workflow UX gap is visible outside chat.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
