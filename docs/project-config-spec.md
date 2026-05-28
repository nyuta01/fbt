# fbt Project Config Spec

Status: MVP-ready
Created: 2026-05-28
Audience: authors of `fbt` project YAML files

## 1. Overview

An `fbt` project is defined by `fs_project.yml` and resource YAML files. Users
declare filesystem sources, output artifacts, transform contracts, runners,
transform assets, policies, and evals. `fbt` core parses these files into a
manifest and delegates execution to external runners.

Project diagnostics include a stable diagnostic code, file, line when fbt can
locate the resource in YAML, resource name, and an actionable hint for common
fixes. `fbt doctor`, `fbt plan`, and `fbt build` all parse the project before
doing their main work.

## 2. Standard Layout

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

`prompts/` is a conventional directory. In the manifest, prompts are
`transform_asset` resources with `asset_type: prompt`.

## 3. fs_project.yml

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

state:
  backend: local
  path: .fbt/state

execution:
  mode: local
  max_workers: 4
  fail_fast: false

defaults:
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
```

Provider and agent integrations are optional external runner packages, not fbt
core dependencies. See [Runner Adapter Packaging](runner-adapters.md).

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

## 4. Sources

Sources point to input files or directories outside the project's managed
outputs.

```yaml
sources:
  - name: support
    description: Primary customer support documents
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tags: ["support", "raw"]

      - name: raw_chats
        type: markdown_directory
        path: data/support/chats/
        tags: ["support", "raw"]
```

For local file, directory, and glob paths, fbt fingerprints the resolved file
set and file contents. Adding a new file under a declared glob or directory
source changes the source fingerprint and makes dependent transforms dirty.

## 5. Artifacts

Artifacts are logical outputs managed by the project. They are often created
implicitly from transform outputs, but may also be declared explicitly.

```yaml
artifacts:
  - name: case_summaries
    type: markdown_directory
    path: target/artifacts/support/case_summaries/
    contract:
      required_sections:
        - Summary
        - Customer Impact
    owner: support_ops
    tags: ["support", "knowledge"]
```

## 6. Transform Assets

```yaml
assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md

  - name: support_style_guide
    type: style_guide
    path: assets/support_style_guide.md
```

Asset types include `prompt`, `template`, `script`, `style_guide`, `rubric`,
`examples`, `glossary`, `schema`, `config`, and `tool_manifest`.

## 7. Transforms

A transform is a contract for producing output artifacts from inputs. It is not
the implementation itself.

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
    tags: ["support", "llm"]
```

Transform fields:

| Field | Required | Meaning |
|---|---:|---|
| `name` | yes | Transform name |
| `type` | yes | `command`, `extract`, `template`, `llm`, `agent`, or `compose` |
| `runner` | yes | Runner reference |
| `inputs` | yes | `source` or `ref` dependencies |
| `outputs` | yes | Output artifact declarations |
| `assets` | no | Transform assets |
| `model` | no | LLM / agent model config |
| `tools` | no | Agent tools |
| `policy` | no | Policy reference |
| `evals` | no | Eval references |
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
    require:
      confidence: structural
      evals:
        required_sections: pass
```

`ref` can require confidence and eval results. Human approval state is not part
of the fbt project config.

## 8. Policies

Policies define security, tool, network, cost, and write-scope constraints.

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
      deny:
        - write_source_files
        - shell
    limits:
      timeout_seconds: 600
      max_cost_usd: 3.00
      max_tool_calls: 40
      max_output_bytes: 10485760
```

## 9. Evals

```yaml
evals:
  - name: required_sections
    type: deterministic
    config:
      sections:
        - Summary
        - Customer Impact
    grants_confidence: structural

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

MVP core executes deterministic evals. Other eval types are recorded as skipped
until delegated eval runners are implemented.

## 10. Runners

```yaml
runners:
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
```

Runner resolution is defined in
[Runner Discovery Spec](runner-discovery-spec.md). Project `runners` entries
with explicit `command` take precedence over plugin manifests and `PATH`
conventions.

## 11. Selectors

```yaml
selectors:
  - name: support_daily
    definition:
      union:
        - method: tag
          value: support
        - method: path
          value: transforms/support/
```

Initial selector methods:

- `name`
- `tag`
- `path`
- `resource_type`
- `state`
- `parent`
- `child`

## 12. Validation Rules

Project parsing validates at least:

- Unique resource names
- Resolvable `source` and `ref` references
- Output paths under the configured artifact path
- Resolvable runner, policy, and eval references
- No duplicate declared outputs
- Existing transform asset paths
- Source path existence according to source policy
- Write-scope policy for agent transforms
- Supported `config_version`
- Supported artifact type aliases
- Resolvable runner command or plugin manifest
- Policy path scopes that do not escape the project unless explicitly allowed

Validation failures exit with code `2`.
