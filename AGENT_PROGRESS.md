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
release-binary smoke checks. The MVP source default is `0.1.0`, and release
builds can stamp version, commit, and build date metadata into the CLI.
Plan and build output now include concrete `next:` commands for blocked and
skipped work, and `fbt artifact explain TARGET` gives a focused explanation of
one artifact's plan decision. Artifact inspection now includes `artifact path`,
enriched `artifact show`, and `artifact history`. Review inspection now
includes `review show` and pending review guidance before approval. Standard
export contracts are defined for OpenLineage, OpenTelemetry, OpenMetadata, and
standard-compatible visualization, and `fbt export openlineage` now emits
OpenLineage RunEvent NDJSON with fbt lineage facets for artifact versions.
`fbt export otel` now emits local-first OTLP/JSON traces from run results,
including invocation/transform spans, usage attributes, and safe runner events.
The conformance suite now checks OpenLineage and OTel standard-export payload
shape plus redaction of raw source content and marker secrets.
`fbt doctor` now checks project readiness, state writability/lock acquisition,
runner discovery, and runner protocol initialization. YAML authoring diagnostics
now include line numbers where available and actionable hints for common parse
errors.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

The practical local MVP tasks are complete. Remaining tracked work is release
readiness, user-facing workflow hardening, and post-MVP depth:
repository/release publication, opt-in real LLM smoke, command-surface cleanup,
OpenMetadata evaluation, standard-compatible visualization recipes, expanded
conformance, full policy-decision records, and semantic descriptors.
`FBT-REL-002` is blocked on
maintainer release credentials and signing setup: no Git remote, signing config,
or `v0.1.0` tag is present locally. `FBT-REL-003` is blocked until that signed
release baseline exists.

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
2. Complete maintainer-owned `FBT-REL-002` when release credentials and signing
   setup are available; otherwise continue with the next unblocked P0 agent
   task.
3. Continue with `FBT-STD-006` if prioritizing standard visualization docs.
4. Keep fbt-native state as the internal source of truth and delegate graph,
   trace, and catalog visualization to standard-compatible tools where
   possible.
5. Keep expanding the Go CLI only when a task has a spec-backed acceptance
   criterion.
6. Keep `make verify` green after each bounded task.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
