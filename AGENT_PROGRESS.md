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
(`FBT-CONFIG-002`, now done), CLI-agent adapter safety (`FBT-RUNNER-010`, now
done), and stale current-state docs cleanup (`FBT-DOCS-DRIFT-001`, now done).

The latest full review added a new post-MVP hardening backlog. Real CLI-agent
policy enforcement (`FBT-RUNNER-022`) and explicit large JSON-RPC JSONL frame
handling (`FBT-RUNNER-023`) are done. Visible CLI-agent staged-input truncation
failures (`FBT-RUNNER-024`) and bounded stderr/exit diagnostics for runner
protocol failures (`FBT-RUNNER-025`) are also done.

The latest documentation review found that README is stronger than the docs
site in concrete source-to-artifact evidence, and that the public OG image still
contains stale `review gates` language. The new docs backlog is
`FBT-DOCS-DRIFT-002` for stale public asset language, `FBT-DOCS-UX-012` for
manual-generation example concreteness, and `FBT-DOCS-UX-013` for runnable
runner/standards/reference docs examples.

`FBT-DOCS-DRIFT-002` is done. The public OG image no longer says `review
gates`, and `scripts/harness_drift.py` now scans public docs assets for stale
current-state review/approval phrases.

`FBT-DOCS-UX-012` is done. The manual-generation docs, practical example
READMEs, and reference example guide now show concrete source records,
response/product evidence, prompt and format assets, runner wiring, output
paths, expected artifact excerpts, and `artifact explain` receipt excerpts for
both incident runbook and support manual examples.

`FBT-DOCS-UX-013` is done. Runner, OpenAI adapter, runner authoring, lineage,
standard export, visualization, and project-config docs now include runnable
commands, minimal YAML snippets, expected conformance/doctor/export output,
and concrete example file pointers.

The latest product/UX review added six follow-up tasks without expanding fbt's
core scope:

- `FBT-UX-015`: make the first own-files success path self-service.
- `FBT-RUNNER-026`: harden official adapter install and live verification UX.
- `FBT-OPS-001`: document daily high-volume source operations patterns.
- `FBT-EVAL-001`: add external semantic/evidence-quality check examples.
- `FBT-REL-004`: simplify end-user install path for core and adapters.
- `FBT-STATE-003`: validate retention guidance with a high-volume fixture.

`FBT-UX-015` is done. README, docs-site quickstart/manual-generation pages,
examples index, and `docs/examples/first-own-files-success-path.md` now show a
copy-paste path for replacing template source files and instructions with a
user's own files, proving the loop with demo runners, inspecting receipts, and
then switching to an external runner. `make verify` includes
`own-files-smoke`.

`FBT-POLICY-001` is done. Directory artifact descriptors now record aggregate
regular-file byte size, so `limits.max_output_bytes` applies to
`directory`/`markdown_directory` artifacts as well as file artifacts. Build
regressions cover denied oversized directory outputs and verify they do not
advance current artifact pointers.

`FBT-RUNNER-021` is done. Official adapter modules no longer carry local
`replace ../../sdk/go` directives. They depend on a normal VCS-resolved
`sdk/go` module version, keep local development convenience through `go.work`,
and `make adapter-install-smoke` verifies clean `go install module@commit`
installation for command, OpenAI, Codex CLI, and Claude Code adapters through a
temporary bare VCS remote.

`FBT-RUNNER-022` is done. Codex CLI and Claude Code adapters now treat policy
mapping as executable behavior. Codex CLI runs with a read-only sandbox and
timeout mapping, then fails before invoking Codex for unsupported network,
tool, cost, or tool-call policies. Claude Code maps tool allow/deny lists,
timeout, and max budget where supported, then fails before invoking Claude for
unsupported network, unknown tool, or tool-call policies. Adapter conformance
now runs both positive safe-policy paths and negative unsupported-policy paths.

`FBT-RUNNER-023` is done. Core protocol reads and the Go SDK stdio server now
set an explicit 16 MiB JSONL frame limit instead of inheriting Go scanner's
default token limit. Core and SDK tests cover messages above the old default,
and the protocol spec documents that raw documents should stay in files rather
than JSON-RPC frames.

`FBT-RUNNER-024` is done. Codex CLI and Claude Code adapters no longer stage
source or asset files through a truncating `LimitReader`. Files within the 2
MiB per-file staging limit are copied completely; oversized source or asset
files return an actionable error naming the file and limit before any external
CLI process is invoked.

`FBT-RUNNER-025` is done. The protocol client captures bounded runner stderr
and process exit status for startup and protocol-call failures. Values from
project-configured runner env names are redacted before the diagnostic reaches
CLI output or failed build receipts. CLI errors now include a runner setup hint
that points users back to `fbt doctor`.

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

`FBT-RUNNER-010` hardened the recommended CLI-agent adapter boundary. Runner
conformance now has an opt-in `--agent-adapter` profile that injects a secret
marker, guards the source path, logical artifact path, and `.fbt/state`, and
requires redacted protocol messages, a staging workspace under `work.root` but
outside `work.outputs`, and fail-closed policy mapping. The copyable runner
adapter scaffold emits those markers and `make verify` runs it through the
strict agent-adapter profile.

`FBT-DOCS-DRIFT-001` removed stale current-state docs that made fbt look broader
than its core. The runner protocol no longer says core owns approval state or a
docs-generation surface, `internal/README.md` lists the current public CLI and
actual internal packages, and `scripts/harness_drift.py` rejects those exact
stale phrases if they reappear outside historical notes.

`FBT-RUNNER-009` added an opt-in installed-adapter smoke matrix. `make
runner-adapter-smoke` reads `FBT_RUNNER_ADAPTER_SMOKE_MATRIX` rows in the form
`logical_name|runner_type|artifact_type|command|required_env_csv|agent_adapter`,
then runs conformance, a generated-project `doctor`, and a generated-project
`plan` for each row. Setting `FBT_RUNNER_ADAPTER_SMOKE_BUILD=1` also performs a
real temporary build and artifact inspection. The target is intentionally not
part of `make verify`.

`FBT-STATE-002` defined local retention hygiene without adding a destructive
cleanup surface. MVP policy is `keep_all`. `fbt artifact retention` is a
read-only report for state bytes, immutable artifact bytes, run records,
current versions, historical versions, and missing immutable storage. High
volume projects should archive `.fbt/state/` and `.fbt/artifacts/` together
with external tools; fbt core does not prune history in MVP.

`FBT-STD-007` added opt-in standard backend verification. `make
standard-backend-smoke` generates support-template OpenLineage and OTLP/JSON
exports, validates them locally, and posts to Marquez or an OTLP HTTP endpoint
only when `FBT_MARQUEZ_URL`, `FBT_OTLP_TRACES_URL`, or
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` is set. `FBT_STANDARD_EVIDENCE_DIR` copies
exports and a smoke summary for release/docs evidence. The target is outside
`make verify`, and core still exposes only `fbt export openlineage` and
`fbt export otel`.

A follow-up CLI UX audit is now captured as structured tasks. Help text and
flag-scope clarity (`FBT-UX-009`) is done: root, artifact, export, plan, build,
and retention help now explain what each command returns, and help regression
tests cover the key wording. Actionable common errors (`FBT-UX-010`) is also
done: declared-but-unbuilt artifacts, empty selectors, and `--dry-run` produce
strict errors plus short `Hint:` lines. Copy-paste-safe next commands
(`FBT-UX-011`) is done: `plan`, `build`, and `artifact explain` preserve
`--project-dir` and `--state-dir` in printed next steps. Showing build output
paths (`FBT-UX-012`) is done: successful builds now print each committed
artifact path, version, and contextual `artifact show` command. Scannable
artifact/retention output (`FBT-UX-013`) is done: human output summarizes
semantic descriptors and reports retention sizes with human-readable units
while JSON remains complete. State/artifact directory semantics
(`FBT-CONFIG-003`) is done: only `state.backend: local` is accepted,
`state.path` must remain project-relative, and docs clarify that `--state-dir`
affects receipts/state only, not `.fbt/artifacts` or `artifact_path`. Runner
terminology (`FBT-RUNNER-011`) is done: user-facing docs now describe every
integration as an external command that speaks the fbt runner protocol, while
adapter/protocol/conformance language remains for authors. Standard export
command UX (`FBT-STD-008`) is done: `--output` summaries now name the standard
format, output path, record count, and backend handoff while stdout remains raw
for piping. Grouped doctor diagnostics (`FBT-UX-014`) is done: human `doctor`
output now separates Project, State, and Runners with per-runner nesting while
`--json` keeps the flat diagnostics array for automation.

README first-user journey (`FBT-DOCS-UX-009`) is done: the README now leads
with fbt's value, maps user questions to commands, shows project anatomy, runs
an offline success loop before the real runner example, and routes deeper
reading by goal while staying within the compact README guard.

README concrete mapping (`FBT-DOCS-UX-010`) is done: the offline support loop
and real incident runbook example now name the actual source files, instruction
assets, transform recipes, runner commands, artifact paths, checks, and receipt
locations so users can see what source/instruction/runner produces which
artifact.

README input/output snippets (`FBT-DOCS-UX-011`) is done: examples now show the
actual support ticket record and generated Markdown excerpt, plus incident
evidence excerpts and the expected procedure-style runbook shape.

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
strict CLI-agent adapter conformance check for that scaffold.

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
artifact inspection, confidence/upstream blocking, docs-site build,
OpenLineage/OTel export language, and the current public CLI surface. Remaining
review/approval command references are explicit outside-core or superseded
historical notes.

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

External runner extensibility remains out-of-core. Source-checkout adapter
examples live under `examples/runner_adapters/`, test-only protocol fixtures
live under `tests/runner_fixtures/`, and there is no top-level `runners/`
directory. The optional OpenAI adapter reads `OPENAI_API_KEY`; provider SDKs
and agent runtimes are not part of base core. Runner authoring, adapter
packaging, protocol fixtures, and conformance checks are documented under
`docs/runner-authoring-guide.md`, `docs/runner-adapters.md`, and
`tests/runner-conformance/`. Installed adapter checks use
`make runner-adapter-smoke` with explicit matrix rows and optional real builds.

Official runner package design research is captured in
`docs/research/official-runner-adapter-design-report.md` (`FBT-RUNNER-013`).
The agreed repository strategy is monorepo nested modules (`FBT-RUNNER-014`):
root `go.mod` remains fbt core only, provider-free SDK code starts under
`sdk/go`, official adapters start under `adapters/<name>`, and `go.work` may be
used for local development across modules. Future `sdk/python` or
`sdk/typescript` modules remain possible because the protocol is JSON-RPC over
stdio; the protocol spec and conformance suite remain the source of truth.

The structured backlog now includes the official adapter implementation path:
`FBT-RUNNER-015` for `sdk/go`, `FBT-RUNNER-016` for the command adapter,
`FBT-RUNNER-017` for the OpenAI adapter, and `FBT-RUNNER-018` for Codex CLI and
Claude Code adapters. Provider SDKs and agent runtimes must stay out of fbt
core while those tasks are implemented.

`FBT-RUNNER-015` is done. `sdk/go` is a provider-free nested Go module with
runner protocol types, JSONL stdio JSON-RPC server helpers, output-candidate
helpers, and redaction helpers. `go.work` includes the root module and
`sdk/go`, and `make verify` runs `sdk-go-test` in addition to the root Go
tests.

`FBT-RUNNER-016` is done. The command runner now lives under
`adapters/command` as an official nested module with its own `go.mod`,
`fbt_plugin.yml`, README, SDK-based protocol handling, tests, and conformance
target. `examples/markdown_toolchain` and `examples/data_tool_interop` wrappers
now execute `go run ./adapters/command/cmd/fbt-runner-command`.

`FBT-RUNNER-017` is done. The OpenAI Responses runner now lives under
`adapters/openai` as an official nested module with its own `go.mod`,
`fbt_plugin.yml`, README, SDK-based protocol handling, httptest coverage, and
network-free conformance through `FBT_OPENAI_ADAPTER_FAKE_RESPONSE`.
`examples/incident_response_runbook` and `examples/support_resolution_manual`
wrappers now execute `go run ./adapters/openai/cmd/fbt-runner-openai`.

`FBT-RUNNER-018` is done. `adapters/codex-cli` and
`adapters/claude-code` are official nested modules that wrap `codex exec` and
`claude -p` through staging workspaces, copy final content into `work.outputs`,
and report fail-closed policy markers for `--agent-adapter` conformance. Their
`testdata/*-fixture.sh` scripts are protocol test fixtures only, used to keep
`make verify` deterministic without network, credentials, or paid model calls.

`FBT-RUNNER-019` is done after live testing. Fixture model names are no longer
forwarded to real CLI agents, and Codex/Claude adapter errors include bounded
CLI output for actionable live diagnostics. Claude Code prompt handling no
longer uses variadic flags that consume the prompt. Live Codex conformance
passes with the installed CLI and saved authentication; live Claude Code now
reaches authentication and reports that the local CLI is not logged in.

`FBT-RUNNER-020` is done after OpenAI live testing. The OpenAI adapter no
longer forwards conformance fixture model names to the real Responses API; it
falls back to `FBT_OPENAI_DEFAULT_MODEL` or `gpt-5`. Live OpenAI adapter
conformance passed with a real API key, and a temporary copy of
`examples/incident_response_runbook` completed `doctor -> plan -> build ->
artifact show` against the real OpenAI adapter, committing
`incident_response_runbook@sha256:4bda5ab434cb`.

## Verification

Required gate before calling work done:

```sh
make verify
```

This runs harness, drift, docs validation, Go formatting/tests, CLI smoke,
knowledge-loop smoke, practical examples smoke, docs site build, runner
conformance, product conformance, and distribution smoke checks.

## Next Steps

1. Keep approval, publishing, scheduling, catalog-specific ingestion, and custom
   visualization outside core unless implemented as external tooling.

## Notes For Next Agent

- Do not rely on chat history for product decisions. Update repository docs.
- Keep `AGENTS.md` compact.
- If `make verify` fails, prefer a deterministic guard or spec update over a
  one-off fix.
