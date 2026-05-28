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
  run       1
  skipped   0
  blocked   1

RUN     daily_qa_candidates
        because  no previous successful run
        because  output missing
        output   faq_candidates, manual_patch_candidates, unresolved_questions

BLOCK   promote_manual_update
        blocked  requires manual_patch_candidates current artifact
        output   manual_update
```

Build the daily candidates:

```sh
fbt build --project-dir examples/daily_qa_ops --select daily_qa_candidates
```

Promote the manual update:

```sh
fbt build --project-dir examples/daily_qa_ops --select promote_manual_update
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
