# FBT-RUNNER-022 Enforce CLI-agent policy mapping instead of marker-only safety

## Observation

Codex CLI and Claude Code adapters emit fail-closed policy markers, but the
implementation does not enforce requested fbt policy values. Codex is always
started with `--sandbox workspace-write`, Claude Code is always started with
`--permission-mode dontAsk`, and the conformance profile currently checks the
marker rather than negative policy behavior.

## Decision

Treat policy mapping as executable behavior, not metadata. Map the supported
network, tool, timeout, and sandbox controls for each CLI. If fbt policy cannot
be represented safely by the selected CLI, return a structured runner error
before invoking the external agent.

## Permanent Fix

Extend agent-adapter conformance with negative policy cases, such as denied
network or shell access. Add adapter unit tests that verify CLI arguments and
that unsupported policy requests fail closed.

## Next Check

Run adapter tests, strict agent-adapter conformance for Codex and Claude Code,
then `make verify`.
