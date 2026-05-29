# Semantic Eval Boundary Example

This example is runnable. It shows how to keep semantic judgement and evidence
quality checks outside fbt core while still recording the check as a normal fbt
artifact with lineage.

The project has two transforms:

```text
source evidence
  -> manual_update
  -> evidence_quality_report
```

Both transforms use the command adapter. In a real project, the second
transform could call an LLM judge, an internal policy checker, a retrieval
grounding service, or a CI script. fbt only records the report artifact,
inputs, runner, policy, and lineage.

## Run It

```sh
fbt plan --project-dir examples/semantic_eval_boundary --select tag:quality_boundary
fbt build --project-dir examples/semantic_eval_boundary --select tag:quality_boundary
fbt artifact explain evidence_quality_report --project-dir examples/semantic_eval_boundary
```

Expected output shape:

```text
SUCCESS manual_update
SUCCESS evidence_quality_report
```

The report is written to:

```text
target/artifacts/quality/evidence_quality_report.md
```

Report excerpt:

```md
# Evidence Quality Report

Result: pass

## Grounding Checks
- Required operational claims appear in both the manual artifact and source evidence.
- No blocked unsupported claims were found.
```

`artifact explain` shows the boundary:

```text
Inputs
  ok input   manual_update
  ok input   source.incident_evidence
  ok asset   evidence_quality_rubric
  ok runner  local.command
```

The external command owns the quality logic. fbt records the receipt.

## Core Eval

Use deterministic checks for facts fbt can verify locally:

```yaml
evals:
  - name: required_manual_sections
    type: deterministic
    config:
      sections:
        - Summary
        - Procedure
        - Rollback
    grants_confidence: structural
```

## External Judge Artifact

Use an external runner when a model must judge unsupported claims, tone,
correctness, or risk:

```yaml
transforms:
  - name: judge_manual_update
    type: llm
    runner: openai.responses
    inputs:
      - ref: manual_update
        require:
          confidence: structural
    outputs:
      - name: manual_update_judge_report
        type: markdown
        path: target/artifacts/judges/manual_update_judge_report.md
    assets:
      - ref: judge_rubric
    policy: judge_scope
```

fbt records the judge report as a normal artifact with lineage. The external
runner owns the model call, and Git/CI/publishing tooling owns whether the
report blocks release.

Do not put provider SDKs or model-judge logic in fbt core.

If a project declares `type: semantic` or `type: llm_judge` under `evals`, fbt
records that eval as `skipped`, writes a reason and external-judge hint into the
build receipt, and grants no confidence from it. Use the report-artifact pattern
above when the quality check must actually run.
