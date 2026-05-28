# fbt CLI Reference

Status: MVP-ready  
Created: 2026-05-28  
Audience: users and implementers of the `fbt` command-line interface

## 1. Overview

`fbt` is designed as a local-first single-binary CLI. The base runtime does not
require a daemon, scheduler, metadata database, web server, cloud account, or
approval workflow.

Basic workflow:

```sh
fbt init
fbt doctor
fbt plan
fbt build
fbt artifact show case_summaries
fbt diff case_summaries --against previous
fbt export openlineage
```

## 2. Global Flags

| Flag | Meaning |
|---|---|
| `--project-dir PATH` | Directory containing `fs_project.yml`; defaults to the current directory |
| `--state-dir PATH` | Local state directory; defaults to `.fbt/state` |
| `--select EXPR` | Select transforms for `plan` and `build` |
| `--json` | Machine-readable JSON output |

MVP does not accept `--exclude`, `--target`, `--vars`, `--dry-run`,
`--log-level`, `--no-color`, or `--quiet`.

Commands fail with exit code `2` when they receive unknown flags, unsupported
global flags, extra positional arguments, or a `--select` expression that
matches no transforms. fbt should never silently turn a typo into a broader
build.

## 3. Exit Codes

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Transform or eval failed |
| `2` | Invalid project, config, or parse error |
| `3` | Blocked by policy or confidence requirement |
| `4` | Runner protocol error |
| `5` | State lock or state backend error |
| `6` | Missing dependency or runner not installed |
| `130` | Cancelled by user |

## 4. Selection Syntax

| Expression | Meaning |
|---|---|
| `case_summaries` | Resource with matching name |
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

```sh
fbt version
fbt version --json
```

Human output is intentionally compact:

```text
fbt 0.1.0
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

The runnable templates include deterministic `demo.*` runner wrappers for local
smoke runs from a source checkout. Replace them with external runner commands
for real provider-backed execution.

### 5.2 fbt plan

Compare current project definitions and state to show what will run. `plan` is
read-only: it does not start runners, commit artifacts, or write fbt state.

```sh
fbt plan [--select EXPR]
```

Shows selected transforms, skipped transforms, dirty reasons, blocked reasons,
confidence requirements, and next commands. Use `fbt artifact explain TARGET`
to focus on one artifact.

Example:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked

run transform.knowledge_ops.case_summaries
  reason: no previous successful run

blocked transform.knowledge_ops.weekly_support_insights
  blocked: requires artifact.knowledge_ops.case_summaries current artifact
  next: fbt build --select case_summaries
```

### 5.3 fbt build

Run the lifecycle.

```sh
fbt build [--select EXPR]
```

Lifecycle:

```text
parse -> plan -> run external runner -> eval -> commit -> write state
```

### 5.4 fbt diff

Show differences between artifact versions.

```sh
fbt diff TARGET [--against REF]
```

`--against` values:

- `previous`
- explicit `artifact_version...`

MVP supports raw text diff and Markdown heading-aware diff.

### 5.5 fbt artifact

Inspect artifacts and versions.

```sh
fbt artifact ls
fbt artifact path TARGET
fbt artifact show TARGET
fbt artifact explain TARGET
fbt artifact history TARGET
```

`artifact path` prints the logical output path and immutable storage path for
the current or selected version. `artifact show` includes artifact version,
logical path, immutable storage path, digest, runner/model, confidence,
generating run, and semantic descriptors when available. `artifact history`
lists prior versions for the same logical artifact.

### 5.6 fbt doctor

Run a project readiness check.

```sh
fbt doctor
```

Checks project config parsing, state writability/lock acquisition, runner
discovery, executable availability, and runner protocol `initialize`.

### 5.7 Standard exports

```sh
fbt export openlineage [--output PATH]
fbt export otel [--output PATH]
```

`fbt export openlineage` writes OpenLineage-compatible RunEvent NDJSON for
artifact lineage. `fbt export otel` writes OTLP/JSON-compatible trace payloads
for local execution telemetry.

fbt-specific confidence, eval, descriptor, runner/model, and policy metadata
are included as `fbt_` custom facets or span attributes. Raw artifact content,
raw prompts, raw model responses, credentials, and absolute project paths are
not exported by default.

There is no base `fbt export openmetadata` command. OpenMetadata integration
uses `fbt export openlineage` plus an external OpenMetadata ingestion workflow,
or a future optional publisher outside fbt core.

## 6. JSON Output

With `--json`, stdout contains machine-readable JSON and human logs go to
stderr.

`fbt plan --json` returns the same plan shape used by the text output:

```json
{
  "command": "plan",
  "status": "success",
  "summary": {
    "selected": 2,
    "run": 1,
    "skipped": 0,
    "blocked": 1
  },
  "nodes": [
    {
      "transform_id": "transform.knowledge_ops.case_summaries",
      "name": "case_summaries",
      "action": "run",
      "dirty_reasons": ["source descriptor changed"]
    },
    {
      "transform_id": "transform.knowledge_ops.weekly_support_insights",
      "name": "weekly_support_insights",
      "action": "blocked",
      "blocked_reasons": [
        "requires artifact.knowledge_ops.case_summaries current artifact"
      ],
      "next_steps": ["fbt build --select case_summaries"]
    }
  ]
}
```

## 7. Primary Commands

The main user-facing path stays small:

- `fbt init`
- `fbt doctor`
- `fbt plan`
- `fbt build`
- `fbt artifact`
- `fbt diff`
- `fbt export`

`parse`, `eval`, `docs`, `state`, and `runner` are not public commands. Their
former responsibilities happen inside `doctor`, `plan`, `build`, and
`artifact`. `fbt` does not own human review or approval workflows; use Git,
PRs, CI, release tooling, or catalog systems around the files and standard
exports that fbt produces.
