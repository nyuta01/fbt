# AGENTS.md

This repository is built AI-first. Keep this file compact; it is the routing
layer into repository-local source-of-truth docs, not a full manual.

## Mission

`fbt` is a file build tool: a lightweight local-first control plane for
filesystem artifact transformations, especially LLM and agent-generated
knowledge artifacts. Core does not implement document conversion, OCR, LLM
providers, or agent runtimes; those belong to external runners.

## Start Here

Read these in order before changing code:

1. `README.md` - repository entry point and documentation map.
2. `docs/design-doc.md` - product principles, architecture, roadmap, decisions.
3. `docs/spec.md` - overall semantics and MVP acceptance criteria.
4. `docs/project-config-spec.md` - user-facing YAML contract.
5. `docs/runner-protocol-spec.md` - core/runner boundary.
6. `docs/methodology/harness-engineering.md` - AI-first workflow.
7. `docs/methodology/self-pdca-loop.md` - task loop.
8. `docs/methodology/permanent-fix-protocol.md` - repeated failure handling.
9. `docs/exec-plans/feature-list.json` - structured task state.
10. `AGENT_PROGRESS.md` - latest handoff state.

## Working Rules

- Work one bounded task at a time.
- Prefer boring, inspectable infrastructure that agents can reason about.
- Do not mark work done from code inspection alone; run the relevant checks.
- If behavior changes, update the matching spec, design doc, or plan.
- If an agent failure repeats, promote the fix into docs, scripts, or a
  deterministic check.
- Keep `fbt` core lightweight: no daemon, scheduler, metadata DB, cloud account,
  or heavyweight runner dependency in the base tool.
- Keep runner/plugin behavior outside core unless a spec explicitly says
  otherwise.

## Verification

Use the harness entrypoint:

```bash
make agent-init
```

Before calling work done, run:

```bash
make verify
```

`make verify` is the single gate for harness checks, docs checks, Go formatting,
Go tests, and the CLI smoke.

## Completion Rule

Before ending a task:

1. Update `docs/exec-plans/feature-list.json` when task state changes.
2. Update the active plan with Observation, Decision, Permanent Fix, and Next
   Check.
3. Update `docs/QUALITY_SCORE.md` when quality or risk changed.
4. Update `docs/agent-failures.md` when the task fixes or reveals a repeated
   agent failure.
5. Run `make verify`.
6. Leave `AGENT_PROGRESS.md` with restartable next steps if work is incomplete.

