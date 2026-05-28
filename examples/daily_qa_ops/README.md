# Daily QA Ops Example

This example shows a daily operations shape:

```text
daily source directories -> candidate files -> reviewed manual update
```

It also shows two important fbt boundaries:

- sources do not have to be JSONL; this example uses Markdown directories
- review is not mandatory for every artifact; only the promoted manual update
  requires approval

The checked-in runners are deterministic demo runners. They prove that the fbt
project, dependencies, artifact versions, and review gates work offline. Replace
the runner commands in `fs_project.yml` with an OpenAI, Claude Code, Codex,
Gemini, or other protocol-compatible runner when you want useful generated
business content.

The workflow turns daily question/answer files into several candidate
artifacts:

```text
Markdown questions + Markdown answers + Markdown product docs
  -> faq_candidates.md
  -> manual_patch_candidates.md
  -> unresolved_questions.md
```

Those daily candidate files do not require review. A later promotion step can
combine reviewed-worthy candidates into a formal manual update that does
require review.

## Why This Example Exists

Use this example when your real workflow looks like:

> Every day, customer questions and support answers accumulate in directories.
> Once a day, we want fbt to inspect that material and produce multiple
> structured files: FAQ candidates, manual patch candidates, and unresolved
> questions. Only the final manual update should require human approval.

## Inputs

The sources are plain Markdown directories:

| Source | Path | Purpose |
|---|---|---|
| Questions | `data/qa/inbox/questions/` | Customer questions in the current processing window. |
| Answers | `data/qa/inbox/answers/` | Support replies or internal answers in the current processing window. |
| Product docs | `data/reference/product_docs/` | Current product behavior. |
| Current manual | `data/reference/manual/` | Existing manual sections to patch. |

No JSONL is required. If your real inputs are `.txt`, `.md`, `.html`, `.pdf`,
or mixed files, use an appropriate artifact type such as `text`,
`markdown_directory`, `html`, `pdf`, or `directory`.

## Outputs

The first transform writes multiple artifacts:

```text
target/artifacts/qa/latest/faq_candidates.md
target/artifacts/qa/latest/manual_patch_candidates.md
target/artifacts/qa/latest/unresolved_questions.md
```

These are candidates, so they do not require review.

The second transform writes a promoted artifact:

```text
target/artifacts/manual/latest/approved_update.md
```

This artifact requires review because it is the file a team would use to update
official documentation.

## Run It Offline

This example uses deterministic demo runners, so it works without provider
credentials. Each command is a checkpoint in the workflow.

Preview both transforms:

```sh
fbt plan --project-dir examples/daily_qa_ops --select tag:daily_qa
```

This tells you what fbt would run before any runner is called. On the first
run, only the candidate generation can run because the promotion step depends
on candidate artifacts that do not exist yet.

Expected first-run shape:

```text
Plan: 2 selected, 1 run, 0 skipped, 1 blocked
run transform.daily_qa_ops.daily_qa_candidates
blocked transform.daily_qa_ops.promote_manual_update
  blocked: requires artifact.daily_qa_ops.manual_patch_candidates current artifact
```

Build the daily candidates:

```sh
fbt build --project-dir examples/daily_qa_ops --select daily_qa_candidates
```

This writes three candidate files:

```text
target/artifacts/qa/latest/faq_candidates.md
target/artifacts/qa/latest/manual_patch_candidates.md
target/artifacts/qa/latest/unresolved_questions.md
```

They are intermediate working files, so their artifact history shows:

```text
approval_status: not_required
```

Promote the manual update:

```sh
fbt build --project-dir examples/daily_qa_ops --select promote_manual_update
```

This consumes the candidate files plus the current manual and writes:

```text
target/artifacts/manual/latest/approved_update.md
```

The promoted artifact is pending review because it is the file a docs lead
would use to update official documentation:

```sh
fbt review show approved_manual_update --project-dir examples/daily_qa_ops
fbt review approve approved_manual_update \
  --project-dir examples/daily_qa_ops \
  --comment "Docs lead approved"
```

Explain the generated files:

```sh
fbt artifact history faq_candidates --project-dir examples/daily_qa_ops
fbt artifact history approved_manual_update --project-dir examples/daily_qa_ops
```

These commands show which version is current, where the logical file lives,
which runner produced it, and whether review was required.

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

fbt intentionally does not include a scheduler, daemon, or date partition
engine. Use cron, CI, Airflow, Dagster, or another orchestrator to prepare the
input window and run `fbt plan` and `fbt build` once a day. fbt records each
run as artifact versions, so the logical `latest` files can change while older
versions remain in `.fbt/artifacts/`.
