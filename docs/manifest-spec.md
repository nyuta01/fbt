# fbt Manifest Spec

Status: MVP-ready
Created: 2026-05-28
Updated: 2026-05-29
Audience: implementers of parsed graph metadata

## 1. Overview

The manifest is canonical parsed metadata for an `fbt` project. It represents
the graph used by planning, execution, evals, lineage, standard exports, and
state comparison.

The manifest contains definitions and current state overlays. Runtime history
belongs in `.fbt/state/run_results.jsonl` and related state files.

## 2. Manifest Contains

- Project metadata
- Sources
- Artifacts
- Current artifact-version snapshots where available
- Transforms
- Transform assets
- Policies
- Evals
- Runners
- Parent and child dependency maps
- Selectors
- Disabled resources
- Fingerprints and file checksums

## 3. Manifest Does Not Contain

- Full runner event logs
- Full token or cost history
- Complete tool-call logs
- Failed attempt history
- Full eval-result or policy-decision history
- Human workflow approval state
- Binary artifact content

## 4. Resource Types

| Type | Kind | Meaning |
|---|---|---|
| `source` | definition | External input file, glob, or directory |
| `artifact` | definition | Logical output managed by the project |
| `artifact_version` | state snapshot | Immutable content snapshot; manifest may include current snapshot |
| `transform` | definition | Transform contract |
| `transform_run` | runtime record | Concrete execution, stored in run results |
| `transform_asset` | definition | Prompt, template, script, rubric, style guide, schema, or example |
| `policy` | definition | Path, network, tool, timeout, size, and cost boundaries |
| `policy_decision` | runtime record | Runtime policy result, stored in state |
| `eval` | definition | Deterministic or delegated eval definition |
| `evaluation_result` | runtime record | Eval result, stored in state |
| `runner` | definition | External runner reference |

## 5. Unique IDs

Format:

```text
<resource_type>.<project_name>.<name>
```

Source IDs can include source namespace:

```text
source.<project>.<source_name>.<artifact_name>
```

Examples:

```text
source.knowledge_ops.support.raw_tickets
artifact.knowledge_ops.case_summaries
artifact_version.knowledge_ops.case_summaries.sha256_current
transform.knowledge_ops.case_summaries
transform_asset.knowledge_ops.case_summary_prompt
policy.knowledge_ops.support_summary_scope
eval.knowledge_ops.required_case_sections
runner.knowledge_ops.demo.llm
```

## 6. Top-Level Shape

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

## 7. Metadata

```json
{
  "metadata": {
    "fbt_schema_version": "https://schemas.fbt.dev/fbt/manifest/v1.json",
    "fbt_version": "0.1.0",
    "project_name": "knowledge_ops",
    "project_id": "sha256:...",
    "generated_at": "2026-05-29T10:00:00Z",
    "invocation_id": "inv_...",
    "target_name": "local"
  }
}
```

## 8. Source Resource

```json
{
  "unique_id": "source.knowledge_ops.support.raw_tickets",
  "resource_type": "source",
  "name": "raw_tickets",
  "source_name": "support",
  "artifact_type": "jsonl",
  "path": "data/support/tickets/*.jsonl",
  "resolved_paths": [
    "data/support/tickets/2026-05-28.jsonl"
  ],
  "fingerprint": {
    "method": "directory_listing_and_content",
    "value": "sha256:..."
  },
  "tags": ["support"],
  "meta": {}
}
```

## 9. Artifact Resource

```json
{
  "unique_id": "artifact.knowledge_ops.case_summaries",
  "resource_type": "artifact",
  "name": "case_summaries",
  "artifact_type": "markdown_directory",
  "logical_path": "target/artifacts/support/case_summaries/",
  "contract": {
    "format": "support_case_summary_v1",
    "required_sections": ["Summary", "Customer Impact"]
  },
  "current": {
    "digest": "sha256:current...",
    "version_id": "artifact_version.knowledge_ops.case_summaries.sha256_current",
    "run_id": "run_001",
    "committed_at": "2026-05-29T10:01:00Z"
  },
  "tags": ["support", "knowledge"],
  "meta": {}
}
```

`current` is a state overlay snapshot, not authoritative history.

## 10. Artifact Version Snapshot

```json
{
  "version_id": "artifact_version.knowledge_ops.case_summaries.sha256_current",
  "resource_type": "artifact_version",
  "artifact_id": "artifact.knowledge_ops.case_summaries",
  "descriptor": {
    "media_type": "text/markdown; charset=utf-8",
    "digest": "sha256:current...",
    "size": 12345,
    "artifact_type": "fbt.artifact.markdown_directory.v1"
  },
  "semantic_descriptor": null,
  "generated_by": "transform_run.run_001",
  "confidence": "structural",
  "committed_at": "2026-05-29T10:01:00Z"
}
```

Artifact descriptor canonicalization, artifact type identifiers, and artifact
version ID format are defined in
[Schema and Versioning Spec](schema-and-versioning-spec.md). Runner-supplied
digests are advisory only; core computes authoritative descriptors.

## 11. Transform Resource

```json
{
  "unique_id": "transform.knowledge_ops.case_summaries",
  "resource_type": "transform",
  "name": "case_summaries",
  "transform_type": "llm",
  "runner": "runner.knowledge_ops.demo.llm",
  "inputs": [
    {
      "kind": "source",
      "unique_id": "source.knowledge_ops.support.raw_tickets",
      "name": "raw_tickets"
    }
  ],
  "outputs": [
    {
      "unique_id": "artifact.knowledge_ops.case_summaries",
      "name": "case_summaries",
      "artifact_type": "markdown_directory",
      "declared_path": "target/artifacts/support/case_summaries/",
      "contract": {
        "format": "support_case_summary_v1",
        "required_sections": ["Summary", "Customer Impact"]
      }
    }
  ],
  "assets": [
    "transform_asset.knowledge_ops.case_summary_prompt"
  ],
  "policy": "policy.knowledge_ops.support_summary_scope",
  "evals": [
    "eval.knowledge_ops.required_case_sections"
  ],
  "model": {
    "provider": "demo",
    "name": "deterministic-demo-llm",
    "parameters_hash": "sha256:params..."
  },
  "determinism": "deterministic",
  "fingerprint": {
    "config": "sha256:config...",
    "effective": "sha256:effective..."
  }
}
```

`contract` on artifacts and transform outputs is free-form metadata. It is part
of the parsed graph and runner context; arbitrary document semantics are not
validated by the manifest builder.

Effective fingerprint:

```text
effective = hash(
  transform_config,
  input_source_or_artifact_descriptors,
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

## 12. Policy Resource

```json
{
  "unique_id": "policy.knowledge_ops.support_summary_scope",
  "resource_type": "policy",
  "name": "support_summary_scope",
  "fingerprint": {
    "content": "sha256:policy..."
  },
  "read_scope": ["data/support/", "assets/"],
  "write_scope": ["target/artifacts/support/"],
  "network": false,
  "tools": {
    "allow": ["read_artifact", "write_artifact"],
    "deny": ["write_source_files"]
  },
  "limits": {
    "timeout_seconds": 300,
    "max_cost_usd": 1.0,
    "max_tool_calls": 20
  }
}
```

## 13. Eval Resource

```json
{
  "unique_id": "eval.knowledge_ops.required_case_sections",
  "resource_type": "eval",
  "name": "required_case_sections",
  "eval_type": "deterministic",
  "fingerprint": {
    "config": "sha256:eval..."
  },
  "config": {
    "required_sections": ["Summary", "Signals", "Resolution"]
  },
  "grants_confidence": "structural"
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
  "lockfile": {
    "entry_digest": "sha256:lock-entry...",
    "source": "github.com/nyuta01/fbt/adapters/openai",
    "version": "adapters/openai/v0.1.0",
    "protocol_version": "0.1",
    "command": "fbt-runner-openai"
  },
  "fingerprint": {
    "identity": "sha256:runner..."
  }
}
```

When `fbt.lock.json` is valid and contains a matching runner entry, the runner
fingerprint includes that lock entry digest. Changing the lock entry makes
dependent transforms dirty with `runner identity changed`.

## 15. Graph Maps

```json
{
  "parent_map": {
    "transform.knowledge_ops.case_summaries": [
      "source.knowledge_ops.support.raw_tickets",
      "transform_asset.knowledge_ops.case_summary_prompt",
      "policy.knowledge_ops.support_summary_scope",
      "eval.knowledge_ops.required_case_sections",
      "runner.knowledge_ops.demo.llm"
    ]
  },
  "child_map": {
    "source.knowledge_ops.support.raw_tickets": [
      "transform.knowledge_ops.case_summaries"
    ]
  }
}
```

Transform assets, policies, evals, runners, sources, and upstream artifacts are
graph dependencies.

## 16. Files

```json
{
  "files": {
    "data/support/tickets/2026-05-28.jsonl": {
      "path": "data/support/tickets/2026-05-28.jsonl",
      "checksum": "sha256:...",
      "resource_ids": [
        "source.knowledge_ops.support.raw_tickets"
      ]
    },
    "assets/support_style_guide.md": {
      "path": "assets/support_style_guide.md",
      "checksum": "sha256:...",
      "resource_ids": [
        "transform_asset.knowledge_ops.support_style_guide"
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
- Output missing

When a previous manifest is available, file-backed source changes include an
inspection delta grouped by source resource:

```json
{
  "source_id": "source.knowledge_ops.support.raw_tickets",
  "name": "support.raw_tickets",
  "added": ["data/support/tickets/2026-05-29.jsonl"],
  "changed": ["data/support/tickets/2026-05-28.jsonl"],
  "removed": []
}
```

## 18. Manifest vs State vs Run Results

Manifest:

- Project graph
- Intended transform definitions
- Current metadata snapshot
- Dependency maps
- State comparison metadata

State:

- Current artifact pointers
- Artifact version index
- Latest run pointers
- Eval results
- Policy decisions
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

## 19. MVP Decisions And Post-MVP Boundaries

The MVP manifest contract is implementation-aligned:

- Current artifact-version snapshots use the fields defined by
  [State and Run Results Spec](state-and-run-results-spec.md) and the generated
  state schemas.
- Runner identity and available capabilities are manifest/planning metadata
  when they can be resolved before execution. Concrete runner events, usage,
  costs, errors, and output candidates remain runtime records in run results.
- Transform asset fingerprints are planning metadata because asset changes must
  make dependent transforms dirty. Rendered prompt bodies and runner-specific
  execution evidence remain runtime records.
- OpenLineage and OpenTelemetry are the standard exports covered by MVP.
  Additional formats such as PROV or in-toto are post-MVP integrations and must
  stay outside core until they have a concrete user workflow and conformance
  scenario.
