# fbt CLI Reference

Status: MVP-ready  
Created: 2026-05-28  
Audience: users and implementers of the `fbt` command-line interface

## 1. Overview

`fbt` is designed as a local-first single-binary CLI. The base runtime does not
require a daemon, scheduler, metadata database, web server, cloud account, or
approval workflow.

The command tree and flag parsing are implemented with Cobra. Human output uses
fixed-width key/value rows and tables for scanning; use `--json` when
automation needs full internal IDs and structured fields.

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
| `--state-dir PATH` | Override the local state directory; defaults to `.fbt/state` and does not move immutable artifact storage under `.fbt/artifacts` |
| `--select EXPR` | Select transforms for `plan` and `build`; inspection commands reject it |
| `--json` | Machine-readable JSON output |

MVP does not accept `--exclude`, `--target`, `--vars`, `--dry-run`,
`--log-level`, `--no-color`, or `--quiet`.

Commands fail with exit code `2` when they receive unknown flags, unsupported
global flags, extra positional arguments, or a `--select` expression that
matches no transforms. fbt should never silently turn a typo into a broader
build.

Common user-facing errors include a short `Hint:` line. For example, a declared
artifact that has not been built yet suggests `fbt build --select TARGET`, an
empty selector suggests running `fbt plan` without `--select`, and `--dry-run`
points to the read-only `plan` command.

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
| `+weekly_support_insights` | Selected transform plus upstream transforms |
| `case_summaries+` | Selected transform plus downstream transforms |
| `+case_summaries+` | Selected transform plus upstream and downstream transforms |

Graph operators can wrap any selector expression. fbt expands through the
resource graph but returns only transform IDs to `plan` and `build`.

Examples:

```sh
fbt plan --select tag:support
fbt build --select case_summaries
fbt plan --select +weekly_support_insights
fbt build --select case_summaries+
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
fbt plan [--select EXPR] [--force]
```

Shows selected transforms, skipped transforms, dirty reasons, blocked reasons,
confidence requirements, and next commands. Use `fbt artifact explain TARGET`
to focus on one artifact.

When the current invocation includes `--project-dir` or `--state-dir`, printed
next commands include the same context so they can be copied directly.

`--force` is read-only for `plan`: it previews selected clean transforms as
`RUN` with `because  forced rebuild`.

Example:

```text
Plan
  selected  2
  run       2
  skipped   0
  blocked   0

RUN     case_summaries
        because  no previous successful run
        output   case_summaries
        next     fbt build --select case_summaries

RUN     weekly_support_insights
        because  upstream artifact selected to run
        output   weekly_support_insights
        next     fbt build --select weekly_support_insights
```

### 5.3 fbt build

Produce selected artifacts and write the build receipt.

```sh
fbt build [--select EXPR] [--force]
```

The command is called `build` because fbt treats generated files as build
outputs: it resolves declared inputs, calls the configured external runner,
validates output candidates, commits immutable artifact versions, records eval
results, and writes local state for later inspection and export.

Lifecycle:

```text
parse -> plan -> run external runner -> eval -> commit -> write state
```

Human build output includes the transform run ID, each committed artifact path,
the committed version, and a contextual `fbt artifact show TARGET` next command.

If the runner, output contract, policy check, eval, or cancellation fails after
an invocation has started, `build` still appends failed receipts to
`.fbt/state/run_results.jsonl`. The failed receipt records a safe error kind
and message, but official artifact pointers and artifact versions are not
advanced.

When selected transforms depend on each other, `build` runs them in dependency
order within the same invocation. A downstream selected transform waits for the
upstream selected transform to commit its artifact, then runs if confidence and
policy requirements are satisfied.

`--force` runs selected transforms even when the normal plan would skip them as
clean. It does not bypass upstream artifact, confidence, policy, or output
boundary checks.

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

Inspect generated artifacts and versions. The subcommands are intentionally
split by the question they answer.

```sh
fbt artifact ls
fbt artifact path TARGET
fbt artifact show TARGET
fbt artifact explain TARGET
fbt artifact history TARGET
fbt artifact retention
```

| Subcommand | Answers |
|---|---|
| `ls` | Which artifacts have recorded versions? |
| `path TARGET` | Where is the current logical file and immutable snapshot? |
| `show TARGET` | What is the current artifact version, digest, runner, model, confidence, and metadata? |
| `explain TARGET` | Why would this artifact run, skip, or block right now? |
| `history TARGET` | Which versions have been recorded for this artifact? |
| `retention` | How large are local state and immutable artifact history? |

`artifact path` prints the logical output path and immutable storage path for
the current or selected version. `artifact show` includes artifact version,
logical path, immutable storage path, digest, runner/model, confidence,
generating run, and semantic descriptors when available. `artifact history`
lists prior versions for the same logical artifact. `artifact explain` is the
primary command for plan reasoning: it shows the producing transform, current
version, previous run, decision, input/source fingerprints, upstream artifact
requirements, dirty or blocked reasons, and exact next commands.
`artifact retention` is read-only and reports local state/artifact growth,
current and historical version counts, run-record count, and missing immutable
storage references. It does not remove files.

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

`fbt export openlineage` writes OpenLineage-compatible RunEvent NDJSON from
local artifact lineage. `fbt export otel` writes OTLP/JSON-compatible trace
payloads from local run receipts. Without `--output`, both commands write to
stdout so they can be piped to a backend-specific uploader. With `--output`,
fbt writes the file and prints a short human summary.

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

`parse`, `eval`, `docs`, `state`, and `runner` are not public commands. Project
readiness belongs to `doctor`, preview belongs to read-only `plan`, state writes
belong to `build`, and inspection belongs to `artifact`. `fbt` does not own
human review or approval workflows; use Git, PRs, CI, release tooling, or
catalog systems around the files and standard exports that fbt produces.
