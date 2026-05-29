# FBT-RUNNER-009 Add Opt-In Real Runner Adapter Smoke Matrix

Status: todo
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

Pending. Expected permanent fix:

- Document adapter-specific smoke command shapes and required env vars.
- Add opt-in Make/script entrypoints for locally installed adapters.
- Keep base verification deterministic and service-free.

## Next Check

Run:

```sh
make verify
```

Expected result: base verify remains offline, while adapter authors get a clear
opt-in smoke path for real installed runners.
