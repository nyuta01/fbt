# Daily QA Ops Example

This example shows a daily operations shape:

```text
daily source directories -> candidate files -> manual update
```

It also shows two important fbt boundaries:

- sources do not have to be JSONL; this example uses Markdown directories
- fbt records generated artifacts, but does not own approval or publishing

The checked-in runners are deterministic demo runners. They prove that the fbt
project, dependencies, and artifact versions work offline. Replace the runner
commands in `fs_project.yml` with an OpenAI, Claude Code, Codex, Gemini, or
other protocol-compatible runner when you want useful generated business
content.

## Workflow

```text
Markdown questions + Markdown answers + Markdown product docs
  -> faq_candidates.md
  -> manual_patch_candidates.md
  -> unresolved_questions.md
  -> manual_update.md
```

Use this example when customer questions and support answers accumulate in
directories and a daily job should produce structured outputs.

## Production Reference Loop

The most production-shaped entrypoint is:

```sh
examples/daily_qa_ops/ops/run-daily.sh
```

It is a shell/CI wrapper around fbt, not a new fbt feature. It validates the
source window manifest, then writes a run bundle under `target/ops/`:

```text
target/ops/runs/<run-id>/
target/ops/latest/
```

That bundle contains:

| File | Why it exists |
|---|---|
| `source-window.txt` | Proves ingestion prepared the selected source window. |
| `doctor.txt` | Proves the project, state, and runner setup were ready. |
| `plan.txt` | Shows what would run, skip, or block before runner execution. |
| `build.txt` | Shows committed artifact versions and output paths. |
| `manual_update-explain.txt` | Shows source, asset, runner, policy, eval, and lineage evidence. |
| `retention.txt` | Shows the local state/artifact archive boundary. |
| `openlineage.ndjson` | Standard lineage events for external metadata tools. |
| `otel.json` | OTLP/JSON traces for external observability tools. |

Copy `ops/github-actions-daily-fbt.yml` into `.github/workflows/` when you want
CI to be the authoritative daily builder. Add publish, pull-request, or Slack
steps after the fbt loop; fbt intentionally stops at explainable artifacts and
standard exports.

## Inputs

| Source | Path | Purpose |
|---|---|---|
| Questions | `data/qa/inbox/questions/` | Customer questions in the current processing window. |
| Answers | `data/qa/inbox/answers/` | Support replies or internal answers in the current processing window. |
| Product docs | `data/reference/product_docs/` | Current product behavior. |
| Current manual | `data/reference/manual/` | Existing manual sections to patch. |

## Outputs

```text
target/artifacts/qa/latest/faq_candidates.md
target/artifacts/qa/latest/manual_patch_candidates.md
target/artifacts/qa/latest/unresolved_questions.md
target/artifacts/manual/latest/manual_update.md
```

## Run It Offline

Preview both transforms:

```sh
fbt plan --project-dir examples/daily_qa_ops --select tag:daily_qa
```

Expected first-run shape:

```text
Plan
  selected  2
  run       2
  skipped   0
  blocked   0

RUN     daily_qa_candidates
        because  no previous successful run
        because  output missing
        output   faq_candidates, manual_patch_candidates, unresolved_questions

RUN     promote_manual_update
        because  no previous successful run
        because  output missing
        because  upstream artifact selected to run
        output   manual_update
```

Build the selected graph in dependency order:

```sh
fbt build --project-dir examples/daily_qa_ops --select tag:daily_qa
```

Inspect the generated files:

```sh
fbt artifact history faq_candidates --project-dir examples/daily_qa_ops
fbt artifact history manual_update --project-dir examples/daily_qa_ops
fbt artifact explain manual_update --project-dir examples/daily_qa_ops
```

## Daily Operation

Run the same commands every day. Keep the fbt config stable and let your
upstream ingestion step decide what is in the current processing window:

```text
data/qa/inbox/questions/
data/qa/inbox/answers/
```

For a "new items only" run, replace or refresh the inbox with today's batch
before running fbt. For a cumulative knowledge base, keep adding files to the
same inbox and let fbt rebuild when the resolved file set changes.

The stable-path contract is the important part:

```text
data/qa/inbox/questions/
data/qa/inbox/answers/
```

fbt watches those declared paths by fingerprinting their resolved file set and
content. If an external ingestion step replaces the contents with today's
batch, or appends another Markdown file, the next `fbt plan --select
daily_qa_candidates` marks the transform dirty with `source descriptor
changed`.

fbt intentionally does not include a scheduler, daemon, or date partition
engine. Use cron, CI, Airflow, Dagster, or another orchestrator to prepare the
input window and run `fbt plan` and `fbt build` once a day. fbt records each
run as artifact versions, so the logical `latest` files can change while older
versions remain in `.fbt/artifacts/`.

Use source-readiness checks outside fbt as well. For example, have ingestion
write a `_READY` marker, validate expected counts, or fail the external job
before calling fbt. fbt's job starts after the files are ready.

The checked-in production wrapper enforces that convention by requiring a JSON
readiness manifest:

```text
data/qa/inbox/_READY
```

The manifest names the current window and operation mode:

```json
{
  "schema_version": 1,
  "window_id": "2026-05-30",
  "mode": "new_items_only",
  "sources": [
    {"name": "questions", "path": "data/qa/inbox/questions", "min_files": 2},
    {"name": "answers", "path": "data/qa/inbox/answers", "min_files": 2}
  ]
}
```

Use `mode: new_items_only` when ingestion replaces the inbox with today's
batch, `mode: cumulative` when it appends files, `mode: correction` or
`deletion` when existing files are intentionally changed, and `mode: backfill`
when a historical partition is staged into the same stable paths. fbt still
reacts to the resulting file-set fingerprint; ingestion owns the meaning of the
window.

## Day 2 Simulation

Add one new question and one answer:

```sh
cat >examples/daily_qa_ops/data/qa/inbox/questions/Q-1044.md <<'MD'
# Q-1044: Admin export timezone

Customer asks whether scheduled admin exports use the workspace timezone or UTC.
MD

cat >examples/daily_qa_ops/data/qa/inbox/answers/A-1044.md <<'MD'
# A-1044

Scheduled exports use the workspace timezone unless the export job explicitly
sets UTC in the admin settings.
MD
```

Then preview the daily candidate transform:

```sh
fbt plan --project-dir examples/daily_qa_ops --select daily_qa_candidates
```

Expected reason:

```text
RUN     daily_qa_candidates
        because  source descriptor changed
```

Build again and inspect history:

```sh
fbt build --project-dir examples/daily_qa_ops --select daily_qa_candidates
fbt artifact history faq_candidates --project-dir examples/daily_qa_ops
fbt artifact retention --project-dir examples/daily_qa_ops
```

The logical output path remains `target/artifacts/qa/latest/faq_candidates.md`,
while previous versions remain under `.fbt/artifacts/`.

For the full operating model, read
`docs/examples/daily-source-operations.md`.
