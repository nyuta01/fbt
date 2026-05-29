# FBT-RUNNER-017 Promote OpenAI runner into official adapter module

## Observation

The OpenAI runner was still a source-checkout example even though practical
manual-generation examples use it as the real provider-backed path. That made
the supported integration boundary less clear than the new monorepo adapter
strategy.

## Decision

Moved the OpenAI Responses runner into `adapters/openai` as an official nested
Go module with its own command entrypoint, plugin manifest, README, tests, and
conformance target. The adapter uses `sdk/go` protocol/output helpers and keeps
`OPENAI_API_KEY` inside the external adapter process.

## Permanent Fix

`make verify` now runs OpenAI adapter tests and network-free OpenAI adapter
conformance using `FBT_OPENAI_ADAPTER_FAKE_RESPONSE`. Practical example
wrappers invoke `go run ./adapters/openai/cmd/fbt-runner-openai`.

## Next Check

Done. `make verify` passes. The next task should add official Codex CLI and
Claude Code adapter modules under `adapters/`.
