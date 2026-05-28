# fbt Knowledge Loop Example

Status: MVP-ready  
Created: 2026-05-28  
Audience: users applying `fbt` to customer support, incident response, and agile management workflows

## 1. Concept

A core `fbt` use case is turning primary operational documents into reusable, reviewed knowledge artifacts. The goal is a feedback loop for knowledge management and continuous improvement, not a domain-specific support or incident-response product.

A runnable local version of the support flow can be created with
`fbt init --template support`; a committed copy also exists under
`examples/knowledge_ops`. The runnable MVP uses bundled deterministic demo
protocol runners and does not require provider accounts.

Primary documents:

- Tickets
- Chats
- Call notes
- Incident logs
- Investigation notes
- Meeting notes
- Sprint notes
- Issue or PR exports
- Postmortem drafts

Generated artifacts:

- Case summaries
- FAQ candidates
- Runbook update proposals
- Weekly insight reports
- Postmortems
- Action item lists
- Backlog grooming notes
- Retrospective summaries

`fbt` does not encode domain knowledge in core. Domain-specific behavior lives in prompts, style guides, rubrics, evals, policies, and runners.

## 2. Runnable MVP: Customer Support Knowledge Loop

From a source checkout:

```sh
fbt init knowledge_ops --template support
fbt parse --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt review approve case_summaries --project-dir knowledge_ops --comment "Reviewed locally"
fbt build --project-dir knowledge_ops --select weekly_support_insights
fbt docs generate --project-dir knowledge_ops
fbt export openlineage --project-dir knowledge_ops --output knowledge_ops/target/lineage/openlineage.ndjson
fbt export otel --project-dir knowledge_ops --output knowledge_ops/target/telemetry/otel.json
```

The runnable graph is:

```text
support ticket JSONL
  -> case summaries
  -> reviewed approval
  -> weekly support insights
```

Use the OpenLineage export with Marquez to inspect artifact/job/dataset lineage.
Use the OTel export with Jaeger, Tempo, or Grafana to inspect execution traces.
See [Standard Visualization Guide](../standard-visualization-guide.md).

## 3. Extended Pattern: Customer Support Knowledge Loop

Goal:

```text
support ticket / chat / call note
  -> case summary
  -> FAQ candidates
  -> weekly support insights
  -> runbook update proposal
```

Feedback loop:

```text
new support interactions
  -> fbt build
  -> eval / review
  -> approved knowledge artifacts
  -> support team uses artifacts
  -> new interactions become sources
```

## 4. Directory Tree

The committed runnable example is intentionally compact:

```text
knowledge_ops/
  fs_project.yml
  data/support/tickets/2026-05-28.jsonl
  sources/support.yml
  transforms/support/case_summaries.yml
  transforms/support/weekly_insights.yml
  assets/support.yml
  assets/support_style_guide.md
  policies/support.yml
  evals/support.yml
  bin/fbt-demo-llm-runner
  bin/fbt-demo-agent-runner
```

Larger support projects can extend the same layout:

```text
knowledge_ops/
  fs_project.yml
  data/
    support/
      tickets/
        2026-05-28.jsonl
      chats/
        thread-123.md
      call_notes/
        call-456.docx
  sources/
    support.yml
  transforms/
    support/
      case_summaries.yml
      faq_candidates.yml
      weekly_insights.yml
      runbook_update_proposals.yml
  prompts/
    case_summary.md
    faq_candidates.md
    weekly_insights.md
    runbook_update_proposal.md
  assets/
    assets.yml
    support_style_guide.md
    no_unsupported_claims_rubric.md
  policies/
    support.yml
  evals/
    support.yml
  target/
    artifacts/
      support/
        case_summaries/
        faq_candidates.md
        weekly_insights.md
        runbook_update_proposals.md
    docs/
  .fbt/
    state/
```

## 5. Project Config

`fs_project.yml`:

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
  max_workers: 2

defaults:
  cache:
    mode: reuse_if_same_inputs
  review:
    required: false

runners:
  - name: demo.llm
    type: llm
    protocol: stdio_jsonrpc
    command: bin/fbt-demo-llm-runner

  - name: demo.agent
    type: agent
    protocol: stdio_jsonrpc
    command: bin/fbt-demo-agent-runner

selectors:
  - name: support_daily
    definition:
      method: tag
      value: support
```

Provider-backed runners can replace the local commands when available:

```yaml
runners:
  - name: openai.responses
    type: llm
    protocol: stdio_jsonrpc
    command: fbt-openai-runner
    env:
      - OPENAI_API_KEY

  - name: langgraph.agent
    type: agent
    protocol: stdio_jsonrpc
    command: fbt-langgraph-runner
```

The bundled demo examples are deterministic external processes. They emit
usage, cost, provenance, and agent tool-call events for testing the
control-plane flow, but they do not call real model providers. Replace the
`demo.*` runners with external commands before using real provider or agent
execution.

## 6. Sources

`sources/support.yml`:

```yaml
sources:
  - name: support
    description: Customer support interactions exported from operational systems
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

      - name: raw_call_notes
        type: docx_directory
        path: data/support/call_notes/*.docx
        tags: ["support", "raw"]
        tests:
          - exists
```

## 7. Assets

`assets/assets.yml`:

```yaml
assets:
  - name: case_summary_prompt
    type: prompt
    path: prompts/case_summary.md

  - name: faq_candidates_prompt
    type: prompt
    path: prompts/faq_candidates.md

  - name: weekly_insights_prompt
    type: prompt
    path: prompts/weekly_insights.md

  - name: runbook_update_proposal_prompt
    type: prompt
    path: prompts/runbook_update_proposal.md

  - name: support_style_guide
    type: style_guide
    path: assets/support_style_guide.md

  - name: no_unsupported_claims_rubric
    type: rubric
    path: assets/no_unsupported_claims_rubric.md
```

Case summary prompt outline:

```markdown
# Task

Create case summaries from primary support documents.

# Output

Each case must include:

- Summary
- Customer impact
- Cause
- Response
- Next improvement
- Citations
```

## 8. Policies

`policies/support.yml`:

```yaml
policies:
  - name: support_summary_scope
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

## 9. Evals

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

  - name: required_insight_sections
    type: deterministic
    config:
      sections:
        - Executive Summary
        - Top Issues
        - Customer Impact
        - Improvement Actions
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

## 10. Transforms

`case_summaries` and `weekly_support_insights` are the compact runnable MVP
path. `faq_candidates` and `runbook_update_proposals` show how the same graph
can be extended after adding more prompts and eval runners.

### 10.1 Case Summaries

`transforms/support/case_summaries.yml`:

```yaml
transforms:
  - name: case_summaries
    type: llm
    runner: demo.llm
    model:
      provider: demo
      name: deterministic-demo-llm
    inputs:
      - source: support.raw_tickets
    outputs:
      - name: case_summaries
        type: markdown_directory
        path: target/artifacts/support/case_summaries/
    assets:
      - ref: case_summary_prompt
      - ref: support_style_guide
    policy: support_summary_scope
    evals:
      - required_case_sections
    review:
      required: true
      group: support_leads
    cache:
      mode: require_approval_for_reuse
    tags: ["support", "knowledge"]
```

### 10.2 FAQ Candidates

```yaml
transforms:
  - name: faq_candidates
    type: llm
    runner: demo.llm
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
          review:
            status: approved
    outputs:
      - name: faq_candidates
        type: markdown
        path: target/artifacts/support/faq_candidates.md
    assets:
      - ref: faq_candidates_prompt
      - ref: support_style_guide
    policy: support_summary_scope
    evals:
      - citation_coverage
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
    tags: ["support", "faq"]
```

### 10.3 Weekly Insights

```yaml
transforms:
  - name: weekly_support_insights
    type: agent
    runner: demo.agent
    model:
      provider: demo
      name: deterministic-demo-agent
    inputs:
      - ref: case_summaries
        require:
          confidence: reviewed
    tools:
      - read_artifact
      - write_artifact
    outputs:
      - name: weekly_support_insights
        type: markdown
        path: target/artifacts/support/weekly_insights.md
    assets:
      - ref: support_style_guide
    policy: support_summary_scope
    evals:
      - required_insight_sections
    tags: ["support", "weekly"]
```

### 10.4 Runbook Update Proposals

```yaml
transforms:
  - name: runbook_update_proposals
    type: agent
    runner: demo.agent
    inputs:
      - ref: weekly_support_insights
        require:
          confidence: reviewed
      - ref: faq_candidates
        require:
          confidence: reviewed
    tools:
      - read_artifact
      - search_project
      - write_markdown
    outputs:
      - name: runbook_update_proposals
        type: markdown
        path: target/artifacts/support/runbook_update_proposals.md
    assets:
      - ref: runbook_update_proposal_prompt
      - ref: support_style_guide
    policy: support_summary_scope
    evals:
      - no_unsupported_claims
    review:
      required: true
      group: support_leads
    tags: ["support", "runbook"]
```

## 11. Daily Workflow

```sh
fbt parse
fbt plan --select selector:support_daily
fbt build --select case_summaries
fbt review approve case_summaries --comment "Reviewed by support lead"
fbt build --select weekly_support_insights
fbt review status
fbt docs generate
```

Important behavior:

- New tickets, chats, or call notes make `case_summaries` dirty.
- Pending review of `case_summaries` can block `weekly_support_insights`.
- Prompt or style guide changes make downstream transforms dirty.
- Failed evals prevent confidence grants.
- Approval is bound to artifact versions, so regenerated outputs require review.

## 12. Incident Response Variation

Primary documents:

- Alert logs
- Incident channel export
- Investigation notes
- Timeline draft
- Metrics snapshot

Generated artifacts:

- Incident summary
- Timeline
- Postmortem draft
- Action items
- Runbook update proposal

Graph:

```text
source.incident.raw_logs
source.incident.chat_export
source.incident.investigation_notes
  -> transform.incident_timeline
  -> transform.postmortem_draft
  -> transform.action_items
  -> transform.runbook_update_proposals
```

## 13. Agile Development Management Variation

Primary documents:

- Issue export
- Sprint planning notes
- Standup notes
- PR summaries
- Retrospective notes

Generated artifacts:

- Sprint summary
- Risk list
- Decision log
- Retro action items
- Backlog grooming proposal

Graph:

```text
source.agile.issues
source.agile.sprint_notes
source.agile.pr_summaries
  -> transform.sprint_summary
  -> transform.retro_action_items
  -> transform.backlog_grooming_proposal
```

## 14. Why fbt Fits

Operational primary documents vary in format and quality. Reusable knowledge artifacts need consistent structure, evidence, review state, and lineage.

`fbt` provides:

- Logical source definitions for messy files
- LLM / Agent transforms on a graph
- Dependency tracking for prompts, style guides, policies, and evals
- Immutable artifact versions
- Evals and approvals that affect downstream use
- Docs and lineage for continuous improvement
