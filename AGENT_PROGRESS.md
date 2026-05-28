# Agent Progress

Last updated: 2026-05-28

## Current State

The repository contains the English design/specification set for `fbt`, a
baseline AI-first engineering harness, repo governance files, a minimal Go CLI
scaffold, and the first parser implementation baseline. The CLI still exposes
only `help` and `version`; project behavior is currently available through
internal packages and tests.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

`FBT-MVP-001` through `FBT-MVP-008` are complete. The remaining practical-MVP
work is registered as `FBT-MVP-009` through `FBT-MVP-016` in
`docs/exec-plans/feature-list.json`.

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

1. Start `FBT-MVP-009` with a plan for fake and command runners.
2. Provide local runner executables for deterministic conformance and command
   transforms without adding heavy dependencies to the base CLI.
3. Expand the Go CLI only when `FBT-MVP-006` has a spec-backed acceptance
   criterion.
4. Keep `make verify` green after each bounded MVP task.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
