# Agent Progress

Last updated: 2026-05-28

## Current State

The repository contains the English design/specification set for `fbt`, a
baseline AI-first engineering harness, repo governance files, a Go CLI, parser,
manifest graph, planner, descriptor/state primitives, runner discovery,
protocol client, local fake/command/LLM/agent runners, an optional OpenAI
Responses external runner, the first build lifecycle, deterministic evals,
review approvals, confidence promotion, init templates, a runnable local
knowledge-loop example, practical external-runner manual-generation examples,
artifact diffing, and static Markdown docs generation.
The current verification gate also includes practical example parse/plan smoke,
deterministic conformance, and local release-binary smoke checks. The MVP
source default is `0.1.0`, and release builds can stamp version, commit, and
build date metadata into the CLI.
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
`docs/standard-visualization-guide.md` now documents Marquez/OpenLineage,
Jaeger/OTLP, Tempo/Grafana, and OpenMetadata-through-OpenLineage ingestion
recipes without adding a custom fbt graph UI or backend service.
OpenMetadata catalog integration has been evaluated: fbt core should not add a
direct `export openmetadata` command, and any OpenMetadata-specific publisher
belongs outside core.
`make real-llm-smoke` is available as an opt-in external runner smoke gated by
`FBT_REAL_LLM_RUNNER_COMMAND`; it is intentionally outside `make verify`.
External runner extensibility has a dedicated backlog under `FBT-RUNNER-*`.
The backlog keeps provider SDKs and agent runtimes outside fbt core while
hardening the protocol payload, process invocation, capability validation,
safe CLI-agent adapter contract, runner authoring fixtures, demo-runner UX, and
optional provider adapter packaging.
`FBT-RUNNER-001` is complete: build now sends protocol runners resolved source
inputs, current artifact-version inputs, descriptors, semantic descriptors,
declared transform assets, runner config metadata, prior/current state, plan
dirty reasons, and review context.
`FBT-RUNNER-002` is complete: runner config and plugin manifests now support
`args` and optional `cwd`, runner startup passes configured args/cwd plus a
filtered environment, and runner diagnostics report missing declared env names
without printing values.
`FBT-RUNNER-003` is complete: runner `initialize` capabilities are validated
for protocol version, transform types, output artifact types, and output
candidate support in build, doctor, and runner validate paths.
`FBT-RUNNER-004` is complete: external CLI-agent runners now have a documented
adapter contract requiring staging workspaces, fail-closed policy translation,
redacted events, and output candidates under `work.outputs`; Go and conformance
coverage verify outside-work candidates fail before official commit.
`FBT-RUNNER-005` is complete: external runner authors now have
`docs/runner-authoring-guide.md`, protocol fixtures under
`tests/runner-conformance/fixtures`, and `make runner-conformance`, which runs
a strict black-box stdio protocol check against the source fake runner inside
`make verify`.
`FBT-RUNNER-006` is complete: generated support/incident projects and the
checked-in knowledge example now use `demo.llm`, `demo.agent`, and
`bin/fbt-demo-*-runner`, CLI init prints a demo-runner replacement hint, and
the docs describe the shortest path from demo wrappers to external runner
commands.
`FBT-RUNNER-007` is complete: optional provider and CLI-agent adapter package
conventions are documented in `docs/runner-adapters.md`, including package
names, project config, plugin manifests, PATH behavior, credential boundaries,
versioning, and conformance checks while keeping SDKs and runtimes outside
core.
`FBT-EXAMPLE-001` is complete: practical external-runner examples now cover
incident logs to an approved incident response runbook and support
inquiry/response logs to an approved support resolution manual. Both examples
include realistic source records, format assets, policies, deterministic
section evals, review gates, and parse/plan smoke coverage without provider
calls.
`FBT-RUNNER-008` is complete: `runners/openai` now provides an optional
out-of-core OpenAI Responses runner that reads `OPENAI_API_KEY`, calls
`/v1/responses`, writes output candidates under `work.outputs`, and is wired
into the practical examples through project-local wrappers.
The CLI command surface is now closed around implemented commands; `run` and
`debug` placeholders were removed from help and user docs.
The conformance gate now covers schema failures, clean reruns, docs/export
redaction, standard export determinism, and dirty propagation in addition to the
support/review/policy loop.
Build now persists policy decision records for allowed and denied commit checks;
generated docs and OTel export reference those policy decision IDs.
Artifact versions now include first-pass semantic descriptors for normalized
text and Markdown heading/code-block structure, surfaced through `artifact show`
and generated docs while raw descriptors remain artifact version identity.
`fbt doctor` now checks project readiness, state writability/lock acquisition,
runner discovery, and runner protocol initialization. YAML authoring diagnostics
now include line numbers where available and actionable hints for common parse
errors.

The first implementation baseline now pins schema/versioning, artifact type
registry, runner discovery, plugin manifest semantics, security model, and MVP
conformance scenarios.

The practical local MVP is complete. The external runner hardening backlog is
complete. Remaining tracked work is maintainer-owned release publication.
`FBT-REL-002` is blocked on maintainer release credentials and signing setup:
no Git remote, signing config, or `v0.1.0` tag is present locally.
`FBT-REL-003` is blocked until that signed release baseline exists.

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
- `make practical-examples-smoke`
- `make runner-conformance`
- `make conformance`
- `make dist-check`

## Next Steps

1. Keep base runtime free of provider SDKs and heavyweight agent dependencies.
2. Complete maintainer-owned `FBT-REL-002` and `FBT-REL-003` when release
   credentials, signing setup, and release baseline are available.
3. Keep OpenMetadata integration on the OpenLineage ingestion path unless a
   future optional publisher is explicitly requested outside core.
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
