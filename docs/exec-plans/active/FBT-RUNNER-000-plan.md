# FBT-RUNNER-000 Capture external runner hardening backlog

## Observation

fbt is designed as a control plane that delegates transform execution to
external runners, but the current user experience can make the local
deterministic LLM and agent examples look like product runners. The protocol
and discovery specs are external-runner oriented, while build-time payloads,
process invocation, capability validation, and authoring support are not yet
strong enough for Claude Code, Codex, OpenAI, Claude, Gemini, or similar
adapters to feel first-class.

## Decision

Track runner extensibility as a focused post-MVP hardening theme. Keep fbt core
free of provider SDKs and agent runtimes. Make any provider or agent
integration an out-of-process runner that satisfies the fbt stdio JSON-RPC
protocol and writes output candidates into fbt-controlled work directories.

## Permanent Fix

Added prioritized `FBT-RUNNER-*` tasks covering complete runner payloads,
process invocation, capability validation, safe CLI-agent adapters, authoring
fixtures, demo-runner UX, and optional external provider adapter packaging.

## Next Check

Start with `FBT-RUNNER-001` so external runners receive enough context to run
without reparsing fbt project state.
