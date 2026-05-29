# fbt Security and Conformance Spec

Status: Draft  
Created: 2026-05-28  
Audience: implementers of policy enforcement, runner invocation, state safety, and acceptance tests

## 1. Overview

`fbt` manages filesystem transformations that may involve LLMs and AI agents.
The base security model is local-first and assumes external runners are trusted
executables selected by the user or project. Core still owns the commit boundary
and must prevent unsafe runner output from becoming official project state.

This spec defines MVP security responsibilities and conformance scenarios.

## 2. Trust Boundary

Core trusts:

- project files selected by the user
- runner executables configured by the user or project
- local filesystem APIs

Core does not trust:

- runner stdout
- runner stderr
- output candidate paths
- tool-call event contents
- model responses
- generated documents
- document metadata

Core validates structured protocol messages before using them and treats all
runner-provided paths as untrusted until normalized and scope-checked.

## 3. Core-Enforced Guarantees

MVP core must enforce:

- source files are read-only from the perspective of official `fbt` commits
- official artifact pointers update only through the core commit path
- output candidates must live under the invocation work directory
- logical artifact paths must live under `artifact_path`
- failed, cancelled, interrupted, or denied runs do not update official pointers
- `artifact_version` records are immutable
- state files are written with temp-file plus atomic rename
- one local build lock is held per project target
- secrets and raw model responses are not stored by default
- timeout, output-size, and declared cost limits are checked when data is available

Core cannot fully sandbox arbitrary external processes in MVP. OS-level
sandboxing is a post-MVP hardening layer.

## 4. Runner Responsibilities

Runners must:

- respect read and write scope
- respect network, tool, timeout, and cost policy
- write output candidates only under the assigned work directory
- return structured errors for policy denials and execution failures
- redact credentials from structured events
- avoid printing secrets to stdout
- provide usage and cost metadata when available

A runner that cannot satisfy policy must fail closed.

## 5. CLI Agent Adapter Boundary

CLI-agent adapters wrap external tools such as Codex CLI, Claude Code, Gemini
CLI, provider SDK agents, or internal agent launchers. The adapter is the fbt
runner process and owns translation between fbt policy and the external tool's
permission model.

Safe adapters must:

- run the agent in a staging workspace, scoped working tree, or isolated copy
- expose only declared inputs/assets plus policy-approved extra read paths to
  the agent
- map network, shell/tool, timeout, turn, and cost limits to the external
  tool's controls where available
- refuse execution when the requested policy cannot be enforced or represented
- place final candidate files under the assigned `work.outputs` directory
- emit only redacted structured events and declared output candidates

Adapters must not let the external agent write directly to logical artifact
paths, immutable artifact storage, `.fbt/state`, or source paths as part of the
normal fbt build path. Even if an adapter makes a mistake, core treats all
runner paths as untrusted and rejects candidates outside `work.outputs` before
descriptor computation or commit.

## 6. Default Policy

When a transform omits policy:

- read scope is limited to declared inputs and transform assets
- write scope is limited to the invocation work directory
- source writes are denied
- network is denied, except when a runner type explicitly requires network and
  project policy allows it
- shell tools are denied for `llm` and `agent` transforms
- timeout and output-size defaults are applied by core

Agent transforms should declare an explicit policy. Missing policy for an agent
transform is a parse warning in draft mode and should become a parse error before
stable v1.

## 7. Path Rules

Core path validation must:

- resolve relative paths from the project directory
- normalize paths before comparison
- reject absolute output paths unless they are the assigned work directory
- reject `..` escapes
- reject symlink escapes by default
- reject output candidates outside `.fbt/work/<invocation>/outputs`
- reject logical artifact paths outside `artifact_path`

Path validation must run before descriptor computation and before any commit.

## 8. Confidence and Downstream Blocking

Downstream transforms may require a current upstream artifact and a minimum fbt
confidence class such as `structural` or `semantic`. Human approval and release
state are intentionally outside core.

## 9. Conformance Scenarios

The MVP conformance suite in `tests/conformance/run.sh` runs deterministic
local scenarios without external services. It should include the following
scenario classes.

| ID | Area | Scenario | Expected Result |
|---|---|---|---|
| `CONF-SCHEMA-001` | schema | `fs_project.yml` omits `config_version` | project parsing fails with exit code `2` |
| `CONF-SCHEMA-002` | schema | `config_version` is unsupported | project parsing fails with exit code `2` |
| `CONF-DESC-001` | descriptor | same directory content with different mtimes | identical directory digest |
| `CONF-DESC-002` | descriptor | directory contains symlink escape | descriptor computation fails |
| `CONF-RUNNER-001` | runner | transform references missing runner | build/plan fails with exit code `6` |
| `CONF-RUNNER-002` | runner | runner lacks required capability | validation fails before commit |
| `CONF-SEC-001` | path | runner declares output outside work directory | output is denied before descriptor computation; no pointer update |
| `CONF-SEC-002` | policy | transform requires network but policy denies it | transform is blocked with exit code `3` |
| `CONF-STATE-001` | state | runner fails after writing partial output | official pointer remains unchanged |
| `CONF-STATE-002` | state | same digest is committed twice | commit is idempotent |
| `CONF-REDACT-001` | redaction | runner reports env var names and values | values are not persisted |
| `CONF-DIRTY-001` | planning | prompt, policy, or runner config changes | dependent transform is dirty |
| `CONF-DOCS-001` | docs | docs are generated | lineage is shown; secret values are absent |
| `CONF-STD-001` | standard export | OpenLineage export is generated from a support loop | events contain jobs, runs, datasets, fbt facets, schema URL, and UUID-shaped run IDs |
| `CONF-STD-002` | standard export | OTel export is generated from run results | OTLP/JSON contains resource spans, invocation/transform spans, GenAI usage attributes, and runner span events |
| `CONF-STD-003` | redaction | standard exports run on inputs/assets containing a marker secret | exported OpenLineage and OTel payloads do not contain raw source content or the marker secret |
| `CONF-ADAPTER-001` | runner adapter | CLI-agent adapter reports staging workspace and fail-closed policy mapping | strict runner conformance passes only when the staging workspace is under `work.root`, outside `work.outputs`, and policy mapping is fail-closed |
| `CONF-ADAPTER-002` | runner adapter | CLI-agent adapter receives guarded source, logical artifact, and `.fbt/state` files | strict runner conformance fails if the adapter modifies any guarded file directly |
| `CONF-ADAPTER-003` | redaction | CLI-agent adapter receives source and asset files containing a marker secret | strict runner conformance fails if protocol responses or events leak the marker |

## 10. Verification Target

Once product implementation begins, `make verify` should grow a deterministic
conformance target that runs these scenarios without external services.

Current executable coverage:

- schema errors for missing and unsupported `config_version`
- local support template build and artifact commit
- clean rerun skips unchanged artifact work
- incompatible runner capability validation fails before output commit
- runner-declared output candidates outside `work.outputs` fail before output
  commit
- downstream build succeeds after the upstream artifact exists
- docs-site build succeeds after the build loop
- docs-site output does not include the redaction marker
- policy-denied output is not committed to the official artifact path
- prompt/asset changes make dependent transforms dirty again
- OpenLineage export contains standard event keys and fbt facets without raw
  source content
- OTel export contains OTLP/JSON resource spans, transform attributes, usage
  attributes, and runner span events without raw source content
- runner conformance checks candidate containment, redaction, and direct-write
  guards for all runners
- scaffold runner conformance uses the CLI-agent adapter safety profile to
  require staging-workspace and fail-closed policy markers

LLM and agent scenarios should use fake runners for conformance. Real provider
smoke tests belong behind explicit opt-in commands and must not be required for
the base verification gate.
