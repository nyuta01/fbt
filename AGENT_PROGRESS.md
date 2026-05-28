# Agent Progress

Last updated: 2026-05-28

## Current State

The repository contains the English design/specification set for `fbt`, a
baseline AI-first engineering harness, repo governance files, and a minimal Go
CLI scaffold. Product code is intentionally minimal: the CLI exposes `help` and
`version` so verification has an executable target while the MVP implementation
is still being planned.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

The remaining practical-MVP work is registered as `FBT-MVP-001` through
`FBT-MVP-016` in `docs/exec-plans/feature-list.json`.

## Verification

Latest expected gate:

```sh
make verify
```

This runs:

- `make harness-check`
- `make drift-check`
- `make validate-docs`
- `make fmt-check`
- `make go-test`
- `make cli-smoke`

## Next Steps

1. Start `FBT-MVP-001` with a plan for project discovery and `fs_project.yml`
   parsing.
2. Add executable tests for `config_version`, artifact type alias validation,
   and path validation before implementing downstream graph behavior.
3. Expand the Go CLI only when a task has a spec-backed acceptance criterion.
4. Keep `make verify` green after each bounded MVP task.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
