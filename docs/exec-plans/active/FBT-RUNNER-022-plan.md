# FBT-RUNNER-022 Enforce CLI-agent policy mapping instead of marker-only safety

## Observation

Codex CLI and Claude Code adapters emit fail-closed policy markers, but the
implementation does not enforce requested fbt policy values. Codex is always
started with `--sandbox workspace-write`, Claude Code is always started with
`--permission-mode dontAsk`, and the conformance profile currently checks the
marker rather than negative policy behavior.

Follow-up observation: the adapter boundary now needs positive conformance for
the safe policy each CLI can represent, plus negative conformance for policy
requests the CLI cannot enforce.

## Decision

Treat policy mapping as executable behavior, not metadata. Map the supported
network, tool, timeout, and sandbox controls for each CLI. If fbt policy cannot
be represented safely by the selected CLI, return a structured runner error
before invoking the external agent.

Codex CLI uses a read-only sandbox and timeout mapping, then fails closed for
network denial, tool lists, max tool calls, and max cost. Claude Code maps tool
allow/deny lists, timeout, and max budget, then fails closed for network denial,
unknown tools, and max tool calls.

## Permanent Fix

Extend agent-adapter conformance with negative policy cases, such as denied
network or shell access. Add adapter unit tests that verify CLI arguments and
that unsupported policy requests fail closed.

## Next Check

Done. Adapter unit tests, strict positive and negative agent-adapter
conformance for Codex CLI and Claude Code, and `make verify` pass.
