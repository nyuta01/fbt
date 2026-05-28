# Agent Progress

Last updated: 2026-05-28

## Current State

The repository contains the English design/specification set for `fbt`, a
baseline AI-first engineering harness, repo governance files, a Go CLI, parser,
manifest graph, planner, descriptor/state primitives, runner discovery,
protocol client, local fake/command/LLM/agent runners, the first build
lifecycle, deterministic evals, review approvals, confidence promotion, init
templates, a runnable local knowledge-loop example, artifact diffing, and
static Markdown docs generation.
The current verification gate also includes deterministic conformance and local
release-binary smoke checks.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

The practical local MVP tasks are complete. Remaining tracked work is release
readiness, user-facing workflow hardening, and post-MVP depth: version stamping,
repository/release publication, MVP-ready docs, actionable blocked/skipped
guidance, artifact explain/path/show/history, safer review inspection,
project-level doctor checks, stronger YAML diagnostics, opt-in real LLM smoke,
command-surface cleanup, expanded conformance, full policy-decision records,
and semantic descriptors.

## Verification

Latest expected gate:

```sh
make verify
```

This runs:

- `make harness-check`
- `make drift-check`
- `make validate-docs`
- `make fmt-check`
- `make go-test`
- `make cli-smoke`
- `make e2e-smoke`
- `make conformance`
- `make dist-check`

## Next Steps

1. Keep base runtime free of provider SDKs and heavyweight agent dependencies.
2. Start `FBT-UX-001` if prioritizing day-to-day user workflow polish, or
   `FBT-REL-001` if prioritizing release publication readiness.
3. Keep expanding the Go CLI only when a task has a spec-backed acceptance
   criterion.
4. Keep `make verify` green after each bounded task.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
