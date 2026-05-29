# FBT-RUNNER-010 Harden CLI-Agent Adapter Safety Contract

Status: todo
Owner: agent
Updated: 2026-05-29

## Goal

Make CLI-agent adapters safe enough to recommend for Codex CLI, Claude Code,
Gemini CLI, and similar external agents.

## Observation

Core rejects output candidates outside the assigned work output directory before
commit, but core does not sandbox arbitrary external processes. If a CLI-agent
adapter runs an agent directly in the project root, the agent may modify sources,
logical artifact paths, or `.fbt/state` outside the normal fbt commit path.

## Decision

Keep sandboxing and provider-specific controls in external adapters, but make
the safety contract testable. A recommended adapter must prove staging workspace
behavior, fail-closed policy mapping, redacted events, and no direct official
state/source writes.

## Permanent Fix

Pending. Expected permanent fix:

- Extend runner conformance or add adapter conformance scenarios for
  staging-workspace and direct-write denial expectations.
- Update runner authoring docs with concrete adapter patterns.
- Keep OS-level sandboxing out of base core unless a future spec chooses it
  deliberately.

## Next Check

Run:

```sh
make verify
```

Expected result: recommended CLI-agent adapters have a repeatable safety check
that preserves fbt's commit boundary.
