# Agent Progress

Last updated: 2026-05-28

## Current State

The repository contains the English design/specification set for `fbt`, a
baseline AI-first engineering harness, repo governance files, a Go CLI, parser,
manifest graph, planner, descriptor/state primitives, runner discovery,
protocol client, local runners, and the first build lifecycle.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

`FBT-MVP-001` through `FBT-MVP-010` are complete. The remaining practical-MVP
work is registered as `FBT-MVP-011` through `FBT-MVP-016` in
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

1. Start `FBT-MVP-011` with a plan for policy and security enforcement.
2. Harden build-time read/write scope, output candidate, source mutation,
   redaction, timeout, and failed-run state safety checks.
3. Keep expanding the Go CLI only when a task has a spec-backed acceptance
   criterion.
4. Keep `make verify` green after each bounded MVP task.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
