# Agent Progress

Last updated: 2026-05-29

## Current State

The repository contains the current MVP for `fbt`: a local-first file build
tool that parses a filesystem project, plans changed transforms, calls external
runners, commits versioned artifacts, runs deterministic evals, records local
state, explains artifact lineage, and exports standard
OpenLineage / OTLP JSON metadata.

The core intentionally does not implement document conversion, OCR, LLM
providers, agent runtimes, scheduling, publishing, or human approval workflow.
The `review` command, approval state, review gates, `human_review` evals, and
approval facets have been removed from core. Human approval belongs in Git,
PRs, CI, release tooling, ticket systems, or knowledge-base publishing flows.

The primary command surface is now centered on:

- `fbt init`
- `fbt doctor`
- `fbt plan`
- `fbt build`
- `fbt artifact`
- `fbt diff`
- `fbt export openlineage`
- `fbt export otel`

The CLI command tree and flag handling are implemented with Cobra. Default
human output is intentionally not the JSON/state shape: `plan`, `build`, and
`artifact` lead with short names, aligned status labels, summary counts, paths,
confidence, and next commands. Full resource IDs remain available in `Details`
sections and in `--json` output for automation.

Human status output uses fixed-width key/value rows and `text/tabwriter` tables
for dependencies and outputs. Glamour was considered for terminal Markdown
rendering, but it is not part of the default status renderer because the current
problem is structured row alignment rather than Markdown rendering.

The public CLI no longer exposes `parse`, `eval`, `docs`, `state`, or `runner`
subcommands. `doctor` handles readiness diagnostics, `plan` previews without
writes, and `build` handles runner execution, evals, state writes, and artifact
receipts. CLI argument handling is strict: unknown flags, extra arguments, and
selectors that match no transforms fail instead of being ignored. Graph
selectors support `+target`, `target+`, and `+target+` around normal selector
expressions. `plan --force` previews deliberate rebuilds with `reason: forced
rebuild`; `build --force` runs selected clean transforms without bypassing
upstream, confidence, policy, or output-boundary checks.

Docs and examples are aligned with the simpler model:

```text
source files + instructions + external runner
  -> generated artifact
  -> versioned build receipt, evals, lineage, and standard exports
```

`build` is intentionally the execution verb: fbt treats generated files as
build outputs, while external runners own the actual transformation logic.
`plan` is read-only; `build` writes artifact versions and local receipts.

The single-purpose boundary is explicit in README, specs, and docs site: fbt
composes with dbt, DataChain, DVC, Snakemake, remark, Pandoc, schedulers,
provider SDKs, artifact stores, and catalogs, but does not replace them.

A product audit against that boundary is now captured in the structured
backlog. The critical gaps are not review, scheduling, provider SDKs, or custom
visualization; those remain outside core. The new high-priority tasks focus on
build-tool reliability: one-invocation dependency-ordered builds
(`FBT-BUILD-001`, now done), failed-run receipts (`FBT-BUILD-002`, now done),
inert config cleanup (`FBT-CONFIG-001`, now done), strict YAML diagnostics
(`FBT-CONFIG-002`, now done), CLI-agent adapter safety (`FBT-RUNNER-010`), and
stale current-state docs cleanup (`FBT-DOCS-DRIFT-001`).

`FBT-BUILD-001` changed planning/build execution for selected graphs. Selected
transforms are ordered by artifact dependencies. A downstream selected transform
is no longer blocked merely because its selected upstream artifact does not
exist yet; the upstream run intent propagates dirty state, build commits the
upstream first, then rechecks the downstream against current state before
running it. Real blockers such as unselected missing upstreams or unsatisfied
confidence requirements still block.

`FBT-BUILD-002` made failed builds inspectable. Once `build` appends
`invocation_started`, every exit path appends `invocation_completed`; failed
transform attempts append `transform_run` receipts with safe error kind/message
for runner setup, capability, protocol, output-contract, policy, eval, and
cancellation failures. Failed receipts do not advance artifact versions or
current artifact pointers, and OTel exports include failed spans plus
`exception` events.

`FBT-CONFIG-001` reserved no-op config controls instead of pretending they work.
`execution.max_workers`, `execution.fail_fast`, `defaults.cache`,
`defaults.confidence`, and transform-level `cache` now fail with
`CONFIG_FIELD_RESERVED`, line/resource diagnostics, and a hint. Examples and
the project-config spec no longer advertise hidden cache/default/parallel
controls.

`FBT-CONFIG-002` made project YAML strict. Unknown fields in `fs_project.yml`
and resource files now fail with `YAML_FIELD_UNKNOWN`, including file, line,
resource name when available, and a hint. Conformance covers misspelled
top-level, runner, source, transform, policy, and eval fields. Documented
project aliases and free-form `meta`, `contract`, runner `config`, policy
`tools`/`limits`, eval `config`, and model parameter objects remain allowed.

`fbt artifact explain` is the primary single-artifact reasoning surface. It
prints the decision, current version, previous run, dependency fingerprints,
upstream artifact state, dirty or blocked reasons, and next command.

Repeated source growth uses stable source paths. External ingestion prepares
new-items-only, cumulative, or partitioned windows; fbt fingerprints the
resolved file set and content and rebuilds dependent artifacts when it changes.

`examples/markdown_toolchain` demonstrates `type: command` transforms that
wrap remark-style and Pandoc-style document tools through the external command
runner. fbt records the resulting artifact versions and lineage; document
processing remains outside core.

`examples/data_tool_interop` demonstrates dbt/DataChain interoperability by
treating dbt run artifacts and DataChain job outputs as fbt sources for a
versioned Markdown brief. Warehouse transformations and dataset materialization
remain outside core.

`examples/runner_adapter_scaffold` provides a dependency-free Python stdio
JSON-RPC runner skeleton plus `fbt_plugin.yml`. `make verify` now includes a
strict conformance check for that scaffold.

Graph selection now supports `+target`, `target+`, and `+target+` for
upstream, downstream, and bidirectional transform expansion. Both `plan` and
`build` share the same graph selection path.

Explicit rebuild control is intentionally one bit: `--force` on `plan` and
`build`. There is no general cache engine, cache invalidation subcommand, or
full-refresh concept in core.

Semantic and LLM-judge evals remain outside core. MVP core runs deterministic
evals, records `semantic` and `llm_judge` eval declarations as skipped, and
grants no confidence from them. Model-based judging should be implemented as an
external runner transform that produces a normal judge report artifact.

Standard visualization is documented as command-first backend integration.
`examples/standard_visualization` shows how to create OpenLineage and OTLP/JSON
exports from the offline template and post them to Marquez or an OTLP HTTP
endpoint. Docs should use screenshots captured from real standard backends, not
custom fbt diagrams.

Specs and active plans have been cleaned up so current-state docs use
artifact inspection, confidence/upstream blocking, docs-site build, and
OpenLineage/OTel export language. Remaining review/approval command references
are explicit outside-core or superseded historical notes.

The checked-in examples cover:

- `examples/knowledge_ops`: offline support knowledge-loop fixture using demo
  runners.
- `examples/daily_qa_ops`: daily source-growth workflow with stable inbox
  directories and multiple outputs.
- `examples/data_tool_interop`: dbt/DataChain output files to a versioned
  operational brief through a command transform.
- `examples/runner_adapter_scaffold`: minimal external runner adapter skeleton
  with strict conformance coverage.
- `examples/semantic_eval_boundary`: pattern doc for deterministic core evals
  versus external model-judge report artifacts.
- `examples/standard_visualization`: standard OpenLineage and OTel export
  ingestion recipes for Marquez, Jaeger, Tempo, Grafana, or OpenMetadata paths.
- `examples/incident_response_runbook`: optional OpenAI runner flow for turning
  incident evidence into a runbook.
- `examples/support_resolution_manual`: optional OpenAI runner flow for turning
  support evidence into a manual.

External runner extensibility remains out-of-core. `runners/openai` is optional
and reads `OPENAI_API_KEY`; provider SDKs and agent runtimes are not part of
base core. Runner authoring, adapter packaging, protocol fixtures, and
conformance checks are documented under `docs/runner-authoring-guide.md`,
`docs/runner-adapters.md`, and `tests/runner-conformance/`.

## Verification

Required gate before calling work done:

```sh
make verify
```

This runs harness, drift, docs validation, Go formatting/tests, CLI smoke,
knowledge-loop smoke, practical examples smoke, docs site build, runner
conformance, product conformance, and distribution smoke checks.

## Next Steps

1. Harden external-agent usage through `FBT-RUNNER-010` while keeping provider
   SDKs and agent runtimes outside base core.
2. Define local state/artifact retention hygiene through `FBT-STATE-002`.
4. Keep approval, publishing, scheduling, catalog-specific ingestion, and custom
   visualization outside core unless implemented as external tooling.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
