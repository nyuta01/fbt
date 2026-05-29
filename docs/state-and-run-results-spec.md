# fbt State and Run Results Spec

Status: MVP-ready
Created: 2026-05-28
Audience: users and implementers of local state, run results, artifact versions,
eval results, and policy decisions

`fbt` does not require a metadata database. By default, it stores manifest
snapshots, current artifact pointers, execution summaries, artifact versions,
eval results, and policy decisions under `.fbt/state/`.

## Files

```text
.fbt/state/
  manifest.json
  state.json
  run_results.jsonl
  artifact_versions.json
  evaluation_results.json
  policy_decisions.json
```

| File | Meaning |
|---|---|
| `manifest.json` | Last manifest snapshot written by a build invocation |
| `state.json` | Current artifact pointers and latest run pointers |
| `run_results.jsonl` | Append-only invocation and transform run records |
| `artifact_versions.json` | Immutable artifact version index |
| `evaluation_results.json` | Eval results by artifact version and run |
| `policy_decisions.json` | Runtime policy decisions |

## Current Artifact Pointer

```json
{
  "artifact_id": "artifact.knowledge_ops.case_summaries",
  "current_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abc",
  "current_digest": "sha256:abc",
  "logical_path": "target/artifacts/support/case_summaries/",
  "confidence": "structural",
  "committed_at": "2026-05-28T10:30:00Z",
  "generated_by": "transform_run.run_01H..."
}
```

## Artifact Version

```json
{
  "version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abc",
  "artifact_id": "artifact.knowledge_ops.case_summaries",
  "logical_path": "target/artifacts/support/case_summaries/",
  "storage_path": ".fbt/artifacts/artifact_version.../content",
  "descriptor": {
    "digest": "sha256:abc",
    "artifact_type": "fbt.artifact.markdown_directory.v1"
  },
  "generated_by": "transform_run.run_01H...",
  "confidence": "structural",
  "committed_at": "2026-05-28T10:30:00Z"
}
```

Artifact versions are immutable. Reusing a version ID with different content is
an error.

## Run Results

`run_results.jsonl` is append-only and contains invocation start/completion
records plus transform run records. Runner protocol events are stored after
redaction and can be exported to OpenTelemetry.

Every build that writes `invocation_started` must also append
`invocation_completed` with `success`, `failed`, `cancelled`, or `blocked`.
When a transform attempt starts, fbt appends a `transform_run` receipt even if
the runner, output contract, policy check, eval, or cancellation fails.

Failed transform receipts include a safe `error.kind` and `error.message`.
Common kinds are `runner_capability_incompatible`,
`runner_protocol_error`, `runner_contract_violation`, `policy_denied`,
`eval_failed`, `cancelled`, and `failed`. Failed receipts may reference
policy decisions, eval results, runner events, usage, and provenance when those
were available before failure. They must not move current artifact pointers or
write artifact versions.

Example failed receipt:

```json
{
  "record_type": "transform_run",
  "invocation_id": "inv_01H...",
  "run_id": "transform_run.run_01H...",
  "transform_id": "transform.knowledge_ops.case_summaries",
  "status": "policy_denied",
  "started_at": "2026-05-28T10:30:00Z",
  "completed_at": "2026-05-28T10:30:02Z",
  "committed_versions": [],
  "policy_decisions": ["policy_decision.knowledge_ops.case_summaries.1"],
  "error": {
    "kind": "policy_denied",
    "message": "policy denied output case_summaries: output path target/artifacts/support/case_summaries/ is outside declared write scope"
  }
}
```

## Eval Results

```json
{
  "result_id": "evaluation_result.knowledge_ops.required_sections.1",
  "eval_id": "eval.knowledge_ops.required_sections",
  "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abc",
  "transform_run_id": "transform_run.run_01H...",
  "status": "pass",
  "grants_confidence": "structural"
}
```

MVP core executes deterministic evals. Delegated semantic and LLM-judge evals
may be recorded as skipped until external eval runners are implemented.

## Policy Decisions

```json
{
  "decision_id": "policy_decision.knowledge_ops.case_summaries.1",
  "policy_id": "policy.knowledge_ops.support_scope",
  "transform_id": "transform.knowledge_ops.case_summaries",
  "transform_run_id": "transform_run.run_01H...",
  "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abc",
  "status": "allowed",
  "checks": [
    {"name": "write_scope", "status": "pass"}
  ]
}
```

## Store Interface

The local store exposes operations equivalent to:

```text
WriteManifest(manifest)
ReadManifest()
WriteState(snapshot)
ReadState()
PutArtifactVersion(version)
ReadArtifactVersions()
PutEvaluationResult(result)
ReadEvaluationResults()
PutPolicyDecision(decision)
ReadPolicyDecisions()
AppendRunResult(record)
ReadRunResults()
AcquireLock(invocation_id)
```

Human review or approval state is intentionally outside fbt state. Use external
systems such as Git, PRs, CI, release tooling, or catalog metadata for that
workflow.
