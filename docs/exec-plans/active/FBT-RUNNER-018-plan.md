# FBT-RUNNER-018 Add official Codex CLI and Claude Code adapter modules

## Observation

fbt documents Codex CLI and Claude Code as runner-compatible external agents,
but there were no official adapter modules demonstrating that boundary with
real CLI invocation, staging workspaces, and agent-adapter safety markers.

## Decision

Added `adapters/codex-cli` and `adapters/claude-code` as official nested Go
modules. Each adapter speaks the fbt runner protocol through `sdk/go`, stages
source files and assets under `work.root`, invokes the external CLI in
non-interactive mode, copies final content into `work.outputs`, and reports
fail-closed policy mapping for agent-adapter conformance.

## Permanent Fix

`make verify` now runs unit tests and strict `--agent-adapter` conformance for
both modules. The conformance checks use executable fixtures under `testdata/`
so repository verification remains network-free, credential-free, and
deterministic. These fixtures are protocol test fixtures only; normal projects
use `codex exec` or `claude -p`.

## Next Check

Done. `make verify` passes. Future work should improve shared staging helpers
or live opt-in smoke examples without moving agent runtimes into fbt core.
