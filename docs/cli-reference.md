# fbt CLI Reference

Status: MVP-ready  
Created: 2026-05-28  
Audience: users and implementers of the `fbt` command-line interface

## 1. Overview

`fbt` is designed as a local-first single-binary CLI. The base runtime does not require a daemon, scheduler, metadata database, web server, or cloud account.

Basic workflow:

```sh
fbt init
fbt parse
fbt plan
fbt doctor
fbt build
fbt review status
fbt review approve case_summaries
fbt docs generate
```

Release version contract:

- source builds default to `0.1.0`
- release builds may stamp `VERSION`, `COMMIT`, and `BUILD_DATE`
- `fbt version` prints only `fbt VERSION` for stable shell use
- `fbt version --json` includes `version`, `commit`, and `build_date`
- `make build VERSION=... COMMIT=... BUILD_DATE=...` and
  `scripts/dist-check.sh` use the same linker-stamped metadata contract

## 2. Global Flags

| Flag | Meaning |
|---|---|
| `--project-dir PATH` | Directory containing `fs_project.yml`; defaults to the current directory |
| `--state-dir PATH` | Local state directory; defaults to `.fbt/state` |
| `--select EXPR` | Select resources |
| `--json` | Machine-readable JSON output |

MVP does not accept `--exclude`, `--target`, `--vars`, `--dry-run`,
`--log-level`, `--no-color`, or `--quiet`.

## 3. Exit Codes

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Transform or eval failed |
| `2` | Invalid project, config, or parse error |
| `3` | Blocked by policy, review, or confidence requirement |
| `4` | Runner protocol error |
| `5` | State lock or state backend error |
| `6` | Missing dependency or runner not installed |
| `130` | Cancelled by user |

## 4. Selection Syntax

| Expression | Meaning |
|---|---|
| `case_summaries` | Resource with matching name |
| `+case_summaries` | Select the matching transform name; ancestor expansion is reserved |
| `case_summaries+` | Select the matching transform name; descendant expansion is reserved |
| `+case_summaries+` | Select the matching transform name; graph expansion is reserved |
| `tag:support` | Tag selector |
| `path:transforms/support/` | Path selector |
| `resource_type:transform` | Resource type selector |
| `selector:support_daily` | Named project selector |

Examples:

```sh
fbt plan --select tag:support
fbt build --select case_summaries
fbt build --select selector:support_daily
```

## 5. Commands

### 5.0 fbt version

Print the CLI version.

```sh
fbt version
fbt version --json
```

Human output is intentionally compact:

```text
fbt 0.1.0
```

JSON output includes release metadata stamped at build time:

```json
{
  "command": "version",
  "status": "success",
  "version": "0.1.0",
  "commit": "unknown",
  "build_date": "unknown"
}
```

### 5.1 fbt init

Create a new project.

```sh
fbt init [PROJECT_NAME]
```

Flags:

| Flag | Meaning |
|---|---|
| `--template NAME` | `blank`, `knowledge_ops`, `support`, or `incident` |
| `--force` | Allow overwriting existing files |

The `support` and `knowledge_ops` templates include local runner wrappers for
the bundled LLM and agent examples. They are suitable for local smoke runs from
a source checkout without provider credentials.

### 5.2 fbt parse

Parse project files and generate a manifest. Does not start runners.

```sh
fbt parse
```

Main steps:

- Read `fs_project.yml`
- Read source, transform, asset, policy, eval, and runner YAML
- Resolve `ref()` and `source()` dependencies
- Build the graph
- Validate the project
- Write `.fbt/state/manifest.json`

Parse errors exit with code `2`. Human output includes a stable diagnostic
code, message, file, line where available, and a `hint:` line for common YAML
authoring fixes. JSON output includes the diagnostic array.

### 5.3 fbt plan

Compare current manifest and state to show what will run.

```sh
fbt plan [--select EXPR]
```

Shows:

- Selected transforms
- Skipped transforms
- Dirty reasons
- Blocked reasons
- Confidence requirements
- Required review through blocked/pending state

Use `fbt artifact explain TARGET` to focus on one artifact's plan decision,
inputs, current version, previous run evidence, and next steps.

Example:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked

run transform.knowledge_ops.case_summaries
  reason: no previous successful run

blocked transform.knowledge_ops.weekly_support_insights
  blocked: requires artifact.knowledge_ops.case_summaries confidence reviewed
  next: fbt review status case_summaries
  next: fbt review approve case_summaries --comment "reviewed"
```

### 5.4 fbt build

Run the full lifecycle.

```sh
fbt build [--select EXPR]
```

Lifecycle:

```text
parse -> plan -> run -> eval -> review gate -> commit -> write state
```

Example:

```sh
fbt build --select tag:support
fbt build --select +weekly_support_insights
```

### 5.5 fbt eval

Run evals against an artifact or artifact version.

```sh
fbt eval TARGET [--select EVAL_EXPR]
```

Examples:

```sh
fbt eval case_summaries
fbt eval artifact_version.knowledge_ops.case_summaries.sha256_abcd
fbt eval weekly_support_insights --select no_unsupported_claims
```

MVP behavior:

- deterministic evals run in core against the selected artifact version
- semantic, LLM-judge, and human-review evals are recorded as `skipped` until
  delegated eval runners are implemented
- failed deterministic evals return exit code `1`

### 5.6 fbt diff

Show differences between artifact versions.

```sh
fbt diff TARGET [--against REF]
```

`--against` values:

- `previous`
- `last-approved`
- explicit `artifact_version...`

MVP supports raw text diff and Markdown heading-aware diff.

### 5.7 fbt review

Manage review and approval state.

```sh
fbt review status [TARGET]
fbt review show TARGET [--version VERSION_ID]
fbt review approve TARGET [--version VERSION_ID] [--comment TEXT]
fbt review reject TARGET [--version VERSION_ID] [--comment TEXT]
```

Approval is bound to `artifact_version`. If `TARGET` is a logical artifact, the current version is used.

MVP behavior: approving the current version writes an approval record and
promotes the current pointer to `approval_status: approved` and
`confidence: reviewed`. Rejecting the current version writes `rejected` and
keeps downstream reviewed/approved requirements blocked.

Use `fbt review show TARGET` before approval. It prints the selected artifact
version, logical and immutable storage paths, digest, runner/model metadata,
generating run, and inspection commands such as `artifact show`, `artifact
path`, `diff --against last-approved` when available, and approve/reject
commands to run after review.

### 5.8 fbt docs generate

Generate static docs.

```sh
fbt docs generate
```

Output:

```text
target/docs/
```

Docs include graph lineage, runner/model metadata, token/cost summary, review state, confidence, artifact versions, eval results, and diff links.

### 5.9 fbt state

Inspect local state.

```sh
fbt state status
fbt state ls
fbt state current TARGET
```

### 5.10 fbt artifact

Inspect artifacts and versions.

```sh
fbt artifact ls
fbt artifact path TARGET
fbt artifact show TARGET
fbt artifact explain TARGET
fbt artifact history TARGET
fbt artifact versions TARGET
```

`artifact path` prints the logical output path and immutable storage path for
the current or selected version. `artifact show` includes artifact version,
logical path, immutable storage path, digest, runner/model, approval state,
confidence, generating run, and semantic descriptors when available. `artifact
history` lists prior versions for the same logical artifact.

### 5.11 fbt runner

Inspect runner availability and capabilities.

```sh
fbt runner list
fbt runner doctor [RUNNER_NAME]
fbt runner validate RUNNER_NAME
```

Runner discovery order is project config, project-local plugin manifest,
user-local plugin manifest, then `PATH` convention. See
[Runner Discovery Spec](runner-discovery-spec.md).

MVP does not download or install plugins. `fbt plugin install` is reserved for a
future release; use host package managers or checked-in plugin manifests and then
run `fbt runner doctor`.

### 5.12 fbt doctor

Run a project readiness check.

```sh
fbt doctor
```

Checks:

- project config parsing
- state directory writability and lock acquisition
- runner discovery and executable availability
- runner protocol `initialize`

The command exits `0` when all checks pass and `6` when dependency or runner
readiness checks fail. Use `--json` for machine-readable check records.

### 5.13 Standard exports

`fbt export openlineage` writes OpenLineage-compatible run events for artifact
lineage. The default output is newline-delimited JSON on stdout. Use `--output`
to write a file that can be passed to OpenLineage-compatible tooling such as
Marquez.

```sh
fbt export openlineage [--output PATH]
```

The export maps fbt transforms to OpenLineage jobs, transform runs to runs,
source and input artifacts to input datasets, and generated artifacts to output
datasets. fbt-specific confidence, approval, eval, descriptor, runner/model,
and policy metadata are included as `fbt_` custom facets. Raw artifact content,
raw prompts, raw model responses, credentials, and absolute project paths are
not exported by default.

With `--json`, file-based export returns a command envelope with counts and the
output path:

```json
{
  "command": "export openlineage",
  "status": "success",
  "format": "openlineage",
  "events": 1,
  "output_path": "target/lineage/openlineage.ndjson"
}
```

`fbt export otel` writes OpenTelemetry OTLP/JSON-compatible trace payloads for
local execution telemetry. The default output is the OTLP JSON payload on
stdout. Use `--output` to write a file for an OpenTelemetry Collector or
compatible backend workflow.

```sh
fbt export otel [--output PATH]
```

The export maps invocations and transform runs to spans, runner protocol events
to span events, and usage/cost/model metadata to span attributes including
`gen_ai.*` attributes where available. It does not start a network exporter.
Tool-call payload fields are not exported by default; redacted runner event
attributes are exported as span events.

With `--json`, file-based export returns a command envelope with counts and the
output path:

```json
{
  "command": "export otel",
  "status": "success",
  "format": "otel",
  "spans": 2,
  "output_path": "target/telemetry/otel.json"
}
```

There is no base `fbt export openmetadata` command. OpenMetadata integration
uses `fbt export openlineage` plus an external OpenMetadata ingestion workflow,
or a future optional publisher outside fbt core.

The contract keeps fbt-native state as the source of truth and delegates
lineage, trace, and catalog visualization to standard-compatible tools. See
[Standard Export Spec](standard-export-spec.md),
[OpenMetadata Catalog Export Evaluation](research/openmetadata-catalog-export-evaluation.md),
and
[Standard Visualization Guide](standard-visualization-guide.md).

## 6. JSON Output

With `--json`, stdout contains machine-readable JSON and human logs go to stderr.

`fbt plan --json` returns the same plan shape used by the text output:

```json
{
  "command": "plan",
  "status": "success",
  "summary": {
    "selected": 2,
    "run": 1,
    "skipped": 1,
    "blocked": 1
  },
  "nodes": [
    {
      "transform_id": "transform.knowledge_ops.case_summaries",
      "name": "case_summaries",
      "action": "run",
      "dirty_reasons": ["source support.raw_tickets changed"]
    },
    {
      "transform_id": "transform.knowledge_ops.weekly_support_insights",
      "name": "weekly_support_insights",
      "action": "blocked",
      "blocked_reasons": [
        "requires artifact.knowledge_ops.case_summaries confidence reviewed"
      ],
      "next_steps": [
        "fbt review status case_summaries",
        "fbt review approve case_summaries --comment \"reviewed\""
      ]
    }
  ]
}
```

## 7. Error Output

Human-readable:

```text
Error: runner not installed: openai.responses

Install a runner that provides 'openai.responses', or update fs_project.yml.
```

JSON:

```json
{
  "command": "build",
  "status": "error",
  "error": "runner not installed: openai.responses"
}
```

## 8. Primary Commands

The main user-facing path stays small:

- `fbt parse`
- `fbt plan`
- `fbt doctor`
- `fbt build`
- `fbt eval`
- `fbt diff`
- `fbt review`
- `fbt docs generate`

`state`, `artifact`, and `runner` are advanced or diagnostic commands. `run`
and `debug` are reserved command names in the MVP CLI.
