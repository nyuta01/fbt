# FBT-UNIX-006 Provide A Minimal Runner Adapter Scaffold

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make it easy to wrap one external executable, provider, CLI agent, converter,
or internal service as an fbt-compatible runner without adding dependencies to
fbt core.

## Observation

The runner protocol and adapter packaging docs were complete enough for a
careful implementer, but there was no tiny copyable adapter that showed the
exact JSON-RPC loop, output-candidate boundary, and conformance command.

## Decision

Add a dependency-free Python scaffold under `examples/runner_adapter_scaffold`
and wire it into verification through a dedicated conformance target. Keep the
scaffold outside `internal/` so it remains an example of an out-of-process
adapter, not a core runner framework.

## Permanent Fix

- Added `examples/runner_adapter_scaffold` with a minimal stdio JSON-RPC runner
  and plugin manifest.
- Added docs that explain what to replace, how to package the adapter, and how
  to run the conformance harness.
- Added `make runner-scaffold-conformance` and included it in `make verify`.

## Next Check

Run:

```sh
make verify
```

Expected result: both the source fake runner and the scaffold runner pass the
strict conformance harness.
