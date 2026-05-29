# FBT-RUNNER-010 Harden CLI-Agent Adapter Safety Contract

Status: done
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

The runner conformance harness now has an opt-in `--agent-adapter` safety
profile. It injects a redaction marker into source and asset files, creates
guard files at the source path, logical artifact path, and `.fbt/state`, and
fails if a runner leaks the marker or modifies those guarded locations.

The safety profile also requires a CLI-agent adapter to report a staging
workspace under `work.root` but outside `work.outputs`, and to report
fail-closed policy mapping through structured `fbt/event` attributes. The
copyable scaffold emits those markers and `make verify` runs it through the
agent-adapter profile.

Runner authoring, adapter packaging, scaffold, conformance, and security docs
now describe the concrete contract. OS-level sandboxing remains out of base
core unless a future spec chooses it deliberately.

## Next Check

Run:

```sh
make verify
```

Latest result: `make runner-conformance runner-scaffold-conformance` passed.
Expected final gate: `make verify` passes with the scaffold using
`--strict --agent-adapter`.
