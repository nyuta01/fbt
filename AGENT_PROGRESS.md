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

The public CLI no longer exposes `parse`, `eval`, `docs`, `state`, or `runner`
subcommands. `doctor` handles readiness diagnostics, `plan` previews without
writes, and `build` handles runner execution, evals, state writes, and artifact
receipts. CLI argument handling is strict: unknown flags, extra arguments, and
selectors that match no transforms fail instead of being ignored.

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

`fbt artifact explain` is the primary single-artifact reasoning surface. It
prints the decision, current version, previous run, dependency fingerprints,
upstream artifact state, dirty or blocked reasons, and next command.

The checked-in examples cover:

- `examples/knowledge_ops`: offline support knowledge-loop fixture using demo
  runners.
- `examples/daily_qa_ops`: daily source-growth workflow with stable inbox
  directories and multiple outputs.
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

1. Run `FBT-UNIX-003` to define stable source-window operations.
2. Run `FBT-UNIX-004` to add existing-tool runner examples for remark and
   Pandoc.
3. Run `FBT-UNIX-014` to remove stale docs/review/approval references from
   source-of-truth specs.
4. Keep base runtime free of provider SDKs and heavyweight agent dependencies.
5. Keep approval, publishing, scheduling, and catalog-specific ingestion outside
   core unless implemented as external tooling.
6. Improve source-window ergonomics and artifact explanations without turning
   fbt into a scheduler or transform engine.
7. Keep graph, trace, and catalog visualization on standard-compatible exports
   rather than a custom fbt backend.
8. Add CLI surface only when backed by a spec and verification.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
