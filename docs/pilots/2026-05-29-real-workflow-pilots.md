# 2026-05-29 Real Workflow Pilots

Status: completed
Task: `FBT-PILOT-001`

These pilots used realistic local source directories, deterministic checked-in
runners, and temporary project copies. The goal was to observe first-user
friction in the actual `doctor -> plan -> build -> inspect` loop without
adding scheduler, approval, provider SDK, catalog, or storage behavior to fbt
core.

## Pilot 1: Daily QA Operations

Workflow:

```text
questions + answers + product docs + current manual
  -> faq_candidates
  -> manual_patch_candidates
  -> unresolved_questions
  -> manual_update
```

Commands run against a temporary copy of `examples/daily_qa_ops`:

```sh
fbt doctor --project-dir "$project"
fbt plan --project-dir "$project" --select tag:daily_qa
fbt build --project-dir "$project" --select tag:daily_qa
fbt artifact explain manual_update --project-dir "$project"
```

Observed result:

```text
Doctor: ok

Plan
  selected  2
  run       2

SUCCESS daily_qa_candidates
        output     faq_candidates -> target/artifacts/qa/latest/faq_candidates.md
        output     manual_patch_candidates -> target/artifacts/qa/latest/manual_patch_candidates.md
        output     unresolved_questions -> target/artifacts/qa/latest/unresolved_questions.md

SUCCESS promote_manual_update
        output     manual_update -> target/artifacts/manual/latest/manual_update.md
```

Then one new question and one answer were added under the stable source
directories:

```text
data/qa/inbox/questions/Q-1044.md
data/qa/inbox/answers/A-1044.md
```

The next plan showed the expected dirty reason:

```text
RUN     daily_qa_candidates
        because  source descriptor changed
```

Outcome:

- Multiple source directories and multiple artifact outputs work as the daily
  batch mental model expects.
- Stable source paths are enough for a daily new-file flow; fbt does not need a
  date partition engine or watermark store in core.
- `artifact explain` gives a usable receipt for the promoted manual output.

## Pilot 2: External Quality Report Boundary

Workflow:

```text
incident evidence
  -> manual_update
  -> evidence_quality_report
```

Commands run against a temporary copy of `examples/semantic_eval_boundary`:

```sh
fbt doctor --project-dir "$project"
fbt plan --project-dir "$project" --select tag:quality_boundary
fbt build --project-dir "$project" --select tag:quality_boundary
fbt artifact explain evidence_quality_report --project-dir "$project"
```

Observed result:

```text
Doctor: ok

Plan
  selected  2
  run       2

SUCCESS manual_update
        output     manual_update -> target/artifacts/manual/manual_update.md

SUCCESS evidence_quality_report
        output     evidence_quality_report -> target/artifacts/quality/evidence_quality_report.md

Inputs
  ok      input   manual_update
  ok      input   source.incident_evidence
  ok      asset   evidence_quality_rubric
  ok      runner  local.command
```

Outcome:

- The external quality-check pattern is understandable as a normal fbt
  transform producing a report artifact.
- The boundary is still clean: fbt records the report and lineage but does not
  own model-judge logic, approval, or release blocking.

## Friction Backlog

The pilots found one concrete first-user friction point:

| Friction | Decision | Fix |
|---|---|---|
| Copying an example project outside the repository can make demo wrapper commands fail with `go.mod file not found` unless `FBT_SOURCE_ROOT` is declared and set. | Keep examples optimized for checked-out repo usage; copied examples remain supported when the runner environment is made explicit. | `examples/README.md` now documents how to copy examples safely and how to declare `FBT_SOURCE_ROOT` in copied projects. |

No core feature was added from this pilot. The observed workflows still fit the
Unix-style boundary: prepare files outside fbt, run fbt once, inspect the
artifact receipts, then publish or approve with adjacent tools.

