# Daily Source Operations

Status: MVP-ready
Updated: 2026-05-29
Audience: teams running fbt once a day over growing source directories

## Purpose

Daily fbt operation should stay boring:

```text
external ingestion prepares source files
  -> fbt plan shows dirty artifacts
  -> fbt build commits new artifact versions
  -> fbt artifact/export commands explain what changed
```

fbt does not own the scheduler, ingestion database, partition engine, approval
workflow, or publishing destination. It owns the local build receipt for files
that already exist on disk.

## Example Shape

`examples/daily_qa_ops` models a support knowledge workflow:

| Role | Path | Meaning |
|---|---|---|
| Source | `data/qa/inbox/questions/` | Customer questions in the current processing window. |
| Source | `data/qa/inbox/answers/` | Support replies or internal answers in the same window. |
| Source | `data/reference/product_docs/` | Product facts used as grounding context. |
| Source | `data/reference/manual/` | Current manual sections to patch. |
| Artifact | `target/artifacts/qa/latest/faq_candidates.md` | FAQ candidates for review. |
| Artifact | `target/artifacts/qa/latest/manual_patch_candidates.md` | Proposed manual edits. |
| Artifact | `target/artifacts/qa/latest/unresolved_questions.md` | Questions without enough evidence. |
| Artifact | `target/artifacts/manual/latest/manual_update.md` | Promoted manual update candidate. |

The source and artifact sets are both plural. One daily run can create several
candidate artifacts and one downstream manual update.

## Day 1

```sh
fbt plan --project-dir examples/daily_qa_ops --select tag:daily_qa
fbt build --project-dir examples/daily_qa_ops --select tag:daily_qa
fbt artifact explain manual_update --project-dir examples/daily_qa_ops
```

Expected build shape:

```text
SUCCESS daily_qa_candidates
        output     faq_candidates -> target/artifacts/qa/latest/faq_candidates.md
        output     manual_patch_candidates -> target/artifacts/qa/latest/manual_patch_candidates.md
        output     unresolved_questions -> target/artifacts/qa/latest/unresolved_questions.md

SUCCESS promote_manual_update
        output     manual_update -> target/artifacts/manual/latest/manual_update.md
```

Expected receipt shape:

```text
Artifact: manual_update

Inputs
  ok input   manual_patch_candidates
  ok input   unresolved_questions
  ok input   reference.current_manual
  ok runner  demo.agent
```

## Day 2

An external ingestion step can add another question and answer:

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

fbt plan --project-dir examples/daily_qa_ops --select daily_qa_candidates
```

Expected reason:

```text
RUN     daily_qa_candidates
        because  source descriptor changed
```

fbt sees the changed file set and rebuilds dependent artifacts when asked. It
does not decide which date is "today"; the scheduler or ingestion job prepares
the source directory before fbt starts.

## Stable Paths, External Windows

Keep the fbt source declaration stable:

```yaml
sources:
  - name: qa
    artifacts:
      - name: questions
        type: markdown_directory
        path: data/qa/inbox/questions/
      - name: answers
        type: markdown_directory
        path: data/qa/inbox/answers/
```

Change the window outside fbt:

| Operation | Outside fbt | Inside fbt |
|---|---|---|
| New-items-only daily batch | Replace the inbox with today's prepared files. | Same source paths, same transforms. |
| Cumulative evidence base | Append files to the inbox directories. | `plan` detects changed descriptors. |
| Date/service/customer partition | Copy or symlink the selected partition into the inbox. | Select the same transform or tag. |
| Scheduled run | cron, CI, Airflow, Dagster, or another orchestrator. | `doctor`, `plan`, `build`, `artifact`, `export`. |

## Retention

Each successful build creates a new immutable artifact version under
`.fbt/artifacts/` and records state under `.fbt/state/`. Inspect growth with:

```sh
fbt artifact retention --project-dir examples/daily_qa_ops
```

Archive `.fbt/state/` and `.fbt/artifacts/` together when you need to move old
history elsewhere. fbt does not delete history automatically in MVP.
