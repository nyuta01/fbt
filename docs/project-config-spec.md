# fbt Project Config Spec

Status: Draft  
Created: 2026-05-28  
Audience: authors of `fbt` project YAML files

## 1. Overview

An `fbt` project is defined by `fs_project.yml` and resource YAML files. Users declare filesystem artifacts, dependencies, runners, transform assets, policies, evals, and review requirements. `fbt` core parses these files into a manifest and delegates execution to external runners.

`fbt parse` diagnostics are intended for authoring. They include a stable
diagnostic code, file, line when fbt can locate the resource in YAML, resource
name, and an actionable hint for common fixes.

## 2. Naming Policy

Canonical field names use `snake_case`.

Recommended:

```yaml
source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["prompts", "assets"]
target_path: "target"
artifact_path: "target/artifacts"
```

During the draft period, dbt-style kebab-case aliases may be accepted for compatibility, but generated output should use `snake_case`.

`config_version` is not optional. See
[Schema and Versioning Spec](schema-and-versioning-spec.md) for compatibility
rules.

## 3. Standard Layout

```text
fs_project.yml
sources/
transforms/
prompts/
assets/
policies/
evals/
target/
.fbt/
```

`prompts/` is a conventional directory. In the manifest, prompts are `transform_asset` resources with `asset_type: prompt`.

## 4. fs_project.yml

Minimal example:

```yaml
name: knowledge_ops
config_version: 1
version: 0.1.0

source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["prompts", "assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]

target_path: "target"
artifact_path: "target/artifacts"
```

Fuller example:

```yaml
name: knowledge_ops
config_version: 1
version: 0.1.0

source_paths: ["sources"]
transform_paths: ["transforms"]
asset_paths: ["prompts", "assets"]
policy_paths: ["policies"]
eval_paths: ["evals"]

target_path: "target"
artifact_path: "target/artifacts"

state:
  backend: local
  path: .fbt/state

execution:
  mode: local
  max_workers: 4
  fail_fast: false

defaults:
  review:
    required: false
  cache:
    mode: reuse_if_same_inputs
  confidence:
    minimum: structural

runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
    args: ["--profile", "fbt"]
    cwd: .
    env:
      - OPENAI_API_KEY
    config:
      provider: openai
      default_model: gpt-5

selectors:
  - name: support_daily
    definition:
      method: tag
      value: support
```

Provider and agent integrations are optional external runner packages, not fbt
core dependencies. See [Runner Adapter Packaging](runner-adapters.md) for
package naming, plugin manifests, credential handling, and conformance checks.

Top-level fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Project name, used in resource IDs |
| `config_version` | yes | Project config semantics version; MVP requires `1` |
| `version` | no | Project version |
| `source_paths` | no | Directories containing source YAML |
| `transform_paths` | no | Directories containing transform YAML |
| `asset_paths` | no | Directories containing transform assets |
| `policy_paths` | no | Directories containing policy YAML |
| `eval_paths` | no | Directories containing eval YAML |
| `target_path` | no | Generated files root |
| `artifact_path` | no | Official artifact path |
| `state` | no | State backend configuration |
| `execution` | no | Local execution settings |
| `defaults` | no | Resource defaults |
| `runners` | no | Runner references |
| `selectors` | no | Named selections |
| `vars` | no | Project variables |

## 5. Sources

Sources point to input files or directories outside the project’s managed outputs.

```yaml
sources:
  - name: support
    description: Primary customer support documents
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support", "raw"]
        tests:
          - exists
          - min_file_count: 1

      - name: raw_chats
        type: markdown_directory
        path: data/support/chats/
        tags: ["support", "raw"]
        tests:
          - exists
```

Source artifact fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Source artifact name |
| `type` | yes | Input artifact type |
| `path` | yes | File path, glob, directory, or remote URI |
| `description` | no | Docs text |
| `tags` | no | Selection and docs metadata |
| `tests` | no | Source checks |
| `meta` | no | Arbitrary metadata |

Source ID format:

```text
source.<project>.<source_name>.<artifact_name>
```

## 6. Artifacts

Artifacts are logical outputs managed by the project. They are often created implicitly from transform outputs, but may also be declared explicitly.

```yaml
artifacts:
  - name: case_summaries
    type: markdown_directory
    path: target/artifacts/support/case_summaries/
    contract:
      required_sections:
        - Summary
        - Customer Impact
        - Cause
        - Response
        - Next Improvement
      citations:
        required: true
    owner: support_ops
    tags: ["support", "knowledge"]
```

## 7. Transform Assets

Transform assets affect transform behavior. Prompt is an asset type, not a top-level resource type.

```yaml
assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md
    variables:
      - input_documents
      - target_format

  - name: support_style_guide
    type: style_guide
    path: assets/support_style_guide.md

  - name: postmortem_rubric
    type: rubric
    path: assets/postmortem_rubric.md
```

Asset types include:

- `prompt`
- `template`
- `script`
- `style_guide`
- `rubric`
- `examples`
- `glossary`
- `schema`
- `config`
- `tool_manifest`

## 8. Transforms

A transform is a contract for producing output artifacts from inputs. It is not the implementation itself.

```yaml
transforms:
  - name: case_summaries
    type: llm
    runner: openai.responses
    model:
      provider: openai
      name: gpt-5
      parameters:
        temperature: 0.2
    inputs:
      - source: support.raw_tickets
      - source: support.raw_chats
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
      - ref: support_style_guide
    policy: support_summary_scope
    evals:
      - required_sections
      - citation_coverage
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
    cache:
      mode: require_approval_for_reuse
    tags: ["support", "llm"]
```

Transform fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Transform name |
| `type` | yes | `command`, `extract`, `template`, `llm`, `agent`, `compose`, `review` |
| `runner` | yes | Runner reference |
| `inputs` | yes | `source` or `ref` dependencies |
| `outputs` | yes | Output artifact declarations |
| `assets` | no | Transform assets |
| `model` | no | LLM / agent model config |
| `tools` | no | Agent tools |
| `policy` | no | Policy reference |
| `evals` | no | Eval references |
| `review` | no | Review requirement |
| `cache` | no | Reuse policy |
| `contract` | no | Output contract |
| `tags` | no | Selection and docs metadata |
| `meta` | no | Arbitrary metadata |

### Inputs

External input:

```yaml
inputs:
  - source: support.raw_tickets
```

Project artifact input:

```yaml
inputs:
  - ref: case_summaries
```

`ref` can require confidence, eval results, or review state:

```yaml
inputs:
  - ref: case_summaries
    require:
      confidence: reviewed
      evals:
        citation_coverage: pass
      review:
        status: approved
```

### Outputs

```yaml
outputs:
  - name: weekly_support_report
    type: markdown
    path: target/artifacts/support/weekly_report.md
    contract:
      required_sections:
        - Executive Summary
        - Top Issues
        - Proposed Improvements
```

### LLM Transform

```yaml
transforms:
  - name: faq_candidates
    type: llm
    runner: openai.responses
    model:
      provider: openai
      name: gpt-5
      parameters:
        temperature: 0.1
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
    outputs:
      - name: faq_candidates
        type: markdown
        path: target/artifacts/support/faq_candidates.md
    assets:
      - type: prompt
        path: prompts/faq_candidates.md
      - ref: support_style_guide
    evals:
      - no_unsupported_claims
      - citation_coverage
```

### Agent Transform

```yaml
transforms:
  - name: weekly_support_insights
    type: agent
    runner: langgraph.agent
    agent: support_insight_writer
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
    tools:
      - read_artifact
      - search_project
      - write_markdown
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    policy: support_agent_scope
    evals:
      - required_sections
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
```

## 9. Policies

Policies define security, tool, network, cost, write-scope, and review constraints.

```yaml
policies:
  - name: support_agent_scope
    read:
      - data/support/
      - target/artifacts/support/
    write:
      - .fbt/work/
      - target/artifacts/support/
    network: true
    tools:
      allow:
        - read_artifact
        - search_project
        - write_markdown
      deny:
        - write_source_files
        - shell
    limits:
      timeout_seconds: 600
      max_cost_usd: 3.00
      max_tool_calls: 40
      max_output_bytes: 10485760
    review:
      required: true
      group: support_leads
```

## 10. Evals

Evals judge artifact quality.

```yaml
evals:
  - name: required_sections
    type: deterministic
    config:
      sections:
        - Summary
        - Customer Impact
        - Cause
        - Response
        - Next Improvement
    grants_confidence: structural

  - name: citation_coverage
    type: semantic
    runner: openai.responses
    config:
      min: 0.9
      require_source_links: true
    grants_confidence: semantic

  - name: no_unsupported_claims
    type: llm_judge
    runner: openai.responses
    config:
      rubric: assets/no_unsupported_claims_rubric.md
      threshold: pass
    grants_confidence: semantic
```

Eval types:

- `deterministic`
- `semantic`
- `llm_judge`
- `human_review`

## 11. Runners

```yaml
runners:
  - name: command.local
    type: command
    protocol: stdio_jsonrpc
    command: fbt-command-runner

  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
    args: ["--profile", "fbt"]
    env:
      - OPENAI_API_KEY
    config:
      provider: openai
      default_model: gpt-5

  - name: langgraph.agent
    type: agent
    protocol: stdio_jsonrpc
    command: fbt-langgraph-runner
    cwd: .
```

Runner resolution is defined in
[Runner Discovery Spec](runner-discovery-spec.md). Project `runners` entries
with explicit `command` take precedence over plugin manifests and `PATH`
conventions.

Runner fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Logical runner name referenced by transforms |
| `type` | yes | Runner class such as `command`, `llm`, `agent`, `eval`, or `converter` |
| `protocol` | yes | `stdio_jsonrpc` for MVP |
| `command` | no | Executable name or path; required unless discovery finds a plugin or `PATH` command |
| `args` | no | Static process arguments passed after the runner command |
| `cwd` | no | Working directory for the runner process; relative paths resolve from the project directory |
| `env` | no | Environment variable names passed through to the runner; values are never written to manifest or diagnostics |
| `config` | no | Runner-specific configuration passed to protocol runners and included in fingerprints |
| `capabilities` | no | Static expected capabilities checked against `initialize` in runner validation |

Runner fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Logical runner name used by transforms |
| `type` | yes | Runner category, such as `command`, `llm`, `agent`, `eval`, or `converter` |
| `protocol` | yes | `stdio_jsonrpc` for MVP |
| `command` | no | Executable name or path; required unless a plugin manifest provides it |
| `env` | no | Environment variable names that core may pass through |
| `config` | no | Runner-specific settings included in effective fingerprints |
| `capabilities` | no | Expected capabilities, verified during runner initialization |

## 12. Selectors

```yaml
selectors:
  - name: support_daily
    definition:
      union:
        - method: tag
          value: support
        - method: path
          value: transforms/support/

  - name: needs_review
    definition:
      method: state
      value: pending_review
```

Initial selector methods:

- `name`
- `tag`
- `path`
- `resource_type`
- `state`
- `parent`
- `child`

Use with:

```sh
fbt build --select selector:support_daily
```

## 13. Validation Rules

`fbt parse` validates at least:

- Unique resource names
- Resolvable `source()` and `ref()` references
- Output paths under the configured artifact path
- Resolvable runner, policy, and eval references
- No duplicate declared outputs
- Existing transform asset paths
- Source path existence according to source policy
- Write-scope policy for agent transforms
- Review handling for review-required transforms
- Supported `config_version`
- Supported artifact type aliases
- Resolvable runner command or plugin manifest
- Policy path scopes that do not escape the project unless explicitly allowed

Validation failures exit with code `2`.
