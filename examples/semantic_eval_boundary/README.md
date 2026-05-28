# Semantic Eval Boundary Example

This example is intentionally a pattern, not a standalone model-judge fixture.
It shows how to keep semantic judgement outside fbt core.

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
