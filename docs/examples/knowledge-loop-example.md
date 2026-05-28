# fbt Knowledge Loop Example

Status: MVP-ready
Created: 2026-05-28
Updated: 2026-05-29
Audience: users applying `fbt` to customer support, incident response, and agile management workflows

## 1. Concept

A core `fbt` use case is turning primary operational files into reusable
knowledge artifacts. The goal is a feedback loop for knowledge management and
continuous improvement, not a domain-specific support product.

`fbt` does not encode domain knowledge in core. Domain-specific behavior lives
in prompts, style guides, rubrics, evals, policies, and runners.

## 2. Runnable MVP

From a source checkout:

```sh
fbt init knowledge_ops --template support
fbt parse --project-dir knowledge_ops
fbt doctor --project-dir knowledge_ops
fbt plan --project-dir knowledge_ops --select tag:support
fbt build --project-dir knowledge_ops --select case_summaries
fbt build --project-dir knowledge_ops --select weekly_support_insights
fbt docs generate --project-dir knowledge_ops
mkdir -p knowledge_ops/target/lineage knowledge_ops/target/telemetry
fbt export openlineage --project-dir knowledge_ops --output knowledge_ops/target/lineage/openlineage.ndjson
fbt export otel --project-dir knowledge_ops --output knowledge_ops/target/telemetry/otel.json
```

The runnable graph is:

```text
support ticket source files
  -> case summaries
  -> weekly support insights
```

Use `fbt artifact explain case_summaries` and
`fbt artifact explain weekly_support_insights` to inspect what produced each
artifact. Use the OpenLineage export with Marquez or OpenMetadata, and the OTel
export with Jaeger, Tempo, or Grafana.

## 3. Directory Tree

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
  data/
    support/
      tickets/
      chats/
      call_notes/
  transforms/
    support/
      case_summaries.yml
      faq_candidates.yml
      weekly_insights.yml
      runbook_update_proposals.yml
  prompts/
  assets/
  policies/
  evals/
  target/artifacts/
  .fbt/state/
```

## 4. Transform Shape

The first transform reads source files and produces case summaries:

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
    tags: ["support", "knowledge"]
```

The downstream transform consumes the current case-summary artifact:

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
          confidence: structural
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

## 5. Daily Workflow

```sh
fbt parse
fbt doctor
fbt plan --select tag:support
fbt build --select case_summaries
fbt build --select weekly_support_insights
fbt artifact explain weekly_support_insights
fbt docs generate
```

Important behavior:

- New tickets, chats, or call notes make `case_summaries` dirty.
- `weekly_support_insights` blocks until `case_summaries` exists with the
  required confidence.
- Prompt, style-guide, policy, eval, runner, or model changes make dependent
  transforms dirty.
- Failed evals prevent confidence grants.
- Artifact versions are immutable, so generated outputs remain inspectable over
  time.

## 6. Variations

Incident response:

```text
incident logs + response notes + postmortems
  -> incident summary
  -> runbook update proposal
```

Agile management:

```text
issues + sprint notes + PR summaries
  -> sprint summary
  -> risk list
  -> backlog grooming proposal
```

## 7. Why fbt Fits

Operational primary documents vary in format and quality. Reusable knowledge
artifacts need consistent structure, evidence, version history, eval state, and
lineage.

`fbt` provides:

- Logical source definitions for messy files
- LLM and agent transforms on a graph
- Dependency tracking for prompts, style guides, policies, and evals
- Immutable artifact versions
- Deterministic evals that affect downstream confidence requirements
- Docs and standard lineage exports for continuous improvement

Human approval and publishing stay outside fbt in Git, PR, CI, release, or
knowledge-base workflows.
