# fbt Manifest Spec

Status: Draft  
Created: 2026-05-28  
Audience: implementers of parsed graph metadata

## 1. Overview

The manifest is canonical metadata for an `fbt` project. Since `fbt` core is a control plane, the manifest represents the parsed definition graph used by planning, execution, evals, docs, and state comparison.

The manifest represents:

- Filesystem artifact graph
- Sources
- Transforms
- Transform assets
- Policies
- Evals
- Runners
- Dependencies
- Optional current artifact version snapshots from state
- Fingerprints
- Confidence requirements
- Approval requirements
- Docs and lineage metadata

## 2. Responsibilities

Manifest contains:

- Project metadata
- Resource definitions
- Dependency graph
- Logical artifact definitions
- Current artifact pointer snapshots where available
- Fingerprints and checksums
- Transform asset, policy, eval, and runner dependencies
- Selection and state comparison metadata
- Docs metadata

Manifest does not contain:

- Full runner event logs
- Full token usage history
- Complete tool call logs
- Full review comments
- Failed attempt logs
- Full transform_run history
- Full evaluation_result or policy_decision history
- Binary artifact content

Runtime history belongs in state and run results.

## 3. Resource Types

| Type | Kind | Meaning |
|---|---|---|
| `source` | definition | External input file or directory |
| `artifact` | definition | Logical output managed by the project |
| `artifact_version` | state snapshot | Immutable content snapshot; manifest may include current snapshot |
| `transform` | definition | Transform contract |
| `transform_run` | runtime record | Concrete execution; not stored as full manifest history |
| `transform_asset` | definition | Prompt, template, script, rubric, style guide, examples |
| `policy` | definition | Tool scope, review, security, cost |
| `policy_decision` | runtime record | Runtime policy result; not full manifest history |
| `eval` | definition | Deterministic, semantic, or human eval definition |
| `evaluation_result` | runtime record | Eval result; not full manifest history |
| `runner` | definition | External runner reference |

## 4. Unique IDs

Format:

```text
<resource_type>.<project_name>.<name>
```

Examples:

```text
source.knowledge_ops.legal_docs.raw_contracts
artifact.knowledge_ops.contract_summaries
artifact_version.knowledge_ops.contract_summaries.sha256_current
transform.knowledge_ops.contract_summaries
transform_asset.knowledge_ops.contract_summary_prompt
policy.knowledge_ops.legal_summary
eval.knowledge_ops.citation_coverage
runner.knowledge_ops.openai.responses
```

Source IDs can include source namespace:

```text
source.<project>.<source_name>.<artifact_name>
```

## 5. Top-Level Shape

```json
{
  "metadata": {},
  "sources": {},
  "artifacts": {},
  "artifact_versions": {},
  "transforms": {},
  "transform_assets": {},
  "policies": {},
  "evals": {},
  "runners": {},
  "parent_map": {},
  "child_map": {},
  "selectors": {},
  "disabled": {},
  "state_snapshot": {},
  "files": {}
}
```

## 6. Metadata

```json
{
  "metadata": {
    "fbt_schema_version": "https://schemas.fbt.dev/fbt/manifest/v1.json",
    "fbt_version": "0.1.0",
    "project_name": "knowledge_ops",
    "project_id": "sha256:...",
    "generated_at": "2026-05-28T10:00:00Z",
    "invocation_id": "inv_01H...",
    "target_name": "local"
  }
}
```

Schema URI and compatibility rules are defined in
[Schema and Versioning Spec](schema-and-versioning-spec.md). Manifest readers
must reject unsupported major schema versions.

## 7. Source Resource

```json
{
  "unique_id": "source.knowledge_ops.legal_docs.raw_contracts",
  "resource_type": "source",
  "name": "raw_contracts",
  "source_name": "legal_docs",
  "artifact_type": "docx_directory",
  "path": "data/legal/contracts/*.docx",
  "resolved_paths": [
    "data/legal/contracts/a.docx",
    "data/legal/contracts/b.docx"
  ],
  "fingerprint": {
    "method": "directory_listing_and_content",
    "value": "sha256:abc..."
  },
  "tags": ["legal"],
  "meta": {}
}
```

## 8. Artifact Resource

```json
{
  "unique_id": "artifact.knowledge_ops.contract_summaries",
  "resource_type": "artifact",
  "name": "contract_summaries",
  "artifact_type": "markdown_directory",
  "logical_path": "target/artifacts/contracts/summaries/",
  "current": {
    "digest": "sha256:current...",
    "version_id": "artifact_version.knowledge_ops.contract_summaries.sha256_current",
    "run_id": "run_001",
    "committed_at": "2026-05-28T10:01:00Z"
  },
  "contract": {
    "required_sections": ["Summary", "Key Terms", "Risks", "Questions"],
    "citations": {
      "required": true
    }
  },
  "tags": ["legal", "llm-generated"],
  "meta": {}
}
```

`current` is a state overlay snapshot, not authoritative history.

## 9. Artifact Version Snapshot

```json
{
  "version_id": "artifact_version.knowledge_ops.contract_summaries.sha256_current",
  "resource_type": "artifact_version",
  "artifact_id": "artifact.knowledge_ops.contract_summaries",
  "descriptor": {
    "media_type": "text/markdown; charset=utf-8",
    "digest": "sha256:current...",
    "size": 12345,
    "artifact_type": "fbt.artifact.markdown_directory.v1"
  },
  "semantic_descriptor": null,
  "generated_by": "transform_run.run_001",
  "confidence": "semantic",
  "approval_status": "pending",
  "committed_at": "2026-05-28T10:01:00Z"
}
```

Artifact descriptor canonicalization, artifact type identifiers, and artifact
version ID format are defined in
[Schema and Versioning Spec](schema-and-versioning-spec.md). Runner-supplied
digests are advisory only; core computes authoritative descriptors.

## 10. Transform Resource

```json
{
  "unique_id": "transform.knowledge_ops.contract_summaries",
  "resource_type": "transform",
  "name": "contract_summaries",
  "transform_type": "llm",
  "runner": "runner.knowledge_ops.openai.responses",
  "inputs": [
    {
      "kind": "ref",
      "unique_id": "artifact.knowledge_ops.normalized_contracts",
      "name": "normalized_contracts",
      "require": {
        "confidence": "structural"
      }
    }
  ],
  "outputs": [
    {
      "unique_id": "artifact.knowledge_ops.contract_summaries",
      "name": "contract_summaries",
      "artifact_type": "markdown_directory",
      "declared_path": "target/artifacts/contracts/summaries/"
    }
  ],
  "assets": [
    "transform_asset.knowledge_ops.contract_summary_prompt"
  ],
  "policy": "policy.knowledge_ops.legal_summary",
  "evals": [
    "eval.knowledge_ops.citation_coverage"
  ],
  "model": {
    "provider": "openai",
    "name": "gpt-5",
    "parameters_hash": "sha256:params..."
  },
  "determinism": "stochastic",
  "cache": {
    "mode": "require_approval_for_reuse"
  },
  "fingerprint": {
    "config": "sha256:config...",
    "effective": "sha256:effective..."
  }
}
```

Effective fingerprint:

```text
effective = hash(
  transform_config,
  input_artifact_version_descriptors,
  transform_asset_fingerprints,
  policy_fingerprint,
  eval_fingerprints,
  runner_identity,
  runner_config,
  model_identity,
  model_parameters,
  declared_external_dependencies
)
```

## 11. Transform Asset Resource

```json
{
  "unique_id": "transform_asset.knowledge_ops.contract_summary_prompt",
  "resource_type": "transform_asset",
  "name": "contract_summary_prompt",
  "asset_type": "prompt",
  "path": "prompts/contract_summary.md",
  "fingerprint": {
    "content": "sha256:asset..."
  },
  "variables": ["input_documents", "style_guide"],
  "tags": ["legal"],
  "meta": {}
}
```

## 12. Policy Resource

```json
{
  "unique_id": "policy.knowledge_ops.legal_summary",
  "resource_type": "policy",
  "name": "legal_summary",
  "fingerprint": {
    "content": "sha256:policy..."
  },
  "read_scope": ["target/artifacts/contracts/normalized/"],
  "write_scope": ["target/quarantine/contracts/summaries/"],
  "network": true,
  "tools": {
    "allow": ["read_artifact", "search_project"],
    "deny": ["write_source_files"]
  },
  "limits": {
    "timeout_seconds": 300,
    "max_cost_usd": 1.0,
    "max_tool_calls": 20
  },
  "review": {
    "required": true,
    "group": "legal"
  }
}
```

## 13. Eval Resource

```json
{
  "unique_id": "eval.knowledge_ops.citation_coverage",
  "resource_type": "eval",
  "name": "citation_coverage",
  "eval_type": "semantic",
  "runner": "runner.knowledge_ops.openai.responses",
  "fingerprint": {
    "config": "sha256:eval..."
  },
  "config": {
    "min": 0.9
  },
  "grants_confidence": "semantic"
}
```

## 14. Runner Resource

```json
{
  "unique_id": "runner.knowledge_ops.openai.responses",
  "resource_type": "runner",
  "name": "openai.responses",
  "runner_type": "llm",
  "protocol": "stdio-jsonrpc",
  "command": "fbt-openai-runner",
  "version": "0.1.0",
  "capabilities": {
    "transform_types": ["llm"],
    "artifact_types": ["markdown", "markdown_directory"],
    "stream_events": true,
    "usage_reporting": true,
    "cost_estimation": true
  },
  "fingerprint": {
    "identity": "sha256:runner..."
  }
}
```

## 15. Graph Maps

```json
{
  "parent_map": {
    "transform.knowledge_ops.contract_summaries": [
      "artifact.knowledge_ops.normalized_contracts",
      "transform_asset.knowledge_ops.contract_summary_prompt",
      "policy.knowledge_ops.legal_summary",
      "eval.knowledge_ops.citation_coverage",
      "runner.knowledge_ops.openai.responses"
    ]
  },
  "child_map": {
    "transform_asset.knowledge_ops.contract_summary_prompt": [
      "transform.knowledge_ops.contract_summaries"
    ]
  }
}
```

Transform assets, policies, evals, and runners are graph nodes.

## 16. Files

```json
{
  "files": {
    "prompts/contract_summary.md": {
      "path": "prompts/contract_summary.md",
      "checksum": "sha256:...",
      "resource_ids": [
        "transform_asset.knowledge_ops.contract_summary_prompt"
      ]
    }
  }
}
```

## 17. State Comparison

Modified reasons include:

- Source descriptor changed
- Upstream artifact version pointer changed
- Transform asset fingerprint changed
- Policy fingerprint changed
- Eval fingerprint changed
- Runner identity changed
- Model parameters changed
- Declared external dependency changed
- Retrieved context changed
- Tool identity changed
- Approval invalidated
- Output missing

## 18. Manifest vs Run Results

Manifest:

- Project graph
- Intended transform definitions
- Current metadata snapshot
- Dependency maps
- State comparison metadata

State:

- Current artifact pointers
- Artifact version index
- Approval state
- Latest run pointers
- Previous manifest checksum

Run results:

- What actually ran
- Transform run records
- Runner event summaries
- Token and cost summary
- Tool-call summary
- Output candidates
- Committed artifact versions
- Eval and policy results
- Warnings and errors

## 19. Remaining Manifest Decisions

The manifest is the definition graph plus state overlay snapshot. Runtime history is not authoritative in the manifest. Schema/versioning and descriptor registry decisions are fixed for this draft. Remaining decisions:

1. Minimum fields for current artifact version snapshots.
2. How much approval state to include beyond `approval_status`.
3. Whether runner capabilities are parse-time manifest snapshots or runtime run-results records.
4. Whether rendered transform asset fingerprints belong in planning metadata or only run results.
5. Whether PROV or in-toto exports should be added after the OpenLineage and
   OpenTelemetry contracts in [Standard Export Spec](standard-export-spec.md).
