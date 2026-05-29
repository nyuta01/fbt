# FBT-MVP-009 Implement fake and command runners for local MVP

## Observation

Core has a JSON-RPC protocol client, but there are no local runner executables
that implement the protocol for deterministic tests, conformance scenarios, or
simple command transforms.

## Decision

Add two out-of-process runners:

- `tests/runner_fixtures/fake`: deterministic protocol runner that creates simple output
  candidates under the assigned work directory without external services
- `examples/runner_adapters/command`: protocol runner that executes a configured local command
  with `FBT_WORK_ROOT`, `FBT_WORK_TEMP`, and `FBT_WORK_OUTPUTS`
- tests that start each runner as an external process through the protocol
  client
- a `tests/fixtures` directory for future conformance fixtures

## Permanent Fix

Added process-level Go tests that start `tests/runner_fixtures/fake` and `examples/runner_adapters/command`
through the protocol client and verify initialize, runTransform, output files,
and output-candidate notifications.

## Next Check

Run:

```sh
make verify
```

Result on 2026-05-28: pass.
