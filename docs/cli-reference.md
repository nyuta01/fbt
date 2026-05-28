# fbt CLI Reference

Status: Draft  
Created: 2026-05-28  
Audience: users and implementers of the `fbt` command-line interface

## 1. Overview

`fbt` is designed as a local-first single-binary CLI. The base runtime does not require a daemon, scheduler, metadata database, web server, or cloud account.

Basic workflow:

```sh
fbt init
fbt parse
fbt plan
fbt build
fbt review status
fbt review approve case_summaries
fbt docs generate
```

## 2. Global Flags

| Flag | Meaning |
|---|---|
| `--project-dir PATH` | Directory containing `fs_project.yml`; defaults to the current directory |
| `--state-dir PATH` | Local state directory; defaults to `.fbt/state` |
| `--target NAME` | Target name; defaults to `local` |
| `--select EXPR` | Select resources |
| `--exclude EXPR` | Exclude resources |
| `--vars JSON_OR_YAML` | Runtime variables |
| `--dry-run` | Validate without committing outputs |
| `--json` | Machine-readable JSON output |
| `--log-level LEVEL` | `debug`, `info`, `warn`, or `error` |
| `--no-color` | Disable color output |
| `--quiet` | Suppress non-summary human output |

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
| `+case_summaries` | Include ancestors |
| `case_summaries+` | Include descendants |
| `+case_summaries+` | Include ancestors and descendants |
| `tag:support` | Tag selector |
| `path:transforms/support/` | Path selector |
| `source:support.raw_tickets` | Source selector |
| `state:modified` | Dirty resources |
| `state:pending_review` | Artifacts waiting for review |
| `selector:support_daily` | Named project selector |

Examples:

```sh
fbt plan --select state:modified
fbt build --select tag:support --exclude state:pending_review
fbt run --select +weekly_support_insights
```

## 5. Commands

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

### 5.3 fbt plan

Compare current manifest and state to show what will run.

```sh
fbt plan [--select EXPR] [--exclude EXPR]
```

Shows:

- Selected transforms
- Skipped transforms
- Dirty reasons
- Blocked reasons
- Estimated token and cost
- Required review
- Confidence requirements
- Output commit mode

Example:

```text
Plan: 2 selected, 1 skipped, 1 blocked

run transform.knowledge_ops.case_summaries
  reason: source support.raw_tickets changed
  runner: openai.responses
  estimate: 13k tokens, $0.42
  review: required, group support_leads

blocked transform.knowledge_ops.weekly_support_insights
  reason: requires artifact.knowledge_ops.case_summaries confidence reviewed
```

### 5.4 fbt build

Run the full lifecycle.

```sh
fbt build [--select EXPR] [--exclude EXPR]
```

Lifecycle:

```text
parse -> plan -> run -> eval -> review gate -> commit -> write state
```

Example:

```sh
fbt build --select tag:support
fbt build --select state:modified
fbt build --select +weekly_support_insights
```

### 5.5 fbt run

Run selected transforms directly. In MVP, `build` is the primary user-facing command and `run` is an advanced command.

```sh
fbt run --select EXPR
```

### 5.6 fbt eval

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

### 5.7 fbt diff

Show differences between artifact versions.

```sh
fbt diff TARGET [--against REF]
```

`--against` values:

- `current`
- `last-run`
- `last-approved`
- explicit `artifact_version...`
- file path

MVP should prioritize raw text diff and Markdown heading-aware diff.

### 5.8 fbt review

Manage review and approval state.

```sh
fbt review status [TARGET]
fbt review approve TARGET [--version VERSION_ID] [--comment TEXT]
fbt review reject TARGET [--version VERSION_ID] [--comment TEXT]
```

Approval is bound to `artifact_version`. If `TARGET` is a logical artifact, the current version is used.

MVP behavior: approving the current version writes an approval record and
promotes the current pointer to `approval_status: approved` and
`confidence: reviewed`. Rejecting the current version writes `rejected` and
keeps downstream reviewed/approved requirements blocked.

### 5.9 fbt docs generate

Generate static docs.

```sh
fbt docs generate
```

Output:

```text
target/docs/
```

Docs include graph lineage, runner/model metadata, token/cost summary, review state, confidence, artifact versions, eval results, and diff links.

### 5.10 fbt state

Inspect local state.

```sh
fbt state status
fbt state ls
fbt state current TARGET
fbt state clean
```

`state clean` cleans cache and work directories. It should not delete artifact versions or approvals by default.

### 5.11 fbt artifact

Inspect artifacts and versions.

```sh
fbt artifact ls
fbt artifact show TARGET
fbt artifact versions TARGET
```

### 5.12 fbt runner

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

### 5.13 fbt debug

Print diagnostics.

```sh
fbt debug
```

Checks:

- fbt version
- project directory
- state directory
- target
- OS / architecture
- runner executable availability
- state lock
- config validation

## 6. JSON Output

With `--json`, stdout contains machine-readable JSON and human logs go to stderr.

`fbt plan --json` example:

```json
{
  "command": "plan",
  "status": "success",
  "summary": {
    "selected": 2,
    "skipped": 1,
    "blocked": 1
  },
  "nodes": [
    {
      "unique_id": "transform.knowledge_ops.case_summaries",
      "action": "run",
      "dirty_reasons": ["source support.raw_tickets changed"],
      "estimated_cost_usd": 0.42,
      "review_required": true
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
  "error": {
    "code": "RUNNER_NOT_INSTALLED",
    "message": "runner not installed: openai.responses",
    "retryable": false
  }
}
```

## 8. Primary Commands

The main user-facing path should stay small:

- `fbt parse`
- `fbt plan`
- `fbt build`
- `fbt eval`
- `fbt diff`
- `fbt review`
- `fbt docs generate`

`run`, `state`, `artifact`, `runner`, and `debug` are advanced or diagnostic commands.
