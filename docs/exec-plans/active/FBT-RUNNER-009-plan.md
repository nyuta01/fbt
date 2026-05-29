# FBT-RUNNER-009 Add Opt-In Real Runner Adapter Smoke Matrix

Status: done
Owner: agent
Updated: 2026-05-29

## Goal

Make real external runner adoption easier without adding provider SDKs or agent
runtimes to fbt core.

## Observation

The core runner boundary is correct: runners are external processes. The
repository includes demo runners, a command runner, an optional OpenAI runner,
and a copyable scaffold. A user evaluating fbt with OpenAI, Claude Code, Codex,
Gemini, or an internal CLI still needs more concrete adapter validation guidance
than a protocol document alone.

## Decision

Keep real provider checks opt-in. Add an adapter smoke matrix and docs that show
how to validate external runners through `doctor`, conformance, and a small
real project without making credentials or SDKs part of `make verify`.

## Permanent Fix

Added `make runner-adapter-smoke`, backed by
`scripts/smoke-runner-adapters.sh`. The target is opt-in and is not part of
`make verify`.

The matrix format is:

```text
logical_name|runner_type|artifact_type|command|required_env_csv|agent_adapter
```

For each row, the script runs runner conformance, adds `--agent-adapter` when
requested, generates a temporary fbt project, runs `fbt doctor`, and runs
`fbt plan --select adapter_smoke`. If
`FBT_RUNNER_ADAPTER_SMOKE_BUILD=1` is set, it also builds the temporary
artifact and inspects it with `fbt artifact show`.

Docs now show command shapes for OpenAI, Codex CLI, Claude Code, Gemini, and an
internal CLI adapter without adding provider SDKs or agent runtimes to fbt core.

## Next Check

Run:

```sh
make verify
```

Latest targeted result:

```sh
FBT_RUNNER_ADAPTER_SMOKE_MATRIX='scaffold.agent|agent|markdown|examples/runner_adapter_scaffold/bin/fbt-runner-example||true' \
FBT_RUNNER_ADAPTER_SMOKE_BUILD=1 \
make runner-adapter-smoke
```

passed. Final gate: `make verify` passed and base verification remains offline.
