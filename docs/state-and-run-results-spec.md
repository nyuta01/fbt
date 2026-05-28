# fbt State and Run Results Spec

Status: Draft  
Created: 2026-05-28  
Audience: users and implementers of local state, run results, artifact versions, approvals, eval results, and policy decisions

## 1. Overview

`fbt` does not require a metadata database. By default, it stores manifest snapshots, current artifact pointers, execution summaries, artifact versions, approval state, eval results, and policy decisions under `.fbt/state/`.

This spec defines:

- What files are updated after `fbt build`
- Which files are authoritative history
- How failed or interrupted runs avoid corrupting official artifacts
- Which objects approvals, evals, and policies attach to
- The minimum boundary for future external state backends

Standard export mappings for this state are defined in
[Standard Export Spec](standard-export-spec.md). The native state files remain
the source of truth; OpenLineage, OpenTelemetry, and OpenMetadata integrations
are export views over these files.

## 2. State Directory

```text
.fbt/
  state/
    manifest.json
    state.json
    run_results.jsonl
    artifact_versions.json
    approvals.json
    evaluation_results.json
    policy_decisions.json
  cache/
  logs/
  work/
```

| File | Responsibility | Updated When |
|---|---|---|
| `manifest.json` | Parsed definition resources and dependency graph | `parse`, `plan`, `build` |
| `state.json` | Current artifact pointers and planner snapshot | Successful commit |
| `run_results.jsonl` | Append-only invocation and transform-run summaries | Invocation start/end and transform completion |
| `artifact_versions.json` | Local artifact version index | Output candidate descriptor creation |
| `approvals.json` | Approval state by artifact version | `fbt review approve/reject` |
| `evaluation_results.json` | Eval execution results | Eval completion |
| `policy_decisions.json` | Policy check results | Policy check completion |

## 3. Core Invariants

1. `artifact_version` is immutable. A descriptor for an existing `version_id` must not change.
2. Official artifact pointers are updated only on commit.
3. Runners never update official artifact pointers directly.
4. Failed, cancelled, or interrupted runs do not update official pointers.
5. Approval is bound to `artifact_version`, not file path.
6. `state.json` is a snapshot, not complete history.
7. `run_results.jsonl` is append-only execution summary history.
8. `manifest.json` is not the authoritative runtime history.
9. Commit is idempotent. Re-committing the same digest does not corrupt state.
10. Secrets, raw prompts, raw model responses, and credentials are not saved by default.
11. State files include schema metadata and reject unsupported major schema versions.

## 4. state.json

`state.json` stores current pointers and planner snapshots.

```json
{
  "metadata": {
    "fbt_schema_version": "https://schemas.fbt.dev/fbt/state/v1.json",
    "fbt_version": "0.1.0",
    "project_name": "knowledge_ops",
    "project_id": "sha256:project...",
    "updated_at": "2026-05-28T10:10:00Z",
    "last_invocation_id": "inv_01H..."
  },
  "current_artifacts": {
    "artifact.knowledge_ops.case_summaries": {
      "artifact_id": "artifact.knowledge_ops.case_summaries",
      "current_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abcd",
      "current_digest": "sha256:abcd...",
      "logical_path": "target/artifacts/support/case_summaries/",
      "confidence": "reviewed",
      "approval_status": "approved",
      "committed_at": "2026-05-28T10:08:00Z",
      "generated_by": "transform_run.run_01H..."
    }
  },
  "latest_runs": {
    "transform.knowledge_ops.case_summaries": {
      "latest_run_id": "transform_run.run_01H...",
      "latest_successful_run_id": "transform_run.run_01H...",
      "latest_status": "success",
      "latest_effective_fingerprint": "sha256:effective..."
    }
  },
  "previous_manifest": {
    "path": ".fbt/state/manifest.json",
    "checksum": "sha256:manifest..."
  }
}
```

## 5. artifact_versions.json

`artifact_versions.json` is the local index of immutable content snapshots.

```json
{
  "metadata": {
    "fbt_schema_version": "https://schemas.fbt.dev/fbt/artifact-versions/v1.json",
    "project_name": "knowledge_ops",
    "updated_at": "2026-05-28T10:10:00Z"
  },
  "artifact_versions": {
    "artifact_version.knowledge_ops.case_summaries.sha256_abcd": {
      "version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abcd",
      "artifact_id": "artifact.knowledge_ops.case_summaries",
      "logical_path": "target/artifacts/support/case_summaries/",
      "storage_path": ".fbt/artifacts/artifact_version.knowledge_ops.case_summaries.sha256_abcd/content",
      "descriptor": {
        "media_type": "inode/directory",
        "digest": "sha256:abcd...",
        "size": null,
        "artifact_type": "fbt.artifact.markdown_directory.v1"
      },
      "semantic_descriptor": {
        "method": "markdown_ast_v1",
        "digest": "sha256:semantic..."
      },
      "generated_by": "transform_run.run_01H...",
      "confidence": "reviewed",
      "approval_status": "approved",
      "created_at": "2026-05-28T10:08:00Z",
      "committed_at": "2026-05-28T10:08:10Z",
      "materials": [
        {
          "resource_id": "source.knowledge_ops.support.raw_tickets",
          "digest": "sha256:input..."
        }
      ]
    }
  }
}
```

`descriptor.digest` is the core content identity. Raw digest and semantic digest are separate for formats such as Word, Excel, and PDF. Descriptor canonicalization and semantic descriptor method names are defined in [Schema and Versioning Spec](schema-and-versioning-spec.md).

## 6. run_results.jsonl

`run_results.jsonl` uses JSON Lines: one JSON object per line. This makes append-only writes and interrupted runs easier to handle.

Record types:

- `invocation_started`
- `transform_run`
- `invocation_completed`

`invocation_started`:

```json
{
  "record_type": "invocation_started",
  "invocation_id": "inv_01H...",
  "started_at": "2026-05-28T10:05:00Z",
  "command": "build",
  "args": ["build", "--select", "case_summaries"],
  "project_name": "knowledge_ops",
  "target_name": "local",
  "manifest_checksum": "sha256:manifest..."
}
```

`transform_run`:

```json
{
  "record_type": "transform_run",
  "invocation_id": "inv_01H...",
  "run_id": "transform_run.run_01H...",
  "transform_id": "transform.knowledge_ops.case_summaries",
  "status": "success",
  "started_at": "2026-05-28T10:05:10Z",
  "completed_at": "2026-05-28T10:08:00Z",
  "duration_ms": 170000,
  "selection": {
    "selected": true,
    "dirty_reasons": [
      "source descriptor changed",
      "transform asset prompts/case_summary.md changed"
    ]
  },
  "runner": {
    "id": "runner.knowledge_ops.openai.responses",
    "name": "openai.responses",
    "version": "0.1.0"
  },
  "materials": [
    {
      "resource_id": "source.knowledge_ops.support.raw_tickets",
      "artifact_version": null,
      "digest": "sha256:input..."
    }
  ],
  "output_candidates": [
    {
      "artifact_id": "artifact.knowledge_ops.case_summaries",
      "path": ".fbt/work/req_123/outputs/case_summaries/",
      "descriptor": {
        "media_type": "inode/directory",
        "digest": "sha256:abcd...",
        "size": null,
        "artifact_type": "fbt.artifact.markdown_directory.v1"
      }
    }
  ],
  "committed_versions": [
    "artifact_version.knowledge_ops.case_summaries.sha256_abcd"
  ],
  "evaluation_results": [
    "evaluation_result.knowledge_ops.citation_coverage.01H..."
  ],
  "policy_decisions": [
    "policy_decision.knowledge_ops.support_summary_scope.01H..."
  ],
  "usage": {
    "gen_ai.usage.input_tokens": 12000,
    "gen_ai.usage.output_tokens": 1800,
    "fbt.usage.total_tokens": 13800,
    "fbt.estimated_cost_usd": 0.42
  },
  "events": [
    {
      "request_id": "req_123",
      "transform_run_id": "transform_run.run_01H...",
      "time": "2026-05-28T10:07:55Z",
      "event_type": "usage",
      "level": "info",
      "message": "LLM request completed",
      "attributes": {
        "gen_ai.usage.input_tokens": 12000,
        "gen_ai.usage.output_tokens": 1800,
        "fbt.usage.total_tokens": 13800
      }
    }
  ],
  "trace": {
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  },
  "warnings": [],
  "error": null
}
```

`events` stores runner protocol events that are safe for telemetry export.
Default state records omit raw `tool_call` payloads; redacted event attributes
are sufficient for `fbt export otel` span events.

`invocation_completed`:

```json
{
  "record_type": "invocation_completed",
  "invocation_id": "inv_01H...",
  "completed_at": "2026-05-28T10:08:20Z",
  "status": "success",
  "summary": {
    "selected": 1,
    "success": 1,
    "failed": 0,
    "blocked": 0,
    "skipped": 0,
    "pending_review": 0
  }
}
```

## 7. Evaluation Results

```json
{
  "evaluation_results": {
    "evaluation_result.knowledge_ops.citation_coverage.01H...": {
      "result_id": "evaluation_result.knowledge_ops.citation_coverage.01H...",
      "eval_id": "eval.knowledge_ops.citation_coverage",
      "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abcd",
      "transform_run_id": "transform_run.run_01H...",
      "status": "pass",
      "score": 0.94,
      "threshold": 0.9,
      "grants_confidence": "semantic",
      "runner": "runner.knowledge_ops.openai.responses",
      "details_path": ".fbt/logs/evals/citation_coverage_01H.json"
    }
  }
}
```

Eval status values:

- `pass`
- `warn`
- `fail`
- `error`
- `skipped`

## 8. Policy Decisions

```json
{
  "policy_decisions": {
    "policy_decision.knowledge_ops.support_summary_scope.01H...": {
      "decision_id": "policy_decision.knowledge_ops.support_summary_scope.01H...",
      "policy_id": "policy.knowledge_ops.support_summary_scope",
      "transform_id": "transform.knowledge_ops.case_summaries",
      "transform_run_id": "transform_run.run_01H...",
      "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abcd",
      "status": "allowed",
      "checks": [
        {
          "name": "write_scope",
          "status": "pass"
        }
      ],
      "decided_at": "2026-05-28T10:08:12Z"
    }
  }
}
```

Policy status values:

- `allowed`
- `denied`
- `warn`
- `error`

## 9. Approvals

```json
{
  "approvals": {
    "artifact_version.knowledge_ops.case_summaries.sha256_abcd": {
      "artifact_version_id": "artifact_version.knowledge_ops.case_summaries.sha256_abcd",
      "artifact_id": "artifact.knowledge_ops.case_summaries",
      "digest": "sha256:abcd...",
      "status": "approved",
      "review_group": "support_leads",
      "reviewer": "user@example.com",
      "approved_at": "2026-05-28T10:30:00Z",
      "expires_at": null,
      "comment": "Citations are sufficient for FAQ reuse",
      "superseded_by": null
    }
  }
}
```

Approval status values:

- `pending`
- `approved`
- `rejected`
- `expired`
- `superseded`
- `not_required`

Approval for one artifact version does not automatically apply to a later version unless the digest is identical or an explicit policy allows semantic-equivalence carryover.

## 10. Build Lifecycle and State Updates

```text
1. parse project files
2. write .fbt/state/manifest.json
3. append invocation_started to run_results.jsonl
4. plan selected transforms
5. create scoped work directory
6. invoke runner
7. runner writes output candidates under .fbt/work/
8. fbt computes descriptors / digests
9. fbt runs evals and policy checks
10. fbt copies immutable artifact version content under `.fbt/artifacts/`
11. fbt commits allowed artifact_versions to logical paths
12. fbt records artifact_versions, evaluation_results, policy_decisions, and approvals
13. fbt applies review gate
14. fbt updates state.json current pointers
15. append transform_run and invocation_completed to run_results.jsonl
```

For review-required artifacts:

- `commit_pending`: write to logical path but block downstream requirements until approved
- `quarantine_until_approved`: keep outputs in quarantine until approval

MVP default: `commit_pending`.

## 11. Atomicity and Locking

- Snapshot files are written to a temp file and atomically renamed.
- `run_results.jsonl` is append-only.
- A build acquires `.fbt/state/.lock`.
- Stale locks are detected.
- Official pointer updates and artifact version index updates happen in the same critical section.

Concurrent builds in the same project are not allowed by default.

## 12. Retention and Redaction

Recommended defaults:

- `run_results.jsonl`: 90 days or 10,000 transform runs
- `.fbt/work/`: removable after successful commit
- `.fbt/logs/`: 30 days
- `artifact_versions.json`: enough history for current and last approved versions
- Approval records: keep while the artifact version exists

Secrets and sensitive data:

- Do not store env var values
- Do not store raw prompts or raw model responses by default
- Store redacted tool-call arguments
- Store full traces only with explicit opt-in

## 13. External State Backend Boundary

Future state backends should satisfy:

```text
GetManifest(project_id, target)
PutManifest(project_id, target, manifest)
GetState(project_id, target)
UpdateState(project_id, target, compare_and_swap_token, state_patch)
AppendRunResult(project_id, target, record)
PutArtifactVersion(project_id, artifact_version)
GetArtifactVersion(version_id)
PutApproval(artifact_version_id, approval)
PutEvaluationResult(result)
PutPolicyDecision(decision)
AcquireLock(project_id, target, invocation_id)
ReleaseLock(project_id, target, invocation_id)
```

Base `fbt` implements this interface on the local filesystem.

External state backends must preserve the same artifact immutability,
idempotent commit, schema-version rejection, and approval-by-version semantics
as the local backend.
