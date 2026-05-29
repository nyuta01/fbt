# External Semantic And Evidence Quality Checks

Status: MVP-ready
Updated: 2026-05-29
Audience: teams that need grounding, no-invention, or model-judge checks for
generated artifacts

## Principle

fbt core runs deterministic checks such as required sections. It does not own
semantic judges, model graders, grounding services, retrieval systems, or risk
policies.

When a generated artifact needs a semantic or evidence-quality check, model it
as another transform:

```text
source evidence + generated artifact + rubric
  -> external runner
  -> judge or evidence report artifact
```

fbt records the report artifact, its inputs, runner, policy, and lineage. Git,
CI, pull requests, or publishing tooling decide whether the report blocks a
release.

## Runnable Example

`examples/semantic_eval_boundary` uses command transforms to keep the example
provider-free:

| Transform | Inputs | Output |
|---|---|---|
| `manual_update` | `source.incident_evidence` | `target/artifacts/manual/manual_update.md` |
| `evidence_quality_report` | `manual_update`, `source.incident_evidence`, `evidence_quality_rubric` | `target/artifacts/quality/evidence_quality_report.md` |

Run:

```sh
fbt plan --project-dir examples/semantic_eval_boundary --select tag:quality_boundary
fbt build --project-dir examples/semantic_eval_boundary --select tag:quality_boundary
fbt artifact explain evidence_quality_report --project-dir examples/semantic_eval_boundary
```

Expected report excerpt:

```md
# Evidence Quality Report

Result: pass

## Grounding Checks
- Required operational claims appear in both the manual artifact and source evidence.
- No blocked unsupported claims were found.
```

Expected receipt shape:

```text
Artifact: evidence_quality_report

Inputs
  ok input   manual_update
  ok input   source.incident_evidence
  ok asset   evidence_quality_rubric
  ok runner  local.command
```

## Replacing The Checker

The example checker is `bin/check-evidence-quality`. Replace that command with:

- an LLM judge runner such as `openai.responses`
- a Codex or Claude Code adapter that writes a report
- an internal grounding service wrapper
- a CI script that checks citations, forbidden claims, or risk rules

Keep the output as a normal artifact:

```yaml
outputs:
  - name: evidence_quality_report
    type: markdown
    path: target/artifacts/quality/evidence_quality_report.md
```

This keeps fbt focused on declared files, build state, and lineage instead of
embedding a judge framework in core.
