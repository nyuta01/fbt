# fbt State and Run Results Spec

Status: MVP-ready
Created: 2026-05-28
Audience: users and implementers of local state, run results, artifact versions,
eval results, and policy decisions

`fbt` does not require a metadata database. By default, it stores manifest
snapshots, current artifact pointers, execution summaries, artifact versions,
eval results, and policy decisions under `.fbt/state/`.

State is local-only in MVP. `state.backend` must be `local` when set. The
project `state.path` config and CLI `--state-dir` override only the state and
receipt directory. They do not move immutable artifact snapshots under
`.fbt/artifacts`, and they do not change the current logical output root
configured by `artifact_path`.

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

## Retention Hygiene

MVP retention policy is `keep_all`: fbt never deletes artifact versions, run
results, eval results, or policy decisions automatically. This preserves local
lineage and makes repeated builds inspectable.

The read-only retention inspection command is:

```sh
fbt artifact retention
```

High-volume fixture coverage is provided by:

```sh
make retention-high-volume-smoke
```

The fixture creates multiple artifact versions, verifies current and historical
version counts, checks JSON archive roots, and confirms no files are removed.

It reports human-readable state size, immutable artifact size, run-record
count, artifact version count, current-version count, historical-version count,
and missing storage references. The command removes no files.

For high-volume projects, archive `.fbt/state/` and `.fbt/artifacts/` together
before any external cleanup. Current logical artifacts under `artifact_path`
can be regenerated from current pointers only when the referenced immutable
storage still exists. Historical lineage requires the corresponding state
records and immutable storage to remain available.

fbt core intentionally does not expose a destructive prune command in MVP. If a
future prune command is added, it must be explicit, default to dry-run, preserve
current artifact pointers, record a cleanup receipt, and have conformance
coverage before it can remove files.

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

## Orphaned Declarations

Deleting or renaming source, transform, or artifact declarations does not delete
state. Current artifact pointers, artifact versions, run results, eval results,
and policy decisions remain inspectable.

An artifact version is `orphaned` when its `artifact_id` no longer appears in
the current manifest as a declared artifact or transform output. fbt reports
that state instead of pretending the artifact is still declared:

- `fbt artifact show TARGET` and `fbt artifact history TARGET` still resolve
  recorded versions by artifact ID or short artifact name.
- Human output prints `Declared: no (orphaned)` for recorded versions whose
  declaration is gone.
- JSON output includes `declared: false` and `orphaned: true`.
- `fbt artifact explain TARGET` remains a current-graph planning command. It
  requires a current producer transform and is not the inspection surface for
  orphaned historical artifacts.
- Recreating a declaration with the same artifact ID makes future builds use
  the normal declared semantics again. Historical versions are still immutable.

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
`runner_lock_incompatible`, `runner_protocol_error`,
`runner_contract_violation`, `policy_denied`, `eval_failed`, `cancelled`, and
`failed`. Failed receipts may reference policy decisions, eval results, runner
events, usage, and provenance when those were available before failure. They
must not move current artifact pointers or write artifact versions.

The latest transform status in `state.json` is the recovery index for local and
CI usage. `fbt plan --failed` and `fbt build --failed` select transforms whose
latest status is not `success`, add `latest run failed` as the explicit run
reason, and append new receipts for the retry. Older failed receipts remain in
`run_results.jsonl`; fbt does not rewrite, hide, or garbage-collect them during
recovery.

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
are recorded as skipped until external eval runners are implemented. Skipped
delegated eval results include `reason` and `hint` fields so users can tell they
are declarations, not active quality gates.

```json
{
  "result_id": "evaluation_result.knowledge_ops.semantic_check.1",
  "eval_id": "eval.knowledge_ops.semantic_check",
  "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abc",
  "transform_run_id": "transform_run.run_01H...",
  "status": "skipped",
  "reason": "semantic evals are not executed by fbt core",
  "hint": "Use an external judge transform that produces a report artifact when this should be an active quality gate."
}
```

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
