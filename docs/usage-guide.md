# fbt Usage Guide

Status: Draft  
Created: 2026-05-28  
Audience: users defining and operating an `fbt` filesystem transformation project

## 1. Assumptions

This guide describes the target `fbt` user experience. `fbt` core is a control plane, not a transform engine. LLM calls, agent runtimes, document converters, OCR, and SaaS connectors are provided by external runners or plugins.

Base usage assumes:

- No daemon
- No scheduler
- No metadata database
- No cloud account
- Local filesystem state
- `fbt build` as the primary command

## 2. Initialize a Project

```sh
fbt init knowledge_ops --template support
cd knowledge_ops
```

Generated layout:

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

Minimal `fs_project.yml`:

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
```

## 3. Add Primary Documents

For a customer support workflow, primary documents may include tickets, chat exports, and call notes.

```text
data/support/tickets/2026-05-28.jsonl
data/support/chats/thread-123.md
data/support/call_notes/call-456.docx
```

`sources/support.yml`:

```yaml
sources:
  - name: support
    artifacts:
      - name: raw_tickets
        type: jsonl_directory
        path: data/support/tickets/*.jsonl
        tests:
          - exists
          - min_file_count: 1

      - name: raw_chats
        type: markdown_directory
        path: data/support/chats/
        tests:
          - exists

      - name: raw_call_notes
        type: docx_directory
        path: data/support/call_notes/*.docx
        tests:
          - exists
```

Sources are read-only. Transforms must not mutate source files directly.

## 4. Add Transform Assets

LLM and agent transforms depend on prompts, style guides, rubrics, examples, schemas, and scripts. These are `transform_asset` resources.

`prompts/case_summary.md`:

```markdown
# Role

You organize customer support knowledge.

# Task

Create reusable case summaries from tickets, chats, and call notes.

# Output Requirements

- Output Markdown
- Include citations to primary documents
- Separate facts from assumptions
- Include improvement actions for future cases
```

`assets/support_style_guide.md`:

```markdown
# Support Knowledge Style Guide

- Redact customer names and personal data unless required
- Separate cause, impact, response, and next improvement
- Avoid unsupported claims
- Prefer language that can be reused in FAQ content
```

`assets/assets.yml`:

```yaml
assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md

  - name: support_style_guide
    type: style_guide
    path: assets/support_style_guide.md
```

## 5. Define Policies and Evals

`policies/support.yml`:

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
    review:
      required: true
      group: support_leads
```

`evals/support.yml`:

```yaml
evals:
  - name: required_case_sections
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
    grants_confidence: semantic

  - name: no_unsupported_claims
    type: llm_judge
    runner: openai.responses
    config:
      rubric: assets/no_unsupported_claims_rubric.md
      threshold: pass
    grants_confidence: semantic
```

## 6. Define Transforms

`transforms/support/case_summaries.yml`:

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
      - source: support.raw_call_notes
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
      - ref: support_style_guide
    policy: support_agent_scope
    evals:
      - required_case_sections
      - citation_coverage
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
    cache:
      mode: require_approval_for_reuse
    tags: ["support", "knowledge"]
```

`transforms/support/weekly_insights.yml`:

```yaml
transforms:
  - name: weekly_support_insights
    type: agent
    runner: langgraph.agent
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
          review:
            status: approved
    tools:
      - read_artifact
      - search_project
      - write_markdown
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    assets:
      - ref: support_style_guide
    policy: support_agent_scope
    evals:
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
    tags: ["support", "weekly"]
```

The downstream transform requires the current `case_summaries` artifact version to be reviewed and approved.

## 7. Parse

```sh
fbt parse
```

Expected output:

```text
Parsed 13 resources
Manifest written to .fbt/state/manifest.json
```

Parse errors exit with code `2`.

## 8. Plan

```sh
fbt plan --select tag:support
```

Example output:

```text
Plan: 1 selected, 1 blocked

run transform.knowledge_ops.case_summaries
  reason: output missing
  runner: openai.responses
  review: required, group support_leads

blocked transform.knowledge_ops.weekly_support_insights
  reason: requires artifact.knowledge_ops.case_summaries confidence reviewed
```

`fbt plan` explains both what will run and why something is blocked.

## 9. Build

```sh
fbt build --select case_summaries
```

Example output:

```text
Running transform.knowledge_ops.case_summaries
  runner: openai.responses
  model: gpt-5
  eval: required_case_sections pass
  eval: citation_coverage pass 0.94
  eval: no_unsupported_claims pass

Created artifact_version.knowledge_ops.case_summaries.sha256_abcd
Status: pending review
Output: target/artifacts/support/case_summaries/
```

State files updated:

```text
.fbt/state/manifest.json
.fbt/state/state.json
.fbt/state/run_results.jsonl
.fbt/state/artifact_versions.json
.fbt/state/evaluation_results.json
.fbt/state/policy_decisions.json
```

## 10. Review

```sh
fbt review status case_summaries
```

Example output:

```text
artifact.knowledge_ops.case_summaries
  current: artifact_version.knowledge_ops.case_summaries.sha256_abcd
  status: pending
  group: support_leads
```

Approve the current version:

```sh
fbt review approve case_summaries --comment "Citations and customer impact reviewed"
```

Approval is bound to the current `artifact_version`, not just the logical path.

## 11. Build Downstream Artifacts

```sh
fbt build --select weekly_support_insights
```

Because `case_summaries` is approved, downstream transforms that require reviewed inputs can now run.

## 12. Inspect Diffs

After new source files arrive:

```sh
fbt plan --select tag:support
fbt diff case_summaries --against last-approved
```

This makes AI-generated document changes reviewable.

## 13. Generate Docs

```sh
fbt docs generate
```

Generated docs go to:

```text
target/docs/
```

Docs show:

- Source-to-artifact lineage
- Transform asset, policy, eval, runner, and model dependencies
- Token and cost summaries
- Artifact versions
- Eval results
- Review state
- Downstream block reasons

## 14. Day-2 Operation

The operating loop:

```text
1. Add primary documents
2. Run fbt plan
3. Run fbt build
4. Inspect diffs
5. Run evals
6. Approve or reject artifact versions
7. Build downstream artifacts
8. Use generated knowledge artifacts in the next workflow
```

The purpose is to turn messy primary documents into reusable, reviewed knowledge artifacts that continuously improve operational work.

## 15. What fbt Does Not Do

`fbt` core does not implement:

- Ticket system sync
- Slack or email connectors
- Word / PDF / Excel parsing
- LLM provider APIs
- Agent runtimes
- Domain-specific business judgment

These belong in runners, connectors, plugins, or existing business systems.
